package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/nemdull/sf-meter-api/registry"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("起動できなかった、または異常終了した", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	// Ctrl+C や、コンテナを止めるときの SIGTERM を受け取ったら、ctx が終わる
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Get port from environment, default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if _, err := strconv.Atoi(port); err != nil {
		return errors.New("PORT は数値で指定してください: " + port)
	}

	// Initialize registry and dependencies
	startup, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cfg := registry.ConfigFromEnv()
	app, err := registry.New(startup, cfg, logger)
	if err != nil {
		return err
	}
	defer func() {
		if err := app.Close(); err != nil {
			logger.Error("保存先との接続を閉じられなかった", "error", err)
		}
	}()

	server := &http.Server{
		Addr:    net.JoinHostPort("0.0.0.0", port),
		Handler: app.Handler,
		// 遅い・終わらない接続にいつまでも付き合わないように、時間の上限を決めておく
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	failed := make(chan error, 1)
	go func() {
		logger.Info("sf-meter API を起動した",
			"url", "http://localhost:"+port,
			"store", app.Store,
			"writes_enabled", cfg.AdminToken != "",
		)
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			failed <- err
		}
	}()

	select {
	case err := <-failed:
		return err
	case <-ctx.Done():
	}

	// 受け付けを止め、処理中のリクエストが終わるのを 10 秒まで待ってから終わる
	logger.Info("停止の合図を受け取った。処理中のリクエストが終わるのを待つ")
	shutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	return server.Shutdown(shutdown)
}
