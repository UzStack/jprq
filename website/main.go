package main

import (
	"crypto/rand"
	"crypto/subtle"
	_ "embed"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/azimjohn/jprq/server/github"
)

var oauth github.Authenticator

//go:embed static/index.html
var html string

//go:embed static/config.json
var config string

//go:embed static/install.sh
var installer string

//go:embed static/token.html
var tokenHtml string

func main() {
	clientId := os.Getenv("GITHUB_CLIENT_ID")
	clientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
	redirectURI := os.Getenv("GITHUB_REDIRECT_URI")
	if clientId == "" || clientSecret == "" || redirectURI == "" {
		log.Fatalf("missing github client id/secret or redirect URI")
	}
	oauth = github.NewSelfHosted(clientId, clientSecret, redirectURI)

	http.HandleFunc("/", contentHandler([]byte(html), "text/html"))
	http.HandleFunc("/config.json", contentHandler([]byte(config), "application/json"))
	http.HandleFunc("/install.sh", contentHandler([]byte(installer), "text/x-shellscript"))
	http.HandleFunc("/auth", authHandler)
	http.HandleFunc("/oauth-callback", oauthCallback)

	addr := os.Getenv("JPRQ_WEBSITE_ADDR")
	if addr == "" {
		addr = "127.0.0.1:3300"
	}
	log.Printf("Listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func contentHandler(content []byte, contentType string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Write(content)
	}
}

func authHandler(w http.ResponseWriter, r *http.Request) {
	app := r.URL.Query().Get("app")
	callback := r.URL.Query().Get("callback")
	oauthURL := oauth.OAuthUrl()
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		http.Error(w, "failed to initialize authentication", http.StatusInternalServerError)
		return
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)
	parsedOAuthURL, err := url.Parse(oauthURL)
	if err != nil {
		http.Error(w, "invalid authentication configuration", http.StatusInternalServerError)
		return
	}
	query := parsedOAuthURL.Query()
	query.Set("state", state)
	parsedOAuthURL.RawQuery = query.Encode()
	oauthURL = parsedOAuthURL.String()
	http.SetCookie(w, &http.Cookie{
		Name: "jprq_oauth_state", Value: state, Path: "/", MaxAge: 300,
		HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode,
	})

	if app != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     "jprq_app",
			Value:    app,
			Path:     "/",
			MaxAge:   300,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
	}

	if callback != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     "jprq_callback",
			Value:    callback,
			Path:     "/",
			MaxAge:   300,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
	}

	http.Redirect(w, r, oauthURL, http.StatusFound)
}

func oauthCallback(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil || r.FormValue("code") == "" {
		http.Redirect(w, r, "/auth", http.StatusTemporaryRedirect)
		return
	}
	stateCookie, err := r.Cookie("jprq_oauth_state")
	if err != nil || subtle.ConstantTimeCompare([]byte(stateCookie.Value), []byte(r.FormValue("state"))) != 1 {
		http.Error(w, "invalid OAuth state", http.StatusBadRequest)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: "jprq_oauth_state", Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode,
	})
	token, err := oauth.ObtainToken(r.FormValue("code"))
	if err != nil || token == "" {
		fmt.Printf("error obtaining token: %s\n", err)
		http.Redirect(w, r, "/auth", http.StatusTemporaryRedirect)
		return
	}

	// Check if this is an app-based authentication
	appCookie, err := r.Cookie("jprq_app")
	callbackCookie, _ := r.Cookie("jprq_callback")

	if err == nil && appCookie.Value != "" {
		// Clear cookies
		http.SetCookie(w, &http.Cookie{
			Name: "jprq_app", Value: "", Path: "/", MaxAge: -1, HttpOnly: true,
		})
		http.SetCookie(w, &http.Cookie{
			Name: "jprq_callback", Value: "", Path: "/", MaxAge: -1, HttpOnly: true,
		})

		// If callback URL provided, redirect there instead of deep link.
		// Parse the URL so we preserve any pre-existing query params (e.g. state)
		// and append `token` correctly using `&` instead of a second `?`.
		if callbackCookie != nil && callbackCookie.Value != "" {
			parsed, perr := url.Parse(callbackCookie.Value)
			if perr == nil && parsed.Scheme != "" && parsed.Host != "" {
				q := parsed.Query()
				q.Set("token", token)
				parsed.RawQuery = q.Encode()
				http.Redirect(w, r, parsed.String(), http.StatusFound)
				return
			}
			fmt.Printf("invalid callback URL %q: %v\n", callbackCookie.Value, perr)
		}

		// Fall back to deep link
		switch appCookie.Value {
		case "mac", "windows", "linux":
			appURL := fmt.Sprintf("jprq://auth/callback?token=%s", token)
			http.Redirect(w, r, appURL, http.StatusFound)
		default:
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(fmt.Sprintf(tokenHtml, token)))
		}
		return
	}

	// Default: show token in web page
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; script-src 'unsafe-inline'")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Write([]byte(fmt.Sprintf(tokenHtml, token)))
}
