package repository

import (
	"context"
	"time"

	"bibliothek/pkg/schulzeit"
)

// Die Größen des Staffel-Vorschlags für EIN Exemplar — für den Dialog „Verlust/Schaden
// melden" (OFFEN.md 9.3 a, Sichtung des Medienzentrums vom 16.09.2026).
//
// Warum das nötig ist: Der Bescheid-Dialog rechnet seinen Vorschlag aus OffeneForderung,
// also aus einer Forderung, die es schon GIBT. Beim Melden entsteht sie gerade erst — es
// gibt keine Forderung, aus der man lesen könnte, und bis zum 17.09.2026 stand im Feld
// „Ersatzbetrag" deshalb eine feste 15,00 € ohne jeden Bezug zum Buch. Das Protokoll des
// Medienzentrums nennt genau das: „weder Einkaufs- bzw. Listenpreise noch
// Beschädigungsgrade hinterlegt, fehlende Restwertberechnung z.Zt. Handeingabe."
//
// Die Zählung ist BEWUSST dieselbe wie in OffeneForderungen und nicht eine zweite: beide
// zählen Schuljahre über schuljahrVon und schuljahreMitAusleihe. Zwei Auslegungen
// derselben Grenze verschöben den Betrag um eine ganze Stufe der Staffel — und diese
// Zahl steht am Ende in einem Bescheid an Erziehungsberechtigte.

// ErsatzwertGroessen sind die Zahlen, aus denen pkg/ersatzwert seinen Vorschlag rechnet.
// Der Vorschlag selbst entsteht nicht hier: Das Repository liefert Zahlen, die Staffel
// liegt in pkg/ersatzwert, und die Herleitung formuliert die API-Schicht.
type ErsatzwertGroessen struct {
	// Kaufpreis ist der Einkaufspreis des Exemplars; 0, wenn keiner erfasst ist.
	Kaufpreis float64
	// Die beiden Größen für das Verleihjahr, beide in SCHULJAHREN — dieselbe Bedeutung
	// wie in OffeneForderung, weil ersatzwert.Verleihjahr das Maximum von beiden nimmt.
	SchuljahreMitAusleihe int
	SchuljahreImBestand   int
	// IstLernmittel entscheidet, ob die Staffel überhaupt gilt: Sie steht in der
	// Arbeitshilfe für Schulbücher der Lernmittelfreiheit. Für einen Roman aus der
	// Schülerbücherei gibt es keine Vorgabe des Landes.
	IstLernmittel bool
}

// GroessenFuerExemplar liest die Größen des Staffel-Vorschlags für ein Exemplar.
//
// Ein unbekanntes Exemplar ist kein Fehler, sondern ein leeres Ergebnis: Der Dialog
// fragt beim Öffnen, und ein Exemplar kann zwischen Anzeige und Klick ausgesondert oder
// gelöscht worden sein. Ein 500 an dieser Stelle nähme dem Personal die Möglichkeit,
// den Schaden überhaupt zu melden — es soll dann nur den Betrag selbst eintragen.
func (r *pgBescheidRepository) GroessenFuerExemplar(ctx context.Context, exemplarID string) (ErsatzwertGroessen, error) {
	var g ErsatzwertGroessen
	var erworben *time.Time

	err := r.db.QueryRow(ctx, `
		SELECT coalesce(e.einkaufspreis, 0)::float8,
		       coalesce(t.ist_lernmittel, false),
		       e.erworben_am
		FROM buecher_exemplare e
		JOIN buecher_titel t ON t.id = e.titel_id
		WHERE e.id = $1`, exemplarID).Scan(&g.Kaufpreis, &g.IstLernmittel, &erworben)
	if err != nil {
		return ErsatzwertGroessen{}, err
	}

	// Dieselbe Rechnung wie in OffeneForderungen: Schuljahre, nicht Kalenderjahre.
	if erworben != nil {
		g.SchuljahreImBestand = schuljahrVon(schulzeit.Jetzt()) - schuljahrVon(*erworben)
	}

	// Geteilter Zähler mit dem Bescheid-Weg statt einer zweiten Abfrage daneben.
	schuljahre, err := r.schuljahreMitAusleihe(ctx, []string{exemplarID})
	if err != nil {
		return ErsatzwertGroessen{}, err
	}
	g.SchuljahreMitAusleihe = schuljahre[exemplarID]

	return g, nil
}
