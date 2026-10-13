package repository

// Was eine Seite ohne Anmeldung von einem Titel zeigen darf, steht hier einmal.
//
// Zwei Seiten lesen den Bestand ohne Sitzung: der Katalog (/katalog) und der Flur-Monitor
// (/monitor). Beide setzen dieselbe Bedingung ein; wer die Regel ändert, ändert sie für beide,
// und api/portal/monitor_pg_test.go hält das Paar zusammen. Mit zwei Regeln stünde auf einem
// Schulserver, dessen Bestand zum größten Teil aus Schulbüchern besteht, das Mathebuch der 7
// als „Buch des Monats" auf dem Monitor.

import (
	"context"
	"fmt"

	"bibliothek/db"
)

// OeffentlichSichtbar liefert das SQL-Prädikat, unter dem ein Titel (Tabelle
// buecher_titel unter dem Alias titelAlias) auf einer Seite ohne Anmeldung erscheinen darf:
//
//  1. kein Lernmittel (buecher_titel.ist_lernmittel).
//     Schulbücher werden klassensatzweise zugeteilt, niemand recherchiert freiwillig nach
//     dem Biologiebuch der 8. Klasse; im Katalog stehend würden sie die Treffer der
//     eigentlichen Freihand-Bibliothek zuschütten, auf dem Monitor jede Folie gewinnen.
//  2. mindestens ein Exemplar ist im Haus: nicht ausgesondert und nicht nur bestellt.
//     Ein Titel, dessen Exemplare alle abgegangen sind, ist kein „Buch des Monats" mehr;
//     einer, der erst im Zulauf ist, ist noch nicht „Neu eingetroffen".
//
// Den Alias schreibt der Programmcode und nie eine Eingabe; er lässt sich deshalb einsetzen.
func OeffentlichSichtbar(titelAlias string) string {
	return "(NOT " + titelAlias + `.ist_lernmittel
		AND EXISTS (
			SELECT 1 FROM buecher_exemplare oe
			WHERE oe.titel_id = ` + titelAlias + `.id
			  AND oe.ist_ausgesondert = false
			  AND oe.bestellstatus IS NULL
		))`
}

// ── Flur-Monitor ─────────────────────────────────────────────────────────────

// MonitorTitel ist, was der Flur-Monitor von einem Titel zeigt: Angaben zum Titel und nie,
// wer ihn ausgeliehen hat.
type MonitorTitel struct {
	ID       string `json:"id"`
	Titel    string `json:"titel"`
	Autor    string `json:"autor"`
	CoverURL string `json:"cover_url"`
	// ISBN ist der Schlüssel, unter dem der Cover-Proxy (/api/images/cover) das Bild führt.
	// Ohne sie müsste der Monitor die fremde Cover-URL selbst einbinden, und das lässt die
	// Content-Security-Policy nicht zu (img-src ohne https:).
	ISBN string `json:"isbn"`
}

// MonitorFolien ist die ganze Antwort für den Flur-Monitor: drei Folien.
type MonitorFolien struct {
	BuchDesMonats   *MonitorTitel  `json:"buch_des_monats"`
	NeuEingetroffen []MonitorTitel `json:"neu_eingetroffen"`
	Beliebt         []MonitorTitel `json:"beliebt"`
}

// Fenster und Längen der Folien (docs/FACHKONZEPT.md §16.2).
const (
	monitorBuchDesMonatsTage = 30
	monitorBeliebtTage       = 7
	monitorNeuAnzahl         = 10
	monitorBeliebtAnzahl     = 5
)

// monitorSpalten ist die Auswahl jeder Folie, in der Reihenfolge, in der titelListe sie liest.
const monitorSpalten = `bt.id, bt.titel, COALESCE(bt.autor, ''), COALESCE(bt.cover_url, ''), COALESCE(bt.isbn, '')`

// monitorMitCover hält Titel ohne Cover von „Buch des Monats" und „Neu eingetroffen" fern:
// Eine Folie ohne Bild taugt auf einem Bildschirm im Flur nicht. „Beliebt" zeigt auch Titel
// ohne Cover, mit einem Platzhalter; sonst wäre die Liste nur kürzer.
const monitorMitCover = `bt.cover_url IS NOT NULL AND bt.cover_url <> ''`

// MonitorAbfragen liest die drei Folien des Flur-Monitors.
type MonitorAbfragen struct {
	pool db.PgxPoolIface
}

// NewMonitorAbfragen liefert die Abfragen des Monitors am übergebenen Pool.
func NewMonitorAbfragen(pool db.PgxPoolIface) *MonitorAbfragen {
	return &MonitorAbfragen{pool: pool}
}

