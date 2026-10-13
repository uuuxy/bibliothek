package safehttp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Die Adressen, die der Dialer ablehnt, und zwei öffentliche als Gegenprobe.
func TestVerbieteInterneZieladressen(t *testing.T) {
	faelle := []struct {
		name     string
		adresse  string
		gesperrt bool
	}{
		{"Öffentliche IPv4", "93.184.216.34:443", false},
		{"Öffentliche IPv6", "[2606:2800:220:1:248:1893:25c8:1946]:443", false},
		{"Loopback IPv4", "127.0.0.1:443", true},
		{"Loopback IPv6", "[::1]:443", true},
		{"Loopback als IPv4-in-IPv6", "[::ffff:127.0.0.1]:443", true},
		{"Privat 10/8", "10.0.0.5:80", true},
		{"Privat 172.16/12", "172.16.0.1:80", true},
		{"Privat 192.168/16", "192.168.1.10:443", true},
		{"Link-Local (Cloud-Metadaten)", "169.254.169.254:80", true},
		{"IPv6 Unique Local", "[fd00::1]:443", true},
		{"Unspezifiziert", "0.0.0.0:80", true},
		{"Keine IP", "example.com:80", true},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			err := verbieteInterneZieladressen("tcp4", f.adresse, nil)
			if gesperrt := err != nil; gesperrt != f.gesperrt {
				t.Errorf("verbieteInterneZieladressen(%q) gesperrt = %v (err: %v); want %v", f.adresse, gesperrt, err, f.gesperrt)
			}
		})
	}
}

// Der Angriffsweg am Client selbst: Ein erlaubter Host leitet auf ein internes Ziel weiter.
// Der Standard-Client folgt dem, dieser nicht.
//
// Die Server des Tests lauschen auf 127.0.0.1; der Client lehnt deshalb schon die erste
// Verbindung ab. Dass der Standard-Client bis ins Interne kommt, zeigt die Gegenprobe.
func TestNewClient_FolgtKeinerWeiterleitungAufsLoopback(t *testing.T) {
	intern := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("interne Daten")); err != nil {
			t.Errorf("Attrappen-Antwort schreiben: %v", err)
		}
	}))
	defer intern.Close()

	umleiter := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, intern.URL+"/geheim", http.StatusFound)
	}))
	defer umleiter.Close()

	// Die Gegenprobe zuerst: Der Standard-Client folgt der Weiterleitung bis ins Interne. Ohne
	// sie stünde nicht fest, ob der Test etwas prüft.
	if resp, err := http.Get(umleiter.URL); err == nil {
		defer func() {
			if cerr := resp.Body.Close(); cerr != nil {
				t.Errorf("Body schließen: %v", cerr)
			}
		}()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Vorbedingung: Standard-Client kam nicht durch (Status %d) — der Test prüft dann nichts", resp.StatusCode)
		}
	} else {
		t.Fatalf("Vorbedingung: Standard-Client scheiterte bereits (%v) — der Test prüft dann nichts", err)
	}

	// Es zählt, dass die Ablehnung aus der Prüfung der Adresse kommt und nicht aus einem
	// anderen Fehler.
	client := NewClient(5 * time.Second)
	resp, err := client.Get(umleiter.URL)
	if err == nil {
		if cerr := resp.Body.Close(); cerr != nil {
			t.Errorf("Body schließen: %v", cerr)
		}
		t.Fatal("gehärteter Client hat eine Loopback-Adresse erreicht; want Ablehnung")
	}
	if !strings.Contains(err.Error(), "nicht öffentlich") {
		t.Errorf("Ablehnung kam nicht aus der Adressprüfung: %v", err)
	}
}
