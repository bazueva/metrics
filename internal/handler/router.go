package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"

	models "github.com/bazueva/metrics/internal/model"
	"github.com/bazueva/metrics/internal/notifier"
	memStorage "github.com/bazueva/metrics/internal/storage"
	"github.com/samber/lo"
	"go.uber.org/zap"
)

// Storage определяет методы для хранения и получения метрик.
type Storage interface {
	GetMetric(name string) (models.Metrics, error)
	GetAllMetrics() []models.Metrics
	UpdateMetric(metric models.Metrics, needSave bool) error
	CreateMetric(metricType string, name string, value string) (models.Metrics, error)
	UpdatesMetrics([]models.Metrics) error
}

// Database определяет методы для работы с базой данных.
type Database interface {
	Ping() error
}

// Notifier определяет методы для отправки событий.
type Notifier interface {
	Notify(event notifier.MetricsSavedEvent) error
}

// Handler обрабатывает HTTP-запросы для работы с метриками.
type Handler struct {
	storage  Storage
	logger   *zap.Logger
	db       Database
	notifier Notifier
}

// NewHandler создаёт новый Handler.
func NewHandler(
	memStorage Storage,
	logger *zap.Logger,
	db Database,
	notifier Notifier,
) *Handler {
	return &Handler{
		storage:  memStorage,
		logger:   logger,
		db:       db,
		notifier: notifier,
	}
}

// UpdateHandler обрабатывает обновление метрики, переданной в параметрах URL.
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

// GetMetricHandler возвращает значение запрошенной метрики.
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

// GetAllMetricsHandler возвращает все сохранённые метрики в формате HTML.
func (h *Handler) GetAllMetricsHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")

	for _, metric := range h.storage.GetAllMetrics() {
		switch metric.MType {
		case models.Counter:
			_, _ = fmt.Fprintf(writer, "%s - %d <br>", metric.ID, *metric.Delta)
		case models.Gauge:
			_, _ = fmt.Fprintf(writer, "%s - %f <br>", metric.ID, *metric.Value)
		}
	}
}

// UpdateMetricHandler обрабатывает обновление метрики, переданной в формате JSON.
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

// ValueMetricHandler возвращает запрошенную метрику в формате JSON.
func (h *Handler) ValueMetricHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	if request.ContentLength == 0 {
		h.jsonErrorHandler(writer, fmt.Errorf("не указана метрика"), http.StatusBadRequest)

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

	resultMetricJSON, err := json.Marshal(resultMetric)
	if err != nil {
		h.logger.Error("ошибка json unmarshal", zap.Error(err))
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	writer.Write(resultMetricJSON)
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

// PingHandler проверяет соединение с базой данных.
func (h *Handler) PingHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")

	if err := h.db.Ping(); err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		writer.Write([]byte("Ошибка соединения с БД"))

		return
	}

	writer.WriteHeader(http.StatusOK)
}

// UpdatesMetricHandler обрабатывает пакетное обновление метрик, переданных в формате JSON.
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
		h.jsonErrorHandler(writer, fmt.Errorf("не переданы метрики"), http.StatusBadRequest)

		return
	}

	err = h.storage.UpdatesMetrics(metrics)
	if err != nil {
		h.jsonErrorHandler(writer, err, 0)

		return
	}

	event := notifier.MetricsSavedEvent{
		TS: time.Now().Unix(),
		Metrics: lo.Map(metrics, func(metric models.Metrics, _ int) string {
			return metric.ID
		}),
		IPAddress: getIPAddress(request),
	}

	if err := h.notifier.Notify(event); err != nil {
		h.logger.Error("ошибка отправки сообщения аудита", zap.Error(err))
	}

	writer.WriteHeader(http.StatusOK)
}

func getIPAddress(request *http.Request) string {
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		return request.RemoteAddr
	}

	return host
}
