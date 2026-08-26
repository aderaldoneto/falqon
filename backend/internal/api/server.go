package api

import (
	"context"
	"crypto/hmac"
	"errors"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"

	formauth "github.com/aderaldo/falqon/backend/internal/auth"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"golang.org/x/crypto/bcrypt"
)

const (
	oauthCookieName   = "falqon_oauth"
	sessionCookieName = "falqon_session"
)

type databaseHealthChecker interface {
	Ping(context.Context) error
}

type googleAuthenticator interface {
	Enabled() bool
	AuthorizationURL(state, nonce string) (string, error)
	Authenticate(context.Context, string, string) (formauth.GoogleIdentity, error)
}

type authRepository interface {
	CreateEmailUser(context.Context, string, string, string) (formauth.User, error)
	UpsertGoogleUser(context.Context, formauth.GoogleIdentity) (formauth.User, error)
	CreateSession(context.Context, int64, time.Duration) (string, time.Time, error)
	UserBySession(context.Context, string) (formauth.User, error)
	RevokeSession(context.Context, string) error
}

func (server *Server) RegisterUser(
	ctx context.Context,
	request RegisterUserRequestObject,
) (RegisterUserResponseObject, error) {
	if request.Body == nil {
		return invalidRegistration("registration data is required"), nil
	}

	name := strings.TrimSpace(request.Body.Name)
	email := strings.ToLower(strings.TrimSpace(string(request.Body.Email)))
	password := request.Body.Password
	parsedEmail, emailError := mail.ParseAddress(email)
	if len(name) < 2 || len(name) > 160 || emailError != nil || parsedEmail.Address != email ||
		len(email) > 320 || len(password) < 8 || len(password) > 72 {
		return invalidRegistration("name, email, or password is invalid"), nil
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user, err := server.authRepository.CreateEmailUser(ctx, name, email, string(passwordHash))
	if errors.Is(err, formauth.ErrEmailAlreadyExists) {
		return RegisterUser409JSONResponse{
			Code:    "email_already_registered",
			Message: "email is already registered",
		}, nil
	}
	if err != nil {
		return nil, err
	}

	token, expiresAt, err := server.authRepository.CreateSession(
		ctx,
		user.ID,
		server.config.SessionDuration,
	)
	if err != nil {
		return nil, err
	}

	return RegisterUser201JSONResponse{
		Body: User{
			Id:    user.ID,
			Name:  user.Name,
			Email: openapi_types.Email(user.Email),
		},
		Headers: RegisterUser201ResponseHeaders{
			SetCookie: server.sessionCookie(token, expiresAt).String(),
		},
	}, nil
}

type ServerConfig struct {
	WebURL          string
	CookieSecure    bool
	SessionDuration time.Duration
}

type Server struct {
	database       databaseHealthChecker
	google         googleAuthenticator
	authRepository authRepository
	flowCodec      *formauth.FlowCodec
	config         ServerConfig
}

func NewServer(
	database databaseHealthChecker,
	google googleAuthenticator,
	authRepository authRepository,
	flowCodec *formauth.FlowCodec,
	config ServerConfig,
) *Server {
	return &Server{
		database:       database,
		google:         google,
		authRepository: authRepository,
		flowCodec:      flowCodec,
		config:         config,
	}
}

func (server *Server) GetHealth(
	ctx context.Context,
	_ GetHealthRequestObject,
) (GetHealthResponseObject, error) {
	if err := server.database.Ping(ctx); err != nil {
		return GetHealth503JSONResponse{
			Code:    "database_unavailable",
			Message: "database is unavailable",
		}, nil
	}
	return GetHealth200JSONResponse{Status: Ok}, nil
}

func (server *Server) BeginGoogleLogin(
	_ context.Context,
	_ BeginGoogleLoginRequestObject,
) (BeginGoogleLoginResponseObject, error) {
	if !server.google.Enabled() {
		return BeginGoogleLogin503JSONResponse{
			Code:    "google_auth_not_configured",
			Message: "Google authentication is not configured",
		}, nil
	}

	flow, encodedFlow, err := server.flowCodec.New()
	if err != nil {
		return nil, err
	}
	authorizationURL, err := server.google.AuthorizationURL(flow.State, flow.Nonce)
	if err != nil {
		return nil, err
	}

	return BeginGoogleLogin307Response{Headers: BeginGoogleLogin307ResponseHeaders{
		Location:  authorizationURL,
		SetCookie: server.oauthCookie(encodedFlow).String(),
	}}, nil
}

func (server *Server) CompleteGoogleLogin(
	ctx context.Context,
	request CompleteGoogleLoginRequestObject,
) (CompleteGoogleLoginResponseObject, error) {
	if request.Params.Error != nil {
		return server.authRedirect("google_denied"), nil
	}
	if request.Params.Code == nil || request.Params.State == nil || request.Params.FalqonOauth == nil {
		return invalidCallback("missing OAuth callback parameters"), nil
	}

	flow, err := server.flowCodec.Decode(*request.Params.FalqonOauth)
	if err != nil || !hmac.Equal([]byte(flow.State), []byte(*request.Params.State)) {
		return invalidCallback("invalid OAuth state"), nil
	}

	identity, err := server.google.Authenticate(ctx, *request.Params.Code, flow.Nonce)
	if err != nil {
		return server.authRedirect("google_authentication_failed"), nil
	}
	user, err := server.authRepository.UpsertGoogleUser(ctx, identity)
	if err != nil {
		return nil, err
	}
	token, expiresAt, err := server.authRepository.CreateSession(
		ctx,
		user.ID,
		server.config.SessionDuration,
	)
	if err != nil {
		return nil, err
	}

	return CompleteGoogleLogin302Response{Headers: CompleteGoogleLogin302ResponseHeaders{
		Location:  server.config.WebURL,
		SetCookie: server.sessionCookie(token, expiresAt).String(),
	}}, nil
}

func (server *Server) GetAuthSession(
	ctx context.Context,
	request GetAuthSessionRequestObject,
) (GetAuthSessionResponseObject, error) {
	if request.Params.FalqonSession == nil {
		return unauthenticatedResponse(), nil
	}

	user, err := server.authRepository.UserBySession(ctx, *request.Params.FalqonSession)
	if errors.Is(err, formauth.ErrUnauthenticated) {
		return unauthenticatedResponse(), nil
	}
	if err != nil {
		return nil, err
	}

	return GetAuthSession200JSONResponse{
		Id:    user.ID,
		Name:  user.Name,
		Email: openapi_types.Email(user.Email),
	}, nil
}

func (server *Server) Logout(
	ctx context.Context,
	request LogoutRequestObject,
) (LogoutResponseObject, error) {
	if request.Params.FalqonSession != nil {
		if err := server.authRepository.RevokeSession(ctx, *request.Params.FalqonSession); err != nil {
			return nil, err
		}
	}

	return Logout204Response{Headers: Logout204ResponseHeaders{
		SetCookie: server.expiredSessionCookie().String(),
	}}, nil
}

func (server *Server) oauthCookie(value string) *http.Cookie {
	return &http.Cookie{
		Name:     oauthCookieName,
		Value:    value,
		Path:     "/auth/google/callback",
		MaxAge:   600,
		HttpOnly: true,
		Secure:   server.config.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	}
}

func (server *Server) sessionCookie(value string, expiresAt time.Time) *http.Cookie {
	return &http.Cookie{
		Name:     sessionCookieName,
		Value:    value,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   server.config.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	}
}

func (server *Server) expiredSessionCookie() *http.Cookie {
	return &http.Cookie{
		Name:     sessionCookieName,
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
		HttpOnly: true,
		Secure:   server.config.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	}
}

func (server *Server) authRedirect(code string) CompleteGoogleLogin302Response {
	location, err := url.Parse(server.config.WebURL)
	if err == nil {
		query := location.Query()
		query.Set("auth_error", code)
		location.RawQuery = query.Encode()
	}

	return CompleteGoogleLogin302Response{Headers: CompleteGoogleLogin302ResponseHeaders{
		Location:  location.String(),
		SetCookie: server.oauthCookie("").String(),
	}}
}

func invalidCallback(message string) CompleteGoogleLogin400JSONResponse {
	return CompleteGoogleLogin400JSONResponse{
		Code:    "invalid_google_callback",
		Message: message,
	}
}

func unauthenticatedResponse() GetAuthSession401JSONResponse {
	return GetAuthSession401JSONResponse{
		Code:    "unauthenticated",
		Message: "authentication is required",
	}
}

func invalidRegistration(message string) RegisterUser422JSONResponse {
	return RegisterUser422JSONResponse{
		Code:    "invalid_registration",
		Message: message,
	}
}
