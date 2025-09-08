package server

import (
	"log/slog"
	"net/http"

	"github.com/Blazingkevin/loki/internal/openapi"
	"github.com/getkin/kin-openapi/openapi3"
)

type Router struct {
	spec     *openapi3.T
	specInfo *openapi.SpecInfo
	loggger  *slog.Logger
	mux      *http.ServeMux
}

func NewRouter(spec *openapi3.T, specInfo *openapi.SpecInfo, logger *slog.Logger) *Router {
	router := &Router{
		spec:     spec,
		specInfo: specInfo,
		loggger:  logger,
		mux:      http.NewServeMux(),
	}

	// TODO: register routes

	return router
}
