package users

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

type Service struct {
	repo   Repository
	logger *slog.Logger
}

func NewService(
	repo *Repository,
	logger *slog.Logger,
) *Service {
	return &Service{
		repo:   *repo,
		logger: logger.With("component", "users.service"),
	}
}

func (s *Service) GetMe(
	ctx context.Context,
	userID uuid.UUID,
) (Me, error) {
	return s.repo.GetMe(ctx, userID)
}

func (s *Service) UpdateProfile(
	ctx context.Context,
	userID uuid.UUID,
	req UpdateProfileInput,
) (Profile, error) {
	return s.repo.UpsertUserProfile(
		ctx,
		userID,
		req,
	)
}

func (s *Service) CreateAddress(
	ctx context.Context,
	userID uuid.UUID,
	req CreateAddressRequest) (Address, error) {
	return s.repo.CreateAddress(ctx, userID, req)
}

func (s *Service) UpdateAddress(
	ctx context.Context, userID,
	addressID uuid.UUID,
	req UpdateAddressRequest,
) (Address, error) {
	return s.repo.UpdateAddress(ctx, userID, addressID, req)
}

func (s *Service) ListAddresses(ctx context.Context, userID uuid.UUID) ([]Address, error) {
	return s.repo.ListAddresses(ctx, userID)
}

func (s *Service) DeleteAddress(ctx context.Context, userID, addressID uuid.UUID) error {
	return s.repo.DeleteAddress(ctx, userID, addressID)
}

func (s *Service) RegisterDeviceToken(ctx context.Context, userID uuid.UUID, req RegisterDeviceTokenRequest) (DeviceToken, error) {
	return s.repo.UpsertDeviceToken(ctx, userID, req.Token, req.Platform)
}

func (s *Service) ListDeviceTokens(ctx context.Context, userID uuid.UUID) ([]DeviceToken, error) {
	return s.repo.ListDeviceTokens(ctx, userID)
}

func (s *Service) DeleteDeviceToken(ctx context.Context, userID uuid.UUID, token string) error {
	return s.repo.DeleteDeviceToken(ctx, userID, token)
}
