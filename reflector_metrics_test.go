package llama

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReflectorMetricsRegistration(t *testing.T) {
	RegisterReflectorPrometheus()

	reflectorPacketsReceived.Inc()
	reflectorPacketsReflected.Inc()
	reflectorPacketsBadData.Inc()
	reflectorPacketsThrottled.Inc()
	reflectorTosChanges.Inc()
	reflectorUp.Set(1)
}

func TestReflectorAPIHandlers(t *testing.T) {
	api := NewReflectorAPI(":0")

	req := httptest.NewRequest("GET", "/status", nil)
	w := httptest.NewRecorder()
	api.StatusHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "ok" {
		t.Errorf("expected 'ok', got '%s'", w.Body.String())
	}
}

func TestReflectorAPIEndpoints(t *testing.T) {
	api := NewReflectorAPI(":0")

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	api.PromHandler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}
