package cache

import (
	"time"

	"github.com/vicanso/beginner/config"
	"github.com/vicanso/beginner/helper"
	goCache "github.com/vicanso/go-cache"
	lruttl "github.com/vicanso/lru-ttl"
)

var redisCache = newRedisCache()
var redisCacheWithCompress = newCompressRedisCache()
var redisSession = newRedisSession()
var redisConfig = config.MustGetRedisConfig()

func newRedisCache() *goCache.RedisCache {
	c := goCache.NewRedisCache(helper.RedisGetClient())
	return c
}

func newCompressRedisCache() *goCache.RedisCache {
	// 大于10KB以上的数据压缩
	// 适用于数据量较大，而且数据内容重复较多的场景
	minCompressSize := 10 * 1024
	return goCache.NewCompressRedisCache(
		helper.RedisGetClient(),
		minCompressSize,
	)
}

func newRedisSession() *goCache.RedisSession {
	ss := goCache.NewRedisSession(helper.RedisGetClient())
	// 设置前缀
	ss.SetPrefix("ss:")
	return ss
}

// 获取redis缓存实例
func GetRedisCache() *goCache.RedisCache {
	return redisCache
}

// 获取带缓存的redis缓存实现
func GetRedisCacheWithCompress() *goCache.RedisCache {
	return redisCacheWithCompress
}

// 获取redis session实例
func GetRedisSession() *goCache.RedisSession {
	return redisSession
}

// 创建指定大小与时间的lru缓存
func NewLRUCache(maxEntries int, defaultTTL time.Duration) *lruttl.Cache {
	return lruttl.New(maxEntries, defaultTTL)
}
