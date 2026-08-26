package api

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	formauth "github.com/aderaldo/falqon/backend/internal/auth"
)

type databaseHealthCheckerStub struct{ err error }

func (stub databaseHealthCheckerStub) Ping(context.Context) error { return stub.err }

type googleAuthenticatorStub struct {
	enabled  bool
	identity formauth.GoogleIdentity
}

func (stub googleAuthenticatorStub) Enabled() bool { return stub.enabled }

func (stub googleAuthenticatorStub) AuthorizationURL(state, nonce string) (string, error) {
	return "https://accounts.google.com/auth?state=" + state + "&nonce=" + nonce, nil
}

func (stub googleAuthenticatorStub) Authenticate(
	context.Context,
	string,
	string,
) (formauth.GoogleIdentity, error) {
	return stub.identity, nil
}

type authRepositoryStub struct {
	user         formauth.User
	sessionToken string
	expiresAt    time.Time
	createError  error
}

func (stub authRepositoryStub) CreateEmailUser(
	context.Context,
	string,
	string,
	string,
) (formauth.User, error) {
	return stub.user, stub.createError
}

func (stub authRepositoryStub) UpsertGoogleUser(
	context.Context,
	formauth.GoogleIdentity,
) (formauth.User, error) {
	return stub.user, nil
}

func (stub authRepositoryStub) CreateSession(
	context.Context,
	int64,
	time.Duration,
) (string, time.Time, error) {
	return stub.sessionToken, stub.expiresAt, nil
}

func (stub authRepositoryStub) UserBySession(context.Context, string) (formauth.User, error) {
	return stub.user, nil
}

func (stub authRepositoryStub) RevokeSession(context.Context, string) error { return nil }

func TestGetHealthReturnsOKWhenDatabaseIsAvailable(t *testing.T) {
	t.Parallel()

	server := newHealthTestServer(databaseHealthCheckerStub{})
	response, err := server.GetHealth(context.Background(), GetHealthRequestObject{})
	if err != nil {
		t.Fatalf("GetHealth() error = %v", err)
	}
	if _, ok := response.(GetHealth200JSONResponse); !ok {
		t.Fatalf("GetHealth() response = %T, want GetHealth200JSONResponse", response)
	}
}

func TestGetHealthReturnsUnavailableWhenDatabaseFails(t *testing.T) {
	t.Parallel()

	server := newHealthTestServer(databaseHealthCheckerStub{err: errors.New("database error")})
	response, err := server.GetHealth(context.Background(), GetHealthRequestObject{})
	if err != nil {
		t.Fatalf("GetHealth() error = %v", err)
	}
	if _, ok := response.(GetHealth503JSONResponse); !ok {
		t.Fatalf("GetHealth() response = %T, want GetHealth503JSONResponse", response)
	}
}

func TestGeneratedOpenAPIContractIsValid(t *testing.T) {
	t.Parallel()

	contract, err := GetSwagger()
	if err != nil {
		t.Fatalf("GetSwagger() error = %v", err)
	}
	if err := contract.Validate(context.Background()); err != nil {
		t.Fatalf("OpenAPI contract validation error = %v", err)
	}
}

func TestBeginGoogleLoginCreatesSecureFlow(t *testing.T) {
	t.Parallel()

	server := newAuthTestServer(googleAuthenticatorStub{enabled: true}, authRepositoryStub{})
	response, err := server.BeginGoogleLogin(context.Background(), BeginGoogleLoginRequestObject{})
	if err != nil {
		t.Fatalf("BeginGoogleLogin() error = %v", err)
	}
	redirect, ok := response.(BeginGoogleLogin307Response)
	if !ok {
		t.Fatalf("BeginGoogleLogin() response = %T, want BeginGoogleLogin307Response", response)
	}
	if !strings.HasPrefix(redirect.Headers.Location, "https://accounts.google.com/") {
		t.Fatalf("Location = %q, want Google URL", redirect.Headers.Location)
	}
	if !strings.Contains(redirect.Headers.SetCookie, "falqon_oauth=") ||
		!strings.Contains(redirect.Headers.SetCookie, "HttpOnly") {
		t.Fatalf("Set-Cookie = %q, want HttpOnly OAuth cookie", redirect.Headers.SetCookie)
	}
}

