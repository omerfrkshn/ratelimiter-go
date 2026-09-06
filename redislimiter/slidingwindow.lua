-- Atomic sliding-window-log rate limit check backed by a Redis sorted set.
-- Running this as a single script (rather than separate ZREMRANGEBYSCORE /
-- ZCARD / ZADD calls from the client) is what makes it safe for multiple
-- app instances to share one limit without a race between "check" and "add".
--
-- KEYS[1] = redis key for this rate-limit key (e.g. "ratelimit:user-42")
-- ARGV[1] = now, unix nanoseconds (score for the new entry)
-- ARGV[2] = window start, unix nanoseconds (now - period); entries older
--           than this no longer count against the limit
-- ARGV[3] = rate: max requests allowed per period
-- ARGV[4] = key TTL in seconds, so an idle key expires instead of
--           lingering in Redis forever
-- ARGV[5] = unique member suffix (client-generated), so two requests
--           landing on the same nanosecond timestamp don't collide as
--           the same sorted-set member
local key = KEYS[1]
local now = tonumber(ARGV[1])
local windowStart = tonumber(ARGV[2])
local rate = tonumber(ARGV[3])
local ttl = tonumber(ARGV[4])
local member = now .. "-" .. ARGV[5]

redis.call("ZREMRANGEBYSCORE", key, "-inf", windowStart)

local count = redis.call("ZCARD", key)
if count >= rate then
    return 0
end

redis.call("ZADD", key, now, member)
redis.call("EXPIRE", key, math.ceil(ttl))
return 1
