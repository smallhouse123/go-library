package redis

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"time"

	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
	"github.com/smallhouse123/go-library/service/config"
	"go.uber.org/zap"
)

type Impl struct {
	name   string
	client *redis.Client
	sugar  *zap.SugaredLogger
	config config.Config
}

func New(name string, client *redis.Client, config config.Config) Redis {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}

	//TODO move logger to service
	sugar := logger.Sugar()
	return &Impl{
		name:   name,
		client: client,
		sugar:  sugar,
		config: config,
	}
}

func (im *Impl) Set(ctx context.Context, key string, val []byte, expire time.Duration, zip bool) error {
	var newVal []byte
	if zip {
		buf := &bytes.Buffer{}
		writer := gzip.NewWriter(buf)
		writer.Write(val)
		writer.Flush()
		writer.Close()
		b := buf.Bytes()
		newVal = append(newVal, b...)
	} else {
		newVal = append(newVal, val...)
	}

	if expire == Forever {
		expire = 0
	}

	_, err := im.client.Set(ctx, key, newVal, expire).Result()
	if err != nil {
		im.sugar.Errorw("SET redis failed", "err", err)
	}
	return err
}

func (im *Impl) Expire(ctx context.Context, key string, ttl time.Duration) error {
	var err error
	var val bool

	if ttl == Forever {
		val, err = im.client.Persist(ctx, key).Result()
	} else {
		val, err = im.client.Expire(ctx, key, ttl).Result()
	}

	if err != nil {
		im.sugar.Errorw("EXPIRE redis failed", "err", err)
		return err
	}

	// Return value will be false if key does not exist
	// or does not have an associated timeout.
	if !val {
		return ErrExpireNotExistOrTimeout
	}
	return nil
}

func (im *Impl) Get(ctx context.Context, key string, zip bool) ([]byte, error) {
	val, err := im.client.Get(ctx, key).Bytes()
	if err != nil {
		if err != ErrNotFound {
			im.sugar.Errorw("GET redis failed", "err", err)
		}
		return nil, err
	}
	if !zip {
		return val, err
	}

	buf := bytes.NewBuffer(val)
	rb, err := gzip.NewReader(buf)
	if err != nil {
		im.sugar.Warnw("new gzip reader failed", "err", err)
		return val, nil
	}
	res, err := io.ReadAll(rb)
	rb.Close()
	return res, err
}

func (im *Impl) Del(ctx context.Context, keys ...string) (int, error) {
	if len(keys) == 0 {
		return 0, errors.New("length of keys is 0")
	}

	// Use pipeline to implement multi-key Del to prevent error CROSSSLOT
	pipe := im.client.Pipeline()
	for _, key := range keys {
		pipe.Del(ctx, key)
	}

	dels, err := pipe.Exec(ctx)
	if err != nil {
		im.sugar.Errorw("DEL redis failed", "err", err)
		return 0, err
	}

	affected := 0
	for _, del := range dels {
		affected += int(del.(*redis.IntCmd).Val())
	}

	return affected, nil
}

func (im *Impl) Incr(ctx context.Context, key string) (int64, error) {
	res, err := im.client.Incr(ctx, key).Result()
	if err != nil {
		im.sugar.Errorw("INCR redis failed", "err", err)
	}
	return res, err
}

func (im *Impl) Exists(ctx context.Context, key string) (int64, error) {
	res, err := im.client.Exists(ctx, key).Result()
	if err != nil {
		im.sugar.Errorw("EXISTS redis failed", "err", err)
	}
	return res, err
}

func (im *Impl) TTL(ctx context.Context, key string) (int, error) {

	val, err := im.client.TTL(ctx, key).Result()
	if err != nil {
		im.sugar.Errorw("TTL redis failed", "err", err)
		return 0, err
	}

	if val == TTLNoKey {
		return int(val), ErrNotFound
	} else if val == TTLNoExpire {
		return int(val), ErrNoTTL
	}

	return int(val / time.Second), err
}

func (im *Impl) Name() string {
	return im.name
}

func (im *Impl) Rename(ctx context.Context, oldKey, newKey string) error {
	_, err := im.client.Rename(ctx, oldKey, newKey).Result()
	if err != nil {
		im.sugar.Errorw("RENAME redis failed", "err", err)
	}
	return nil
}

func (im *Impl) MGet(ctx context.Context, keys []string) ([]MVal, error) {
	if len(keys) == 0 {
		return []MVal{}, nil
	}

	values, err := im.client.MGet(ctx, keys...).Result()
	if err != nil {
		im.sugar.Errorw("MGET redis failed", "err", err)
	}
	return im.processMGetValues(ctx, values), nil
}

func (im *Impl) processMGetValues(ctx context.Context, values []interface{}) []MVal {
	size := 0
	mvals := []MVal{}
	for k := range values {
		if values[k] == nil {
			mvals = append(mvals, MVal{
				Valid: false,
				Value: []byte(""),
			})
			continue
		}

		mval := MVal{Valid: true}
		mval.Value = []byte(values[k].(string))

		size += len(mval.Value)
		mvals = append(mvals, mval)
	}

	return mvals
}

func (im *Impl) HMGet(ctx context.Context, key string, fields []string, removeNil bool) (map[string]interface{}, error) {
	// Convert the fields slice to []interface{} required by HMGet
	interfaceFields := make([]interface{}, len(fields))
	for i, field := range fields {
		interfaceFields[i] = field
	}

	// Perform HMGet
	values, err := im.client.HMGet(ctx, key, fields...).Result()
	if err != nil {
		im.sugar.Errorw("HMGET redis failed", "err", err)
		return nil, err
	}

	// Construct the result map
	result := make(map[string]interface{})
	for i, field := range fields {
		if removeNil && values[i] == nil {
			continue
		} else {
			result[field] = values[i]
		}
	}

	return result, nil
}