func TestCompleteGoogleLoginCreatesSession(t *testing.T) {
	t.Parallel()

	google := googleAuthenticatorStub{
		enabled: true,
		identity: formauth.GoogleIdentity{
			Subject: "google-subject",
			Name:    "User",
			Email:   "user@example.com",
		},
	}
	repository := authRepositoryStub{
		user:         formauth.User{ID: 2, Name: "User", Email: "user@example.com"},
		sessionToken: "session-token",
		expiresAt:    time.Now().Add(time.Hour),
	}
	server := newAuthTestServer(google, repository)
	flow, encodedFlow, err := server.flowCodec.New()
	if err != nil {
		t.Fatalf("flowCodec.New() error = %v", err)
	}
	code := "authorization-code"

	response, err := server.CompleteGoogleLogin(context.Background(), CompleteGoogleLoginRequestObject{
		Params: CompleteGoogleLoginParams{
			Code:        &code,
			State:       &flow.State,
			FalqonOauth: &encodedFlow,
		},
	})
	if err != nil {
		t.Fatalf("CompleteGoogleLogin() error = %v", err)
	}
	redirect, ok := response.(CompleteGoogleLogin302Response)
	if !ok {
		t.Fatalf("CompleteGoogleLogin() response = %T, want CompleteGoogleLogin302Response", response)
	}
	if redirect.Headers.Location != "http://localhost:5173" {
		t.Fatalf("Location = %q, want frontend URL", redirect.Headers.Location)
	}
	if !strings.Contains(redirect.Headers.SetCookie, "falqon_session=session-token") ||
		!strings.Contains(redirect.Headers.SetCookie, "HttpOnly") {
		t.Fatalf("Set-Cookie = %q, want HttpOnly session cookie", redirect.Headers.SetCookie)
	}
}

func TestRegisterUserCreatesUserAndSession(t *testing.T) {
	t.Parallel()

	repository := authRepositoryStub{
		user:         formauth.User{ID: 3, Name: "Maria", Email: "maria@example.com"},
		sessionToken: "session-token",
		expiresAt:    time.Now().Add(time.Hour),
	}
	server := newAuthTestServer(googleAuthenticatorStub{}, repository)

	response, err := server.RegisterUser(context.Background(), RegisterUserRequestObject{
		Body: &RegisterUserJSONRequestBody{
			Name:     " Maria ",
			Email:    "MARIA@example.com",
			Password: "password123",
		},
	})
	if err != nil {
		t.Fatalf("RegisterUser() error = %v", err)
	}
	created, ok := response.(RegisterUser201JSONResponse)
	if !ok {
		t.Fatalf("RegisterUser() response = %T, want RegisterUser201JSONResponse", response)
	}
	if created.Body.Id != 3 || !strings.Contains(created.Headers.SetCookie, "falqon_session=") {
		t.Fatalf("RegisterUser() response = %#v, want user and session cookie", created)
	}
}

func TestRegisterUserRejectsInvalidData(t *testing.T) {
	t.Parallel()

	server := newAuthTestServer(googleAuthenticatorStub{}, authRepositoryStub{})
	response, err := server.RegisterUser(context.Background(), RegisterUserRequestObject{
		Body: &RegisterUserJSONRequestBody{
			Name:     "A",
			Email:    "not-an-email",
			Password: "short",
		},
	})
	if err != nil {
		t.Fatalf("RegisterUser() error = %v", err)
	}
	if _, ok := response.(RegisterUser422JSONResponse); !ok {
		t.Fatalf("RegisterUser() response = %T, want RegisterUser422JSONResponse", response)
	}
}

func newHealthTestServer(database databaseHealthChecker) *Server {
	return NewServer(database, nil, nil, nil, ServerConfig{})
}

func newAuthTestServer(google googleAuthenticator, repository authRepository) *Server {
	codec, err := formauth.NewFlowCodec("a-secret-with-at-least-32-characters")
	if err != nil {
		panic(err)
	}
	return NewServer(
		databaseHealthCheckerStub{},
		google,
		repository,
		codec,
		ServerConfig{
			WebURL:          "http://localhost:5173",
			SessionDuration: 7 * 24 * time.Hour,
		},
	)
}
