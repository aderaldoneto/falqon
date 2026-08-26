package api

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	formauth "github.com/aderaldo/falqon/backend/internal/auth"
	formdomain "github.com/aderaldo/falqon/backend/internal/forms"
)

type databaseHealthCheckerStub struct{ err error }

func (stub formRepositoryStub) ListPublished(context.Context) ([]formdomain.Summary, error) {
	return stub.forms, stub.err
}
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
	userError    error
}

type formRepositoryStub struct {
	forms           []formdomain.Summary
	err             error
	submission      formdomain.Submission
	formSubmissions formdomain.FormSubmissions
}

func (stub formRepositoryStub) CreateSubmission(context.Context, int64, []formdomain.Answer) (formdomain.Submission, error) {
	return stub.submission, stub.err
}

func (stub formRepositoryStub) ListSubmissions(context.Context, int64, int64) (formdomain.FormSubmissions, error) {
	return stub.formSubmissions, stub.err
}

func (stub formRepositoryStub) Create(
	context.Context,
	int64,
	string,
	string,
	*string,
	[]formdomain.FieldDefinition,
) (formdomain.Summary, error) {
	if len(stub.forms) == 0 {
		return formdomain.Summary{}, stub.err
	}
	return stub.forms[0], stub.err
}

func (stub formRepositoryStub) ListByOwner(context.Context, int64) ([]formdomain.Summary, error) {
	return stub.forms, stub.err
}

func (stub formRepositoryStub) FindByOwner(context.Context, int64, int64) (formdomain.PublicForm, error) {
	return stub.FindPublishedBySlug(context.Background(), "")
}

func (stub formRepositoryStub) Update(_ context.Context, _ int64, _ int64, title string, slug string, _ *string, _ []formdomain.FieldDefinition) (formdomain.Summary, error) {
	if len(stub.forms) == 0 {
		return formdomain.Summary{}, stub.err
	}
	updated := stub.forms[0]
	updated.Title = title
	updated.Slug = slug
	return updated, stub.err
}

func (stub formRepositoryStub) Delete(context.Context, int64, int64) error {
	return stub.err
}

func (stub formRepositoryStub) Publish(context.Context, int64, int64) (formdomain.Summary, error) {
	if len(stub.forms) == 0 {
		return formdomain.Summary{}, stub.err
	}
	return stub.forms[0], stub.err
}

