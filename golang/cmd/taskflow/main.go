package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/aditya/taskflow-go/internal/config"
	"github.com/aditya/taskflow-go/internal/mailer"
	"github.com/aditya/taskflow-go/internal/queue"
	"github.com/aditya/taskflow-go/internal/storage"
	"github.com/aditya/taskflow-go/internal/store"
	"github.com/aditya/taskflow-go/internal/web"
	"github.com/redis/go-redis/v9"
)

type redisSessions struct {
	client *redis.Client
}

func (r *redisSessions) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *redisSessions) Get(ctx context.Context, sid string) (int64, error) {
	val, err := r.client.Get(ctx, "session:"+sid).Result()
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(val, 10, 64)
}

func (r *redisSessions) Set(ctx context.Context, sid string, userID int64, ttl time.Duration) error {
	return r.client.Set(ctx, "session:"+sid, strconv.FormatInt(userID, 10), ttl).Err()
}

func (r *redisSessions) Delete(ctx context.Context, sid string) error {
	return r.client.Del(ctx, "session:"+sid).Err()
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg := config.Load()

	cmd := "web"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if cmd == "init-bucket" {
		s3, err := storage.NewS3(ctx, cfg.AWSEndpoint, cfg.AWSRegion, cfg.AWSKey, cfg.AWSSecret, cfg.AWSBucket)
		if err != nil {
			logger.Error("s3 client init failed", "error", err)
			os.Exit(1)
		}
		for i := 0; i < 15; i++ {
			if err := s3.EnsureBucket(ctx); err == nil {
				logger.Info("bucket ready", "bucket", cfg.AWSBucket)
				return
			}
			time.Sleep(1 * time.Second)
		}
		logger.Error("failed to initialize bucket", "bucket", cfg.AWSBucket)
		os.Exit(1)
	}

	db, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	switch cmd {
	case "migrate":
		if err := db.Migrate(ctx); err != nil {
			logger.Error("migration failed", "error", err)
			os.Exit(1)
		}
		logger.Info("migrations applied")
		return

	case "seed":
		if err := db.Seed(ctx); err != nil {
			logger.Error("seed failed", "error", err)
			os.Exit(1)
		}
		logger.Info("database seeded", "email", "test@example.com")
		return

	case "scheduler":
		logger.Info("scheduler started")
		emitHeartbeat(ctx, db, logger)
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				logger.Info("scheduler shutting down")
				return
			case <-ticker.C:
				emitHeartbeat(ctx, db, logger)
			}
		}
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPass,
	})
	defer rdb.Close()

	q := queue.NewRedisQueue(rdb)

	switch cmd {
	case "worker":
		mail := mailer.New(cfg.MailHost, cfg.MailPort, cfg.MailFrom)
		logger.Info("worker started")
		for {
			if ctx.Err() != nil {
				logger.Info("worker shutting down gracefully")
				return
			}
			msg, err := q.Pop(ctx, 2*time.Second)
			if err != nil {
				if ctx.Err() != nil {
					logger.Info("worker shutting down gracefully")
					return
				}
				logger.Error("queue pop failed", "error", err)
				time.Sleep(1 * time.Second)
				continue
			}
			if msg == nil {
				continue
			}
			subject := fmt.Sprintf("[TaskFlow] Task %s: %s", msg.Event, msg.Title)
			body := fmt.Sprintf("Task '%s' in project '%s' was %s.\nStatus: %s\nPriority: %s",
				msg.Title, msg.ProjectName, msg.Event, msg.Status, msg.Priority)
			if err := mail.Send(msg.Recipient, subject, body); err != nil {
				logger.Error("mail delivery failed", "task_id", msg.TaskID, "error", err)
				continue
			}
			logger.Info("async task notification sent", "task_id", msg.TaskID, "event", msg.Event, "recipient", msg.Recipient)
		}

	case "web":
		s3, err := storage.NewS3(ctx, cfg.AWSEndpoint, cfg.AWSRegion, cfg.AWSKey, cfg.AWSSecret, cfg.AWSBucket)
		if err != nil {
			logger.Error("s3 init failed", "error", err)
			os.Exit(1)
		}
		srv, err := web.NewServer(db, &redisSessions{client: rdb}, s3, q, logger)
		if err != nil {
			logger.Error("web server init failed", "error", err)
			os.Exit(1)
		}
		httpSrv := &http.Server{
			Addr:              ":" + cfg.Port,
			Handler:           srv.Handler(),
			ReadHeaderTimeout: 10 * time.Second,
		}
		go func() {
			logger.Info("web server listening", "port", cfg.Port)
			if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Error("http server error", "error", err)
				os.Exit(1)
			}
		}()
		<-ctx.Done()
		logger.Info("web server shutting down gracefully")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdownCtx)
	default:
		logger.Error("unknown command", "command", cmd)
		os.Exit(1)
	}
}

func emitHeartbeat(ctx context.Context, db *store.Store, logger *slog.Logger) {
	st, err := db.GetTaskStats(ctx)
	if err != nil {
		logger.Error("scheduler heartbeat failed", "error", err)
		return
	}
	logger.Info("scheduler heartbeat", "total_tasks", st.Total, "open_tasks", st.Open, "overdue_tasks", st.Overdue)
}
