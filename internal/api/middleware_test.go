package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCORSMiddleware(t *testing.T) {
	if err := os.Setenv("ALLOWED_ORIGINS", "https://allowed.com,http://another.com"); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("ALLOWED_ORIGINS"); err != nil {
			t.Fatalf("failed to unset env: %v", err)
		}
	}()

	handler := CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name           string
		origin         string
		method         string
		expectedStatus int
		expectedCORS   string
	}{
		{
			name:           "Allowed Origin",
			origin:         "https://tokengoblin.com",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			expectedCORS:   "https://tokengoblin.com",
		},
		{
			name:           "Allowed Origin from Env",
			origin:         "https://allowed.com",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			expectedCORS:   "https://allowed.com",
		},
		{
			name:           "Disallowed Origin",
			origin:         "https://evil.com",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			expectedCORS:   "",
		},
		{
			name:           "Empty Origin",
			origin:         "",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			expectedCORS:   "",
		},
		{
			name:           "OPTIONS Allowed Origin",
			origin:         "https://tokengoblin.com",
			method:         http.MethodOptions,
			expectedStatus: http.StatusOK,
			expectedCORS:   "https://tokengoblin.com",
		},
		{
			name:           "OPTIONS Disallowed Origin",
			origin:         "https://evil.com",
			method:         http.MethodOptions,
			expectedStatus: http.StatusOK,
			expectedCORS:   "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/", nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code)

			corsHeaders := rr.Header().Values("Access-Control-Allow-Origin")
			if tc.expectedCORS == "" {
				assert.Empty(t, corsHeaders, "Should not emit CORS header for disallowed origins")
			} else {
				assert.Equal(t, []string{tc.expectedCORS}, corsHeaders)
			}
		})
	}
}
