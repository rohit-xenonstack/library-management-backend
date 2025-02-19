package api

import (
	"encoding/json"
	"library-management/backend/internal/api/handler"
	"library-management/backend/internal/config"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Response struct {
	Message string `json:"message"`
}

func TestNewApi(t *testing.T) {
	var cfg = &config.SampleEnv
	var h *handler.Handler

	api := NewAPI(cfg, h)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ping", nil)
	api.Router.ServeHTTP(w, req)

	response := Response{
		Message: "pong",
	}
	responseJson, _ := json.Marshal(response)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, string(responseJson), w.Body.String())
}
