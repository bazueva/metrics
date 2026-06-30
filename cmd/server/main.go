package main

import (
	"fmt"
	"net/http"

	"github.com/bazueva/metrics/cmd/config"
	"github.com/bazueva/metrics/internal/handler"
	"github.com/bazueva/metrics/internal/storage"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/go-chi/chi/v5"
)

func main() {
	serverAddr := config.ServerAddr{
		Host: "localhost",
		Port: 8080,
	}
	parseFlags(&serverAddr)

	memStorage := storage.NewMemStorage()
	httpHandler := handler.NewHandler(memStorage)

	router := chi.NewRouter()
	router.Use(middleware.Logger)

	router.Post("/update/{metricType}/{metricName}/{metricValue}", httpHandler.UpdateHandler)
	router.Get("/value/{metricType}/{metricName}", httpHandler.GetMetricHandler)
	router.Get("/", httpHandler.GetAllMetricsHandler)

	if err := http.ListenAndServe(serverAddr.String(), router); err != nil {
		fmt.Println(err)
	}
}
