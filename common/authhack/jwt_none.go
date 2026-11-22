package authhack

import (
	"errors"

	"github.com/golang-jwt/jwt/v4"
)

// Implements the none signing method.  This is required by the spec
// but you probably should never use it.
type AuthHackJWTSigningNone struct{}

func (m *AuthHackJWTSigningNone) Alg() string {
	return "None"
}

// Only allow 'none' alg type if UnsafeAllowNoneSignatureType is specified as the key
func (m *AuthHackJWTSigningNone) Verify(signingString, signature string, key interface{}) (err error) {
	if key == jwt.UnsafeAllowNoneSignatureType {
		return nil
	}
	return errors.New("'none' algorithm is not allowed unless key is UnsafeAllowNoneSignatureType")
}

// Only allow 'none' signing if UnsafeAllowNoneSignatureType is specified as the key
func (m *AuthHackJWTSigningNone) Sign(signingString string, key interface{}) (string, error) {
	return "", nil
}
