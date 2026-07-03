package handlers

import (
	"net/http"

	"project-prism/server/internal/respond"
)

func Health(w http.ResponseWriter, r *http.Request) {
	respond.JSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "project-prism-api",
	})
}
