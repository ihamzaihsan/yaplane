package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	database "forum/database"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const oauthLifetime = 10 * time.Minute

type oauthProvider struct {
	clientID, secret, authorize, token, user, emails, scope string
}

type OAuth struct {
	baseURL   string
	providers map[string]oauthProvider
	key       [32]byte
	err       error
	client    *http.Client
}

func oauthSettings() (string, map[string]oauthProvider, error) {
	providers := map[string]oauthProvider{
		"google": {authorize: "https://accounts.google.com/o/oauth2/v2/auth", token: "https://oauth2.googleapis.com/token", user: "https://openidconnect.googleapis.com/v1/userinfo", scope: "openid email profile"},
		"github": {authorize: "https://github.com/login/oauth/authorize", token: "https://github.com/login/oauth/access_token", user: "https://api.github.com/user", emails: "https://api.github.com/user/emails", scope: "user:email"},
	}
	for name, p := range providers {
		prefix := strings.ToUpper(name)
		p.clientID, p.secret = os.Getenv(prefix+"_CLIENT_ID"), os.Getenv(prefix+"_CLIENT_SECRET")
		if p.clientID == "" && p.secret == "" {
			delete(providers, name)
			continue
		}
		if p.clientID == "" || p.secret == "" {
			return "", nil, fmt.Errorf("%s requires both a client ID and secret", prefix)
		}
		providers[name] = p
	}
	base := os.Getenv("OAUTH_BASE_URL")
	if len(providers) != 0 {
		u, err := url.Parse(base)
		if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" {
			return "", nil, errors.New("OAUTH_BASE_URL must be the forum's HTTPS origin")
		}
	}
	return strings.TrimSuffix(base, "/"), providers, nil
}

