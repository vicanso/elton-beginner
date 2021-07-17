---
description: redis是大部分系统缓存的基石，合理的使用缓存能大幅度提升系统性能
---

# 缓存 

缓存在系统中的应用主要有两类：一、利用缓存来提升系统性能；二、利用缓存来临时缓存业务数据。在高访问量高并发的系统中，利用缓存提升性能时还需要考虑缓存穿透、击穿等场景。

## redis

redis是缓存的首选方案，下面来讲解在使用redis时应该考虑的要求。

- `性能统计`：能针对各请求计算处理时长
- `出错统计`：提交便利的方式记录出错处理
- `熔断控制`：提供熔断控制手段，方便根据系统运行状态熔断redis调用服务

## redis配置

redis连接通过uri连接串的形式配置，下面是配置的处理逻辑：

默认配置文件：
```yaml
# redis 配置
redis:
  # 可以配置为下面的形式，则从env中获取REDIS_URI对应的字符串来当redis连接串
  # uri: REDIS_URI
  # uri: redis://:pass@127.0.0.1:6379/?slow=200ms&maxProcessing=1000
  uri: redis://127.0.0.1:6379/?slow=200ms&maxProcessing=1000
```

生产配置文件：
```yaml
redis:
  # 从env中读取REDIS_URI的值
  uri: REDIS_URI
```

配置定义：
```go
	// RedisConfig redis配置
	RedisConfig struct {
		// 连接地址
		Addrs []string `validate:"required,dive,hostname_port"`
		// 用户名
		Username string
		// 密码
		Password string
		// 慢请求时长
		Slow time.Duration `validate:"required"`
		// 最大的正在处理请求量
		MaxProcessing uint32 `validate:"required"`
		// 连接池大小
		PoolSize int
		// sentinel模式下使用的master name
		Master string
	}
```

获取redis配置，从配置文件或env中读取之后，将uri连接串转换为对应的struct：

```go
// MustGetRedisConfig 获取redis的配置
func MustGetRedisConfig() *RedisConfig {
	prefix := "redis."
	uri := defaultViperX.GetStringFromENV(prefix + "uri")
	uriInfo, err := url.Parse(uri)
	if err != nil {
		panic(err)
	}
	// 获取密码
	password, _ := uriInfo.User.Password()
	username := uriInfo.User.Username()

	query := uriInfo.Query()
	// 获取slow设置的时间间隔
	slowValue := query.Get("slow")
	slow := 100 * time.Millisecond
	if slowValue != "" {
		slow, err = time.ParseDuration(slowValue)
		if err != nil {
			panic(err)
		}
	}

	// 获取最大处理数的配置
	maxProcessing := 1000
	maxValue := query.Get("maxProcessing")
	if maxValue != "" {
		maxProcessing, err = strconv.Atoi(maxValue)
		if err != nil {
			panic(err)
		}
	}

	// 转换失败则为0
	poolSize, _ := strconv.Atoi(query.Get("poolSize"))

	redisConfig := &RedisConfig{
		Addrs:         strings.Split(uriInfo.Host, ","),
		Username:      username,
		Password:      password,
		Slow:          slow,
		MaxProcessing: uint32(maxProcessing),
		PoolSize:      poolSize,
		Master:        query.Get("master"),
	}

	mustValidate(redisConfig)
	return redisConfig
}
```

## redis模块

redis的driver选择[go-redis](https://github.com/go-redis/redis)，它提供支持三种模式的Client(单实例、sentinel以及cluster)，Limiter提供了Allow方法用于熔断，ReportResult方法用于记录出错，Hook提供了四个hook的处理(BeforeProcess、AfterProcess、BeforeProcessPipeline以及AfterProcessPipeline)，能统计各请求的处理时间、命令以及结果。

此模块的主要实现是redisHook，它包括了hook与limiter的实现，主要用于记录请求处理的时长以及熔断，具体实现如下：

```go
// 对于慢或出错请求输出日志并写入influxdb
func (rh *redisHook) logSlowOrError(ctx context.Context, cmd, err string) {
	t := ctx.Value(startedAtKey).(*time.Time)
	d := time.Since(*t)
	if d > rh.slow || err != "" {
		log.Default().Info().
			Str("category", "redisSlowOrErr").
			Str("cmd", cmd).
			Str("use", d.String()).
			Str("error", err).
			Msg("")
	}
}

// BeforeProcess redis处理命令前的hook函数
func (rh *redisHook) BeforeProcess(ctx context.Context, cmd redis.Cmder) (context.Context, error) {
	t := time.Now()
	ctx = context.WithValue(ctx, startedAtKey, &t)
	rh.processing.Inc()
	rh.total.Inc()
	return ctx, nil
}

// AfterProcess redis处理命令后的hook函数
func (rh *redisHook) AfterProcess(ctx context.Context, cmd redis.Cmder) error {
	// allow返回error时也触发
	message := ""
	err := cmd.Err()
	if err != nil {
		message = err.Error()
	}
	rh.logSlowOrError(ctx, cmd.FullName(), message)
	rh.processing.Dec()
	return nil
}

// BeforeProcessPipeline redis pipeline命令前的hook函数
func (rh *redisHook) BeforeProcessPipeline(ctx context.Context, cmds []redis.Cmder) (context.Context, error) {
	t := time.Now()
	ctx = context.WithValue(ctx, startedAtKey, &t)
	rh.pipeProcessing.Inc()
	rh.total.Inc()
	return ctx, nil
}

// AfterProcessPipeline redis pipeline命令后的hook函数
func (rh *redisHook) AfterProcessPipeline(ctx context.Context, cmds []redis.Cmder) error {
	// allow返回error时也触发
	cmdSb := new(strings.Builder)
	message := ""
	for index, cmd := range cmds {
		if index != 0 {
			cmdSb.WriteString(",")
		}
		cmdSb.WriteString(cmd.Name())
		err := cmd.Err()
		if err != nil {
			message += err.Error()
		}
	}
	rh.logSlowOrError(ctx, cmdSb.String(), message)
	rh.pipeProcessing.Dec()
	return nil
}

// getProcessingAndTotal 获取正在处理中的请求与总请求量
func (rh *redisHook) getProcessingAndTotal() (uint32, uint32, uint64) {
	processing := rh.processing.Load()
	pipeProcessing := rh.pipeProcessing.Load()
	total := rh.total.Load()
	return processing, pipeProcessing, total
}

// Allow 是否允许继续执行redis
func (rh *redisHook) Allow() error {
	// 如果处理请求量超出，则不允许继续请求
	if rh.processing.Load()+rh.pipeProcessing.Load() > rh.maxProcessing {
		return ErrRedisTooManyProcessing
	}
	return nil
}

// ReportResult 记录结果
func (*redisHook) ReportResult(result error) {
	// allow返回error时不触发
	if result != nil && !RedisIsNilError(result) {
		log.Default().Error().
			Str("category", "redisProcessFail").
			Err(result).
			Msg("")
	}
}
```

## cache模块

redis模块提供了性能统计、熔断等手段，通过redis client可以使用redis提供的各类丰富命令实现各种缓存，[go-cache](https://github.com/vicanso/go-cache)提供了几类常用的缓存方式，可以参考使用。下面是使用go-cache与lruttl初始化的几种常用缓存。

```go
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
```

缓存模块中提供了常用的redis缓存实例，此实例提供了几类常用的缓存函数，但都必须指定缓存时间，如果不指定则使用默认缓存时间，因为对于缓存来讲就算再长的有效期也应该有，而不是不指定。还提供了snappy压缩的缓存实例，可对于较大的数据执行snappy压缩，以及基于内存的lru ttl缓存。