package main

import (
"fmt"
errorHandler "forum/Error"
database "forum/database"
handlers "forum/handlers"
middleware "forum/middleware"
"log"
"net/http"
"time"
)

func main() {
err := database.InitDB()
if err != nil {
log.Fatalf("Failed to initialize the database: %v", err)
}
defer func() {
if err := database.DBInstance.DB.Close(); err != nil {
log.Fatal("Error closing the database:", err)
}
}()

fs := http.FileServer(http.Dir("static"))
http.Handle("/static/", http.StripPrefix("/static/", fs))

rateLimiter := middleware.NewRateLimiter(1 * time.Minute)

http.Handle("/", rateLimiter.RateLimitMiddleware(http.HandlerFunc(handlers.ServeMainForum)))
http.Handle("/login", rateLimiter.RateLimitMiddleware(http.HandlerFunc(handlers.ServeLogin)))
http.Handle("/register", rateLimiter.RateLimitMiddleware(http.HandlerFunc(handlers.ServeRegister)))
http.Handle("/profile", rateLimiter.RateLimitMiddleware(http.HandlerFunc(handlers.ServeProfile)))
http.Handle("/logout", rateLimiter.RateLimitMiddleware(http.HandlerFunc(handlers.ServeLogout)))
http.Handle("/post/", rateLimiter.RateLimitMiddleware(http.HandlerFunc(handlers.ServeIndividualPost)))
http.Handle("/posts/create", rateLimiter.RateLimitMiddleware(http.HandlerFunc(handlers.ServePost)))
http.Handle("/add-comment", rateLimiter.RateLimitMiddleware(http.HandlerFunc(handlers.AddComment)))
http.Handle("/likeDislike", rateLimiter.RateLimitMiddleware(http.HandlerFunc(handlers.LikeDislike)))
http.Handle("/likeDislike/comment", rateLimiter.RateLimitMiddleware(http.HandlerFunc(handlers.LikeDislikeComment)))
http.Handle("/post/myPosts", rateLimiter.RateLimitMiddleware(http.HandlerFunc(handlers.ServeUserPosts)))
http.Handle("/post/likedPosts", rateLimiter.RateLimitMiddleware(http.HandlerFunc(handlers.ServeLikedPosts)))
http.Handle("/delete-post", rateLimiter.RateLimitMiddleware(http.HandlerFunc(handlers.ServeDeletePost)))
http.Handle("/post/comment", rateLimiter.RateLimitMiddleware(http.HandlerFunc(handlers.AddComment)))

http.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
errorHandler.ServeError(w, r, http.StatusInternalServerError)
})

fmt.Println("Starting server on :8080")
err = http.ListenAndServeTLS("0.0.0.0:8080", "HTTPS/server.crt", "HTTPS/server.key", nil)
if err != nil {
log.Fatal("Server failed to start:", err)
}
}