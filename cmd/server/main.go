package main

import (
	"net/http"

	"github.com/bazueva/metrics/internal/handler"
	"github.com/bazueva/metrics/internal/storage"
)

func main() {
	memStorage := storage.NewMemStorage()
	httpHandler := handler.NewHandler(memStorage)

	mux := http.NewServeMux()
	mux.Handle("/update/", http.StripPrefix("/update/", http.HandlerFunc(httpHandler.UpdateHandler)))

	if err := http.ListenAndServe(":8080", mux); err != nil {
		panic(err)
	}
}
