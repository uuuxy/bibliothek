package repository

import (
	"context"
	"time"

	"bibliothek/pkg/betrag"
	"bibliothek/pkg/schulzeit"

	"github.com/jackc/pgx/v5"
)

// ErsatzwertGroessen sind die Zahlen, aus denen pkg/ersatzwert seinen Vorschlag rechnet. Der
// Vorschlag selbst entsteht nicht hier: Die Abfrage liefert Zahlen, die Staffel liegt in
// pkg/ersatzwert, und die Herleitung formuliert die Tür.
type ErsatzwertGroessen struct {
	// Kaufpreis ist der Einkaufspreis des Exemplars; 0, wenn keiner erfasst ist.
	Kaufpreis betrag.Cent
	// Listenpreis ist, was ein Ersatz heute kostet, der „Neupreis zum Zeitpunkt des Verlusts"
	// der Arbeitshilfe. 0 heißt „nicht erfasst"; die Staffel weicht dann auf den Kaufpreis aus
	// und sagt das in der Herleitung.
	Listenpreis betrag.Cent
	// ZustandAbschlag ist der Prozentsatz für den Zustand dieses Exemplars (0 bis 100).
	ZustandAbschlag int
	// SchuljahreMitAusleihe und SchuljahreImBestand sind die beiden Größen für das Verleihjahr,
	// beide in Schuljahren wie in OffeneForderung: ersatzwert.Verleihjahr nimmt das Maximum von
	// beiden.
	SchuljahreMitAusleihe int
	SchuljahreImBestand   int
	// Topf ist das Eigentum des Exemplars (ExemplarTopfSQL) und entscheidet die Regel des
	// Betrags: die Staffel der Arbeitshilfe für ein Buch des Landes, der Neuwert für alles
	// andere. Das Eigentum entscheidet und nicht ist_lernmittel des Titels: Auch eine Lektüre
	// aus Mitteln der Lernmittelfreiheit gehört dem Land, und das Land beschreibt in der
	// Arbeitshilfe selbst, was ein angemessener Betrag ist.
	Topf string
}

// LadeGroessenFuerExemplar liest die Größen des Staffel-Vorschlags für ein Exemplar, für den
// Dialog „Verlust/Schaden melden": Beim Melden entsteht die Forderung erst, aus der der Dialog
// des Bescheids seinen Vorschlag rechnet. Gezählt wird wie in LadeOffeneForderungen
// (schuljahrVon, zaehleSchuljahreMitAusleihe); zwei Auslegungen derselben Grenze verschöben den
// Betrag um eine Stufe der Staffel.
//
// Ein unbekanntes Exemplar liefert pgx.ErrNoRows: Es kann zwischen Anzeige und Klick
// ausgesondert oder gelöscht worden sein.
func (a *BescheidAbfragen) LadeGroessenFuerExemplar(ctx context.Context, exemplarID string) (ErsatzwertGroessen, error) {
	alle, err := a.ladeGroessen(ctx, "e.id = $1", exemplarID)
	if err != nil {
		return ErsatzwertGroessen{}, err
	}
	g, da := alle[exemplarID]
	if !da {
		// pgx.ErrNoRows und kein eigener Fehler: Der Rufer unterscheidet daran „Exemplar gibt
		// es nicht" (404) von einer gescheiterten Abfrage (500).
		return ErsatzwertGroessen{}, pgx.ErrNoRows
	}
	return g, nil
}

// LadeGroessenFuerTitel liest dieselben Größen für alle Exemplare eines Titels in einer
// Abfrage: Die Buchakte zeigt den heutigen Buchwert an jedem Exemplar, und ein Aufruf je
// Exemplar wären bei einem Klassensatz mit 30 Bänden 30 Abfragen für eine Seite.
func (a *BescheidAbfragen) LadeGroessenFuerTitel(ctx context.Context, titelID string) (map[string]ErsatzwertGroessen, error) {
	return a.ladeGroessen(ctx, "e.titel_id = $1", titelID)
}

// ladeGroessen ist die eine Abfrage hinter beiden Wegen: Zwei Fassungen wären zwei Auslegungen
// derselben Staffel, und die Zahl steht in einem Bescheid. coalesce auf den Listenpreis, weil
// die Spalte leer sein kann und 0 für die Staffel dasselbe bedeutet (sie weicht dann auf den
// Kaufpreis aus).
func (a *BescheidAbfragen) ladeGroessen(ctx context.Context, bedingung string, wert any) (map[string]ErsatzwertGroessen, error) {
	rows, err := a.db.Query(ctx, `
		SELECT e.id,
		       coalesce(e.einkaufspreis, 0),
		       coalesce(t.listenpreis, 0),
		       e.zustand_abwertung_prozent,
		       `+ExemplarTopfSQL+`,
		       -- Der Zugang, nicht der Bestelltag: Im Bestellweg entsteht die
		       -- Zeile beim Bestellen, in den Bestand kommt das Buch mit der Lieferung — bei
		       -- Lernmitteln regelmäßig im nächsten Schuljahr. Altbestand trägt in zugang_am
		       -- dasselbe Datum wie in erworben_am; NULL (noch im Zulauf) kann hier nicht
		       -- vorkommen, der Rückfall steht für Zeilen aus der Zeit vor der Migration.
		       COALESCE(e.zugang_am, e.erworben_am)
		FROM buecher_exemplare e
		JOIN buecher_titel t ON t.id = e.titel_id
		`+ExemplarTopfJoin+`
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
		var zugang *time.Time
		if err := rows.Scan(&id, &g.Kaufpreis, &g.Listenpreis, &g.ZustandAbschlag,
			&g.Topf, &zugang); err != nil {
			return nil, err
		}
		// Dieselbe Rechnung wie in LadeOffeneForderungen: Schuljahre, nicht Kalenderjahre.
		if zugang != nil {
			g.SchuljahreImBestand = schuljahrVon(schulzeit.Jetzt()) - schuljahrVon(*zugang)
		}
		alle[id] = g
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Derselbe Zähler wie beim Bescheid, statt einer zweiten Abfrage daneben.
	schuljahre, err := a.zaehleSchuljahreMitAusleihe(ctx, ids)
	if err != nil {
		return nil, err
	}
	for id, g := range alle {
		g.SchuljahreMitAusleihe = schuljahre[id]
		alle[id] = g
	}

	return alle, nil
}
