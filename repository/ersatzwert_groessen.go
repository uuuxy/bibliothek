package repository

import (
	"context"
	"time"

	"bibliothek/pkg/schulzeit"

	"github.com/jackc/pgx/v5"
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
	// Listenpreis ist, was ein Ersatz HEUTE kostet (Migration 127) — der „Neupreis zum
	// Zeitpunkt des Verlusts" der Arbeitshilfe. 0 heißt „nicht erfasst"; die Staffel
	// weicht dann auf den Kaufpreis aus und sagt das in der Herleitung.
	Listenpreis float64
	// ZustandAbschlag ist der Prozentsatz für den Zustand DIESES Exemplars (0–100).
	ZustandAbschlag int
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
	alle, err := r.groessen(ctx, "e.id = $1", exemplarID)
	if err != nil {
		return ErsatzwertGroessen{}, err
	}
	g, da := alle[exemplarID]
	if !da {
		// pgx.ErrNoRows, nicht ein eigener Fehler: Der Aufrufer unterscheidet daran
		// „Exemplar gibt es nicht" (404) von „Abfrage kaputt" (500).
		return ErsatzwertGroessen{}, pgx.ErrNoRows
	}
	return g, nil
}

// GroessenFuerTitel liest dieselben Größen für ALLE Exemplare eines Titels — in einer
// Abfrage, nicht einer pro Exemplar.
//
// Die Buchakte zeigt den heutigen Buchwert an jedem Exemplar (OFFEN.md 9.8, Stufe 2b).
// Ein Aufruf je Karte wäre bei einem Klassensatz mit 30 Bänden 30 Abfragen für eine
// Seite; die Zählung der Schuljahre nimmt ohnehin eine Liste.
func (r *pgBescheidRepository) GroessenFuerTitel(ctx context.Context, titelID string) (map[string]ErsatzwertGroessen, error) {
	return r.groessen(ctx, "e.titel_id = $1", titelID)
}

// groessen ist die EINE Abfrage hinter beiden Wegen. Zwei Fassungen desselben SELECTs
// wären zwei Auslegungen derselben Staffel — und die Zahl steht in einem Bescheid.
//
// coalesce auf den Listenpreis: Die Spalte ist nullbar (NULL = nicht erfasst), und
// 0 bedeutet für die Staffel dasselbe — sie weicht dann auf den Kaufpreis aus. Ein
// Scan in float64 ohne coalesce wäre ein 500 beim ersten Titel ohne Preis.
func (r *pgBescheidRepository) groessen(ctx context.Context, bedingung string, wert any) (map[string]ErsatzwertGroessen, error) {
	rows, err := r.db.Query(ctx, `
		SELECT e.id,
		       coalesce(e.einkaufspreis, 0)::float8,
		       coalesce(t.listenpreis, 0)::float8,
		       e.zustand_abwertung_prozent,
		       coalesce(t.ist_lernmittel, false),
		       e.erworben_am
		FROM buecher_exemplare e
		JOIN buecher_titel t ON t.id = e.titel_id
		WHERE `+bedingung, wert)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	alle := map[string]ErsatzwertGroessen{}
	ids := []string{}
	for rows.Next() {
		var id string
		var g ErsatzwertGroessen
		var erworben *time.Time
		if err := rows.Scan(&id, &g.Kaufpreis, &g.Listenpreis, &g.ZustandAbschlag,
			&g.IstLernmittel, &erworben); err != nil {
			return nil, err
		}
		// Dieselbe Rechnung wie in OffeneForderungen: Schuljahre, nicht Kalenderjahre.
		if erworben != nil {
			g.SchuljahreImBestand = schuljahrVon(schulzeit.Jetzt()) - schuljahrVon(*erworben)
		}
		alle[id] = g
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Geteilter Zähler mit dem Bescheid-Weg statt einer zweiten Abfrage daneben.
	schuljahre, err := r.schuljahreMitAusleihe(ctx, ids)
	if err != nil {
		return nil, err
	}
	for id, g := range alle {
		g.SchuljahreMitAusleihe = schuljahre[id]
		alle[id] = g
	}

	return alle, nil
}
