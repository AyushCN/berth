package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDevLoginRefusesInProduction(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// The route is no longer registered in production, but the handler refuses
	// as well: it hands out a token signed with the real JWT_SECRET, so relying
	// on registration alone is a single point of failure.
	r := gin.New()
	r.GET("/auth/dev-login", DevLogin("a-real-secret", "production"))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/dev-login", nil))

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 in production, got %d", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not json: %v", err)
	}
	if _, leaked := body["token"]; leaked {
		t.Error("a token was returned in production")
	}
}

func TestDevLoginWorksOutsideProduction(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/auth/dev-login", DevLogin("a-real-secret", "development"))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/dev-login", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 in development, got %d", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not json: %v", err)
	}
	token, _ := body["token"].(string)
	if token == "" {
		t.Error("expected a token in development")
	}
	if len(body) != 1 {
		t.Errorf("unexpected fields in response: %v", body)
	}
}
