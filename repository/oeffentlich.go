package repository

// oeffentlich.go — was eine Seite OHNE Anmeldung von einem Titel zeigen darf, genau einmal
// aufgeschrieben.
//
// Zwei Seiten lesen den Bestand ohne Sitzung: der Katalog (/katalog, api/opac.go) und der
// Flur-Monitor (/monitor, api/monitor.go). Bis zum 30.08.2026 hatte jede ihre eigene
// Regel: Der Katalog ließ Lernmittel weg und zeigte nur Titel mit einem Exemplar im Haus;
// der Monitor kannte beides nicht — ein Paar, das nur zufällig einig war (docs/sweeps.md,
// Geschwister-Asymmetrie). Auf einem Schulserver, dessen Bestand zum größten Teil aus
// Schulbüchern besteht, wäre das Mathebuch der 7 „Buch des Monats" gewesen; das
// Demo-Seed musste genau darum herumtricksen („sonst gewinnen Lernmittel").
//
// Deshalb: Beide Seiten setzen DENSELBEN String ein. Wer die Regel ändert, ändert sie für
// beide — und api/monitor_pg_test.go hält das Paar zusammen.

import (
	"context"
	"fmt"

	"bibliothek/db"
)

// OeffentlichSichtbar liefert das SQL-Prädikat, unter dem ein Titel (Tabelle
// buecher_titel unter dem Alias titelAlias) auf einer Seite ohne Anmeldung erscheinen darf:
//
//  1. kein Lernmittel (buecher_titel.ist_lernmittel, Migration 093).
//     Schulbücher werden klassensatzweise zugeteilt, niemand recherchiert freiwillig nach
//     dem Biologiebuch der 8. Klasse; im Katalog stehend würden sie die Treffer der
//     eigentlichen Freihand-Bibliothek zuschütten, auf dem Monitor jede Folie gewinnen.
//  2. mindestens ein Exemplar ist im Haus: nicht ausgesondert und nicht nur bestellt.
//     Ein Titel, dessen Exemplare alle abgegangen sind, ist kein „Buch des Monats" mehr;
//     einer, der erst im Zulauf ist, ist noch nicht „Neu eingetroffen".
//
// Der Alias ist entwickler-definiert (nie nutzergesteuert), die Einbettung daher sicher.
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

// MonitorTitel ist die schlanke Titelansicht des Flur-Monitors — Titeldaten, nie Ausleiher.
type MonitorTitel struct {
	ID       string `json:"id"`
	Titel    string `json:"titel"`
	Autor    string `json:"autor"`
	CoverURL string `json:"cover_url"`
	// ISBN dient allein als Cache-Schlüssel des Cover-Proxys (/api/images/cover). Ohne
	// sie müsste der Monitor eine externe Cover-URL direkt einbinden — genau das
	// Hotlinking, das die Content-Security-Policy seit dem 06.08.2026 nicht mehr
	// zulässt (img-src ohne https:).
	ISBN string `json:"isbn"`
}

