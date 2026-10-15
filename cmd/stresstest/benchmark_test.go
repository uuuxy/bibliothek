package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkLastlauf(b *testing.B) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	anfrageDaten := aktionsAnfrage{Suchtext: "test"}
	rumpf, err := json.Marshal(anfrageDaten)
	if err != nil {
		b.Fatalf("Request-Körper: %v", err)
	}
	token := "dummy"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lauf := newLastlauf(ts.URL, token, rumpf, 50)
		lauf.starteAlle()
		lauf.gibStartsignal()
		lauf.alle.Wait()
	}
}
