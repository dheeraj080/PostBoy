// testserver is a minimal JSON API server used by the VHS tape.
// Run with: go run ./cmd/testserver
package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/issues", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		resp := map[string]interface{}{
			"id":       84019284,
			"node_id":   "I_kwDOAG7x4858",
			"number":    84,
			"title":     "CLI headless test runner fails in CI",
			"state":     "open",
			"user": map[string]interface{}{
				"login": "kambledheerajkumar",
				"id":    491823,
			},
			"labels":     []string{"bug", "terminal"},
			"created_at": "2026-10-08T02:14:00Z",
		}
		json.NewEncoder(w).Encode(resp)
	})
	log.Println("testserver listening on :18080")
	log.Fatal(http.ListenAndServe(":18080", mux))
}