// MonitorSlides ist die vollständige Antwort für den Flur-Monitor: drei Folien.
type MonitorSlides struct {
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

// monitorSpalten ist die Auswahl jeder Folie — in der Reihenfolge, in der titelListe
// sie liest.
const monitorSpalten = `bt.id, bt.titel, COALESCE(bt.autor, ''), COALESCE(bt.cover_url, ''), COALESCE(bt.isbn, '')`

// monitorMitCover: Auf „Buch des Monats" und „Neu eingetroffen" kommen Titel ohne Cover
// bewusst nicht vor — eine Folie ohne Bild ist auf einem Flurbildschirm wertlos. „Beliebt"
// zeigt auch Titel ohne Cover (mit Platzhalter-Kachel), sonst wäre die Liste kürzer, ohne
// dass jemand etwas gewinnt.
const monitorMitCover = `bt.cover_url IS NOT NULL AND bt.cover_url <> ''`

// MonitorRepository liest die drei Folien des Flur-Monitors.
type MonitorRepository struct {
	pool db.PgxPoolIface
}

// NewMonitorRepository bindet das Repository an einen Pool.
func NewMonitorRepository(pool db.PgxPoolIface) *MonitorRepository {
	return &MonitorRepository{pool: pool}
}

// LadeSlides liefert die drei Folien. Fehlt ein „Buch des Monats" (kein Leser im
// Fenster), springt der zuletzt angelegte Titel mit Cover ein — derselbe, der „Neu
// eingetroffen" anführt.
func (r *MonitorRepository) LadeSlides(ctx context.Context) (MonitorSlides, error) {
	slides := MonitorSlides{NeuEingetroffen: []MonitorTitel{}, Beliebt: []MonitorTitel{}}

	meist, err := r.meistgelesen(ctx, monitorBuchDesMonatsTage, 1, true)
	if err != nil {
		return slides, fmt.Errorf("folie buch_des_monats: %w", err)
	}
	neu, err := r.neuEingetroffen(ctx, monitorNeuAnzahl)
	if err != nil {
		return slides, fmt.Errorf("folie neu_eingetroffen: %w", err)
	}
	beliebt, err := r.meistgelesen(ctx, monitorBeliebtTage, monitorBeliebtAnzahl, false)
	if err != nil {
		return slides, fmt.Errorf("folie beliebt: %w", err)
	}

	switch {
	case len(meist) > 0:
		slides.BuchDesMonats = &meist[0]
	case len(neu) > 0:
		ersatz := neu[0]
		slides.BuchDesMonats = &ersatz
	}
	slides.NeuEingetroffen = neu
	slides.Beliebt = beliebt
	return slides, nil
}

// meistgelesen liefert die Titel, die in den letzten tage Tagen die meisten LESER hatten.
//
// Gezählt werden Schüler-Ausleihen, je Schüler einmal — nicht Ausleihzeilen. Der
// Unterschied ist kein Feinschliff: Lehrer-Ausleihen stehen je Exemplar als eigene Zeile
// in ausleihen (repository/loan.go); ein Klassensatz „Die Welle" ×30 an eine Lehrkraft
// hätte mit 30 „Ausleihen" jede Folie beherrscht, obwohl ihn kein Schüler freiwillig
// gelesen hat. Und wer denselben Titel zweimal leiht, ist ein Leser, nicht zwei.
// Anonymisierte Ausleihen (kein Ausleiher mehr) zählen nicht — sie liegen ohnehin weit
// außerhalb der Fenster. Entscheidung 30.08.2026 (Frage-Runde 2c).
func (r *MonitorRepository) meistgelesen(ctx context.Context, tage, anzahl int, nurMitCover bool) ([]MonitorTitel, error) {
	cover := "TRUE"
	if nurMitCover {
		cover = monitorMitCover
	}
	return r.titelListe(ctx, `
		SELECT `+monitorSpalten+`
		FROM ausleihen a
		JOIN buecher_exemplare e ON e.id = a.exemplar_id
		JOIN buecher_titel bt ON bt.id = e.titel_id
		WHERE a.ausgeliehen_am >= NOW() - make_interval(days => $1::int)
		  AND a.schueler_id IS NOT NULL
		  AND `+cover+`
		  AND `+OeffentlichSichtbar("bt")+`
		GROUP BY bt.id, bt.titel, bt.autor, bt.cover_url, bt.isbn
		ORDER BY COUNT(DISTINCT a.schueler_id) DESC, bt.titel ASC
		LIMIT $2::int`, tage, anzahl)
}

// neuEingetroffen liefert die zuletzt angelegten Titel mit Cover.
func (r *MonitorRepository) neuEingetroffen(ctx context.Context, anzahl int) ([]MonitorTitel, error) {
	return r.titelListe(ctx, `
		SELECT `+monitorSpalten+`
		FROM buecher_titel bt
		WHERE `+monitorMitCover+`
		  AND `+OeffentlichSichtbar("bt")+`
		ORDER BY bt.erstellt_am DESC, bt.titel ASC
		LIMIT $1::int`, anzahl)
}

// titelListe führt eine Folien-Abfrage aus und liest die monitorSpalten.
func (r *MonitorRepository) titelListe(ctx context.Context, query string, args ...any) ([]MonitorTitel, error) {
	rows, err := r.pool.Query(ctx, query, args...)
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
