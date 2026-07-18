package github

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGithub_Authenticate(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   string
		wantErr    bool
		wantUser   User
	}{
		{
			name:       "valid token",
			wantErr:    false,
			statusCode: http.StatusOK,
			response:   `{"login": "torvalds", "name": "Linus Torvalds"}`,
			wantUser:   User{Login: "torvalds", Name: "Linus Torvalds"},
		},
		{
			name:       "invalid token",
			wantErr:    true,
			statusCode: http.StatusUnauthorized,
			response:   `{}`,
			wantUser:   User{},
		},
		{
			name:       "request failed",
			wantErr:    true,
			statusCode: 0,
			response:   `{}`,
			wantUser:   User{},
		},
		{
			name:       "decoding response failed",
			wantErr:    true,
			statusCode: http.StatusOK,
			response:   `invalid json`,
			wantUser:   User{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.statusCode == 0 {
					conn, _, err := w.(http.Hijacker).Hijack()
					if err == nil {
						_ = conn.Close()
					}
					return
				}
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.response))
			}))
			defer server.Close()

			g := github{userEndpoint: server.URL}
			user, err := g.Authenticate("token")

			if (err != nil) != tt.wantErr {
				t.Logf("Github.Authenticate() error = %v, wantErr %v", err, tt.wantErr)
				t.Fail()
			}
			if err == nil && user.Login != tt.wantUser.Login {
				t.Logf("Github.Authenticate() = %v, want %v", user, tt.wantUser)
				t.Fail()
			}
		})
	}
}

func TestSelfHostedAllowsAuthenticatedGithubUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"login":"octocat"}`))
	}))
	defer server.Close()

	g := newGithub("client", "secret", "https://example.com/oauth-callback")
	g.userEndpoint = server.URL
	g.allowAuthenticated = true
	user, err := g.Authenticate("token")
	if err != nil {
		t.Fatal(err)
	}
	if user.Login != "octocat" || !user.Allowed {
		t.Fatalf("unexpected user: %+v", user)
	}
}
