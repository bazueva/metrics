package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	dbpkg "github.com/bazueva/metrics/db"
	serverMiddleware "github.com/bazueva/metrics/internal/middleware/server"
	"github.com/bazueva/metrics/internal/repository/db/metrics"
	"github.com/bazueva/metrics/internal/repository/file"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/bazueva/metrics/internal/handler"
	"github.com/bazueva/metrics/internal/logger"
	"github.com/bazueva/metrics/internal/storage"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func main() {
	cfg, err := readConfig()
	if err != nil {
		panic(err)
	}

	cfg.logger, err = zap.NewProduction()
	if err != nil {
		panic(err)
	}

	defer cfg.logger.Sync()

	db, err := sql.Open("pgx", cfg.DatabaseDSN)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	if cfg.DatabaseDSN != "" {
		if err := dbpkg.RunMigrations(db); err != nil {
			log.Fatal("Migration failed:", err)
		}
	}

	var memStorageRepository storage.Repository
	switch {
	case cfg.DatabaseDSN != "":
		memStorageRepository = metrics.NewRepository(db, cfg.logger)
	case cfg.FileStoragePath != "":
		memStorageRepository = file.NewRepository(cfg.FileStoragePath)
	}

	ctxWithCancel, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	memStorage := storage.NewMemStorage(
		memStorageRepository,
		cfg.LoadMetricsFromFile,
		cfg.logger,
		cfg.StoreInterval,
	)
	memStorage.RunSaver(ctxWithCancel)

	startServer(ctxWithCancel, cfg, memStorage, db)

	<-ctxWithCancel.Done()
	cfg.logger.Info("Программа завершена")
}

func startServer(ctx context.Context, cfg config, memStorage *storage.MemStorage, db *sql.DB) {
	httpHandler := handler.NewHandler(memStorage, cfg.logger, db)

	router := chi.NewRouter()
	router.Use(logger.ServerLogger(cfg.logger))
	router.Use(serverMiddleware.UnpackGzip(cfg.logger))
	router.Use(serverMiddleware.ResponseGzip())
	if cfg.SecretKey != "" {
		router.Use(serverMiddleware.CheckSignData(cfg.SecretKey, cfg.logger))
	}

	router.Post("/update/{metricType}/{metricName}/{metricValue}", httpHandler.UpdateHandler)
	router.Get("/value/{metricType}/{metricName}", httpHandler.GetMetricHandler)
	router.Get("/", httpHandler.GetAllMetricsHandler)
	router.Post("/update", httpHandler.UpdateMetricHandler)
	router.Post("/update/", httpHandler.UpdateMetricHandler)
	router.Post("/updates/", httpHandler.UpdatesMetricHandler)
	router.Post("/value/", httpHandler.ValueMetricHandler)
	router.Get("/ping", httpHandler.PingHandler)

	server := &http.Server{
		Addr:    cfg.ServerAddr.String(),
		Handler: router,
	}

	go func() {
		cfg.logger.Info("Сервер запущен", zap.String("addr", cfg.ServerAddr.String()))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			cfg.logger.Error("Ошибка сервера", zap.Error(err))
		}
	}()

	<-ctx.Done()
	cfg.logger.Info("Остановка сервера...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		cfg.logger.Error("Ошибка остановки сервера", zap.Error(err))
	}
}
