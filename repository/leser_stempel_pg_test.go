package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bibliothek/internal/pgtest"
)

// leser.letzter_vorgang_am — die Uhr der Karenz, von Triggern gestempelt (Migration 137).
//
// Der Wert entsteht ausschließlich in der Datenbank: Eine Rückgabe kommt an der Theke, beim
// Nachbuchen eines Offline-Scans und über die Sammelrückgabe zustande, und jeder dieser
// Wege einzeln um „und dann noch den Stempel" zu ergänzen hieße, dass der nächste es
// vergisst. Geprüft wird deshalb das VERHALTEN der Trigger, nicht ihre Anwesenheit —
// repository/schema_gegenrichtung_pg_test.go zählt die Namen, dieser Test misst, was sie tun.
func TestLeserStempel_VorgangSetztUhrOhneAenderungsstempel(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	var titelID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel, autor, medientyp) VALUES ('Stempel-Testband', 'P', 'Buch')
		RETURNING id`).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	t.Cleanup(func() {
		auf := context.Background()
		for _, sql := range []string{
			`DELETE FROM schadensfaelle WHERE exemplar_id IN (SELECT id FROM buecher_exemplare WHERE titel_id = $1)`,
			`DELETE FROM ausleihen WHERE exemplar_id IN (SELECT id FROM buecher_exemplare WHERE titel_id = $1)`,
			`DELETE FROM buecher_exemplare WHERE titel_id = $1`,
			`DELETE FROM buecher_titel WHERE id = $1`,
		} {
			if _, err := pool.Exec(auf, sql, titelID); err != nil {
				t.Errorf("Aufräumen: %v", err)
			}
		}
		if _, err := pool.Exec(auf, `DELETE FROM leser WHERE barcode_id LIKE 'ST-%' || $1`, suffix); err != nil {
			t.Errorf("Aufräumen Leser: %v", err)
		}
	})

	leser := func(name string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, aktualisiert_am)
			VALUES ($1, $2, 'Stempel', '10A', 2026, now() - interval '400 days')
			RETURNING id`, "ST-"+name+"-"+suffix, name).Scan(&id); err != nil {
			t.Fatalf("Leser %s anlegen: %v", name, err)
		}
		return id
	}
	exemplar := func(name string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
			VALUES ($1, $2, true) RETURNING id`, titelID, "ST-EX-"+name+"-"+suffix).Scan(&id); err != nil {
			t.Fatalf("Exemplar anlegen: %v", err)
		}
		return id
	}
	// stand liest beide Uhren des Lesers auf einmal.
	stand := func(id string) (vorgang *time.Time, aktualisiert time.Time) {
		t.Helper()
		if err := pool.QueryRow(ctx,
			`SELECT letzter_vorgang_am, aktualisiert_am FROM leser WHERE id = $1`, id).
			Scan(&vorgang, &aktualisiert); err != nil {
			t.Fatalf("Stand lesen: %v", err)
		}
		return vorgang, aktualisiert
	}

	t.Run("die Rückgabe stempelt die Uhr und lässt aktualisiert_am stehen", func(t *testing.T) {
		// aktualisiert_am ist der Rückfall von AbgangSeit für Altzeilen ohne Abgangsstempel.
		// Schriebe eine Rückgabe ihn mit, schöbe sie genau die Uhr vor, um die es hier geht.
		id := leser("Rueck")
		_, vorher := stand(id)
		var ausleiheID string
		if err := pool.QueryRow(ctx, `
			INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist)
			VALUES ($1, $2, now() - interval '30 days', now() - interval '9 days')
			RETURNING id`, exemplar("Rueck"), id).Scan(&ausleiheID); err != nil {
			t.Fatalf("Ausleihe anlegen: %v", err)
		}
		if uhr, _ := stand(id); uhr != nil {
			t.Errorf("die offene Ausleihe hat die Uhr auf %v gestellt — gestempelt wird erst die Rückgabe", *uhr)
		}
		rueckgabe := time.Now().Add(-2 * time.Hour)
		if _, err := pool.Exec(ctx, `UPDATE ausleihen SET rueckgabe_am = $2 WHERE id = $1`, ausleiheID, rueckgabe); err != nil {
			t.Fatalf("Rückgabe buchen: %v", err)
		}
		uhr, nachher := stand(id)
		if uhr == nil {
			t.Fatal("die Rückgabe hat letzter_vorgang_am nicht gestempelt — die Karenz-Uhr rechnet dann allein ab dem Abgang")
		}
		if !uhr.Equal(rueckgabe) {
			t.Errorf("letzter_vorgang_am steht auf %v, erwartet der Rückgabezeitpunkt %v", *uhr, rueckgabe)
		}
		if !nachher.Equal(vorher) {
			t.Errorf("die Rückgabe hat aktualisiert_am von %v auf %v gezogen — eine Rückgabe ist keine "+
				"Änderung am Leser, und aktualisiert_am ist der Rückfall von AbgangSeit", vorher, nachher)
		}
	})

	t.Run("eine echte Änderung am Leser stempelt aktualisiert_am weiter", func(t *testing.T) {
		// Die Gegenprobe zum Fall darüber: Die Sperre gegen den Änderungsstempel darf nur
		// greifen, wenn sich NICHTS außer der Vorgangsuhr geändert hat.
		id := leser("Aendern")
		_, vorher := stand(id)
		if _, err := pool.Exec(ctx, `UPDATE leser SET klasse = '10B' WHERE id = $1`, id); err != nil {
			t.Fatalf("Klasse ändern: %v", err)
		}
		if _, nachher := stand(id); !nachher.After(vorher) {
			t.Errorf("aktualisiert_am blieb bei %v stehen, obwohl sich die Klasse geändert hat — "+
				"die Sperre aus Migration 137 greift zu weit", vorher)
		}
	})

	t.Run("der bezahlte Schadensfall stempelt", func(t *testing.T) {
		id := leser("Schaden")
		if _, err := pool.Exec(ctx, `
			INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag, ist_bezahlt)
			VALUES ($1, $2, 'Stempel-Probe', 5.00, false)`, exemplar("Schaden"), id); err != nil {
			t.Fatalf("Schadensfall anlegen: %v", err)
		}
		if uhr, _ := stand(id); uhr != nil {
			t.Errorf("der OFFENE Schadensfall hat die Uhr auf %v gestellt — gestempelt wird der Abschluss", *uhr)
		}
		if _, err := pool.Exec(ctx, `UPDATE schadensfaelle SET ist_bezahlt = true WHERE schueler_id = $1`, id); err != nil {
			t.Fatalf("Schadensfall bezahlen: %v", err)
		}
		if uhr, _ := stand(id); uhr == nil {
			t.Error("die Bezahlung hat letzter_vorgang_am nicht gestempelt")
		}
	})

	t.Run("die Uhr geht nicht zurück", func(t *testing.T) {
		// Ein nachgebuchter Offline-Scan trägt eine ÄLTERE Rückgabe nach. Sie darf die Uhr
		// nicht zurückdrehen — sonst verkürzte ein Nachtrag die Karenz eines Abgängers.
		//
		// Hier stehen ZWEI Ausleihen desselben Lesers; die Uhr nennt die spätere. Dass
		// derselbe Stempel auch einer Rückdatierung DERSELBEN Ausleihe nicht folgt, ist die
		// eine Abweichung vom früheren max(rueckgabe_am) — und ein Zustand, den keine Tür
		// der Anwendung herstellt: Alle drei Rückgabewege fassen nur offene Ausleihen an
		// (repository/loan.go ReturnLoanZumTx, repository/schaden_melden.go,
		// internal/service/device_service.go über activeLoan), die Littera-Übernahme fügt
		// nur ein. rueckgabe_am geht von NULL auf einen Wert und nie von einem Wert auf
		// einen anderen (nachgesehen am 23.09.2026).
		id := leser("Rueckwaerts")
		neu := time.Now().Add(-1 * time.Hour)
		alt := time.Now().AddDate(0, 0, -30)
		for _, p := range []struct {
			name string
			zeit time.Time
		}{{"neu", neu}, {"alt", alt}} {
			if _, err := pool.Exec(ctx, `
				INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am)
				VALUES ($1, $2, now() - interval '60 days', now() - interval '39 days', $3)`,
				exemplar("Rueckwaerts-"+p.name), id, p.zeit); err != nil {
				t.Fatalf("Ausleihe %s anlegen: %v", p.name, err)
			}
		}
		uhr, _ := stand(id)
		if uhr == nil || !uhr.Equal(neu) {
			t.Errorf("letzter_vorgang_am steht auf %v, erwartet die SPÄTERE Rückgabe %v — "+
				"ein nachgetragener Scan darf die Karenz-Uhr nicht zurückdrehen", uhr, neu)
		}
	})
}
