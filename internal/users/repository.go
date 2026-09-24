package users

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yabeye/gebeta_api_mvp/common/apperrors"
	"github.com/yabeye/gebeta_api_mvp/common/constants"
	"github.com/yabeye/gebeta_api_mvp/internal/db/sqlc"
	"golang.org/x/sync/errgroup"
)

// Repository defines the persistence operations required by the users service.
type Repository struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

// NewRepository takes the raw pool (not pre-built *sqlc.Queries) since
// withTx needs the pool directly to start transactions; q is built
// internally from the same pool for all non-transactional queries.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{
		q:    sqlc.New(pool),
		pool: pool,
	}
}

// GetMe comment
func (r *Repository) GetMe(
	ctx context.Context,
	userID uuid.UUID,
) (Me, error) {
	var (
		userRow   sqlc.GetUserWithProfileRow
		roleRow   sqlc.UserRole
		roleFound bool
	)

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		row, err := r.q.GetUserWithProfile(gctx, userID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf(
					"user %s: %w",
					userID,
					apperrors.ErrNotFound,
				)
			}

			return fmt.Errorf(
				"querying user with profile: %w",
				err,
			)
		}

		userRow = row
		return nil
	})

	g.Go(func() error {
		row, err := r.q.GetUserRole(gctx, userID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}

			return fmt.Errorf(
				"querying user role: %w",
				err,
			)
		}

		roleRow = row
		roleFound = true

		return nil
	})

	if err := g.Wait(); err != nil {
		return Me{}, err
	}

	me := Me{
		ID:              userRow.ID,
		Email:           userRow.Email,
		Phone:           userRow.Phone,
		Status:          string(userRow.Status),
		FirstName:       userRow.FirstName,
		LastName:        userRow.LastName,
		ProfileImageURL: userRow.ProfileImageUrl,
		BirthDate:       &userRow.BirthDate.Time,
		CreatedAt:       userRow.CreatedAt.Time,
	}

	if roleFound {
		role := string(roleRow.Role)
		me.Role = &role
	}

	return me, nil
}

// UpsertUserProfile Comment
func (r *Repository) UpsertUserProfile(
	ctx context.Context,
	userID uuid.UUID,
	updateProfileInput UpdateProfileInput,
) (Profile, error) {
	row, err := r.q.UpsertUserProfile(
		ctx,
		sqlc.UpsertUserProfileParams{
			UserID:          userID,
			FirstName:       updateProfileInput.FirstName,
			LastName:        updateProfileInput.LastName,
			ProfileImageUrl: updateProfileInput.ProfileImageURL,
			BirthDate: pgtype.Date{
				Time: func() time.Time {
					if updateProfileInput.BirthDate != nil {
						return *updateProfileInput.BirthDate
					}
					return time.Time{}
				}(),
				Valid: updateProfileInput.BirthDate != nil,
			},
		},
	)
	if err != nil {
		return Profile{}, fmt.Errorf(
			"upserting profile: %w",
			err,
		)
	}

	return Profile{
		UserID:          row.UserID,
		FirstName:       row.FirstName,
		LastName:        row.LastName,
		ProfileImageURL: row.ProfileImageUrl,
		BirthDate:       &row.BirthDate.Time,
		UpdatedAt:       row.UpdatedAt.Time,
	}, nil
}

// CreateAddress comment
func (r *Repository) CreateAddress(ctx context.Context, userID uuid.UUID, req CreateAddressRequest) (Address, error) {
	var result sqlc.CreateCustomerAddressRow

	err := r.withTx(ctx, func(qtx *sqlc.Queries) error {
		count, err := qtx.CountAddressesByUser(ctx, userID)
		if err != nil {
			return fmt.Errorf("counting addresses: %w", err)
		}
		if count >= constants.MaxAddressesPerCustomer {
			return fmt.Errorf("%w: you can save up to only %d addresses", apperrors.ErrValidation, constants.MaxAddressesPerCustomer)
		}

		if req.IsDefault {
			if err := qtx.UnsetDefaultAddressForUser(ctx, userID); err != nil {
				return fmt.Errorf("unsetting previous default: %w", err)
			}
		}

		row, err := qtx.CreateCustomerAddress(ctx, sqlc.CreateCustomerAddressParams{
			UserID: userID, Label: req.Label, Address: req.Address,
			Lat: req.Lat, Lng: req.Lng,
			DeliveryNote: req.DeliveryNote, IsDefault: req.IsDefault,
		})
		if err != nil {
			return fmt.Errorf("creating address: %w", err)
		}
		result = row
		return nil
	})
	if err != nil {
		return Address{}, err
	}
	return toAddress(result.ID, result.Label, result.Address, result.Lat, result.Lng, result.DeliveryNote, result.IsDefault), nil
}

// UpdateAddress fetches the existing address (verifying ownership),
// merges any provided fields onto it, enforces the "at least one
// default" rule, then writes the merged result — all inside one
// transaction so the read-merge-write is atomic under concurrent edits.
func (r *Repository) UpdateAddress(ctx context.Context, userID, addressID uuid.UUID, req UpdateAddressRequest) (Address, error) {
	var result sqlc.UpdateCustomerAddressRow

	err := r.withTx(ctx, func(qtx *sqlc.Queries) error {
		existing, err := qtx.GetCustomerAddressByID(ctx, sqlc.GetCustomerAddressByIDParams{
			ID: addressID, UserID: userID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("address %s: %w", addressID, apperrors.ErrNotFound)
			}
			return fmt.Errorf("fetching address: %w", err)
		}

		merged := mergeAddress(existing, req)

		if merged.IsDefault && !existing.IsDefault {
			if err := qtx.UnsetDefaultAddressForUser(ctx, userID); err != nil {
				return fmt.Errorf("unsetting previous default: %w", err)
			}
		}

		row, err := qtx.UpdateCustomerAddress(ctx, sqlc.UpdateCustomerAddressParams{
			ID: addressID, UserID: userID,
			Label: merged.Label, Address: merged.Address,
			Lat: merged.Lat, Lng: merged.Lng,
			DeliveryNote: merged.DeliveryNote, IsDefault: merged.IsDefault,
		})
		if err != nil {
			return fmt.Errorf("updating address: %w", err)
		}
		result = row
		return nil
	})
	if err != nil {
		return Address{}, err
	}
	return toAddress(result.ID, result.Label, result.Address, result.Lat, result.Lng, result.DeliveryNote, result.IsDefault), nil
}

