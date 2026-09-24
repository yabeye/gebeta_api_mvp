package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// contextKey avoids collisions with other packages' context values.
type contextKey string

const userIDContextKey contextKey = "userID"

// supabaseClaims are the JWT claims Supabase Auth issues. Only the
// fields we actually use are declared; unknown claims are ignored.
type supabaseClaims struct {
	jwt.RegisteredClaims
	Email string `json:"email"`
}

// Auth returns chi middleware that verifies the Supabase-issued JWT on
// every request, using keys fetched from jwksURL. Verification is
// fully local (no network call to Supabase per request) — keyfunc
// caches the public keys and refreshes them automatically when a
// token references a key it hasn't seen yet (e.g. after key rotation).
func Auth(jwksURL string) (func(http.Handler) http.Handler, error) {
	k, err := keyfunc.NewDefaultCtx(context.Background(), []string{jwksURL})
	if err != nil {
		return nil, fmt.Errorf("fetching JWKS from %s: %w", jwksURL, err)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString, err := bearerToken(r)
			if err != nil {
				unauthorized(w, err.Error())
				return
			}

			var claims supabaseClaims
			token, err := jwt.ParseWithClaims(
				tokenString,
				&claims,
				k.Keyfunc,
				jwt.WithValidMethods([]string{"ES256", "RS256"}), // reject alg=none and unexpected algs
				jwt.WithExpirationRequired(),
			)
			if err != nil || !token.Valid {
				unauthorized(w, "invalid or expired token")
				return
			}

			userID, err := uuid.Parse(claims.Subject)
			if err != nil {
				unauthorized(w, "token subject is not a valid user id")
				return
			}

			ctx := context.WithValue(r.Context(), userIDContextKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}, nil
}

// bearerToken extracts the raw token from "Authorization: Bearer <token>".
func bearerToken(r *http.Request) (string, error) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return "", fmt.Errorf("missing authorization header")
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", fmt.Errorf("authorization header must be a bearer token")
	}
	return parts[1], nil
}

func unauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(fmt.Sprintf(`{"success":false,"error":{"message":%q}}`, msg)))
}

// UserIDFromContext returns the authenticated user's ID, set by Auth.
// Only call this on routes mounted behind the Auth middleware — it
// panics otherwise, which is deliberate: a handler silently proceeding
// with a zero-value UUID would be a much worse, quieter bug.
func UserIDFromContext(ctx context.Context) uuid.UUID {
	id, ok := ctx.Value(userIDContextKey).(uuid.UUID)
	if !ok {
		panic("middleware.UserIDFromContext: no user id in context — is this route behind Auth middleware?")
	}
	return id
}

var _ = time.Second // keep time import if unused elsewhere; remove if not needed
