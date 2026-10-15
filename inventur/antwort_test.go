package inventur

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSchreibeJSON(t *testing.T) {
	t.Run("ValidPayload", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		nutzlast := map[string]string{"message": "success"}

		schreibeJSON(recorder, http.StatusOK, nutzlast)

		if recorder.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, recorder.Code)
		}

		contentType := recorder.Header().Get("Content-Type")
		expectedContentType := "application/json; charset=utf-8"
		if contentType != expectedContentType {
			t.Errorf("expected Content-Type %q, got %q", expectedContentType, contentType)
		}

		var antwort map[string]string
		if err := json.Unmarshal(recorder.Body.Bytes(), &antwort); err != nil {
			t.Fatalf("failed to parse JSON response: %v", err)
		}

		if antwort["message"] != "success" {
			t.Errorf("expected message 'success', got %v", antwort["message"])
		}
	})

	t.Run("NilPayload", func(t *testing.T) {
		recorder := httptest.NewRecorder()

		schreibeJSON(recorder, http.StatusNoContent, nil)

		if recorder.Code != http.StatusNoContent {
			t.Errorf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
		}

		if recorder.Body.String() != "null\n" {
			t.Errorf("expected body 'null\\n', got %q", recorder.Body.String())
		}
	})

	t.Run("UnencodablePayload", func(t *testing.T) {
		recorder := httptest.NewRecorder()

		// Ein Kanal lässt sich nicht als JSON kodieren.
		nutzlast := make(chan int)

		// Das darf nicht abbrechen.
		schreibeJSON(recorder, http.StatusInternalServerError, nutzlast)

		if recorder.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
		}
	})
}
