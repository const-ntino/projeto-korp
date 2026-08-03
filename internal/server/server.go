package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type response struct {
	Nome    string `json:"nome"`
	Horario string `json:"horario"`
}

func New(now func() time.Time) http.Handler {
	mux := http.NewServeMux()
	registry := prometheus.NewRegistry()
	up := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "http_server_projeto_korp_up",
		Help: "Availability of the Projeto Korp HTTP service.",
	})
	requests := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_server_projeto_korp_requests_total",
		Help: "Total HTTP requests handled by method and status code.",
	}, []string{"method", "status_code"})
	registry.MustRegister(up, requests)
	up.Set(1)

	mux.HandleFunc("/projeto-korp", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response{
			Nome:    "Projeto Korp",
			Horario: now().UTC().Format(time.RFC3339),
		})
	})

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	return instrumentRequests(mux, requests)
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func instrumentRequests(next http.Handler, requests *prometheus.CounterVec) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(rec, r)

		if r.URL.Path != "/metrics" {
			requests.WithLabelValues(r.Method, strconv.Itoa(rec.statusCode)).Inc()
		}
	})
}