// LadeFolien liefert die drei Folien. Fehlt ein „Buch des Monats", weil im Fenster niemand
// gelesen hat, steht dort der zuletzt angelegte Titel mit Cover, derselbe, der „Neu
// eingetroffen" anführt.
func (a *MonitorAbfragen) LadeFolien(ctx context.Context) (MonitorFolien, error) {
	folien := MonitorFolien{NeuEingetroffen: []MonitorTitel{}, Beliebt: []MonitorTitel{}}

	meist, err := a.meistgelesen(ctx, monitorBuchDesMonatsTage, 1, true)
	if err != nil {
		return folien, fmt.Errorf("folie buch_des_monats: %w", err)
	}
	neu, err := a.neuEingetroffen(ctx, monitorNeuAnzahl)
	if err != nil {
		return folien, fmt.Errorf("folie neu_eingetroffen: %w", err)
	}
	beliebt, err := a.meistgelesen(ctx, monitorBeliebtTage, monitorBeliebtAnzahl, false)
	if err != nil {
		return folien, fmt.Errorf("folie beliebt: %w", err)
	}

	switch {
	case len(meist) > 0:
		folien.BuchDesMonats = &meist[0]
	case len(neu) > 0:
		ersatz := neu[0]
		folien.BuchDesMonats = &ersatz
	}
	folien.NeuEingetroffen = neu
	folien.Beliebt = beliebt
	return folien, nil
}

// meistgelesen liefert die Titel, die in den letzten tage Tagen die meisten Leser hatten.
//
// Gezählt wird je Leser einmal und nicht je Ausleihe: Wer denselben Titel zweimal leiht, ist
// ein Leser, und ein Klassensatz an eine Lehrkraft ist ein Vorgang und keine dreißig Leser.
// Anonymisierte Ausleihen zählen nicht; sie liegen ohnehin außerhalb der Fenster.
func (a *MonitorAbfragen) meistgelesen(ctx context.Context, tage, anzahl int, nurMitCover bool) ([]MonitorTitel, error) {
	cover := "TRUE"
	if nurMitCover {
		cover = monitorMitCover
	}
	return a.titelListe(ctx, `
		SELECT `+monitorSpalten+`
		FROM ausleihen a
		JOIN buecher_exemplare e ON e.id = a.exemplar_id
		JOIN buecher_titel bt ON bt.id = e.titel_id
		WHERE a.ausgeliehen_am >= NOW() - make_interval(days => $1::int)
		  AND a.schueler_id IS NOT NULL
		  -- Dauerleihen zählen nicht: Ein Klassensatz ist EIN Vorgang, keine 30 Leser.
		  -- Bis Migration 125 fiel er von selbst heraus — er stand in einer anderen
		  -- Spalte, und die Zählung sah nur schueler_id. Seit alle Leser in einer Tabelle
		  -- stehen, muss die Regel dastehen, sonst führte das Klassenbuch der 7 den
		  -- Flur-Monitor an. Ein Kollege, der sich einen Roman mitnimmt, zählt weiter mit:
		  -- Er ist ein Leser wie jeder andere.
		  AND NOT a.ist_handapparat
		  AND `+cover+`
		  AND `+OeffentlichSichtbar("bt")+`
		GROUP BY bt.id, bt.titel, bt.autor, bt.cover_url, bt.isbn
		ORDER BY COUNT(DISTINCT a.schueler_id) DESC, bt.titel ASC
		LIMIT $2::int`, tage, anzahl)
}

// neuEingetroffen liefert die zuletzt angelegten Titel mit Cover.
func (a *MonitorAbfragen) neuEingetroffen(ctx context.Context, anzahl int) ([]MonitorTitel, error) {
	return a.titelListe(ctx, `
		SELECT `+monitorSpalten+`
		FROM buecher_titel bt
		WHERE `+monitorMitCover+`
		  AND `+OeffentlichSichtbar("bt")+`
		ORDER BY bt.erstellt_am DESC, bt.titel ASC
		LIMIT $1::int`, anzahl)
}

// titelListe führt die Abfrage einer Folie aus und liest die monitorSpalten.
func (a *MonitorAbfragen) titelListe(ctx context.Context, query string, args ...any) ([]MonitorTitel, error) {
	rows, err := a.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	liste := []MonitorTitel{}
	for rows.Next() {
		var t MonitorTitel
		if err := rows.Scan(&t.ID, &t.Titel, &t.Autor, &t.CoverURL, &t.ISBN); err != nil {
			return nil, err
		}
		liste = append(liste, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return liste, nil
}