func NewOAuth() *OAuth {
	o := &OAuth{client: &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
	o.baseURL, o.providers, o.err = oauthSettings()
	if _, err := rand.Read(o.key[:]); err != nil {
		o.err = err
	}
	if o.err != nil {
		log.Printf("OAuth configuration: %v", o.err)
	}
	return o
}

func oauthCookie(provider, value string, age int) *http.Cookie {
	c := &http.Cookie{Name: "__Host-oauth-" + provider, Value: value, Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: age}
	if age > 0 {
		c.Expires = time.Now().Add(oauthLifetime)
	} else {
		c.Expires = time.Unix(1, 0)
	}
	return c
}

func (o *OAuth) signature(provider, value string) string {
	h := hmac.New(sha256.New, o.key[:])
	h.Write([]byte(provider + ":" + value))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func (o *OAuth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Split(strings.TrimPrefix(r.URL.Path, "/auth/"), "/")
	if len(path) > 2 || (len(path) == 2 && path[1] != "callback") {
		http.NotFound(w, r)
		return
	}
	name := path[0]
	if name != "google" && name != "github" {
		http.NotFound(w, r)
		return
	}
	p, enabled := o.providers[name]
	if o.err != nil || !enabled {
		http.Error(w, "This sign-in provider is not configured. Use email and password instead.", http.StatusServiceUnavailable)
		return
	}
	// Use a configured origin, never an untrusted Host header, for redirect URIs.
	u, _ := url.Parse(o.baseURL)
	if r.TLS == nil || !strings.EqualFold(r.Host, u.Host) {
		http.Error(w, "Sign in from the configured forum HTTPS address.", http.StatusBadRequest)
		return
	}
	redirect := o.baseURL + "/auth/" + name + "/callback"
	if len(path) == 1 {
		var random [64]byte
		if _, err := rand.Read(random[:]); err != nil {
			http.Error(w, "Unable to start sign-in", 500)
			return
		}
		state := base64.RawURLEncoding.EncodeToString(random[:32])
		verifier := base64.RawURLEncoding.EncodeToString(random[32:])
		value := state + "." + verifier + "." + strconv.FormatInt(time.Now().Unix(), 10)
		http.SetCookie(w, oauthCookie(name, value+"."+o.signature(name, value), int(oauthLifetime.Seconds())))
		challenge := sha256.Sum256([]byte(verifier))
		query := url.Values{"client_id": {p.clientID}, "redirect_uri": {redirect}, "response_type": {"code"}, "scope": {p.scope}, "state": {state}, "code_challenge": {base64.RawURLEncoding.EncodeToString(challenge[:])}, "code_challenge_method": {"S256"}}
		http.Redirect(w, r, p.authorize+"?"+query.Encode(), http.StatusFound)
		return
	}
	// Clear the short-lived browser binding on every callback, including failures.
	http.SetCookie(w, oauthCookie(name, "", -1))
	cookie, err := r.Cookie("__Host-oauth-" + name)
	parts := []string{}
	if err == nil {
		parts = strings.Split(cookie.Value, ".")
	}
	valid := len(parts) == 4
	if valid {
		issued, err := strconv.ParseInt(parts[2], 10, 64)
		age := time.Now().Unix() - issued
		valid = err == nil && age >= 0 && age < int64(oauthLifetime.Seconds()) &&
			hmac.Equal([]byte(parts[3]), []byte(o.signature(name, strings.Join(parts[:3], ".")))) &&
			len(parts[0]) == 43 && len(parts[1]) == 43 && hmac.Equal([]byte(parts[0]), []byte(r.URL.Query().Get("state")))
	}
	if !valid {
		oauthFailure(w, r, "Sign-in expired or could not be verified. Please try again.")
		return
	}
	if r.URL.Query().Get("error") != "" || r.URL.Query().Get("code") == "" {
		oauthFailure(w, r, "Sign-in was cancelled or no authorization code was returned.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	subject, email, username, err := o.identity(ctx, p, redirect, r.URL.Query().Get("code"), parts[1])
	if err != nil {
		log.Printf("Verify %s sign-in: %v", name, err)
		oauthFailure(w, r, "Unable to verify your account. Ensure your provider has a verified email and try again.")
		return
	}
	email, err = database.OAuthUser(name, subject, email, username)
	if errors.Is(err, database.ErrEmailRegistered) {
		oauthFailure(w, r, "This email already has an account. Use its original sign-in method.")
		return
	}
	if err != nil {
		http.Error(w, "Unable to create the account", 500)
		return
	}
	if err := startSession(w, email); err != nil {
		http.Error(w, "Unable to create a session", 500)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func oauthFailure(w http.ResponseWriter, r *http.Request, message string) {
	http.Redirect(w, r, "/login?"+url.Values{"error": {message}}.Encode(), http.StatusSeeOther)
}

func (o *OAuth) requestJSON(ctx context.Context, method, endpoint, token string, form url.Values, result interface{}) error {
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	r, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return err
	}
	r.Header.Set("Accept", "application/json")
	r.Header.Set("User-Agent", "Yaplane-Community")
	if form != nil {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := o.client.Do(r)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("provider returned HTTP %d", response.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(result)
}

func (o *OAuth) identity(ctx context.Context, p oauthProvider, redirect, code, verifier string) (string, string, string, error) {
	var token struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Error       string `json:"error"`
	}
	err := o.requestJSON(ctx, "POST", p.token, "", url.Values{"grant_type": {"authorization_code"}, "client_id": {p.clientID}, "client_secret": {p.secret}, "redirect_uri": {redirect}, "code": {code}, "code_verifier": {verifier}}, &token)
	if err != nil || token.Error != "" || token.AccessToken == "" || !strings.EqualFold(token.TokenType, "bearer") {
		return "", "", "", errors.New("provider did not issue a bearer token")
	}
	var user struct {
		Subject  string `json:"sub"`
		ID       int64  `json:"id"`
		Email    string `json:"email"`
		Verified bool   `json:"email_verified"`
		Name     string `json:"name"`
		Login    string `json:"login"`
	}
	if err := o.requestJSON(ctx, "GET", p.user, token.AccessToken, nil, &user); err != nil {
		return "", "", "", err
	}
	if p.emails != "" {
		if user.ID <= 0 {
			return "", "", "", errors.New("missing GitHub identity")
		}
		user.Subject = strconv.FormatInt(user.ID, 10)
		user.Name = user.Login
		var emails []struct {
			Email    string `json:"email"`
			Primary  bool   `json:"primary"`
			Verified bool   `json:"verified"`
		}
		if err := o.requestJSON(ctx, "GET", p.emails, token.AccessToken, nil, &emails); err != nil {
			return "", "", "", err
		}
		user.Email, user.Verified = "", false
		for _, email := range emails {
			if email.Primary && email.Verified {
				user.Email, user.Verified = email.Email, true
				break
			}
		}
	}
	user.Email = strings.ToLower(strings.TrimSpace(user.Email))
	if user.Subject == "" || len(user.Subject) > 255 || !user.Verified || !validEmail(user.Email) {
		return "", "", "", errors.New("missing identity or verified email")
	}
	return user.Subject, user.Email, user.Name, nil
}
