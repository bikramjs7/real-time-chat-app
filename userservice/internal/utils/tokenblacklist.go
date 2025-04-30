package utils

import (
    "context"
    "time"

    "github.com/go-redis/redis/v8"
    "go.uber.org/zap"
)

type Blacklist struct {
    client *redis.Client
    logger *zap.Logger
}

func NewBlacklist(redisAddr string, logger *zap.Logger) *Blacklist {
    client := redis.NewClient(&redis.Options{
        Addr: redisAddr,
    })
    return &Blacklist{
        client: client,
        logger: logger,
    }
}

func (b *Blacklist) Set(token string, expiration time.Duration) error {
    ctx := context.Background()
    err := b.client.Set(ctx, token, "blacklisted", expiration).Err()
    if err != nil {
        b.logger.Error("Failed to add token to blacklist", zap.Error(err))
        return err
    }
    if b.logger != nil {
        b.logger.Info("Added token to blacklist", zap.String("token", token))
    }
    return nil
}

func (b *Blacklist) Get(token string) (bool, error) {
    ctx := context.Background()
    val, err := b.client.Get(ctx, token).Result()
    if err == redis.Nil {
        return false, nil
    } else if err != nil {
        b.logger.Error("Failed to check token in blacklist", zap.Error(err))    
        return false, err
    }
    if val == "blacklisted" {
        b.logger.Info("Token is blacklisted", zap.String("token", token))
        return true, nil
    }
    return false, nil
}