// mergedAddress is an internal working type for the fetch-merge step.
type mergedAddress struct {
	Label, Address string
	Lat, Lng       float64
	DeliveryNote   *string
	IsDefault      bool
}

// mergeAddress overlays only the fields present in req onto existing,
// leaving everything else unchanged — this is what makes the update
// genuinely partial.
func mergeAddress(existing sqlc.GetCustomerAddressByIDRow, req UpdateAddressRequest) mergedAddress {
	m := mergedAddress{
		Label: existing.Label, Address: existing.Address,
		Lat: existing.Lat, Lng: existing.Lng,
		DeliveryNote: existing.DeliveryNote, IsDefault: existing.IsDefault,
	}
	if req.Label != nil {
		m.Label = *req.Label
	}
	if req.Address != nil {
		m.Address = *req.Address
	}
	if req.Lat != nil {
		m.Lat = *req.Lat
	}
	if req.Lng != nil {
		m.Lng = *req.Lng
	}
	if req.DeliveryNote != nil {
		m.DeliveryNote = req.DeliveryNote
	}
	if req.IsDefault != nil {
		m.IsDefault = *req.IsDefault
	}
	return m
}

func toAddress(id uuid.UUID, label, address string, lat, lng float64, note *string, isDefault bool) Address {
	return Address{ID: id, Label: label, Address: address, Lat: lat, Lng: lng, DeliveryNote: note, IsDefault: isDefault}
}

func (r *Repository) ListAddresses(ctx context.Context, userID uuid.UUID) ([]Address, error) {
	rows, err := r.q.ListCustomerAddressesByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("listing addresses: %w", err)
	}
	addrs := make([]Address, 0, len(rows))
	for _, row := range rows {
		addrs = append(addrs, toAddress(row.ID, row.Label, row.Address, row.Lat, row.Lng, row.DeliveryNote, row.IsDefault))
	}
	return addrs, nil
}

func (r *Repository) DeleteAddress(ctx context.Context, userID, addressID uuid.UUID) error {
	return r.withTx(ctx, func(qtx *sqlc.Queries) error {
		existing, err := qtx.GetCustomerAddressByID(ctx, sqlc.GetCustomerAddressByIDParams{ID: addressID, UserID: userID})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("address %s: %w", addressID, apperrors.ErrNotFound)
			}
			return fmt.Errorf("fetching address: %w", err)
		}
		if existing.IsDefault {
			count, err := qtx.CountAddressesByUser(ctx, userID)
			if err != nil {
				return fmt.Errorf("counting addresses: %w", err)
			}
			if count > 1 {
				return fmt.Errorf("%w: reassign the default address before deleting it", apperrors.ErrValidation)
			}
		}
		if err := qtx.DeleteCustomerAddress(ctx, sqlc.DeleteCustomerAddressParams{ID: addressID, UserID: userID}); err != nil {
			return fmt.Errorf("deleting address: %w", err)
		}
		return nil
	})
}

func (r *Repository) withTx(ctx context.Context, fn func(qtx *sqlc.Queries) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := fn(r.q.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// UpsertDeviceToken registers or re-associates an FCM token with
// userID. Upserting on the unique token handles reinstalls and token
// rotation cleanly — if the token already exists (possibly under a
// different user, e.g. a shared/reset device), ownership transfers to
// the current caller.
func (r *Repository) UpsertDeviceToken(ctx context.Context, userID uuid.UUID, token, platform string) (DeviceToken, error) {
	row, err := r.q.UpsertFCMDeviceToken(ctx, sqlc.UpsertFCMDeviceTokenParams{
		UserID:   userID,
		Token:    token,
		Platform: sqlc.DevicePlatform(platform),
	})
	if err != nil {
		return DeviceToken{}, fmt.Errorf("upserting device token: %w", err)
	}

	return DeviceToken{
		ID:       row.ID,
		Token:    row.Token,
		Platform: string(row.Platform),
	}, nil
}

// ListDeviceTokens returns all FCM tokens registered for userID —
// typically more than one, since a user may be logged in on several
// devices at once.
func (r *Repository) ListDeviceTokens(ctx context.Context, userID uuid.UUID) ([]DeviceToken, error) {
	rows, err := r.q.ListDeviceTokensByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("listing device tokens: %w", err)
	}

	tokens := make([]DeviceToken, 0, len(rows))
	for _, row := range rows {
		tokens = append(tokens, DeviceToken{
			ID:       row.ID,
			Token:    row.Token,
			Platform: string(row.Platform),
		})
	}
	return tokens, nil
}

// DeleteDeviceToken removes a specific token for userID — typically
// called on logout, so a signed-out device stops receiving push
// notifications for that account.
func (r *Repository) DeleteDeviceToken(ctx context.Context, userID uuid.UUID, token string) error {
	if err := r.q.DeleteDeviceToken(ctx, sqlc.DeleteDeviceTokenParams{
		Token: token, UserID: userID,
	}); err != nil {
		return fmt.Errorf("deleting device token: %w", err)
	}
	return nil
}
