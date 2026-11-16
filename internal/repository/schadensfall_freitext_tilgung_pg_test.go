package repository

import (
	"context"
	"testing"
)

// Der Freitext eines erledigten Schadensfalls fällt mit der Anonymisierung. Der Fall bleibt als
// Beleg stehen (Betrag, Datum, bezahlt oder storniert); daran hängen Kassenbuch und Bescheid.
// Die Geschichte dazu bleibt nicht: „Buch im Bus liegen gelassen, Mutter angerufen" ist
// Personenbezug, der die Anonymisierung sonst überlebte, und die Löschfrist des Schadensfalls
// ist die des Belegs, nicht die der Erzählung.
//
// Die Gegenprobe: Eine offene Forderung behält ihren Text. Sie wird noch gebraucht, weil jemand
// sie einziehen, erklären oder stornieren können muss.
//
// Am echten Postgres, über TilgeSchuelerSpuren, den Weg, den das endgültige Löschen, der Abgang
// aus der LUSD und der nächtliche Lauf gemeinsam fahren.
func TestTilgeSchuelerSpuren_FreitextNurBeiErledigtenFaellen(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	schueler := legeSchuelerAn(t, pool, "S-FREI-1", "Freitext", "9c")
	exemplare := legeSignaturMitExemplarenAn(t, pool, "Freitext", 2)

	const erzaehlung = "Buch im Bus liegen gelassen, Mutter angerufen"

	var bezahlt, offen string
	if err := pool.QueryRow(ctx,
		`INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag, ist_bezahlt)
		 VALUES ($1, $2, $3, 20.00, true) RETURNING id`,
		exemplare[0], schueler, erzaehlung).Scan(&bezahlt); err != nil {
		t.Fatalf("bezahlten Schadensfall anlegen: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO schadensfaelle (exemplar_id, schueler_id, beschreibung, betrag, ist_bezahlt)
		 VALUES ($1, $2, $3, 15.00, false) RETURNING id`,
		exemplare[1], schueler, erzaehlung).Scan(&offen); err != nil {
		t.Fatalf("offenen Schadensfall anlegen: %v", err)
	}

	if err := TilgeSchuelerSpuren(ctx, pool, schueler, "Test"); err != nil {
		t.Fatalf("Spuren tilgen: %v", err)
	}

	var textBezahlt, textOffen string
	var betragBezahlt float64
	if err := pool.QueryRow(ctx,
		`SELECT beschreibung, betrag FROM schadensfaelle WHERE id = $1`, bezahlt).
		Scan(&textBezahlt, &betragBezahlt); err != nil {
		t.Fatalf("bezahlten Fall lesen: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`SELECT beschreibung FROM schadensfaelle WHERE id = $1`, offen).Scan(&textOffen); err != nil {
		t.Fatalf("offenen Fall lesen: %v", err)
	}

	if textBezahlt != "" {
		t.Errorf("der Freitext des erledigten Falls steht noch da: %q — er überlebt die "+
			"Anonymisierung und macht die Person über ihre Geschichte wieder erkennbar", textBezahlt)
	}
	if betragBezahlt != 20.00 {
		t.Errorf("der Beleg selbst muss bleiben, Betrag ist jetzt %v", betragBezahlt)
	}
	if textOffen != erzaehlung {
		t.Errorf("die offene Forderung hat ihre Begründung verloren (%q) — ohne sie steht ein "+
			"Betrag ohne Grund vor einer Familie", textOffen)
	}

	// Zweimal tilgen ist erlaubt und ändert nichts: Das endgültige Löschen und der nächtliche
	// Lauf können denselben Schüler nacheinander treffen.
	if err := TilgeSchuelerSpuren(ctx, pool, schueler, "Test"); err != nil {
		t.Fatalf("zweite Tilgung: %v", err)
	}
}
