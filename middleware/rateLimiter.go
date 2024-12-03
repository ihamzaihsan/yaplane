package middleware

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	requestBurst    = 20
	maxRateClients  = 4096
	clientRetention = 10 * time.Minute
)

type ipLimiter struct {
	tokens       float64
	updated      time.Time
	blockedUntil time.Time
}

type RateLimiter struct {
	mux         sync.Mutex
	clients     map[string]*ipLimiter
	cooldown    time.Duration
	nextCleanup time.Time
	now         func() time.Time
}

func NewRateLimiter(cooldown time.Duration) *RateLimiter {
	if cooldown <= 0 {
		cooldown = time.Minute
	}
	return &RateLimiter{clients: make(map[string]*ipLimiter), cooldown: cooldown, now: time.Now}
}

// allow uses a token bucket: one request per second, with room for a page-load burst.
// Cleanup is lazy and the map is bounded, so there is no permanent cleanup goroutine.
func (rl *RateLimiter) allow(ip string) time.Duration {
	rl.mux.Lock()
	defer rl.mux.Unlock()
	now := rl.now()
	if !now.Before(rl.nextCleanup) {
		for key, client := range rl.clients {
			if now.Sub(client.updated) >= clientRetention && !now.Before(client.blockedUntil) {
				delete(rl.clients, key)
			}
		}
		rl.nextCleanup = now.Add(time.Minute)
	}
	client := rl.clients[ip]
	if client == nil {
		if len(rl.clients) >= maxRateClients {
			return rl.cooldown
		}
		client = &ipLimiter{tokens: requestBurst, updated: now}
		rl.clients[ip] = client
	}
	if now.Before(client.blockedUntil) {
		return client.blockedUntil.Sub(now)
	}
	client.tokens = math.Min(requestBurst, client.tokens+now.Sub(client.updated).Seconds())
	client.updated = now
	if client.tokens < 1 {
		client.blockedUntil = now.Add(rl.cooldown)
		return rl.cooldown
	}
	client.tokens--
	return 0
}

func (rl *RateLimiter) RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}
		// Do not trust user-supplied forwarding headers as client identities.
		if parsed := net.ParseIP(ip); parsed != nil {
			ip = parsed.String()
		}
		if retry := rl.allow(ip); retry > 0 {
			w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(retry.Seconds()))))
			http.Error(w, "Too many requests, please try again later.", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
