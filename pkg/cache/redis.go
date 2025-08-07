package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"creative-studio-server/config"
	"creative-studio-server/pkg/logger"
)

type RedisClient struct {
	client *redis.Client
	ctx    context.Context
}

var Cache *RedisClient

func InitRedis(cfg *config.Config) error {
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.GetRedisAddr(),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		DialTimeout:  10 * time.Second,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		PoolSize:     10,
		PoolTimeout:  30 * time.Second,
	})

	ctx := context.Background()
	
	// Test connection
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	Cache = &RedisClient{
		client: rdb,
		ctx:    ctx,
	}

	logger.Info("Redis connected successfully")
	return nil
}

func (r *RedisClient) Set(key string, value interface{}, expiration time.Duration) error {
	var data []byte
	var err error

	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		data, err = json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal value: %w", err)
		}
	}

	err = r.client.Set(r.ctx, key, data, expiration).Err()
	if err != nil {
		return fmt.Errorf("failed to set cache key %s: %w", key, err)
	}

	return nil
}

func (r *RedisClient) Get(key string) (string, error) {
	val, err := r.client.Get(r.ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("key %s not found", key)
	} else if err != nil {
		return "", fmt.Errorf("failed to get cache key %s: %w", key, err)
	}

	return val, nil
}

func (r *RedisClient) GetJSON(key string, dest interface{}) error {
	val, err := r.Get(key)
	if err != nil {
		return err
	}

	err = json.Unmarshal([]byte(val), dest)
	if err != nil {
		return fmt.Errorf("failed to unmarshal cached value: %w", err)
	}

	return nil
}

func (r *RedisClient) Delete(key string) error {
	err := r.client.Del(r.ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete cache key %s: %w", key, err)
	}

	return nil
}

func (r *RedisClient) Exists(key string) (bool, error) {
	exists, err := r.client.Exists(r.ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check if key %s exists: %w", key, err)
	}

	return exists > 0, nil
}

func (r *RedisClient) SetWithTTL(key string, value interface{}, ttl time.Duration) error {
	return r.Set(key, value, ttl)
}

func (r *RedisClient) Increment(key string) (int64, error) {
	val, err := r.client.Incr(r.ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment key %s: %w", key, err)
	}

	return val, nil
}

func (r *RedisClient) SetHash(key string, field string, value interface{}) error {
	var data string
	var err error

	switch v := value.(type) {
	case string:
		data = v
	default:
		jsonData, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal value: %w", err)
		}
		data = string(jsonData)
	}

	err = r.client.HSet(r.ctx, key, field, data).Err()
	if err != nil {
		return fmt.Errorf("failed to set hash field %s:%s: %w", key, field, err)
	}

	return nil
}

func (r *RedisClient) GetHash(key string, field string) (string, error) {
	val, err := r.client.HGet(r.ctx, key, field).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("hash field %s:%s not found", key, field)
	} else if err != nil {
		return "", fmt.Errorf("failed to get hash field %s:%s: %w", key, field, err)
	}

	return val, nil
}

func (r *RedisClient) GetAllHash(key string) (map[string]string, error) {
	val, err := r.client.HGetAll(r.ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get all hash fields for %s: %w", key, err)
	}

	return val, nil
}

func (r *RedisClient) DeleteHash(key string, field string) error {
	err := r.client.HDel(r.ctx, key, field).Err()
	if err != nil {
		return fmt.Errorf("failed to delete hash field %s:%s: %w", key, field, err)
	}

	return nil
}

func (r *RedisClient) SetList(key string, values ...interface{}) error {
	err := r.client.RPush(r.ctx, key, values...).Err()
	if err != nil {
		return fmt.Errorf("failed to set list %s: %w", key, err)
	}

	return nil
}

func (r *RedisClient) GetList(key string, start, stop int64) ([]string, error) {
	val, err := r.client.LRange(r.ctx, key, start, stop).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get list %s: %w", key, err)
	}

	return val, nil
}

func (r *RedisClient) PopList(key string) (string, error) {
	val, err := r.client.LPop(r.ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("list %s is empty", key)
	} else if err != nil {
		return "", fmt.Errorf("failed to pop from list %s: %w", key, err)
	}

	return val, nil
}

func (r *RedisClient) GetKeys(pattern string) ([]string, error) {
	keys, err := r.client.Keys(r.ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get keys with pattern %s: %w", pattern, err)
	}

	return keys, nil
}

func (r *RedisClient) FlushDB() error {
	err := r.client.FlushDB(r.ctx).Err()
	if err != nil {
		return fmt.Errorf("failed to flush database: %w", err)
	}

	return nil
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}

// Cache key helpers
func UserCacheKey(userID uint) string {
	return fmt.Sprintf("user:%d", userID)
}

func AtomicClipCacheKey(clipID uint) string {
	return fmt.Sprintf("clip:%d", clipID)
}

func ProjectCacheKey(projectID uint) string {
	return fmt.Sprintf("project:%d", projectID)
}

func SearchCacheKey(query string, filters map[string]interface{}) string {
	// Create a cache key based on search parameters
	// In practice, you'd hash the parameters for a cleaner key
	return fmt.Sprintf("search:%s", query)
}

func RenderTaskCacheKey(taskID string) string {
	return fmt.Sprintf("render_task:%s", taskID)
}