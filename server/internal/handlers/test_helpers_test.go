package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/go-chi/chi/v5"
)

func newRequestWithURLParams(method, target, body string, params map[string]string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	routeCtx := chi.NewRouteContext()
	for key, value := range params {
		routeCtx.URLParams.Add(key, value)
	}
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
}
