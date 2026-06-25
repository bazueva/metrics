package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	models "github.com/bazueva/metrics/internal/model"
	memStorage "github.com/bazueva/metrics/internal/storage"
)

type Handler struct {
	memStorage memStorage.Storage
}

func NewHandler(memStorage memStorage.Storage) *Handler {
	return &Handler{memStorage: memStorage}
}

func (h *Handler) UpdateHandler(w http.ResponseWriter, request *http.Request) {
	err := h.memStorage.UpdateMetric(
		request.PathValue("metricType"),
		request.PathValue("metricName"),
		request.PathValue("metricValue"),
	)
	if err != nil {
		errorHandler(w, err)

		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetMetricHandler(writer http.ResponseWriter, request *http.Request) {
	result, err := h.memStorage.GetMetric(request.PathValue("metricName"))
	if err != nil {
		errorHandler(writer, err)

		return
	}

	switch result.MType {
	case models.Counter:
		writer.Write([]byte(strconv.Itoa(int(*result.Delta))))
	case models.Gauge:
		writer.Write([]byte(strconv.FormatFloat(*result.Value, 'f', -1, 64)))
	default:
		http.Error(writer, "Undefined type", http.StatusNotFound)

		return
	}

	writer.WriteHeader(http.StatusOK)
}

func (h *Handler) GetAllMetricsHandler(writer http.ResponseWriter, request *http.Request) {
	for _, metric := range h.memStorage.GetAllMetrics() {
		switch metric.MType {
		case models.Counter:
			writer.Write([]byte(fmt.Sprintf("%s - %d \n", metric.ID, *metric.Delta)))
		case models.Gauge:
			writer.Write([]byte(fmt.Sprintf("%s - %f \n", metric.ID, *metric.Value)))
		}
	}

	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.WriteHeader(http.StatusOK)
}

func errorHandler(writer http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, memStorage.ErrEmptyMetricName),
		errors.Is(err, memStorage.ErrNotFoundMetric):
		http.Error(writer, err.Error(), http.StatusNotFound)
	default:
		http.Error(writer, err.Error(), http.StatusBadRequest)
	}
}
