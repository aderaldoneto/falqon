package api

import (
	"context"
	"errors"
	"testing"
)

type databaseHealthCheckerStub struct {
	err error
}

func (stub databaseHealthCheckerStub) Ping(context.Context) error {
	return stub.err
}

func TestGetHealthReturnsOKWhenDatabaseIsAvailable(t *testing.T) {
	t.Parallel()

	server := NewServer(databaseHealthCheckerStub{})
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

	server := NewServer(databaseHealthCheckerStub{err: errors.New("database error")})
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
