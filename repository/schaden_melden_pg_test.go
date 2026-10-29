package repository

import (
	"bibliothek/internal/testdaten"
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Der Grund der Aussonderung folgt der Fallgruppe: „Nicht zurückgegeben" ist VERLUST und nicht
// BESCHAEDIGUNG, weil der Fund und das endgültige Löschen im Fehlbestandsbericht nur VERLUST
// kennen.
func TestSchadenMelden_NichtZurueckgegebenIstVerlust(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewSchadensAbfragen(pool)

	copyID := legeSignaturMitExemplarenAn(t, pool, "VerlustGrund", 1)[0]
	schueler := legeSchuelerAn(t, pool, "VG-A", "Dana", "8b")
	bearbeiter := legeBearbeiterAn(t, pool)
	loan := legeAusleiheAn(t, pool, copyID, schueler, bearbeiter)

	if _, err := repo.Melde(ctx, loan, bearbeiter, "nicht zurückgegeben", SchadensArtNichtZurueck, 0); err != nil {
		t.Fatalf("Verlust melden: %v", err)
	}
	if grund := aussonderungsGrund(t, pool, copyID); grund != "VERLUST" {
		t.Errorf("aussonderung_grund = %q, erwartet VERLUST", grund)
	}
	var beendet bool
	if err := pool.QueryRow(ctx, `SELECT rueckgabe_am IS NOT NULL FROM ausleihen WHERE id = $1`, loan).Scan(&beendet); err != nil {
		t.Fatal(err)
	}
	if !beendet {
		t.Error("die Ausleihe läuft nach dem Verlust weiter")
	}
}

// Das Buch taucht beim Nachsuchen wieder auf: Der Fund im Fehlbestandsbericht beendet auch die
// Forderung, die das Buch abgerechnet hatte, dieselbe Regel wie an der Theke. Sonst käme das
// Exemplar zurück in den Umlauf, und die Forderung bliebe offen.
func TestMarkiereVerlustAlsGefunden_BeendetForderung(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	copyID := legeSignaturMitExemplarenAn(t, pool, "FundForderung", 1)[0]
	schueler := legeSchuelerAn(t, pool, "FF-A", "Emil", "9c")
	bearbeiter := legeBearbeiterAn(t, pool)
	loan := legeAusleiheAn(t, pool, copyID, schueler, bearbeiter)
	schadensID, err := NewSchadensAbfragen(pool).Melde(ctx, loan, bearbeiter, "nicht zurückgegeben", SchadensArtNichtZurueck, 1850)
	if err != nil {
		t.Fatalf("Verlust melden: %v", err)
	}
	// Unabhängig vom Grund, den SchadensAbfragen.Melde setzt: Der Fund verlangt VERLUST.
	if _, err := pool.Exec(ctx, `UPDATE buecher_exemplare SET aussonderung_grund = 'VERLUST' WHERE id = $1`, copyID); err != nil {
		t.Fatal(err)
	}

	gefunden, befund, err := NewInventurAbfragen(pool).MarkiereVerlustAlsGefunden(ctx, copyID, bearbeiter)
	if err != nil {
		t.Fatalf("MarkiereVerlustAlsGefunden: %v", err)
	}
	if !gefunden {
		t.Fatal("gefunden=false, erwartet true")
	}
	if befund.StornierteForderungen != 1 || befund.StornierterBetrag != 1850 {
		t.Errorf("Befund = %+v, erwartet 1 stornierte Forderung über 18,50", befund)
	}
	var storniert bool
	var grund string
	if err := pool.QueryRow(ctx,
		`SELECT storniert_am IS NOT NULL AND ist_bezahlt, coalesce(stornierungsgrund, '') FROM schadensfaelle WHERE id = $1`,
		schadensID).Scan(&storniert, &grund); err != nil {
		t.Fatal(err)
	}
	if !storniert {
		t.Errorf("Forderung nach dem Fund noch offen (Grund %q)", grund)
	}
}

// aussonderungsGrund liest den Grund eines Exemplars (leerer Text = keiner).
func aussonderungsGrund(t *testing.T, pool *pgxpool.Pool, copyID string) string {
	t.Helper()
	var grund string
	if err := pool.QueryRow(context.Background(),
		`SELECT coalesce(aussonderung_grund, '') FROM buecher_exemplare WHERE id = $1`, copyID).Scan(&grund); err != nil {
		t.Fatalf("Aussonderungsgrund lesen: %v", err)
	}
	return grund
}

// Ein Kollege bekommt keine Forderung: Der Bescheid ist ein Schreiben an Erziehungsberechtigte
// und braucht Klasse und Anschrift, die von einer Lehrkraft nirgends stehen; gehaftet wird
// gegenüber dem Dienstherrn und nur bei Vorsatz oder grober Fahrlässigkeit. Gebucht wird
// trotzdem, was den Bestand angeht. Eine Forderung stünde zudem in keiner Übersicht, weil beide
// Listen die Sicht schueler lesen.
func TestSchadenMelden_KollegeOhneForderung(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	copyID := legeSignaturMitExemplarenAn(t, pool, "KollegeVerlust", 1)[0]
	kollege := legeKollegenAn(t, pool, "A-90001", "Mara")
	bearbeiter := legeBearbeiterAn(t, pool)
	loan := legeAusleiheAn(t, pool, copyID, kollege, bearbeiter)

	schadensID, err := NewSchadensAbfragen(pool).Melde(ctx, loan, bearbeiter,
		"nicht zurückgegeben", SchadensArtNichtZurueck, 1500)
	if err != nil {
		t.Fatalf("Verlust melden: %v", err)
	}
	if schadensID != "" {
		t.Errorf("Kennung eines Schadensfalls = %q, erwartet leer", schadensID)
	}

	var forderungen int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM schadensfaelle WHERE ausleihe_id = $1`, loan).Scan(&forderungen); err != nil {
		t.Fatal(err)
	}
	if forderungen != 0 {
		t.Errorf("%d Forderung(en) gegen einen Kollegen angelegt — sie stünden in keiner Übersicht", forderungen)
	}

	// Der Bestand wird trotzdem geführt: Das Buch ist weg, die Ausleihe ist beendet.
	if grund := aussonderungsGrund(t, pool, copyID); grund != "VERLUST" {
		t.Errorf("aussonderung_grund = %q, erwartet VERLUST", grund)
	}
	var beendet bool
	if err := pool.QueryRow(ctx,
		`SELECT rueckgabe_am IS NOT NULL FROM ausleihen WHERE id = $1`, loan).Scan(&beendet); err != nil {
		t.Fatal(err)
	}
	if !beendet {
		t.Error("die Ausleihe des Kollegen läuft nach dem Verlust weiter")
	}
}

// Ohne Forderung gibt es keine Zeile, an der die Idempotenz hängt. Ein zweiter Klick ändert
// deshalb in keiner Fallgruppe etwas: nicht das Rückgabedatum, nicht die Notiz am
// ausgesonderten Exemplar und nicht das Protokoll.
func TestSchadenMelden_KollegeZweiterKlickVerschiebtNichts(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	bearbeiter := legeBearbeiterAn(t, pool)
	repo := NewSchadensAbfragen(pool)

	for _, fall := range []struct {
		art     SchadensArt
		ausweis string
	}{
		{SchadensArtNichtZurueck, "A-90002"},
		{SchadensArtBeschaedigt, "A-90003"},
	} {
		art := fall.art
		t.Run(string(art), func(t *testing.T) {
			copyID := legeSignaturMitExemplarenAn(t, pool, "KollegeZweimal"+string(art), 1)[0]
			kollege := legeKollegenAn(t, pool, fall.ausweis, "Tom")
			loan := legeAusleiheAn(t, pool, copyID, kollege, bearbeiter)
			stand := func() (rueckgabe time.Time, notiz string, eintraege int) {
				t.Helper()
				if err := pool.QueryRow(ctx, `
					SELECT a.rueckgabe_am, coalesce(e.zustand_notiz, ''),
					       (SELECT count(*) FROM audit_log l WHERE l.datensatz_id = e.id
					          AND l.details->>'action' = $2)
					FROM ausleihen a JOIN buecher_exemplare e ON e.id = a.exemplar_id
					WHERE a.id = $1`, loan, ProtokollAktionAusgesondert).Scan(&rueckgabe, &notiz, &eintraege); err != nil {
					t.Fatal(err)
				}
				return rueckgabe, notiz, eintraege
			}

			if _, err := repo.Melde(ctx, loan, bearbeiter, "Wasserschaden", art, 0); err != nil {
				t.Fatalf("erste Meldung: %v", err)
			}
			ersteRueckgabe, ersteNotiz, _ := stand()

			if _, err := repo.Melde(ctx, loan, bearbeiter, "Kaffeefleck", art, 0); err != nil {
				t.Fatalf("zweite Meldung: %v", err)
			}
			rueckgabe, notiz, eintraege := stand()
			if !rueckgabe.Equal(ersteRueckgabe) {
				t.Errorf("Rückgabedatum verschoben: %s → %s", ersteRueckgabe, rueckgabe)
			}
			if notiz != ersteNotiz {
				t.Errorf("Notiz am Exemplar überschrieben: %q → %q", ersteNotiz, notiz)
			}
			if eintraege != 1 {
				t.Errorf("%d Einträge im Protokoll, erwartet 1", eintraege)
			}
		})
	}
}

// Ein Kollege steht in derselben Tabelle wie ein Schüler, unterschieden nur durch `art`.
func legeKollegenAn(t *testing.T, pool *pgxpool.Pool, barcode, vorname string) string {
	t.Helper()
	id := testdaten.LegeLeserAn(t, pool, barcode, vorname, "Lehrkraft", "lehrkraft")
	return id
}
