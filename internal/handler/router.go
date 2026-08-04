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
	"go.uber.org/zap"
)

type Storage interface {
	GetMetric(name string) (models.Metrics, error)
	GetAllMetrics() []models.Metrics
	UpdateMetric(metric models.Metrics, needSave bool) error
	CreateMetric(metricType string, name string, value string) (models.Metrics, error)
	UpdatesMetrics([]models.Metrics) error
}

type Database interface {
	Ping() error
}

type Handler struct {
	storage Storage
	logger  *zap.Logger
	db      Database
}

func NewHandler(memStorage Storage, logger *zap.Logger, db Database) *Handler {
	return &Handler{
		storage: memStorage,
		logger:  logger,
		db:      db,
	}
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

	err = h.storage.UpdateMetric(metric, true)
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
}

func (h *Handler) GetAllMetricsHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")

	result := make([]byte, 0)

	for _, metric := range h.storage.GetAllMetrics() {
		switch metric.MType {
		case models.Counter:
			result = append(result, []byte(fmt.Sprintf("%s - %d <br>", metric.ID, *metric.Delta))...)
		case models.Gauge:
			result = append(result, []byte(fmt.Sprintf("%s - %f <br>", metric.ID, *metric.Value))...)
		}
	}

	writer.Write(result)
}

func (h *Handler) UpdateMetricHandler(writer http.ResponseWriter, request *http.Request) {
	body, err := io.ReadAll(request.Body)
	defer request.Body.Close()
	if err != nil {
		h.jsonErrorHandler(writer, err, http.StatusBadRequest)

		return
	}

	var metric models.Metrics
	err = json.Unmarshal(body, &metric)
	if err != nil {
		h.jsonErrorHandler(writer, err, http.StatusBadRequest)

		return
	}

	err = h.storage.UpdateMetric(metric, true)
	if err != nil {
		h.jsonErrorHandler(writer, err, 0)

		return
	}

	writer.WriteHeader(http.StatusOK)
}

func (h *Handler) ValueMetricHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	if request.ContentLength == 0 {
		h.jsonErrorHandler(writer, fmt.Errorf("Не указана метрика"), http.StatusBadRequest)

		return
	}

	var metric models.Metrics
	decoder := json.NewDecoder(request.Body)
	if err := decoder.Decode(&metric); err != nil {
		h.jsonErrorHandler(writer, err, http.StatusBadRequest)

		return
	}

	resultMetric, err := h.storage.GetMetric(metric.ID)
	if err != nil {
		h.jsonErrorHandler(writer, err, http.StatusNotFound)

		return
	}

	resultMetricJson, err := json.Marshal(resultMetric)
	if err != nil {
		h.logger.Error("Ошибка json unmarshal", zap.Error(err))
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

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

func (h *Handler) jsonErrorHandler(writer http.ResponseWriter, err error, status int) {
	httpStatus := http.StatusInternalServerError

	if status > 0 {
		httpStatus = status
	} else {
		switch {
		case errors.Is(err, memStorage.ErrEmptyMetricName),
			errors.Is(err, memStorage.ErrNotFoundMetric):
			httpStatus = http.StatusBadRequest
		default:
			httpStatus = http.StatusInternalServerError
			h.logger.Error("Ошибка", zap.Error(err))
			err = errors.New(http.StatusText(httpStatus))
		}
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(httpStatus)
	json.NewEncoder(writer).Encode(map[string]string{
		"error": err.Error(),
	})
}

func (h *Handler) PingHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")

	if err := h.db.Ping(); err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		writer.Write([]byte("Ошибка соединения с БД"))

		return
	}

	writer.WriteHeader(http.StatusOK)
}

func (h *Handler) UpdatesMetricHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")

	decoder := json.NewDecoder(request.Body)
	defer request.Body.Close()

	var metrics []models.Metrics
	err := decoder.Decode(&metrics)
	if err != nil {
		h.jsonErrorHandler(writer, err, http.StatusBadRequest)

		return
	}

	if len(metrics) == 0 {
		h.jsonErrorHandler(writer, fmt.Errorf("Не переданы метрики"), http.StatusBadRequest)

		return
	}

	err = h.storage.UpdatesMetrics(metrics)
	if err != nil {
		h.jsonErrorHandler(writer, err, 0)

		return
	}

	writer.WriteHeader(http.StatusOK)
}
