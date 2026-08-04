package safehttp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Diese Tabelle stand bis zum Audit in api/image_caching_test.go. Sie ist mit der
// Prüffunktion nach pkg/safehttp gewandert, weil dieselbe Schranke jetzt auch die
// Cover- und Metadaten-Abrufe des Inventur-Moduls schützt.
func TestVerbieteInterneZieladressen(t *testing.T) {
	tests := []struct {
		name    string
		address string
		blocked bool
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerbieteInterneZieladressen("tcp4", tt.address, nil)
			if blocked := err != nil; blocked != tt.blocked {
				t.Errorf("VerbieteInterneZieladressen(%q) blocked = %v (err: %v); want %v", tt.address, blocked, err, tt.blocked)
			}
		})
	}
}

// Der eigentliche Angriffsweg aus dem Audit, am echten Client durchgespielt: Ein
// Host, den eine Allowlist durchgewinkt hat, antwortet mit einem Redirect auf ein
// internes Ziel. Der Standard-Client folgt dem wortlos — dieser hier darf es nicht.
//
// Der Test belegt beides in einem Lauf: Der erste Hop (öffentlich erreichbarer
// Testserver) kommt durch, der zweite (Loopback) wird abgelehnt.
func TestNeuerClient_FolgtKeinemRedirectAufsLoopback(t *testing.T) {
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

	// Gegenprobe zuerst: Der Standard-Client folgt der Umleitung bis ins Interne.
	// Ohne diesen Nachweis wüssten wir nicht, ob der Test überhaupt etwas prüft.
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

	// httptest-Server lauschen auf 127.0.0.1. Der gehärtete Client darf deshalb
	// bereits den ERSTEN Hop nicht aufbauen; entscheidend ist, dass die Ablehnung
	// aus der Adressprüfung kommt und nicht aus einem beliebigen anderen Fehler.
	client := NeuerClient(5 * time.Second)
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
