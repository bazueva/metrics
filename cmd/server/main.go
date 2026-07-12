package main

import (
	"fmt"
	"net/http"

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

	memStorage := storage.NewMemStorage()
	httpHandler := handler.NewHandler(memStorage, cfg.logger)

	router := chi.NewRouter()
	router.Use(logger.ServerLogger(cfg.logger))

	router.Post("/update/{metricType}/{metricName}/{metricValue}", httpHandler.UpdateHandler)
	router.Get("/value/{metricType}/{metricName}", httpHandler.GetMetricHandler)
	router.Get("/", httpHandler.GetAllMetricsHandler)
	router.Post("/update", httpHandler.UpdateMetricHandler)
	router.Post("/update/", httpHandler.UpdateMetricHandler)
	router.Post("/value/", httpHandler.ValueMetricHandler)

	if err := http.ListenAndServe(cfg.ServerAddr.String(), router); err != nil {
		fmt.Println(err)
	}
}
