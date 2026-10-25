package katalog

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
			t.Errorf("recorder.Code = %d, erwartet %d", recorder.Code, http.StatusOK)
		}

		contentType := recorder.Header().Get("Content-Type")
		expectedContentType := "application/json; charset=utf-8"
		if contentType != expectedContentType {
			t.Errorf("recorder.Header().Get(…) = %q, erwartet Content-Type %q", contentType, expectedContentType)
		}

		var antwort map[string]string
		if err := json.Unmarshal(recorder.Body.Bytes(), &antwort); err != nil {
			t.Fatalf("failed to parse JSON response: %v", err)
		}

		if antwort["message"] != "success" {
			t.Errorf("Meldung = %v, erwartet 'success'", antwort["message"])
		}
	})

	t.Run("NilPayload", func(t *testing.T) {
		recorder := httptest.NewRecorder()

		schreibeJSON(recorder, http.StatusNoContent, nil)

		if recorder.Code != http.StatusNoContent {
			t.Errorf("recorder.Code = %d, erwartet %d", recorder.Code, http.StatusNoContent)
		}

		if recorder.Body.String() != "null\n" {
			t.Errorf("Rumpf = %q, erwartet 'null\\n'", recorder.Body.String())
		}
	})

	t.Run("UnencodablePayload", func(t *testing.T) {
		recorder := httptest.NewRecorder()

		// Ein Kanal lässt sich nicht als JSON kodieren.
		nutzlast := make(chan int)

		// Das darf nicht abbrechen.
		schreibeJSON(recorder, http.StatusInternalServerError, nutzlast)

		if recorder.Code != http.StatusInternalServerError {
			t.Errorf("recorder.Code = %d, erwartet %d", recorder.Code, http.StatusInternalServerError)
		}
	})
}
