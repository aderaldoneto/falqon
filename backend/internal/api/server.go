package api

import "context"

type databaseHealthChecker interface {
	Ping(context.Context) error
}

type Server struct {
	database databaseHealthChecker
}

func NewServer(database databaseHealthChecker) *Server {
	return &Server{database: database}
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
