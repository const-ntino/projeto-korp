package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestProjetoKorpReturnsNameAndCurrentUTCTime(t *testing.T) {
	fixed := time.Date(2026, 8, 3, 12, 34, 56, 0, time.FixedZone("BRT", -3*60*60))
	handler := New(func() time.Time { return fixed })

	req := httptest.NewRequest(http.MethodGet, "/projeto-korp", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q, want application/json", got)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if body["nome"] != "Projeto Korp" {
		t.Fatalf("nome = %q", body["nome"])
	}
	if body["horario"] != "2026-08-03T15:34:56Z" {
		t.Fatalf("horario = %q", body["horario"])
	}
}

func TestProjetoKorpRejectsNonGET(t *testing.T) {
	handler := New(time.Now)
	req := httptest.NewRequest(http.MethodPost, "/projeto-korp", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestHealthz(t *testing.T) {
	handler := New(time.Now)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "ok\n" {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestMetricsExposeAvailabilityAndRequestVolumeByMethodAndStatus(t *testing.T) {
	handler := New(time.Now)

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/projeto-korp", nil))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/projeto-korp", nil))

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(body, "http_server_projeto_korp_up 1") {
		t.Fatalf("availability metric not found in body:\n%s", body)
	}
	if !strings.Contains(body, `http_server_projeto_korp_requests_total{method="GET",status_code="200"} 1`) {
		t.Fatalf("GET 200 request metric not found in body:\n%s", body)
	}
	if !strings.Contains(body, `http_server_projeto_korp_requests_total{method="POST",status_code="405"} 1`) {
		t.Fatalf("POST 405 request metric not found in body:\n%s", body)
	}
}
