package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type APIServer struct {
	port    int
	manager *SensorManager
	events  *EventBus
	server  *http.Server
}

func NewAPIServer(port int, manager *SensorManager, events *EventBus) *APIServer {
	return &APIServer{
		port:    port,
		manager: manager,
		events:  events,
	}
}

func (a *APIServer) Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", a.handleHealth)
	mux.HandleFunc("/api/sensors", a.handleSensors)
	mux.HandleFunc("/api/aircraft", a.handleAircraft)
	mux.HandleFunc("/api/stats", a.handleStats)
	mux.HandleFunc("/events", a.handleEvents)

	a.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", a.port),
		Handler:           withCORS(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("API server listening on :%d", a.port)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("api server error: %v", err)
		}
	}()
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (a *APIServer) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "service": "blue-glide-mlat"})
}

func (a *APIServer) handleSensors(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.manager.Sensors())
}

func (a *APIServer) handleAircraft(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.manager.Aircraft())
}

func (a *APIServer) handleStats(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.manager.Stats())
}

func (a *APIServer) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := a.events.Subscribe()
	defer a.events.Unsubscribe(ch)

	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case payload := <-ch:
			_, _ = fmt.Fprintf(w, "data: %s\n\n", payload)
			flusher.Flush()
		case <-ticker.C:
			_, _ = fmt.Fprint(w, ": keep-alive\n\n")
			flusher.Flush()
		}
	}
}
