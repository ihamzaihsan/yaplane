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
