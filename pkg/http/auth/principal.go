package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/ottogroup/penelope/pkg/config"
	"github.com/ottogroup/penelope/pkg/http/auth/model"
	"github.com/ottogroup/penelope/pkg/provider"
	"go.opencensus.io/trace"
	"gopkg.in/dc0d/tinykv.v4"
	"net/http"
	"strings"
	"time"
)

var principalCache = tinykv.New(time.Minute * 5)

// PrincipalRetriever retriever to fetch principal of request
type PrincipalRetriever interface {
	RetrieveCurrentPrincipal(ctxIn context.Context, r *http.Request) (*model.Principal, error)
}

type defaultPrincipalRetriever struct {
	principalProvider provider.PrincipalProvider
}

// NewPrincipalRetriever
func NewPrincipalRetriever(provider provider.PrincipalProvider) (PrincipalRetriever, error) {
	return &defaultPrincipalRetriever{
		principalProvider: provider,
	}, nil
}

func (p *defaultPrincipalRetriever) RetrieveCurrentPrincipal(ctxIn context.Context, r *http.Request) (*model.Principal, error) {
	ctx, span := trace.StartSpan(ctxIn, "(*defaultPrincipalRetriever).RetrieveCurrentPrincipal")
	defer span.End()

	if config.SetTestUser.Exist() {
		userEmail := config.SetTestUser.MustGet()
		principal, err := p.principalProvider.GetPrincipalForEmail(ctx, userEmail)
		return principal, err
	}

	if !config.TokenHeaderKey.Exist() {
		return nil, fmt.Errorf("required environment variable %s is not provided", config.TokenHeaderKey)
	}
	tokenKey := config.TokenHeaderKey.MustGet()

	token := r.Header.Get(tokenKey)
	if len(token) == 0 {
		return nil, fmt.Errorf("token not found in request header %s", tokenKey)
	}

	splits := strings.Split(token, ".")
	if len(splits) != 3 {
		return nil, fmt.Errorf("job JWT in malformed expected 3 parts got %d", len(splits))
	}

	jwtToken := jwtToken{Body: &body{}}

	if decodedBody, err := base64.RawStdEncoding.DecodeString(splits[1]); err != nil {
		return nil, err
	} else if err := json.Unmarshal(decodedBody, jwtToken.Body); err != nil {
		return nil, err
	}

	userEmail := jwtToken.Body.Email
	if userEmail == "" {
		return nil, fmt.Errorf("email not found in request token %+v", jwtToken.Body)
	}

	if !config.CompanyDomains.Exist() {
		return nil, fmt.Errorf("required environment variable %s needs to be provided", config.CompanyDomains)
	}

	if !validateUser(userEmail) {
		return nil, fmt.Errorf("user %s is not part of company domain(s): %s", userEmail, config.CompanyDomains.MustGet())
	}

	if cachedPrincipal, ok := principalCache.Get(token); ok {
		return cachedPrincipal.(*model.Principal), nil
	}

	principal, err := p.principalProvider.GetPrincipalForEmail(ctx, userEmail)
	if err != nil {
		return nil, err
	}

	principalCache.Put(token, principal)
	return principal, nil
}

func validateUser(userEmail string) bool {
	domains := config.CompanyDomains.MustGet()

	for _, domain := range strings.Split(domains, ",") {
		if strings.HasSuffix(userEmail, fmt.Sprintf("@%s", strings.TrimSpace(domain))) {
			return true
		}
	}

	return false
}

// jwtToken representation
type jwtToken struct {
	Header  *header `json:"header,omitempty"`
	Body    *body   `json:"body,omitempty"`
	Signing string
}

// header part of jwtToken
type header struct {
	Alg string `json:"alg,omitempty"`
	Kid string `json:"kid,omitempty"`
	Typ string `json:"typ,omitempty"`
}

// body part of jwtToken
type body struct {
	Aud   string `json:"aud,omitempty"`
	Exp   int64  `json:"exp,omitempty"`
	Iat   int64  `json:"iat,omitempty"`
	Iss   string `json:"iss,omitempty"`
	Email string `json:"email,omitempty"`
	Hd    string `json:"hd,omitempty"`
}
