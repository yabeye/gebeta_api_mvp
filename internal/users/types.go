package users

import (
	"time"

	"github.com/google/uuid"
)

// Address is the API shape for a single customer address.
type Address struct {
	ID           uuid.UUID `json:"id"`
	Label        string    `json:"label"`
	Address      string    `json:"address"`
	Lat          float64   `json:"lat"`
	Lng          float64   `json:"lng"`
	DeliveryNote *string   `json:"delivery_note"`
	IsDefault    bool      `json:"is_default"`
}

// Me is the API response shape for GET /users/me — the authenticated
// user's core account fields, profile, role, and addresses, flattened
// for the client's convenience into a single response.
type Me struct {
	ID              uuid.UUID  `json:"id"`
	Email           string     `json:"email"`
	Phone           *string    `json:"phone"`
	Status          string     `json:"status"`
	FirstName       *string    `json:"first_name"`
	LastName        *string    `json:"last_name"`
	ProfileImageURL *string    `json:"profile_image_url"`
	BirthDate       *time.Time `json:"birth_date"`
	CreatedAt       time.Time  `json:"created_at"`

	// Role is nil if no role has been assigned yet — a valid, expected
	// state right after signup, not an error.
	Role *string `json:"role"`
}

// UpdateProfileRequest is the request body for PUT /users/me/profile.
// All fields are optional pointers except where a value is always
// expected — birth_date uses a string so the client can omit it
// entirely without needing to send a sentinel date.
type UpdateProfileRequest struct {
	FirstName       *string `json:"first_name" validate:"omitempty,min=1,max=100"`
	LastName        *string `json:"last_name" validate:"omitempty,min=1,max=100"`
	ProfileImageURL *string `json:"profile_image_url" validate:"omitempty,url"`
	BirthDate       *string `json:"birth_date" validate:"omitempty,datetime=2006-01-02"`
}

// CreateAddressRequest — all fields required; this is a brand new address.
type CreateAddressRequest struct {
	Label        string  `json:"label" validate:"required,min=1,max=100"`
	Address      string  `json:"address" validate:"required,min=1,max=500"`
	Lat          float64 `json:"lat" validate:"required,latitude"`
	Lng          float64 `json:"lng" validate:"required,longitude"`
	DeliveryNote *string `json:"delivery_note" validate:"omitempty,max=500"`
	IsDefault    bool    `json:"is_default"`
}

// UpdateAddressRequest — every field optional; only sent fields change.
// A nil field means "leave as-is," resolved by merging against the
// existing row in the repository, not by the database.
type UpdateAddressRequest struct {
	Label        *string  `json:"label" validate:"omitempty,min=1,max=100"`
	Address      *string  `json:"address" validate:"omitempty,min=1,max=500"`
	Lat          *float64 `json:"lat" validate:"omitempty,latitude"`
	Lng          *float64 `json:"lng" validate:"omitempty,longitude"`
	DeliveryNote *string  `json:"delivery_note" validate:"omitempty,max=500"`
	IsDefault    *bool    `json:"is_default" validate:"omitempty,eq=true"`
}

// UpdateProfileInput comment
type UpdateProfileInput struct {
	FirstName       *string
	LastName        *string
	ProfileImageURL *string
	BirthDate       *time.Time
}

// UpdateCustomerAddressInput comment
type UpdateCustomerAddressInput struct {
	ID           *string
	Label        *string
	Address      *string
	Lat          *float64
	Lng          *float64
	DeliveryNote *string
	IsDefault    *bool
}

// Profile is the API response shape for a profile create/update.
type Profile struct {
	UserID          uuid.UUID  `json:"user_id"`
	FirstName       *string    `json:"first_name"`
	LastName        *string    `json:"last_name"`
	ProfileImageURL *string    `json:"profile_image_url"`
	BirthDate       *time.Time `json:"birth_date"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// RegisterDeviceTokenRequest is the request body for
// POST /users/me/device-tokens, sent whenever a client obtains or
// refreshes its FCM token (app install, reinstall, token rotation).
type RegisterDeviceTokenRequest struct {
	Token    string `json:"token" validate:"required,min=10"`
	Platform string `json:"platform" validate:"required,oneof=ios android web"`
}

// DeviceToken is the API response shape for a registered device token.
type DeviceToken struct {
	ID       uuid.UUID `json:"id"`
	Token    string    `json:"token"`
	Platform string    `json:"platform"`
}
