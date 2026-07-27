package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/benedict-korie/event-pipeline-observability/internal/metrics"
	"github.com/benedict-korie/event-pipeline-observability/internal/processor"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	reg := metrics.New()

	mux := http.NewServeMux()

	mux.HandleFunc("POST /events", func(w http.ResponseWriter, r *http.Request) {
		var job processor.Job
		if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
			http.Error(w, "invalid job payload", http.StatusBadRequest)
			return
		}

		reg.EventReceived()
		duration, err := processor.Process(job)
		reg.EventFinished(err == nil, duration)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"status":"processed"}`))
	})

	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		w.Write([]byte(reg.Render()))
	})

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	log.Printf("event-pipeline-observability listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}