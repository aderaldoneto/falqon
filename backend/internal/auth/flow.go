package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
)

var ErrInvalidOAuthFlow = errors.New("invalid OAuth flow cookie")

type OAuthFlow struct {
	State string
	Nonce string
}

type FlowCodec struct {
	secret []byte
}

func NewFlowCodec(secret string) (*FlowCodec, error) {
	if len(secret) < 32 {
		return nil, errors.New("SESSION_SECRET must contain at least 32 characters")
	}
	return &FlowCodec{secret: []byte(secret)}, nil
}

func (codec *FlowCodec) New() (OAuthFlow, string, error) {
	state, err := randomToken(32)
	if err != nil {
		return OAuthFlow{}, "", err
	}
	nonce, err := randomToken(32)
	if err != nil {
		return OAuthFlow{}, "", err
	}

	flow := OAuthFlow{State: state, Nonce: nonce}
	payload := state + "." + nonce
	return flow, payload + "." + codec.signature(payload), nil
}

func (codec *FlowCodec) Decode(value string) (OAuthFlow, error) {
	parts := strings.Split(value, ".")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" {
		return OAuthFlow{}, ErrInvalidOAuthFlow
	}

	payload := parts[0] + "." + parts[1]
	providedSignature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return OAuthFlow{}, ErrInvalidOAuthFlow
	}
	expectedSignature, _ := base64.RawURLEncoding.DecodeString(codec.signature(payload))
	if !hmac.Equal(providedSignature, expectedSignature) {
		return OAuthFlow{}, ErrInvalidOAuthFlow
	}

	return OAuthFlow{State: parts[0], Nonce: parts[1]}, nil
}

func (codec *FlowCodec) signature(payload string) string {
	mac := hmac.New(sha256.New, codec.secret)
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func randomToken(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}
