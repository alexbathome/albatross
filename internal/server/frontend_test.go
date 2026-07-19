package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestFrontend(t *testing.T) {
	dist := fstest.MapFS{
		"index.html":    {Data: []byte("<html>app shell</html>")},
		"assets/app.js": {Data: []byte("console.log('app')")},
	}
	h := Frontend(dist)

	tests := []struct {
		desc       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{"root serves index", "/", http.StatusOK, "app shell"},
		{"built asset served as-is", "/assets/app.js", http.StatusOK, "console.log"},
		{"client-side route falls back to index", "/holes/66", http.StatusOK, "app shell"},
		{"unmatched api path stays 404", "/api/nope", http.StatusNotFound, ""},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if rec.Code != tc.wantStatus {
				t.Fatalf("GET %s status = %d, want %d", tc.path, rec.Code, tc.wantStatus)
			}
			if tc.wantBody != "" && !strings.Contains(rec.Body.String(), tc.wantBody) {
				t.Errorf("GET %s body = %q, want it to contain %q", tc.path, rec.Body.String(), tc.wantBody)
			}
		})
	}
}

func TestFrontend_NotBuilt(t *testing.T) {
	h := Frontend(fstest.MapFS{".gitkeep": {Data: []byte{}}})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET / with no build status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
