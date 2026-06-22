package handler

import (
	"errors"
	"net/http"
	"strings"

	memStorage "github.com/bazueva/metrics/internal/storage"
)

type Handler struct {
	memStorage memStorage.Storage
}

func NewHandler(memStorage memStorage.Storage) *Handler {
	return &Handler{memStorage: memStorage}
}

func (h *Handler) UpdateHandler(w http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	paths := strings.Split(request.URL.Path, "/")
	if len(paths) != 3 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid path"))
		return
	}

	err := h.memStorage.UpdateMetric(paths[0], paths[1], paths[2])
	if err != nil {
		switch {
		case errors.Is(err, memStorage.ErrEmptyMetricName):
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, err.Error(), http.StatusBadRequest)
		}

		return
	}

	w.WriteHeader(http.StatusOK)
}
