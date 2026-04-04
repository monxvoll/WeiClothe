package ports

import (
	"context"
)

// IdentityProvider defines the strict contract to work with WeiClothe.

// ctx: All methods require a context as their first
// parameter. This is used to manage request deadlines, cancellation signals,
// and timeouts across the system to prevent memory leaks if the external
// identity provider becomes unresponsive.
type IdentityProvider interface {

	// Creates a new user record in the external IAM system.
	// It takes the user's email and password, and returns the unique
	// user ID (uid) or an error if it fails.
	RegisterUser(ctx context.Context, username, email, password, firstName, lastName string) (uid string, err error)

	// Verifies the provided credentials against the IAM system.
	// On successful authentication,  returns a JWT access token or an error
	LoginUser(ctx context.Context, email, password string) (token string, err error)

	// Receives a JWT string, verifies its signature and
	// expiration status, and returns the user ID (uid).
	ValidateToken(ctx context.Context, token string) (uid string, err error)

	// Deletes a user in the IAM system.
	DeleteUser(ctx context.Context, uid string) (err error)
}
