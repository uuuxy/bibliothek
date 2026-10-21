package service

import (
	"context"
	"testing"
	"time"

	"bibliothek/repository"
)

// Ein Scan mit Person ohne Netz trägt die Absicht „Ausleihe". Am Netz nimmt derselbe Scan ein Buch
// zurück, das schon auf dem Konto der Person liegt (BucheAmLeser). Das Nachbuchen tut es auch,
// wenn das Buch schon vor dem Besuch dort lag; was erst in diesem Besuch ausgeliehen wurde,
// kehrt ein zweiter Scan nicht um (bucheAusleiheNach).

func TestBucheNach_RueckgabeMitAusweisNimmtZurueck(t *testing.T) {
	w := nbAufbau(t)
	ctx := context.Background()
	t0 := time.Now().Add(-30 * time.Minute)

	// Anna hat das Buch seit gestern.
	if erg, err := w.dienst.BucheNach(ctx, w.eintrag(NachbuchAbsichtAusleihe, &w.anna, t0.Add(-24*time.Hour))); err != nil || erg.Ergebnis != repository.NachbuchAusgeliehen {
		t.Fatalf("Vorbedingung: %v %+v", err, erg)
	}

	// Ohne Netz: Anna zeigt um t0 ihren Ausweis und gibt 20 s später das Buch ab.
	e := w.eintrag(NachbuchAbsichtAusleihe, &w.anna, t0.Add(20*time.Second))
	seit := 20 * time.Second
	e.BesuchSeit = &seit
	erg, err := w.dienst.BucheNach(ctx, e)
	if err != nil || erg.Ergebnis != repository.NachbuchZurueckgegeben {
		t.Fatalf("Rückgabe mit Ausweis: %v %+v — erwartet zurueckgegeben, das Buch bliebe auf Annas Konto", err, erg)
	}
	if erg.Gebucht == nil || erg.Gebucht.Art != "rueckgabe" || erg.Gebucht.Fremdrueckgabe {
		t.Errorf("gebucht %+v, erwartet die eigene Rückgabe", erg.Gebucht)
	}
	if n, _ := w.offeneAusleihen(t); n != 0 {
		t.Fatalf("nach der Rückgabe noch %d offene Ausleihen", n)
	}
	var zurueck time.Time
	if err := w.pool.QueryRow(ctx, `SELECT rueckgabe_am FROM ausleihen WHERE exemplar_id = $1`, w.exemplarID).Scan(&zurueck); err != nil {
		t.Fatalf("Rückgabe lesen: %v", err)
	}
	if d := zurueck.Sub(t0.Add(20 * time.Second)); d < -time.Second || d > time.Second {
		t.Errorf("rueckgabe_am %v, erwartet den Scan-Zeitpunkt", zurueck)
	}
}

func TestBucheNach_DoppelscanImBesuchKehrtNichtsUm(t *testing.T) {
	w := nbAufbau(t)
	ctx := context.Background()
	t0 := time.Now().Add(-30 * time.Minute)

	// Der erste Scan des Besuchs leiht aus.
	erster := w.eintrag(NachbuchAbsichtAusleihe, &w.anna, t0)
	null := time.Duration(0)
	erster.BesuchSeit = &null
	if erg, err := w.dienst.BucheNach(ctx, erster); err != nil || erg.Ergebnis != repository.NachbuchAusgeliehen {
		t.Fatalf("erster Scan: %v %+v", err, erg)
	}

	// Derselbe Barcode 30 s später im selben Besuch: der Doppelscan.
	zweiter := w.eintrag(NachbuchAbsichtAusleihe, &w.anna, t0.Add(30*time.Second))
	dreissig := 30 * time.Second
	zweiter.BesuchSeit = &dreissig
	if erg, err := w.dienst.BucheNach(ctx, zweiter); err != nil || erg.Ergebnis != repository.NachbuchBereitsAusgeliehen {
		t.Fatalf("Doppelscan: %v %+v — erwartet bereits_ausgeliehen", err, erg)
	}

	// Noch einmal, und der Rechner misst den Besuch eine Millisekunde kürzer als die Uhr der
	// Scans: Zwei Uhren, auf Millisekunden gerundet (besuchsToleranz).
	dritter := w.eintrag(NachbuchAbsichtAusleihe, &w.anna, t0.Add(time.Minute))
	gerundet := time.Minute - time.Millisecond
	dritter.BesuchSeit = &gerundet
	if erg, err := w.dienst.BucheNach(ctx, dritter); err != nil || erg.Ergebnis != repository.NachbuchBereitsAusgeliehen {
		t.Fatalf("Doppelscan mit gerundeter Dauer: %v %+v — erwartet bereits_ausgeliehen", err, erg)
	}

	// Derselbe erste Scan noch einmal, wie nach der Übernahme eines verwaisten Schlüssels.
	if erg, err := w.dienst.BucheNach(ctx, erster); err != nil || erg.Ergebnis != repository.NachbuchBereitsAusgeliehen {
		t.Fatalf("übernommener Schlüssel: %v %+v — erwartet bereits_ausgeliehen", err, erg)
	}
	if n, s := w.offeneAusleihen(t); n != 1 || s != w.anna {
		t.Fatalf("%d offene Ausleihen, bei %s — erwartet eine bei Anna", n, s)
	}
}
