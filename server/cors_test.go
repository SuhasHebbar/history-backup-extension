package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCORS_StatusEndpoint(t *testing.T) {
	const testOrigin = "chrome-extension://mnfpggahldoheaiddleppbafdgjeondn"
	h := buildHandler(newTestDB(t), []string{testOrigin})

	assertHeader := func(t *testing.T, rr *httptest.ResponseRecorder, header, want string) {
		t.Helper()
		if got := rr.Header().Get(header); got != want {
			t.Errorf("%s: got %q, want %q", header, got, want)
		}
	}
	assertNoHeader := func(t *testing.T, rr *httptest.ResponseRecorder, header string) {
		t.Helper()
		if got := rr.Header().Get(header); got != "" {
			t.Errorf("%s: expected absent, got %q", header, got)
		}
	}

	t.Run("no origin — no ACAO", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/status", nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status: got %d, want 200", rr.Code)
		}
		assertNoHeader(t, rr, "Access-Control-Allow-Origin")
	})

	t.Run("matching origin — ACAO reflects origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/status", nil)
		req.Header.Set("Origin", testOrigin)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status: got %d, want 200", rr.Code)
		}
		assertHeader(t, rr, "Access-Control-Allow-Origin", testOrigin)
	})

	t.Run("non-matching origin — no ACAO", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/status", nil)
		req.Header.Set("Origin", "http://evil.example.com")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status: got %d, want 200", rr.Code)
		}
		assertNoHeader(t, rr, "Access-Control-Allow-Origin")
	})

	t.Run("preflight GET (allowed) — ACAO and ACAM present", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/status", nil)
		req.Header.Set("Origin", testOrigin)
		req.Header.Set("Access-Control-Request-Method", http.MethodGet)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		assertHeader(t, rr, "Access-Control-Allow-Origin", testOrigin)
		acam := rr.Header().Get("Access-Control-Allow-Methods")
		if !strings.Contains(acam, http.MethodGet) {
			t.Errorf("Access-Control-Allow-Methods: got %q, want it to contain GET", acam)
		}
	})

	t.Run("preflight PUT (disallowed) — no ACAO", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/status", nil)
		req.Header.Set("Origin", testOrigin)
		req.Header.Set("Access-Control-Request-Method", http.MethodPut)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		assertNoHeader(t, rr, "Access-Control-Allow-Origin")
	})

	t.Run("POST /status — 405", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/status", nil)
		req.Header.Set("Origin", testOrigin)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("status: got %d, want 405", rr.Code)
		}
	})
}
