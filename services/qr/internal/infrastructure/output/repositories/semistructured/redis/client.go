package redis

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"rutadeorigen/qr/internal/platform/exception"
)

const connectTimeout = 10 * time.Second

func Connect(ctx context.Context, url string) (*goredis.Client, error) {
	opts, err := goredis.ParseURL(url)
	if err != nil {
		return nil, exception.New(exception.CodeInternal).WithTechnical("parse redis url").WithCause(err)
	}

	client := goredis.NewClient(opts)

	ctx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, exception.New(exception.CodeInternal).WithTechnical("ping redis").WithCause(err)
	}

	return client, nil
}
