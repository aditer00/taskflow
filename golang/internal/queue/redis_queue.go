package queue

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

const NotificationQueueKey = "taskflow:notifications"

type TaskNotification struct {
	TaskID      int64  `json:"task_id"`
	ProjectName string `json:"project_name"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
	Event       string `json:"event"`
	Recipient   string `json:"recipient"`
}

type RedisQueue struct {
	client *redis.Client
}

func NewRedisQueue(client *redis.Client) *RedisQueue {
	return &RedisQueue{client: client}
}

func (q *RedisQueue) Enqueue(ctx context.Context, msg TaskNotification) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return q.client.LPush(ctx, NotificationQueueKey, payload).Err()
}

func (q *RedisQueue) Pop(ctx context.Context, timeout time.Duration) (*TaskNotification, error) {
	res, err := q.client.BRPop(ctx, timeout, NotificationQueueKey).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(res) < 2 {
		return nil, nil
	}
	var msg TaskNotification
	if err := json.Unmarshal([]byte(res[1]), &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}
