package auth

import (
    "context"
    "fmt"
    "github.com/golang/glog"
    "net/http"
)

// see: https://blog.golang.org/context#TOC_3.2.
type key int

const (
    // CtxPrincipalKey in ctx
    CtxPrincipalKey key = iota
)

// AuthenticationMiddleware is a middleware for auth process
type AuthenticationMiddleware struct {
    tokenValidator     TokenValidator
    principalRetriever PrincipalRetriever
}

// NewAuthenticationMiddleware return instance of AuthenticationMiddleware
func NewAuthenticationMiddleware(validator TokenValidator, principalRetriever PrincipalRetriever) (*AuthenticationMiddleware, error) {
    if validator == nil {
        return nil, fmt.Errorf("token validator must not be nil")
    }
    if principalRetriever == nil {
        return nil, fmt.Errorf("principal retriever must not be nil")
    }

    middleware := &AuthenticationMiddleware{
        tokenValidator:     validator,
        principalRetriever: principalRetriever,
    }

    return middleware, nil
}

// AddAuthentication add new auth method
func (a *AuthenticationMiddleware) AddAuthentication(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        if err := a.tokenValidator.ValidateRequest(r); err != nil {
            glog.Warningf("Error validating request for %s: %s", r.URL.Path, err)
            w.WriteHeader(http.StatusUnauthorized)
            return
        }
        principal, err := a.principalRetriever.RetrieveCurrentPrincipal(ctx, r)
        if err != nil {
            glog.Warningf("could not get current principal: %s", err)
            w.WriteHeader(http.StatusUnauthorized)
            return
        }
        ctx = context.WithValue(r.Context(), CtxPrincipalKey, principal)
        next.ServeHTTP(w, r.WithContext(ctx))
    }
}
