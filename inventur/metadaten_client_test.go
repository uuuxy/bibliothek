package inventur

import (
	"testing"
	"time"
)

func TestNeuerMetadatenClient(t *testing.T) {
	client := NeuerMetadatenClient()

	if client == nil {
		t.Fatal("Erwartete einen initialisierten MetadatenClient, bekam nil")
	}

	if client.httpClient == nil {
		t.Fatal("Erwartete einen initialisierten httpClient, bekam nil")
	}

	expectedTimeout := 8 * time.Second
	if client.httpClient.Timeout != expectedTimeout {
		t.Errorf("Erwartetes Timeout von %v, bekam %v", expectedTimeout, client.httpClient.Timeout)
	}
}