func (stub formRepositoryStub) FindPublishedBySlug(context.Context, string) (formdomain.PublicForm, error) {
	if len(stub.forms) == 0 {
		return formdomain.PublicForm{}, stub.err
	}
	form := stub.forms[0]
	return formdomain.PublicForm{
		ID: form.ID, Title: form.Title, Slug: form.Slug, Description: form.Description,
		Fields: []formdomain.PublicField{{
			ID: 21, Type: formdomain.FieldTypeRating, Label: "Nota", Required: true,
			Configuration: []byte(`{"min":1,"max":5}`),
		}},
	}, stub.err
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
	return stub.user, stub.userError
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

func TestRegisterUserRejectsDuplicateEmail(t *testing.T) {
	t.Parallel()

	server := newAuthTestServer(googleAuthenticatorStub{}, authRepositoryStub{createError: formauth.ErrEmailAlreadyExists})
	response, err := server.RegisterUser(context.Background(), RegisterUserRequestObject{Body: &RegisterUserJSONRequestBody{
		Name: "Maria", Email: "maria@example.com", Password: "password123",
	}})
	if err != nil {
		t.Fatalf("RegisterUser() error = %v", err)
	}
	if _, ok := response.(RegisterUser409JSONResponse); !ok {
		t.Fatalf("RegisterUser() response = %T, want RegisterUser409JSONResponse", response)
	}
}

func TestListFormsReturnsAuthenticatedUsersForms(t *testing.T) {
	t.Parallel()

	description := "Uma pesquisa sobre o filme"
	updatedAt := time.Now().UTC()
	repository := authRepositoryStub{
		user: formauth.User{ID: 7, Name: "Maria", Email: "maria@example.com"},
	}
	formsRepository := formRepositoryStub{forms: []formdomain.Summary{
		{
			ID:          11,
			Title:       "The Godfather",
			Slug:        "the-godfather",
			Description: &description,
			State:       "DRAFT",
			CreatedAt:   updatedAt,
			UpdatedAt:   updatedAt,
		},
	}}
	server := newAuthTestServerWithForms(googleAuthenticatorStub{}, repository, formsRepository)
	token := "session-token"

	response, err := server.ListForms(context.Background(), ListFormsRequestObject{
		Params: ListFormsParams{FalqonSession: &token},
	})
	if err != nil {
		t.Fatalf("ListForms() error = %v", err)
	}
	list, ok := response.(ListForms200JSONResponse)
	if !ok || len(list) != 1 || list[0].Slug != "the-godfather" {
		t.Fatalf("ListForms() response = %#v, want the authenticated user's form", response)
	}
}

func TestGetPublicFormReturnsPublishedDefinition(t *testing.T) {
	t.Parallel()

	server := newAuthTestServerWithForms(
		googleAuthenticatorStub{}, authRepositoryStub{}, formRepositoryStub{forms: []formdomain.Summary{{
			ID: 11, Title: "The Godfather", Slug: "the-godfather", State: "PUBLISHED",
		}}},
	)
	response, err := server.GetPublicForm(context.Background(), GetPublicFormRequestObject{Slug: "the-godfather"})
	if err != nil {
		t.Fatalf("GetPublicForm() error = %v", err)
	}
	form, ok := response.(GetPublicForm200JSONResponse)
	if !ok || len(form.Fields) != 1 || form.Fields[0].Type != RATING {
		t.Fatalf("GetPublicForm() response = %#v, want public form", response)
	}
}

func TestGetPublicFormReturnsNotFound(t *testing.T) {
	t.Parallel()

	server := newAuthTestServerWithForms(
		googleAuthenticatorStub{}, authRepositoryStub{}, formRepositoryStub{err: formdomain.ErrFormNotFound},
	)
	response, err := server.GetPublicForm(context.Background(), GetPublicFormRequestObject{Slug: "draft"})
	if err != nil {
		t.Fatalf("GetPublicForm() error = %v", err)
	}
	if _, ok := response.(GetPublicForm404JSONResponse); !ok {
		t.Fatalf("GetPublicForm() response = %T, want GetPublicForm404JSONResponse", response)
	}
}

func TestListFormsRequiresSession(t *testing.T) {
	t.Parallel()

	server := newAuthTestServer(googleAuthenticatorStub{}, authRepositoryStub{})
	response, err := server.ListForms(context.Background(), ListFormsRequestObject{})
	if err != nil {
		t.Fatalf("ListForms() error = %v", err)
	}
	if _, ok := response.(ListForms401JSONResponse); !ok {
		t.Fatalf("ListForms() response = %T, want ListForms401JSONResponse", response)
	}
}

func TestCreateFormValidatesAndCreatesDraft(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	formsRepository := formRepositoryStub{forms: []formdomain.Summary{
		{ID: 15, Title: "Parasite", Slug: "parasite", State: "DRAFT", CreatedAt: now, UpdatedAt: now},
	}}
	server := newAuthTestServerWithForms(
		googleAuthenticatorStub{},
		authRepositoryStub{user: formauth.User{ID: 7}},
		formsRepository,
	)
	token := "session-token"

	response, err := server.CreateForm(context.Background(), CreateFormRequestObject{
		Params: CreateFormParams{FalqonSession: &token},
		Body: &CreateFormJSONRequestBody{
			Title: " Parasite ",
			Slug:  "parasite",
			Fields: []CreateFormField{
				{
					Type: RATING, Label: "Qual é sua nota?", Required: true,
					Configuration: map[string]interface{}{"min": float64(1), "max": float64(5)},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateForm() error = %v", err)
	}
	created, ok := response.(CreateForm201JSONResponse)
	if !ok || created.Id != 15 || created.State != DRAFT {
		t.Fatalf("CreateForm() response = %#v, want created draft", response)
	}
}

func TestUpdateFormSavesDraft(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	server := newAuthTestServerWithForms(
		googleAuthenticatorStub{},
		authRepositoryStub{user: formauth.User{ID: 7}},
		formRepositoryStub{forms: []formdomain.Summary{{ID: 15, Title: "Original", Slug: "original", State: "DRAFT", CreatedAt: now, UpdatedAt: now}}},
	)
	token := "session-token"
	title := "Updated title"
	slug := "updated-title"

	response, err := server.UpdateForm(context.Background(), UpdateFormRequestObject{
		FormId: 15,
		Params: UpdateFormParams{FalqonSession: &token},
		Body: &UpdateFormJSONRequestBody{
			Title: title,
			Slug:  slug,
			Fields: []CreateFormField{{
				Type:          SHORTTEXT,
				Label:         "Question",
				Required:      false,
				Configuration: map[string]interface{}{},
			}},
		},
	})
	if err != nil {
		t.Fatalf("UpdateForm() error = %v", err)
	}
	updated, ok := response.(UpdateForm200JSONResponse)
	if !ok || updated.Title != title || updated.Slug != slug {
		t.Fatalf("UpdateForm() response = %#v, want updated draft", response)
	}
}

func TestCreateFormRejectsInvalidSlug(t *testing.T) {
	t.Parallel()

	server := newAuthTestServerWithForms(
		googleAuthenticatorStub{}, authRepositoryStub{user: formauth.User{ID: 7}}, formRepositoryStub{},
	)
	token := "session-token"
	response, err := server.CreateForm(context.Background(), CreateFormRequestObject{
		Params: CreateFormParams{FalqonSession: &token},
		Body: &CreateFormJSONRequestBody{
			Title: "Movie", Slug: "Invalid Slug", Fields: []CreateFormField{{
				Type: SHORTTEXT, Label: "Review", Configuration: map[string]interface{}{},
			}},
		},
	})
	if err != nil {
		t.Fatalf("CreateForm() error = %v", err)
	}
	if _, ok := response.(CreateForm422JSONResponse); !ok {
		t.Fatalf("CreateForm() response = %T, want CreateForm422JSONResponse", response)
	}
}

func TestPublishFormPublishesDraft(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	server := newAuthTestServerWithForms(
		googleAuthenticatorStub{},
		authRepositoryStub{user: formauth.User{ID: 7}},
		formRepositoryStub{forms: []formdomain.Summary{{
			ID: 15, Title: "Parasite", Slug: "parasite", State: "PUBLISHED", CreatedAt: now, UpdatedAt: now,
		}}},
	)
	token := "session-token"
	response, err := server.PublishForm(context.Background(), PublishFormRequestObject{
		FormId: 15, Params: PublishFormParams{FalqonSession: &token},
	})
	if err != nil {
		t.Fatalf("PublishForm() error = %v", err)
	}
	published, ok := response.(PublishForm200JSONResponse)
	if !ok || published.State != PUBLISHED {
		t.Fatalf("PublishForm() response = %#v, want published form", response)
	}
}

func TestPublishFormRejectsInvalidState(t *testing.T) {
	t.Parallel()

	server := newAuthTestServerWithForms(
		googleAuthenticatorStub{}, authRepositoryStub{user: formauth.User{ID: 7}},
		formRepositoryStub{err: formdomain.ErrInvalidFormState},
	)
	token := "session-token"
	response, err := server.PublishForm(context.Background(), PublishFormRequestObject{
		FormId: 15, Params: PublishFormParams{FalqonSession: &token},
	})
	if err != nil {
		t.Fatalf("PublishForm() error = %v", err)
	}
	if _, ok := response.(PublishForm409JSONResponse); !ok {
		t.Fatalf("PublishForm() response = %T, want PublishForm409JSONResponse", response)
	}
}

func TestCreateSubmissionAcceptsValidRequiredAnswer(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	server := newAuthTestServerWithForms(googleAuthenticatorStub{}, authRepositoryStub{}, formRepositoryStub{
		forms: []formdomain.Summary{{ID: 11, Slug: "movie"}}, submission: formdomain.Submission{ID: 31, CreatedAt: now},
	})
	response, err := server.CreateSubmission(context.Background(), CreateSubmissionRequestObject{
		Slug: "movie", Body: &CreateSubmissionJSONRequestBody{Answers: []SubmissionAnswer{{FieldId: 21, Value: float64(5)}}},
	})
	if err != nil {
		t.Fatalf("CreateSubmission() error = %v", err)
	}
	created, ok := response.(CreateSubmission201JSONResponse)
	if !ok || created.Id != 31 {
		t.Fatalf("CreateSubmission() response = %#v, want created submission", response)
	}
}

func TestCreateSubmissionRejectsMissingRequiredAnswer(t *testing.T) {
	t.Parallel()

	server := newAuthTestServerWithForms(googleAuthenticatorStub{}, authRepositoryStub{}, formRepositoryStub{forms: []formdomain.Summary{{ID: 11}}})
	response, err := server.CreateSubmission(context.Background(), CreateSubmissionRequestObject{
		Slug: "movie", Body: &CreateSubmissionJSONRequestBody{Answers: []SubmissionAnswer{}},
	})
	if err != nil {
		t.Fatalf("CreateSubmission() error = %v", err)
	}
	if _, ok := response.(CreateSubmission422JSONResponse); !ok {
		t.Fatalf("CreateSubmission() response = %T, want CreateSubmission422JSONResponse", response)
	}
}

func TestListFormSubmissionsReturnsAnswers(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	token := "session-token"
	server := newAuthTestServerWithForms(googleAuthenticatorStub{}, authRepositoryStub{user: formauth.User{ID: 7}}, formRepositoryStub{
		formSubmissions: formdomain.FormSubmissions{FormID: 11, Title: "Movie", Slug: "movie", Submissions: []formdomain.SubmissionResult{{
			ID: 31, CreatedAt: now, Answers: []formdomain.SubmissionAnswerResult{{FieldID: 21, FieldType: formdomain.FieldTypeRating, Label: "Nota", Value: []byte(`5`)}},
		}}},
	})
	response, err := server.ListFormSubmissions(context.Background(), ListFormSubmissionsRequestObject{
		FormId: 11, Params: ListFormSubmissionsParams{FalqonSession: &token},
	})
	if err != nil {
		t.Fatalf("ListFormSubmissions() error = %v", err)
	}
	result, ok := response.(ListFormSubmissions200JSONResponse)
	if !ok || len(result.Submissions) != 1 || len(result.Submissions[0].Answers) != 1 {
		t.Fatalf("ListFormSubmissions() response = %#v, want one answer", response)
	}
}

func TestListFormSubmissionsRequiresSession(t *testing.T) {
	t.Parallel()

	server := newAuthTestServer(googleAuthenticatorStub{}, authRepositoryStub{})
	response, err := server.ListFormSubmissions(context.Background(), ListFormSubmissionsRequestObject{FormId: 11})
	if err != nil {
		t.Fatalf("ListFormSubmissions() error = %v", err)
	}
	if _, ok := response.(ListFormSubmissions401JSONResponse); !ok {
		t.Fatalf("ListFormSubmissions() response = %T, want ListFormSubmissions401JSONResponse", response)
	}
}

func TestGetAuthSessionRejectsExpiredSession(t *testing.T) {
	t.Parallel()

	token := "expired"
	server := newAuthTestServer(googleAuthenticatorStub{}, authRepositoryStub{userError: formauth.ErrUnauthenticated})
	response, err := server.GetAuthSession(context.Background(), GetAuthSessionRequestObject{Params: GetAuthSessionParams{FalqonSession: &token}})
	if err != nil {
		t.Fatalf("GetAuthSession() error = %v", err)
	}
	if _, ok := response.(GetAuthSession401JSONResponse); !ok {
		t.Fatalf("GetAuthSession() response = %T, want GetAuthSession401JSONResponse", response)
	}
}

func TestDeleteFormReturnsNotFound(t *testing.T) {
	t.Parallel()

	token := "session-token"
	server := newAuthTestServerWithForms(googleAuthenticatorStub{}, authRepositoryStub{user: formauth.User{ID: 7}}, formRepositoryStub{err: formdomain.ErrFormNotFound})
	response, err := server.DeleteForm(context.Background(), DeleteFormRequestObject{FormId: 99, Params: DeleteFormParams{FalqonSession: &token}})
	if err != nil {
		t.Fatalf("DeleteForm() error = %v", err)
	}
	if _, ok := response.(DeleteForm404JSONResponse); !ok {
		t.Fatalf("DeleteForm() response = %T, want DeleteForm404JSONResponse", response)
	}
}

func newHealthTestServer(database databaseHealthChecker) *Server {
	return NewServer(database, nil, nil, nil, nil, ServerConfig{})
}

func newAuthTestServer(google googleAuthenticator, repository authRepository) *Server {
	return newAuthTestServerWithForms(google, repository, formRepositoryStub{})
}

func newAuthTestServerWithForms(
	google googleAuthenticator,
	repository authRepository,
	formsRepository formRepository,
) *Server {
	codec, err := formauth.NewFlowCodec("a-secret-with-at-least-32-characters")
	if err != nil {
		panic(err)
	}
	return NewServer(
		databaseHealthCheckerStub{},
		google,
		repository,
		formsRepository,
		codec,
		ServerConfig{
			WebURL:          "http://localhost:5173",
			SessionDuration: 7 * 24 * time.Hour,
		},
	)
}
