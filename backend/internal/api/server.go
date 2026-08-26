package api

import (
	"context"
	"crypto/hmac"
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"

	formauth "github.com/aderaldo/falqon/backend/internal/auth"
	formdomain "github.com/aderaldo/falqon/backend/internal/forms"
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

type formRepository interface {
	Create(context.Context, int64, string, string, *string, []formdomain.FieldDefinition) (formdomain.Summary, error)
	ListByOwner(context.Context, int64) ([]formdomain.Summary, error)
	ListPublished(context.Context) ([]formdomain.Summary, error)
	Publish(context.Context, int64, int64) (formdomain.Summary, error)
	FindPublishedBySlug(context.Context, string) (formdomain.PublicForm, error)
	CreateSubmission(context.Context, int64, []formdomain.Answer) (formdomain.Submission, error)
	ListSubmissions(context.Context, int64, int64) (formdomain.FormSubmissions, error)
}

func (server *Server) ListPublishedForms(
	ctx context.Context,
	_ ListPublishedFormsRequestObject,
) (ListPublishedFormsResponseObject, error) {
	forms, err := server.formRepository.ListPublished(ctx)
	if err != nil {
		return nil, err
	}

	response := make(ListPublishedForms200JSONResponse, 0, len(forms))
	for _, form := range forms {
		response = append(response, FormSummary{
			Id: form.ID, Title: form.Title, Slug: form.Slug,
			Description: form.Description, State: FormState(form.State),
			CreatedAt: form.CreatedAt, UpdatedAt: form.UpdatedAt,
		})
	}
	return response, nil
}

var formSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func (server *Server) GetPublicForm(
	ctx context.Context,
	request GetPublicFormRequestObject,
) (GetPublicFormResponseObject, error) {
	form, err := server.formRepository.FindPublishedBySlug(ctx, request.Slug)
	if errors.Is(err, formdomain.ErrFormNotFound) {
		return GetPublicForm404JSONResponse{Code: "form_not_found", Message: "form was not found"}, nil
	}
	if err != nil {
		return nil, err
	}

	fields := make([]PublicFormField, 0, len(form.Fields))
	for _, field := range form.Fields {
		configuration := make(map[string]interface{})
		if err := json.Unmarshal(field.Configuration, &configuration); err != nil {
			return nil, err
		}
		fields = append(fields, PublicFormField{
			Id: field.ID, Type: FormFieldType(field.Type), Label: field.Label,
			Description: field.Description, Required: field.Required, Configuration: configuration,
		})
	}
	return GetPublicForm200JSONResponse{
		Id: form.ID, Title: form.Title, Slug: form.Slug, Description: form.Description, Fields: fields,
	}, nil
}

func (server *Server) CreateSubmission(
	ctx context.Context,
	request CreateSubmissionRequestObject,
) (CreateSubmissionResponseObject, error) {
	if request.Body == nil {
		return CreateSubmission422JSONResponse{Code: "invalid_answers", Message: "answers are required"}, nil
	}
	form, err := server.formRepository.FindPublishedBySlug(ctx, request.Slug)
	if errors.Is(err, formdomain.ErrFormNotFound) {
		return CreateSubmission404JSONResponse{Code: "form_not_found", Message: "form was not found"}, nil
	}
	if err != nil {
		return nil, err
	}

	fields := make(map[int64]formdomain.PublicField, len(form.Fields))
	for _, field := range form.Fields {
		fields[field.ID] = field
	}
	provided := make(map[int64]struct{}, len(request.Body.Answers))
	answers := make([]formdomain.Answer, 0, len(request.Body.Answers))
	for _, input := range request.Body.Answers {
		field, exists := fields[input.FieldId]
		if !exists {
			return invalidAnswers("answer references an unknown field"), nil
		}
		if _, duplicate := provided[input.FieldId]; duplicate {
			return invalidAnswers("a field cannot be answered more than once"), nil
		}
		value, err := formdomain.ValidateAnswer(field, input.Value)
		if err != nil {
			return invalidAnswers(err.Error()), nil
		}
		provided[input.FieldId] = struct{}{}
		answers = append(answers, formdomain.Answer{FieldID: input.FieldId, Value: value})
	}
	for _, field := range form.Fields {
		if _, answered := provided[field.ID]; field.Required && !answered {
			return invalidAnswers("all required fields must be answered"), nil
		}
	}

	submission, err := server.formRepository.CreateSubmission(ctx, form.ID, answers)
	if err != nil {
		return nil, err
	}
	return CreateSubmission201JSONResponse{Id: submission.ID, CreatedAt: submission.CreatedAt}, nil
}

func (server *Server) ListFormSubmissions(
	ctx context.Context,
	request ListFormSubmissionsRequestObject,
) (ListFormSubmissionsResponseObject, error) {
	if request.Params.FalqonSession == nil {
		return ListFormSubmissions401JSONResponse{Code: "unauthenticated", Message: "authentication is required"}, nil
	}
	user, err := server.authRepository.UserBySession(ctx, *request.Params.FalqonSession)
	if errors.Is(err, formauth.ErrUnauthenticated) {
		return ListFormSubmissions401JSONResponse{Code: "unauthenticated", Message: "authentication is required"}, nil
	}
	if err != nil {
		return nil, err
	}
	result, err := server.formRepository.ListSubmissions(ctx, request.FormId, user.ID)
	if errors.Is(err, formdomain.ErrFormNotFound) {
		return ListFormSubmissions404JSONResponse{Code: "form_not_found", Message: "form was not found"}, nil
	}
	if err != nil {
		return nil, err
	}

	submissions := make([]AdminSubmission, 0, len(result.Submissions))
	for _, item := range result.Submissions {
		answers := make([]AdminSubmissionAnswer, 0, len(item.Answers))
		for _, answer := range item.Answers {
			var value interface{}
			if err := json.Unmarshal(answer.Value, &value); err != nil {
				return nil, err
			}
			answers = append(answers, AdminSubmissionAnswer{
				FieldId: answer.FieldID, FieldType: FormFieldType(answer.FieldType), Label: answer.Label, Value: value,
			})
		}
		submissions = append(submissions, AdminSubmission{Id: item.ID, CreatedAt: item.CreatedAt, Answers: answers})
	}
	return ListFormSubmissions200JSONResponse{
		FormId: result.FormID, Title: result.Title, Slug: result.Slug, Submissions: submissions,
	}, nil
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
	formRepository formRepository
	flowCodec      *formauth.FlowCodec
	config         ServerConfig
}

func NewServer(
	database databaseHealthChecker,
	google googleAuthenticator,
	authRepository authRepository,
	formRepository formRepository,
	flowCodec *formauth.FlowCodec,
	config ServerConfig,
) *Server {
	return &Server{
		database:       database,
		google:         google,
		authRepository: authRepository,
		formRepository: formRepository,
		flowCodec:      flowCodec,
		config:         config,
	}
}

func (server *Server) ListForms(
	ctx context.Context,
	request ListFormsRequestObject,
) (ListFormsResponseObject, error) {
	if request.Params.FalqonSession == nil {
		return unauthenticatedFormsResponse(), nil
	}
	user, err := server.authRepository.UserBySession(ctx, *request.Params.FalqonSession)
	if errors.Is(err, formauth.ErrUnauthenticated) {
		return unauthenticatedFormsResponse(), nil
	}
	if err != nil {
		return nil, err
	}

	forms, err := server.formRepository.ListByOwner(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	response := make(ListForms200JSONResponse, 0, len(forms))
	for _, form := range forms {
		response = append(response, formSummaryResponse(form))
	}
	return response, nil
}

func (server *Server) CreateForm(
	ctx context.Context,
	request CreateFormRequestObject,
) (CreateFormResponseObject, error) {
	if request.Params.FalqonSession == nil {
		return unauthenticatedCreateFormResponse(), nil
	}
	user, err := server.authRepository.UserBySession(ctx, *request.Params.FalqonSession)
	if errors.Is(err, formauth.ErrUnauthenticated) {
		return unauthenticatedCreateFormResponse(), nil
	}
	if err != nil {
		return nil, err
	}
	if request.Body == nil {
		return invalidForm("form data is required"), nil
	}

	title := strings.TrimSpace(request.Body.Title)
	slug := strings.TrimSpace(request.Body.Slug)
	if len(title) < 2 || len(title) > 255 || len(slug) < 2 || len(slug) > 255 ||
		!formSlugPattern.MatchString(slug) || len(request.Body.Fields) == 0 {
		return invalidForm("title, slug, and at least one field are required"), nil
	}
	if request.Body.Description != nil {
		trimmed := strings.TrimSpace(*request.Body.Description)
		if len(trimmed) > 2000 {
			return invalidForm("description must contain at most 2000 characters"), nil
		}
		request.Body.Description = nullableAPIString(trimmed)
	}

	fields := make([]formdomain.FieldDefinition, 0, len(request.Body.Fields))
	for position, input := range request.Body.Fields {
		configuration, err := json.Marshal(input.Configuration)
		if err != nil {
			return invalidForm("field configuration is invalid"), nil
		}
		field := formdomain.FieldDefinition{
			Type: formdomain.FieldType(input.Type), Label: strings.TrimSpace(input.Label),
			Required: input.Required, Position: position, Configuration: configuration,
		}
		if input.Description != nil {
			field.Description = strings.TrimSpace(*input.Description)
		}
		if err := field.Validate(); err != nil {
			return invalidForm(err.Error()), nil
		}
		fields = append(fields, field)
	}

	form, err := server.formRepository.Create(
		ctx, user.ID, title, slug, request.Body.Description, fields,
	)
	if errors.Is(err, formdomain.ErrSlugAlreadyExists) {
		return CreateForm409JSONResponse{
			Code: "slug_already_exists", Message: "slug is already in use",
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return CreateForm201JSONResponse(formSummaryResponse(form)), nil
}

func (server *Server) PublishForm(
	ctx context.Context,
	request PublishFormRequestObject,
) (PublishFormResponseObject, error) {
	if request.Params.FalqonSession == nil {
		return PublishForm401JSONResponse{Code: "unauthenticated", Message: "authentication is required"}, nil
	}
	user, err := server.authRepository.UserBySession(ctx, *request.Params.FalqonSession)
	if errors.Is(err, formauth.ErrUnauthenticated) {
		return PublishForm401JSONResponse{Code: "unauthenticated", Message: "authentication is required"}, nil
	}
	if err != nil {
		return nil, err
	}

	form, err := server.formRepository.Publish(ctx, request.FormId, user.ID)
	if errors.Is(err, formdomain.ErrFormNotFound) {
		return PublishForm404JSONResponse{Code: "form_not_found", Message: "form was not found"}, nil
	}
	if errors.Is(err, formdomain.ErrInvalidFormState) {
		return PublishForm409JSONResponse{Code: "invalid_form_state", Message: "only draft forms can be published"}, nil
	}
	if err != nil {
		return nil, err
	}
	return PublishForm200JSONResponse(formSummaryResponse(form)), nil
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

func unauthenticatedFormsResponse() ListForms401JSONResponse {
	return ListForms401JSONResponse{
		Code:    "unauthenticated",
		Message: "authentication is required",
	}
}

func unauthenticatedCreateFormResponse() CreateForm401JSONResponse {
	return CreateForm401JSONResponse{Code: "unauthenticated", Message: "authentication is required"}
}

func invalidForm(message string) CreateForm422JSONResponse {
	return CreateForm422JSONResponse{Code: "invalid_form", Message: message}
}

func invalidAnswers(message string) CreateSubmission422JSONResponse {
	return CreateSubmission422JSONResponse{Code: "invalid_answers", Message: message}
}

func nullableAPIString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func formSummaryResponse(form formdomain.Summary) FormSummary {
	return FormSummary{
		Id: form.ID, Title: form.Title, Slug: form.Slug, Description: form.Description,
		State: FormState(form.State), CreatedAt: form.CreatedAt, UpdatedAt: form.UpdatedAt,
	}
}

func invalidRegistration(message string) RegisterUser422JSONResponse {
	return RegisterUser422JSONResponse{
		Code:    "invalid_registration",
		Message: message,
	}
}
