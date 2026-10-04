package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	models "github.com/bazueva/metrics/internal/model"
	"github.com/bazueva/metrics/internal/notifier"
	"github.com/bazueva/metrics/internal/storage"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type exampleDatabase struct{}

func (exampleDatabase) Ping() error {
	return nil
}

func ExampleHandler_UpdateHandler() {
	memStorage := storage.NewMemStorage(
		nil,
		false,
		zap.NewNop(),
		0,
	)

	h := NewHandler(
		memStorage,
		zap.NewNop(),
		nil,
		nil,
	)

	router := chi.NewRouter()
	router.Post(
		"/update/{metricType}/{metricName}/{metricValue}",
		h.UpdateHandler,
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/update/gauge/temperature/42.5",
		nil,
	)

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	fmt.Println(recorder.Code)

	// Output:
	// 200
}

func ExampleHandler_GetMetricHandler() {
	memStorage := storage.NewMemStorage(
		nil,
		false,
		zap.NewNop(),
		0,
	)

	value := 42.5
	err := memStorage.UpdateMetric(models.Metrics{
		ID:    "temperature",
		MType: models.Gauge,
		Value: &value,
	}, false)
	if err != nil {
		fmt.Println(err)
		return
	}

	h := NewHandler(
		memStorage,
		zap.NewNop(),
		nil,
		nil,
	)

	router := chi.NewRouter()
	router.Get(
		"/value/{metricType}/{metricName}",
		h.GetMetricHandler,
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/value/gauge/temperature",
		nil,
	)

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	fmt.Println(recorder.Code)
	fmt.Println(recorder.Body.String())

	// Output:
	// 200
	// 42.5
}

func ExampleHandler_GetAllMetricsHandler() {
	memStorage := storage.NewMemStorage(
		nil,
		false,
		zap.NewNop(),
		0,
	)

	gaugeValue := 42.5
	counterValue := int64(10)

	_ = memStorage.UpdateMetric(models.Metrics{
		ID:    "temperature",
		MType: models.Gauge,
		Value: &gaugeValue,
	}, false)

	_ = memStorage.UpdateMetric(models.Metrics{
		ID:    "requests",
		MType: models.Counter,
		Delta: &counterValue,
	}, false)

	h := NewHandler(
		memStorage,
		zap.NewNop(),
		nil,
		nil,
	)

	router := chi.NewRouter()
	router.Get("/", h.GetAllMetricsHandler)

	request := httptest.NewRequest(
		http.MethodGet,
		"/",
		nil,
	)

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	fmt.Println(recorder.Code)
	fmt.Println(recorder.Body.String())

	// Output:
	// 200
	// requests - 10 <br>temperature - 42.500000 <br>
}

func ExampleHandler_UpdateMetricHandler() {
	memStorage := storage.NewMemStorage(
		nil,
		false,
		zap.NewNop(),
		0,
	)

	h := NewHandler(
		memStorage,
		zap.NewNop(),
		nil,
		nil,
	)

	router := chi.NewRouter()
	router.Post("/update", h.UpdateMetricHandler)

	body := strings.NewReader(`{
		"id": "temperature",
		"type": "gauge",
		"value": 42.5
	}`)

	request := httptest.NewRequest(
		http.MethodPost,
		"/update",
		body,
	)
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	fmt.Println(recorder.Code)

	// Output:
	// 200
}

func ExampleHandler_ValueMetricHandler() {
	memStorage := storage.NewMemStorage(
		nil,
		false,
		zap.NewNop(),
		0,
	)

	value := 42.5
	_ = memStorage.UpdateMetric(models.Metrics{
		ID:    "temperature",
		MType: models.Gauge,
		Value: &value,
	}, false)

	h := NewHandler(
		memStorage,
		zap.NewNop(),
		nil,
		nil,
	)

	router := chi.NewRouter()
	router.Post("/value/", h.ValueMetricHandler)

	body := strings.NewReader(`{
		"id": "temperature",
		"type": "gauge"
	}`)

	request := httptest.NewRequest(
		http.MethodPost,
		"/value/",
		body,
	)
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	fmt.Println(recorder.Code)
	fmt.Println(recorder.Body.String())

	// Output:
	// 200
	// {"id":"temperature","type":"gauge","value":42.5}
}

func ExampleHandler_UpdatesMetricHandler() {
	memStorage := storage.NewMemStorage(
		nil,
		false,
		zap.NewNop(),
		0,
	)

	h := NewHandler(
		memStorage,
		zap.NewNop(),
		nil,
		notifier.NewNotifier(),
	)

	router := chi.NewRouter()
	router.Post("/updates/", h.UpdatesMetricHandler)

	body := strings.NewReader(`[
		{
			"id": "temperature",
			"type": "gauge",
			"value": 42.5
		},
		{
			"id": "requests",
			"type": "counter",
			"delta": 10
		}
	]`)

	request := httptest.NewRequest(
		http.MethodPost,
		"/updates/",
		body,
	)
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	fmt.Println(recorder.Code)

	// Output:
	// 200
}

func ExampleHandler_PingHandler() {
	h := NewHandler(
		nil,
		zap.NewNop(),
		exampleDatabase{},
		nil,
	)

	router := chi.NewRouter()
	router.Get("/ping", h.PingHandler)

	request := httptest.NewRequest(
		http.MethodGet,
		"/ping",
		nil,
	)

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	fmt.Println(recorder.Code)

	// Output:
	// 200
}
