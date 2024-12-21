package handlers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
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
