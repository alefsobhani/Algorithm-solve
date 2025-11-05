package middleware

import (
	"context"
	"errors"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateConfig struct {
	Rate  float64
	Burst float64
}

type RateLimiter struct {
	client    *redis.Client
	readCfg   RateConfig
	writeCfg  RateConfig
	luaScript *redis.Script
}

func NewRateLimiter(client *redis.Client, read RateConfig, write RateConfig) *RateLimiter {
	if client == nil {
		return nil
	}
	script := redis.NewScript(tokenBucketLua)
	return &RateLimiter{client: client, readCfg: read, writeCfg: write, luaScript: script}
}

func (l *RateLimiter) Middleware(next http.Handler) http.Handler {
	if l == nil || (l.readCfg.Rate <= 0 && l.writeCfg.Rate <= 0) {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg := l.writeCfg
		scope := "write"
		if isReadMethod(r.Method) {
			cfg = l.readCfg
			scope = "read"
		}

		if cfg.Rate <= 0 || cfg.Burst <= 0 {
			next.ServeHTTP(w, r)
			return
		}

		identifier := clientIdentifier(r)
		if identifier == "" {
			identifier = "anonymous"
		}
		allowed, retryAfter, err := l.allow(r.Context(), scope, identifier, cfg)
		if err != nil {
			http.Error(w, "rate limit error", http.StatusInternalServerError)
			return
		}

		if !allowed {
			if retryAfter > 0 {
				w.Header().Set("Retry-After", formatRetryAfter(retryAfter))
			}
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (l *RateLimiter) allow(ctx context.Context, scope string, identifier string, cfg RateConfig) (bool, time.Duration, error) {
	now := time.Now()
	key := strings.Join([]string{"rl", scope, identifier}, ":")
	result, err := l.luaScript.Run(ctx, l.client, []string{key}, now.UnixMilli(), cfg.Rate, cfg.Burst, 1).Result()
	if err != nil {
		return false, 0, err
	}

	values, ok := result.([]interface{})
	if !ok || len(values) != 3 {
		return false, 0, errors.New("invalid redis response")
	}

	allowedInt, err := toInt64(values[0])
	if err != nil {
		return false, 0, err
	}
	waitSeconds, err := toFloat64(values[2])
	if err != nil {
		return false, 0, err
	}
	if allowedInt != 1 {
		return false, time.Duration(math.Ceil(waitSeconds*1000)) * time.Millisecond, nil
	}
	return true, 0, nil
}

func isReadMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func clientIdentifier(r *http.Request) string {
	if id := strings.TrimSpace(r.Header.Get("X-Client-ID")); id != "" {
		return id
	}
	if fwd := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); fwd != "" {
		parts := strings.Split(fwd, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func formatRetryAfter(d time.Duration) string {
	seconds := int(math.Ceil(d.Seconds()))
	if seconds < 1 {
		seconds = 1
	}
	return strconv.Itoa(seconds)
}

func toFloat64(v interface{}) (float64, error) {
	switch val := v.(type) {
	case int64:
		return float64(val), nil
	case float64:
		return val, nil
	case string:
		return strconv.ParseFloat(val, 64)
	default:
		return 0, errors.New("unsupported type")
	}
}

func toInt64(v interface{}) (int64, error) {
	switch val := v.(type) {
	case int64:
		return val, nil
	case float64:
		return int64(val), nil
	case string:
		parsed, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return 0, err
		}
		return parsed, nil
	default:
		return 0, errors.New("unsupported type")
	}
}

const tokenBucketLua = `
local key = KEYS[1]
local now_ms = tonumber(ARGV[1])
local rate = tonumber(ARGV[2])
local capacity = tonumber(ARGV[3])
local requested = tonumber(ARGV[4])

if rate <= 0 then
  return {1, capacity, 0}
end

local state = redis.call('HMGET', key, 'tokens', 'timestamp')
local tokens = tonumber(state[1])
local last = tonumber(state[2])

if tokens == nil then
  tokens = capacity
end
if last == nil then
  last = now_ms
end

local delta = now_ms - last
if delta < 0 then
  delta = 0
end
local refill = delta * rate / 1000
if refill > 0 then
  tokens = math.min(capacity, tokens + refill)
  last = now_ms
end

local allowed = tokens >= requested
local wait = 0
if allowed then
  tokens = tokens - requested
else
  wait = (requested - tokens) / rate
end

redis.call('HMSET', key, 'tokens', tokens, 'timestamp', last)
local ttl = math.ceil((capacity / rate) * 1000)
redis.call('PEXPIRE', key, ttl)

if allowed then
  return {1, tokens, 0}
else
  return {0, tokens, wait}
end
`
