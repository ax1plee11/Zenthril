package spam

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/redis/go-redis/v9"
)

var (
	ErrRateLimited     = errors.New("rate_limited")
	ErrTooLong         = errors.New("message_too_long")
	ErrTooManyMentions = errors.New("too_many_mentions")
	ErrBlockedLink     = errors.New("blocked_link")
)

var blocklist = []string{
	"spam-site.example",
	"malware.example",
	"phishing.example",
}

var mentionRegex = regexp.MustCompile(`@\w+|<@[^>]+>`)

type Guard struct {
	redis *redis.Client
}

func NewGuard(rdb *redis.Client) *Guard {
	return &Guard{redis: rdb}
}

func (g *Guard) CheckMessage(ctx context.Context, userID, channelID string) error {
	bucketKey := "spam:bucket:" + userID + ":" + channelID
	blockKey := "spam:block:" + userID + ":" + channelID
	violationsKey := "spam:violations:" + userID

	blocked, err := g.redis.Exists(ctx, blockKey).Result()
	if err != nil {
		return nil
	}
	if blocked > 0 {
		return ErrRateLimited
	}

	luaScript := redis.NewScript(`
local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

local data = redis.call('HMGET', key, 'tokens', 'last_refill')
local tokens = tonumber(data[1]) or capacity
local last_refill = tonumber(data[2]) or now

local elapsed = now - last_refill
local refill = math.floor(elapsed / window * capacity)
tokens = math.min(capacity, tokens + refill)
if refill > 0 then
    last_refill = now
end

if tokens > 0 then
    tokens = tokens - 1
    redis.call('HMSET', key, 'tokens', tokens, 'last_refill', last_refill)
    redis.call('EXPIRE', key, window * 2)
    return 1
else
    return 0
end
`)

	nowMs := g.redis.Time(ctx).Val().UnixMilli()
	result, err := luaScript.Run(ctx, g.redis, []string{bucketKey}, 5, 5000, nowMs).Int()
	if err != nil {
		return nil
	}

	if result == 0 {
		g.redis.Set(ctx, blockKey, "1", 30*1000*1000*1000) //nolint:errcheck

		violations, _ := g.redis.Incr(ctx, violationsKey).Result()
		g.redis.Expire(ctx, violationsKey, 10*60*1000*1000*1000) //nolint:errcheck

		if violations >= 3 {
			g.redis.Set(ctx, blockKey, "1", 10*60*1000*1000*1000) //nolint:errcheck
			g.redis.Del(ctx, violationsKey)                        //nolint:errcheck
		}

		return ErrRateLimited
	}

	return nil
}

func (g *Guard) CheckContent(content string) error {
	if len([]rune(content)) > 4000 {
		return ErrTooLong
	}

	mentions := mentionRegex.FindAllString(content, -1)
	if len(mentions) > 5 {
		return ErrTooManyMentions
	}

	lower := strings.ToLower(content)
	for _, blocked := range blocklist {
		if strings.Contains(lower, blocked) {
			return ErrBlockedLink
		}
	}

	return nil
}

func (g *Guard) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}
