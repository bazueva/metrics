package collector

import (
	"fmt"
	randV2 "math/rand/v2"
	"runtime"

	models "github.com/bazueva/metrics/internal/model"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

type collector struct {
	metricsSnapshot []models.Metrics
}

func (c *collector) ExtendedMetricSnapshot() ([]models.Metrics, error) {
	virtualMemory, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}

	percentages, err := cpu.Percent(0, true)
	if err != nil {
		return nil, err
	}

	result := make([]models.Metrics, 0, len(percentages)+2)

	result = append(result, []models.Metrics{
		{
			ID:    "TotalMemory",
			MType: models.Gauge,
			Value: new(float64(virtualMemory.Total)),
		},
		{
			ID:    "FreeMemory",
			MType: models.Gauge,
			Value: new(float64(virtualMemory.Free)),
		},
	}...)

	for i, v := range percentages {
		result = append(result, models.Metrics{
			ID:    fmt.Sprintf("CPUutilization%d", i),
			MType: models.Gauge,
			Value: &v,
		})
	}

	return result, nil
}

func NewCollector() *collector {
	return &collector{}
}

func (c *collector) MetricsSnapshot(counter int64) []models.Metrics {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	return []models.Metrics{
		{
			ID:    "Alloc",
			MType: models.Gauge,
			Value: new(float64(ms.Alloc)),
		},
		{
			ID:    "BuckHashSys",
			MType: models.Gauge,
			Value: new(float64(ms.BuckHashSys)),
		},
		{
			ID:    "Frees",
			MType: models.Gauge,
			Value: new(float64(ms.Frees)),
		},
		{
			ID:    "GCCPUFraction",
			MType: models.Gauge,
			Value: new(ms.GCCPUFraction),
		},
		{
			ID:    "GCSys",
			MType: models.Gauge,
			Value: new(float64(ms.GCSys)),
		},
		{
			ID:    "HeapAlloc",
			MType: models.Gauge,
			Value: new(float64(ms.HeapAlloc)),
		},
		{
			ID:    "HeapIdle",
			MType: models.Gauge,
			Value: new(float64(ms.HeapIdle)),
		},
		{
			ID:    "HeapInuse",
			MType: models.Gauge,
			Value: new(float64(ms.HeapInuse)),
		},
		{
			ID:    "HeapObjects",
			MType: models.Gauge,
			Value: new(float64(ms.HeapObjects)),
		},
		{
			ID:    "HeapReleased",
			MType: models.Gauge,
			Value: new(float64(ms.HeapReleased)),
		},
		{
			ID:    "HeapSys",
			MType: models.Gauge,
			Value: new(float64(ms.HeapSys)),
		},
		{
			ID:    "LastGC",
			MType: models.Gauge,
			Value: new(float64(ms.LastGC)),
		},
		{
			ID:    "Lookups",
			MType: models.Gauge,
			Value: new(float64(ms.Lookups)),
		},
		{
			ID:    "MCacheInuse",
			MType: models.Gauge,
			Value: new(float64(ms.MCacheInuse)),
		},
		{
			ID:    "MCacheSys",
			MType: models.Gauge,
			Value: new(float64(ms.MCacheSys)),
		},
		{
			ID:    "MSpanInuse",
			MType: models.Gauge,
			Value: new(float64(ms.MSpanInuse)),
		},
		{
			ID:    "MSpanSys",
			MType: models.Gauge,
			Value: new(float64(ms.MSpanSys)),
		},
		{
			ID:    "Mallocs",
			MType: models.Gauge,
			Value: new(float64(ms.Mallocs)),
		},
		{
			ID:    "NextGC",
			MType: models.Gauge,
			Value: new(float64(ms.NextGC)),
		},
		{
			ID:    "NumForcedGC",
			MType: models.Gauge,
			Value: new(float64(ms.NumForcedGC)),
		},
		{
			ID:    "NumGC",
			MType: models.Gauge,
			Value: new(float64(ms.NumGC)),
		},
		{
			ID:    "OtherSys",
			MType: models.Gauge,
			Value: new(float64(ms.OtherSys)),
		},
		{
			ID:    "PauseTotalNs",
			MType: models.Gauge,
			Value: new(float64(ms.PauseTotalNs)),
		},
		{
			ID:    "StackInuse",
			MType: models.Gauge,
			Value: new(float64(ms.StackInuse)),
		},
		{
			ID:    "StackSys",
			MType: models.Gauge,
			Value: new(float64(ms.StackSys)),
		},
		{
			ID:    "Sys",
			MType: models.Gauge,
			Value: new(float64(ms.Sys)),
		},
		{
			ID:    "TotalAlloc",
			MType: models.Gauge,
			Value: new(float64(ms.TotalAlloc)),
		},
		{
			ID:    "RandomValue",
			MType: models.Gauge,
			Value: new(randV2.Float64()),
		},
		{
			ID:    "PollCount",
			MType: models.Counter,
			Delta: &counter,
		},
	}
}
