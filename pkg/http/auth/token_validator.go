package auth

import (
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"go.opencensus.io/trace"
	"io"
	"net/http"
)

// tokenValidator  defines operations for JWT token validation
type TokenValidator interface {
	ValidateRequest(req *http.Request) error
}

// NewEmptyTokenValidator return instance of jwtTokenValidator
func NewEmptyTokenValidator() TokenValidator {
	return &voidTokenValidator{}
}

// voidTokenValidator handle JWT token validation
type voidTokenValidator struct {
}

// ValidateRequest check http request
func (voidTokenValidator) ValidateRequest(_ *http.Request) error {
	return nil
}

const (
	publicKeysURL  = "https://www.gstatic.com/iap/verify/public_key"
	algorithm      = "ES256"
	algorithmClaim = "alg"
	keyIDClaim     = "kid"
	issuerClaim    = "https://cloud.google.com/iap"
)

type publicKey []byte

// NewTokenValidator return instance of jwtTokenValidator
func NewTokenValidator(keyForToken string, audience string) (TokenValidator, error) {
	publicKeys, err := fetchPublicKeys()
	if err != nil {
		return nil, err
	}

	return &jwtTokenValidator{publicKeys, keyForToken, audience}, nil
}

// jwtTokenValidator validates Jwt tokens
type jwtTokenValidator struct {
	publicKeys     map[string]publicKey
	keyForToken    string
	appJWTAudience string
}

// ValidateRequest checks the validity of the claims in the request.
func (t *jwtTokenValidator) ValidateRequest(req *http.Request) error {
	_, span := trace.StartSpan(req.Context(), "jwtTokenValidator.ValidateRequest")
	defer span.End()

	token := req.Header.Get(t.keyForToken)
	if len(token) == 0 {
		return fmt.Errorf("jwtTokenValidator: token not found in request header %s", t.keyForToken)
	}

	claims := &tokenClaims{publicKeys: t.publicKeys, appJWTAudience: t.appJWTAudience}
	_, err := jwt.ParseWithClaims(token, claims, t.tokenKey)
	return err
}

// FetchPublicKeys downloads and decodes all public keys from Google.
func fetchPublicKeys() (map[string]publicKey, error) {
	r, err := http.Get(publicKeysURL)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	return decodePublicKeys(r.Body)
}

// DecodePublicKeys decodes all public keys from the given Reader.
func decodePublicKeys(r io.Reader) (map[string]publicKey, error) {
	var skeys map[string]string
	if err := json.NewDecoder(r).Decode(&skeys); err != nil {
		return nil, err
	}
	bkeys := make(map[string]publicKey)
	for k, v := range skeys {
		if len(v) != 0 {
			bkeys[k] = []byte(v)
		}
	}
	return bkeys, nil
}

func (t *jwtTokenValidator) tokenKey(token *jwt.Token) (interface{}, error) {
	if _, ok := t.tokenMethod(token); !ok {
		return nil, fmt.Errorf("invalid algorithm: %v", token.Header[algorithmClaim])
	}
	keyID, _ := token.Header[keyIDClaim].(string)
	key := token.Claims.(*tokenClaims).publicKeys[keyID]
	if len(key) == 0 {
		return nil, fmt.Errorf("no public key for %q", keyID)
	}
	parsedKey, err := jwt.ParseECPublicKeyFromPEM(key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse key: %v", err)
	}
	return parsedKey, nil
}

func (t *jwtTokenValidator) tokenMethod(token *jwt.Token) (jwt.SigningMethod, bool) {
	if token.Header[algorithmClaim] != algorithm {
		return nil, false
	}
	method, ok := token.Method.(*jwt.SigningMethodECDSA)
	if !ok {
		return nil, false
	}
	return method, true
}

// tokenClaims represents parsed JWT Token tokenClaims.
type tokenClaims struct {
	jwt.StandardClaims
	publicKeys     map[string]publicKey
	Email          string `json:"email,omitempty"`
	appJWTAudience string
}

// Valid validates the tokenClaims.
func (c tokenClaims) Valid() error {
	if err := (c.StandardClaims).Valid(); err != nil {
		return err
	}
	if c.Issuer != issuerClaim {
		return fmt.Errorf("invalid issuer: %q", c.Issuer)
	}
	if c.Audience != c.appJWTAudience {
		return fmt.Errorf("unexpected audience: %q", c.Audience)
	}

	return nil
}
