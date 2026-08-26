package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const (
	googleIssuer   = "https://accounts.google.com"
	googleJWKSURL  = "https://www.googleapis.com/oauth2/v3/certs"
	googleAuthURL  = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenURL = "https://oauth2.googleapis.com/token"
)

var ErrGoogleNotConfigured = errors.New("google authentication is not configured")

type GoogleIdentity struct {
	Subject string
	Name    string
	Email   string
}

type GoogleAuthenticator struct {
	enabled     bool
	oauthConfig oauth2.Config
	verifier    *oidc.IDTokenVerifier
}

func NewGoogleAuthenticator(clientID, clientSecret, redirectURL string) *GoogleAuthenticator {
	enabled := clientID != "" && clientSecret != "" && redirectURL != ""
	keySet := oidc.NewRemoteKeySet(context.Background(), googleJWKSURL)

	return &GoogleAuthenticator{
		enabled: enabled,
		oauthConfig: oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint: oauth2.Endpoint{
				AuthURL:  googleAuthURL,
				TokenURL: googleTokenURL,
			},
			Scopes: []string{oidc.ScopeOpenID, oidc.ScopeProfile, oidc.ScopeEmail},
		},
		verifier: oidc.NewVerifier(googleIssuer, keySet, &oidc.Config{ClientID: clientID}),
	}
}

func (authenticator *GoogleAuthenticator) Enabled() bool {
	return authenticator.enabled
}

func (authenticator *GoogleAuthenticator) AuthorizationURL(state, nonce string) (string, error) {
	if !authenticator.enabled {
		return "", ErrGoogleNotConfigured
	}

	return authenticator.oauthConfig.AuthCodeURL(
		state,
		oauth2.SetAuthURLParam("nonce", nonce),
	), nil
}

func (authenticator *GoogleAuthenticator) Authenticate(
	ctx context.Context,
	code string,
	nonce string,
) (GoogleIdentity, error) {
	if !authenticator.enabled {
		return GoogleIdentity{}, ErrGoogleNotConfigured
	}

	token, err := authenticator.oauthConfig.Exchange(ctx, code)
	if err != nil {
		return GoogleIdentity{}, fmt.Errorf("exchange authorization code: %w", err)
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return GoogleIdentity{}, errors.New("google response did not include an ID token")
	}

	idToken, err := authenticator.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return GoogleIdentity{}, fmt.Errorf("verify ID token: %w", err)
	}
	if idToken.Nonce != nonce {
		return GoogleIdentity{}, errors.New("ID token nonce does not match")
	}

	var claims struct {
		Subject       string `json:"sub"`
		Name          string `json:"name"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return GoogleIdentity{}, fmt.Errorf("decode ID token claims: %w", err)
	}
	if claims.Subject == "" || claims.Email == "" || !claims.EmailVerified {
		return GoogleIdentity{}, errors.New("google account does not have a verified identity and email")
	}

	return GoogleIdentity{
		Subject: claims.Subject,
		Name:    claims.Name,
		Email:   claims.Email,
	}, nil
}
