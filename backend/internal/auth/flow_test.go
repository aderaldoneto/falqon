package auth

import (
	"errors"
	"testing"
)

func TestFlowCodecRoundTrip(t *testing.T) {
	t.Parallel()

	codec, err := NewFlowCodec("a-secret-with-at-least-32-characters")
	if err != nil {
		t.Fatalf("NewFlowCodec() error = %v", err)
	}
	created, encoded, err := codec.New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	decoded, err := codec.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if decoded != created {
		t.Fatalf("Decode() = %#v, want %#v", decoded, created)
	}
}

func TestFlowCodecRejectsTampering(t *testing.T) {
	t.Parallel()

	codec, err := NewFlowCodec("a-secret-with-at-least-32-characters")
	if err != nil {
		t.Fatalf("NewFlowCodec() error = %v", err)
	}
	_, encoded, err := codec.New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = codec.Decode(encoded + "changed")
	if !errors.Is(err, ErrInvalidOAuthFlow) {
		t.Fatalf("Decode() error = %v, want ErrInvalidOAuthFlow", err)
	}
}

func TestFlowCodecRequiresStrongSecret(t *testing.T) {
	t.Parallel()

	if _, err := NewFlowCodec("short"); err == nil {
		t.Fatal("NewFlowCodec() error = nil, want an error")
	}
}
