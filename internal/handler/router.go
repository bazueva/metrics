package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	models "github.com/bazueva/metrics/internal/model"
	memStorage "github.com/bazueva/metrics/internal/storage"
)

type Storage interface {
	GetMetric(name string) (models.Metrics, error)
	GetAllMetrics() []models.Metrics
	UpdateMetric(metric models.Metrics) error
	CreateMetric(metricType string, name string, value string) (models.Metrics, error)
}

type Handler struct {
	storage Storage
}

func NewHandler(memStorage Storage) *Handler {
	return &Handler{storage: memStorage}
}

func (h *Handler) UpdateHandler(w http.ResponseWriter, request *http.Request) {
	metric, err := h.storage.CreateMetric(
		request.PathValue("metricType"),
		request.PathValue("metricName"),
		request.PathValue("metricValue"),
	)
	if err != nil {
		errorHandler(w, err)

		return
	}

	err = h.storage.UpdateMetric(metric)
	if err != nil {
		errorHandler(w, err)

		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetMetricHandler(writer http.ResponseWriter, request *http.Request) {
	result, err := h.storage.GetMetric(request.PathValue("metricName"))
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
	for _, metric := range h.storage.GetAllMetrics() {
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

func (h *Handler) UpdateMetricHandler(writer http.ResponseWriter, request *http.Request) {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
	}

	defer request.Body.Close()

	var metric models.Metrics
	err = json.Unmarshal(body, &metric)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)

		return
	}

	err = h.storage.UpdateMetric(metric)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)

		return
	}

	writer.WriteHeader(http.StatusOK)
}

func (h *Handler) ValueMetricHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	decoder := json.NewDecoder(request.Body)

	var metric models.Metrics
	if err := decoder.Decode(&metric); err != nil {
		writeJsonError(writer, http.StatusBadRequest, err)

		return
	}

	resultMetric, err := h.storage.GetMetric(metric.ID)
	if err != nil {
		writeJsonError(writer, http.StatusNotFound, err)

		return
	}

	resultMetricJson, err := json.Marshal(resultMetric)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)

		return
	}

	writer.WriteHeader(http.StatusOK)
	writer.Write(resultMetricJson)
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

func writeJsonError(writer http.ResponseWriter, status int, err error) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)

	json.NewEncoder(writer).Encode(map[string]string{
		"error": err.Error(),
	})
}
