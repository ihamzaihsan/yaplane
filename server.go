package main

import (
	"crypto/tls"
	"fmt"
	errorHandler "forum/Error"
	database "forum/database"
	"forum/handlers"
	"forum/middleware"
	"forum/models"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

type serverConfig struct{ address, certFile, keyFile string }

func readServerConfig() (serverConfig, error) {
	value := func(name, fallback string) string {
		if v := os.Getenv(name); v != "" {
			return v
		}
		return fallback
	}
	c := serverConfig{value("FORUM_ADDR", ":8080"), value("TLS_CERT_FILE", "HTTPS/server.crt"), value("TLS_KEY_FILE", "HTTPS/server.key")}
	_, port, err := net.SplitHostPort(c.address)
	n, portErr := strconv.Atoi(port)
	if err != nil || portErr != nil || n < 0 || n > 65535 {
		return c, fmt.Errorf("FORUM_ADDR must be a host:port address")
	}
	return c, nil
}

func newServer(c serverConfig) *http.Server {
	mux := http.NewServeMux()
	add := func(path string, fn http.HandlerFunc, private bool, methods ...string) {
		var handler http.Handler = fn
		if private {
			handler = middleware.RequireSession(handler)
		}
		mux.Handle(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, method := range methods {
				if r.Method == method || (method == http.MethodGet && r.Method == http.MethodHead) {
					handler.ServeHTTP(w, r)
					return
				}
			}
			w.Header().Set("Allow", strings.Join(methods, ", "))
			errorHandler.ServeError(w, r, http.StatusMethodNotAllowed)
		}))
	}
	add("/", handlers.ServeMainForum, false, "GET")
	add("/login", handlers.ServeLogin, false, "GET", "POST")
	add("/register", handlers.ServeRegister, false, "GET", "POST")
	add("/auth/", handlers.NewOAuth().ServeHTTP, false, "GET")
	add("/profile", handlers.ServeProfile, true, "GET")
	add("/activity", handlers.ServeActivity, true, "GET")
	add("/notifications", handlers.ServeNotifications, true, "GET")
	add("/notifications/count", handlers.ServeNotificationCount, true, "GET")
	add("/notifications/read", handlers.ServeReadNotifications, true, "POST")
	add("/posts/edit", handlers.ServeEditPost, true, "GET", "POST")
	add("/comments/edit", handlers.ServeEditComment, true, "GET", "POST")
	add("/comments/delete", handlers.ServeDeleteComment, true, "POST")
	add("/moderation", handlers.ServeModeration, true, "GET")
	add("/moderation/action", handlers.ServeModerationAction, true, "POST")
	add("/logout", handlers.ServeLogout, false, "POST")
	add("/post/", handlers.ServeIndividualPost, false, "GET")
	add("/posts/create", handlers.ServePost, true, "GET", "POST")
	add("/add-comment", handlers.AddComment, true, "POST")
	add("/post/comment", handlers.AddComment, true, "POST")
	add("/likeDislike", handlers.LikeDislike, true, "POST")
	add("/likeDislike/comment", handlers.LikeDislikeComment, true, "POST")
	add("/post/myPosts", handlers.ServeUserPosts, true, "GET")
	add("/post/likedPosts", handlers.ServeLikedPosts, true, "GET")
	add("/delete-post", handlers.ServeDeletePost, true, "POST")
	add("/error", func(w http.ResponseWriter, r *http.Request) { errorHandler.ServeError(w, r, 500) }, false, "GET")
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.Handle("/static/uploads/", http.StripPrefix("/static/uploads/", http.HandlerFunc(serveUploadedImage)))
	return &http.Server{
		Addr:              c.address,
		Handler:           middleware.Security(middleware.NewRateLimiter(time.Minute).RateLimitMiddleware(mux)),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second,
		WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20,
		TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12, CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256, tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256, tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
		}},
	}
}

// Serve uploads with their detected image type, even if the filename ends in .html.
func serveUploadedImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var viewer *models.User
	w.Header().Set("Cache-Control", "no-store")
	if cookie, err := r.Cookie("session_token"); err == nil {
		viewer, _ = database.GetUserBySession(cookie.Value)
	}
	if err := database.CanViewUpload(path.Clean("uploads/"+r.URL.Path), viewer); err != nil {
		if err == database.ErrNotFound {
			http.NotFound(w, r)
		} else {
			http.Error(w, "Unable to read image", 500)
		}
		return
	}
	f, err := http.Dir("static/uploads").Open(r.URL.Path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	var header [512]byte
	n, err := f.Read(header[:])
	if err != nil && err != io.EOF {
		http.Error(w, "Unable to read image", 500)
		return
	}
	kind := http.DetectContentType(header[:n])
	if !strings.HasPrefix(kind, "image/") {
		http.Error(w, "Unsupported image type", 415)
		return
	}
	w.Header().Set("Content-Type", kind)
	w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'")
	http.ServeContent(w, r, info.Name(), info.ModTime(), f)
}
