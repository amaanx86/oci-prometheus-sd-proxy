package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBearerAuth(t *testing.T) {
	const validToken = "super-secret-token"

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	handler := BearerAuth(validToken, next)

	cases := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{
			name:       "valid bearer token",
			authHeader: "Bearer " + validToken,
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing authorization header",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "wrong token",
			authHeader: "Bearer wrong-token",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "no Bearer prefix",
			authHeader: validToken,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "Basic auth scheme",
			authHeader: "Basic dXNlcjpwYXNz",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "empty token value",
			authHeader: "Bearer ",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "Bearer prefix only - no space after",
			authHeader: "Bearer",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "token with extra whitespace",
			authHeader: "Bearer " + validToken + " ",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("status: got %d, want %d", rr.Code, tc.wantStatus)
			}
		})
	}
}

// TestBearerAuth_TimingSafety verifies the handler responds with 401 for all
// invalid tokens - we can't test constant-time directly, but we confirm the
// behavior is consistent regardless of how similar the bad token is.
func TestBearerAuth_TimingSafety(t *testing.T) {
	const validToken = "aaaaaaaaaaaaaaaa"
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := BearerAuth(validToken, next)

	badTokens := []string{
		"aaaaaaaaaaaaaaab", // differs only in last char
		"baaaaaaaaaaaaaaa", // differs only in first char
		"",
		"a",
		"aaaaaaaaaaaaaaaaa", // one char longer
		"aaaaaaaaaaaaaaa",   // one char shorter
	}

	for _, bad := range badTokens {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+bad)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("token %q: got %d, want 401", bad, rr.Code)
		}
	}
}

// TestBearerAuth_NextCalledOnlyOnSuccess ensures the downstream handler is
// called exactly once on a valid request and never on an invalid one.
func TestBearerAuth_NextCalledOnlyOnSuccess(t *testing.T) {
	const token = "tok"

	calls := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
	})
	handler := BearerAuth(token, next)

	// Valid request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if calls != 1 {
		t.Errorf("next called %d times on valid request, want 1", calls)
	}

	// Invalid request
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("Authorization", "Bearer bad")
	handler.ServeHTTP(httptest.NewRecorder(), req2)
	if calls != 1 {
		t.Errorf("next called %d times total after invalid request, want still 1", calls)
	}
}
