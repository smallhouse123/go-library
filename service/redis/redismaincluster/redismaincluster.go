package redismaincluster

import (
	"github.com/redis/go-redis/v9"
	"github.com/smallhouse123/go-library/service/config"
	redisService "github.com/smallhouse123/go-library/service/redis"
	"go.uber.org/fx"
)

var (
	Service = fx.Provide(NewRedisMainCluster)
)

func NewRedisMainCluster(config config.Config) redisService.Redis {
	var client *redis.Client
	addr, err := config.Get("ENVOY_REDIS_ADDRESS")
	if err != nil {
		return nil
	}
	client, err = redisService.ConnectRedis(addr.(string), "", "")
	if err != nil {
		return nil
	}
	return redisService.New("redisMainCluster", client, config)
}
