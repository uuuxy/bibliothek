package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkStresstestRunner(b *testing.B) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	reqBody := ActionRequest{Query: "test"}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		b.Fatalf("Request-Körper: %v", err)
	}
	token := "dummy"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runner := newStresstestRunner(ts.URL, token, jsonData, 50)
		runner.starteAlle()
		runner.gibStartsignal()
		runner.wg.Wait()
	}
}
