package redis

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/redis/go-redis/extra/rediscensus/v9"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	Forever = time.Duration(-1)

	TTLNoExpire = -1
	TTLNoKey    = -2
)

var (
	ErrExpireNotExistOrTimeout = errors.New(
		"key does not exist or does not have an associated timeout")

	ErrNotFound = redis.Nil
	ErrNoTTL    = errors.New("No ttl")
)

// MVal is structure for redis multiple return values
type MVal struct {
	Valid bool
	Value []byte
}

type Redis interface {
	Set(ctx context.Context, key string, val []byte, ttl time.Duration, zip bool) error

	// Expire set a expire time to a key.
	Expire(ctx context.Context, key string, ttl time.Duration) error

	// Get gets the value of a key
	Get(ctx context.Context, key string, zip bool) (val []byte, err error)

	// Del Removes the specified keys and return the number of keys that were removed.
	// A key is ignored if it does not exist.
	Del(ctx context.Context, keys ...string) (int, error)

	// Incr Increments the number stored at key by one. If the key does not exist, it is set to 0 before performing the operation.
	Incr(ctx context.Context, key string) (int64, error)

	// Exists Returns if the key exists.
	Exists(ctx context.Context, key string) (int64, error)

	// TTL returns key's ttl in terms of second
	TTL(ctx context.Context, key string) (int, error)

	// Name return redis name
	Name() string

	// Rename rename the key from old to new
	Rename(ctx context.Context, oldKey, newKey string) error

	// MGet gets values of a set of keys
	// If key does not exist, you will not get ErrNotFound
	// You will get false value in `Valid` field in return MVal
	MGet(ctx context.Context, keys []string) ([]MVal, error)

	// HMGet return a map of field names to their values, with given key
	HMGet(ctx context.Context, key string, fields []string, removeNil bool) (map[string]interface{}, error)
}

func ConnectRedisCluster(addr, username, password string) (*redis.ClusterClient, error) {
	ctx := context.Background()

	config := zap.NewProductionConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	logger, err := config.Build()
	if err != nil {
		panic(err)
	}
	sugar := logger.Sugar()
	defer func() {
		if err := logger.Sync(); err != nil {
			// catch path stdout/stderr bug of zap package
			// https://github.com/uber-go/zap/issues/880
			if _, ok := err.(*os.PathError); !ok {
				logger.Error("logger sync failed, err")
			}
		}
	}()

	options := &redis.ClusterOptions{
		Addrs:    []string{addr},
		Username: username,
		Password: password,

		NewClient: func(opt *redis.Options) *redis.Client {
			node := redis.NewClient(opt)
			node.AddHook(rediscensus.NewTracingHook())
			return node
		},

		MaxRetries:      3,
		MinRetryBackoff: 1 * time.Second,
		MaxRetryBackoff: 2 * time.Second,

		DialTimeout:  2 * time.Second,
		ReadTimeout:  1500 * time.Millisecond,
		WriteTimeout: 1500 * time.Millisecond,

		ConnMaxIdleTime: 240 * time.Second,
	}

	rdb := redis.NewClusterClient(options)
	rdb.AddHook(rediscensus.NewTracingHook())

	_, err = rdb.Ping(ctx).Result()
	if err != nil {
		sugar.Errorw(
			"fail to connect to redis cluster",
			"redisAddr", addr,
			"redisUser", username,
			"err", err,
		)
		panic(err)
	}

	sugar.Desugar().Info("redis cluster connected")
	return rdb, nil
}

func ConnectRedis(addr, username, password string) (*redis.Client, error) {
	ctx := context.Background()

	config := zap.NewProductionConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	logger, err := config.Build()
	if err != nil {
		panic(err)
	}
	sugar := logger.Sugar()
	defer func() {
		if err := logger.Sync(); err != nil {
			// catch path stdout/stderr bug of zap package
			// https://github.com/uber-go/zap/issues/880
			if _, ok := err.(*os.PathError); !ok {
				logger.Error("logger sync failed, err")
			}
		}
	}()

	// Define Redis client options
	options := &redis.Options{
		Addr:     addr,     // Redis server address
		Password: password, // No password set
		DB:       0,        // Use default DB

		// Optional timeout settings
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}

	// Create a new Redis client
	rdb := redis.NewClient(options)

	// Ping the Redis server to check the connection
	_, err = rdb.Ping(ctx).Result()
	if err != nil {
		sugar.Errorw(
			"fail to connect to redis cluster",
			"redisAddr", addr,
			"redisUser", username,
			"err", err,
		)
		panic(err)
	}

	sugar.Desugar().Info("redis instance connected")
	return rdb, nil
}
