package middleware

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type RateLimiter struct {
	limiters    map[string]*ipLimiter
	blockedIps  map[string]time.Time
	mux         sync.Mutex
	cooldown    time.Duration 
}

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func NewRateLimiter(cooldown time.Duration) *RateLimiter {
	rl := &RateLimiter{
		limiters:   make(map[string]*ipLimiter),
		blockedIps: make(map[string]time.Time),
		cooldown:   cooldown,
	}
	go rl.cleanupStaleEntries()
	return rl
}

func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mux.Lock()
	defer rl.mux.Unlock()

	if limiterEntry, exists := rl.limiters[ip]; exists {
		limiterEntry.lastSeen = time.Now() 
		return limiterEntry.limiter
	}

	limiter := rate.NewLimiter(1, 5) 
	rl.limiters[ip] = &ipLimiter{limiter: limiter, lastSeen: time.Now()}
	return limiter
}

func (rl *RateLimiter) RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr

		rl.mux.Lock()
		if unblockTime, blocked := rl.blockedIps[ip]; blocked {
			if time.Now().Before(unblockTime) {
				http.Error(w, "Too many requests, please try again later.", http.StatusTooManyRequests)
				rl.mux.Unlock()
				return
			}
			delete(rl.blockedIps, ip)
		}
		rl.mux.Unlock()

		limiter := rl.getLimiter(ip)

		if !limiter.Allow() {
			rl.mux.Lock()
			rl.blockedIps[ip] = time.Now().Add(rl.cooldown)
			rl.mux.Unlock()
			http.Error(w, "Too many requests, please try again later.", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) cleanupStaleEntries() {
	for {
		time.Sleep(10 * time.Minute)
		rl.mux.Lock()
		for ip, limiterEntry := range rl.limiters {
			if time.Since(limiterEntry.lastSeen) > 30*time.Minute {
				delete(rl.limiters, ip)
			}
		}
		for ip, unblockTime := range rl.blockedIps {
			if time.Now().After(unblockTime) {
				delete(rl.blockedIps, ip)
			}
		}
		rl.mux.Unlock()
	}
}
