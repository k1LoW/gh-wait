package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIsLoopbackHost(t *testing.T) {
	tests := []struct {
		host string
		want bool
	}{
		{"127.0.0.1:9248", true},
		{"127.0.0.1", true},
		{"localhost:9248", true},
		{"LOCALHOST:9248", true},
		{"[::1]:9248", true},
		{"[::1]", true},
		{"127.0.0.2:9248", true},
		{"", false},
		{"attacker.example:9248", false},
		{"localhost.attacker.example:9248", false},
		{"192.168.0.1:9248", false},
		{"0.0.0.0:9248", false},
	}
	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			if got := isLoopbackHost(tt.host); got != tt.want {
				t.Errorf("isLoopbackHost(%q) = %v, want %v", tt.host, got, tt.want)
			}
		})
	}
}

func TestWithLoopbackHostOnly(t *testing.T) {
	handler := WithLoopbackHostOnly(NewHandler(NewState(0)))

	tests := []struct {
		host string
		want int
	}{
		{"127.0.0.1:9248", http.StatusOK},
		{"localhost:9248", http.StatusOK},
		{"rebind.attacker.example:9248", http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/_/api/rules", nil)
			req.Host = tt.host
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code != tt.want {
				t.Errorf("status = %d, want %d", w.Code, tt.want)
			}
		})
	}
}

func TestCrossOriginStateChangingRequestRejected(t *testing.T) {
	body := `{"id":"r1","type":"pr","repo":"o/r","number":1,"conditions":["approved"],"action":"open","status":"watching"}`

	tests := []struct {
		name    string
		headers map[string]string
		want    int
	}{
		{"cli without browser headers", nil, http.StatusCreated},
		{"same origin", map[string]string{"Sec-Fetch-Site": "same-origin"}, http.StatusCreated},
		{"cross site", map[string]string{"Sec-Fetch-Site": "cross-site"}, http.StatusForbidden},
		{"cross origin without Sec-Fetch-Site", map[string]string{"Origin": "https://attacker.example"}, http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(NewState(0))
			req := httptest.NewRequest(http.MethodPost, "/_/api/rules", strings.NewReader(body))
			req.Header.Set("Content-Type", "text/plain")
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code != tt.want {
				t.Errorf("status = %d, want %d", w.Code, tt.want)
			}
		})
	}
}
