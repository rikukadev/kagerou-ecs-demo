// api は内部サービス。ALB からは見えず、gateway だけが呼ぶ。
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	port := env("PORT", "8081")
	started := time.Now()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/items", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"service": "api",
			"env":     env("KAGEROU_ENV", "local"),
			"uptime":  time.Since(started).Round(time.Second).String(),
			"items":   []string{"alpha", "beta", "gamma"},
		})
	})

	log.Printf("api listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
