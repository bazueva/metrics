package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	dbpkg "github.com/bazueva/metrics/db"
	"github.com/bazueva/metrics/internal/repository/db/metrics"
	"github.com/bazueva/metrics/internal/repository/file"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/bazueva/metrics/internal/handler"
	"github.com/bazueva/metrics/internal/logger"
	"github.com/bazueva/metrics/internal/middleware"
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

	memStorage := storage.NewMemStorage(
		memStorageRepository,
		cfg.LoadMetricsFromFile,
		cfg.logger,
		cfg.StoreInterval,
	)
	memStorage.RunSaver()

	startServer(cfg, memStorage, db)
}

func startServer(cfg config, memStorage *storage.MemStorage, db *sql.DB) {
	httpHandler := handler.NewHandler(memStorage, cfg.logger, db)

	router := chi.NewRouter()
	router.Use(logger.ServerLogger(cfg.logger))
	router.Use(middleware.ServerUnpackGzip(cfg.logger))
	router.Use(middleware.ServerResponseGzip())

	router.Post("/update/{metricType}/{metricName}/{metricValue}", httpHandler.UpdateHandler)
	router.Get("/value/{metricType}/{metricName}", httpHandler.GetMetricHandler)
	router.Get("/", httpHandler.GetAllMetricsHandler)
	router.Post("/update", httpHandler.UpdateMetricHandler)
	router.Post("/update/", httpHandler.UpdateMetricHandler)
	router.Post("/updates/", httpHandler.UpdatesMetricHandler)
	router.Post("/value/", httpHandler.ValueMetricHandler)
	router.Get("/ping", httpHandler.PingHandler)

	if err := http.ListenAndServe(cfg.ServerAddr.String(), router); err != nil {
		fmt.Println(err)
	}
}
