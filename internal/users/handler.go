package users

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/yabeye/gebeta_api_mvp/common/apperrors"
	"github.com/yabeye/gebeta_api_mvp/common/httpx"
	appmiddleware "github.com/yabeye/gebeta_api_mvp/internal/middleware"
)

// Handler Comment
type Handler struct {
	service *Service
	logger  *slog.Logger
}

// NewHandler Comment
func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger.With("component", "users.handler")}
}

// GetMe handles GET /api/v1/users/me. The user's identity comes from
// the verified JWT (set by the Auth middleware), never from a URL
// param or request body — a client can only ever fetch their own info
// through this route.
func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID := appmiddleware.UserIDFromContext(r.Context())

	me, err := h.service.GetMe(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, me)
}

// UpdateProfile handles PUT /api/v1/users/me/profile. Like GetMe, the
// user's identity comes only from the verified JWT — never from the
// request body — so a user can only ever update their own profile.
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := appmiddleware.UserIDFromContext(r.Context())

	var req UpdateProfileRequest
	if err := httpx.DecodeAndValidate(r, &req); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}

	var birthDate *time.Time

	if req.BirthDate != nil {
		parsed, err := time.Parse(
			"2006-01-02",
			*req.BirthDate,
		)
		if err != nil {
			httpx.WriteError(w, r, h.logger, err)
			return
		}

		birthDate = &parsed
	}

	profile, err := h.service.UpdateProfile(r.Context(), userID,
		UpdateProfileInput{
			FirstName:       req.FirstName,
			LastName:        req.LastName,
			ProfileImageURL: req.ProfileImageURL,
			BirthDate:       birthDate,
		})
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, profile)
}

// CreateAddress comment
func (h *Handler) CreateAddress(w http.ResponseWriter, r *http.Request) {
	userID := appmiddleware.UserIDFromContext(r.Context())
	var req CreateAddressRequest
	if err := httpx.DecodeAndValidate(r, &req); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	addr, err := h.service.CreateAddress(r.Context(), userID, req)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, addr)
}

// UpdateAddress comment
func (h *Handler) UpdateAddress(w http.ResponseWriter, r *http.Request) {
	userID := appmiddleware.UserIDFromContext(r.Context())
	addressID, err := httpx.URLParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}

	var req UpdateAddressRequest
	if err := httpx.DecodeAndValidate(r, &req); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}

	addr, err := h.service.UpdateAddress(r.Context(), userID, addressID, req)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, addr)
}

// ListAddresses handles GET /api/v1/users/me/addresses.
func (h *Handler) ListAddresses(w http.ResponseWriter, r *http.Request) {
	userID := appmiddleware.UserIDFromContext(r.Context())

	addrs, err := h.service.ListAddresses(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, addrs)
}

// DeleteAddress handles DELETE /api/v1/users/me/addresses/{id}.
func (h *Handler) DeleteAddress(w http.ResponseWriter, r *http.Request) {
	userID := appmiddleware.UserIDFromContext(r.Context())

	addressID, err := httpx.URLParamUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}

	if err := h.service.DeleteAddress(r.Context(), userID, addressID); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RegisterDeviceToken handles POST /api/v1/users/me/device-tokens.
func (h *Handler) RegisterDeviceToken(w http.ResponseWriter, r *http.Request) {
	userID := appmiddleware.UserIDFromContext(r.Context())

	var req RegisterDeviceTokenRequest
	if err := httpx.DecodeAndValidate(r, &req); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}

	token, err := h.service.RegisterDeviceToken(r.Context(), userID, req)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, token)
}

// ListDeviceTokens handles GET /api/v1/users/me/device-tokens.
func (h *Handler) ListDeviceTokens(w http.ResponseWriter, r *http.Request) {
	userID := appmiddleware.UserIDFromContext(r.Context())

	tokens, err := h.service.ListDeviceTokens(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, tokens)
}

// DeleteDeviceToken handles DELETE /api/v1/users/me/device-tokens.
// The token is passed as a query param (?token=...) rather than a
// path segment, since FCM tokens are long, URL-unfriendly strings —
// not a good fit for a REST path segment.
func (h *Handler) DeleteDeviceToken(w http.ResponseWriter, r *http.Request) {
	userID := appmiddleware.UserIDFromContext(r.Context())

	token := r.URL.Query().Get("token")
	if token == "" {
		httpx.WriteError(w, r, h.logger, fmt.Errorf("%w: missing token query parameter", apperrors.ErrValidation))
		return
	}

	if err := h.service.DeleteDeviceToken(r.Context(), userID, token); err != nil {
		httpx.WriteError(w, r, h.logger, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
