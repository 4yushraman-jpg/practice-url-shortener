package middleware

import (
	"net"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimitMiddleware struct {
	RDB *redis.Client
}

// RateLimit acts as a "Factory". You give it the rules, and it builds the middleware for you.
func (h *RateLimitMiddleware) RateLimit(limit int64, window time.Duration, prefix string) func(http.HandlerFunc) http.HandlerFunc {

	// It returns the actual middleware function
	return func(next http.HandlerFunc) http.HandlerFunc {

		// Which in turn returns the standard HTTP handler
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			ip := r.Header.Get("X-Forwarded-For")
			if ip == "" {
				var err error
				ip, _, err = net.SplitHostPort(r.RemoteAddr)
				if err != nil {
					ip = r.RemoteAddr
				}
			}

			// Use the dynamic prefix to keep keys separate in Redis
			key := prefix + ":" + ip

			count, err := h.RDB.Incr(ctx, key).Result()
			if err != nil {
				next(w, r)
				return
			}

			if count == 1 {
				// Use the dynamic window
				h.RDB.Expire(ctx, key, window)
			}

			if count > limit {
				// Use the dynamic limit
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			next(w, r)
		}
	}
}
