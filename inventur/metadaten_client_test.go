package inventur

import (
	"testing"
	"time"
)

func TestNeuerMetadatenClient(t *testing.T) {
	client := NeuerMetadatenClient()

	if client == nil {
		t.Fatal("Erwartet MetadatenClient Instanz, aber nil erhalten")
	}

	if client.httpClient == nil {
		t.Fatal("Erwartet initialisierten httpClient, aber nil erhalten")
	}

	expectedTimeout := 10 * time.Second
	if client.httpClient.Timeout != expectedTimeout {
		t.Errorf("Erwartet Timeout von %v, aber %v erhalten", expectedTimeout, client.httpClient.Timeout)
	}
}
