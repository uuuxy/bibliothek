package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
)

// Zwei Titel, ein Buch. Dasselbe Buch steht zweimal im Katalog, wenn zwei Wege es verschieden
// anlegen, etwa eine Liste ohne ISBN und der Bestands-Import mit ISBN. Zusammenführen hängt
// alles, was am einen Titel hängt, an den anderen und löscht ihn danach. Zwei Ausgaben eines
// Buchs sind dagegen Auflagen (auflagen.go) und bleiben eigene Titel.
//
// Regeln:
//   - Das Ziel bleibt mit seiner Kennung. Die Exemplare der Quelle wandern mit Nummer, Etikett
//     und Ausleihen, dazu Vormerkungen, Klassensatz-Reservierungen, Bestellpositionen,
//     Anliegen, Schlagworte und die Zuordnung zu Klassen; danach wird die Quelle gelöscht.
//   - Was am Ziel steht, bleibt; leere Felder füllt die Quelle auf, und ist einer der beiden ein
//     Lernmittel, ist es das Ziel danach auch (titel_zusammenfuehren_felder.go).
//   - Tragen beide eine ISBN, sind es zwei Ausgaben, und das Zusammenführen wird abgelehnt.
//   - Ein Eintrag im Protokoll hält die Quelle und alle gewanderten Zeilen fest, damit sich ein
//     falsches Paar von Hand trennen lässt.

var (
	// ErrTitelZusammenfuehrenGleich meldet: Ziel und Quelle sind derselbe Titel.
	ErrTitelZusammenfuehrenGleich = errors.New("ein titel lässt sich nicht mit sich selbst zusammenführen")
	// ErrTitelZusammenfuehrenZweiISBN meldet: Beide Titel tragen eine ISBN, sind also zwei
	// Ausgaben.
	ErrTitelZusammenfuehrenZweiISBN = errors.New("beide titel tragen eine isbn")
)

// protokollAktionTitelZusammengefuehrt kennzeichnet den Eintrag in details->>'action'.
const protokollAktionTitelZusammengefuehrt = "titel_zusammengefuehrt"

// TitelZusammenfuehrenAuftrag benennt die beiden Titel. NurVorschau führt alles aus und nimmt es
// am Ende zurück: Die Vorschau zeigt so genau, was das Zusammenführen täte.
type TitelZusammenfuehrenAuftrag struct {
	ZielID, QuelleID, BearbeiterID string
	NurVorschau                    bool
}

// TitelZusammenfuehrung sagt, was gewandert ist oder in der Vorschau wandern würde.
type TitelZusammenfuehrung struct {
	TitelID     string `json:"titel_id"`
	Titel       string `json:"titel"`
	QuelleID    string `json:"quelle_id"`
	QuelleTitel string `json:"quelle_titel"`
	QuelleISBN  string `json:"quelle_isbn"`
	Exemplare   int    `json:"exemplare"`
	// Vormerkungen wandern; VormerkungenDoppelt entfallen, weil derselbe Leser beide Titel
	// vorgemerkt hatte (es bleibt die weiter fortgeschrittene, sonst die ältere).
	Vormerkungen        int `json:"vormerkungen"`
	VormerkungenDoppelt int `json:"vormerkungen_doppelt"`
	Klassensaetze       int `json:"klassensaetze"`
	Bestellpositionen   int `json:"bestellpositionen"`
	Anliegen            int `json:"anliegen"`
	// Uebernommen sind die Felder, die das Ziel von der Quelle bekommt; Abweichend die, in
	// denen beide verschieden sind und das Ziel seinen Wert behält.
	Uebernommen []TitelFeld       `json:"uebernommen"`
	Abweichend  []TitelAbweichung `json:"abweichend"`
	// ISBN und ExemplareDanach beschreiben den bleibenden Titel danach, für die eine Zeile der
	// Vorschau; gezählt wie „gesamt“ im Katalog (SQLExemplarImBestand). An der Zahl sieht man,
	// wenn beide Titel dieselben Bücher trugen.
	ISBN            string `json:"isbn"`
	ExemplareDanach int    `json:"exemplare_danach"`
	// coverDateien sind lokale Cover der Quelle, die nach dem Commit gelöscht werden.
	coverDateien []string
}

// gewanderteZeilen sind die Kennungen der Zeilen, die den Titel gewechselt haben; der Eintrag im
// Protokoll hält sie fest.
type gewanderteZeilen struct {
	Exemplare, Vormerkungen, VormerkungenDoppelt, Klassensaetze, Bestellpositionen, Anliegen []string
	Schlagworte, Klassen                                                                     []string
}

// FuehreTitelZusammen führt die Quelle im Ziel auf, in einer Transaktion; mit NurVorschau nimmt
// es am Ende alles zurück. Lokale Cover der Quelle, die niemand mehr zeigt, löscht es nach dem
// Commit.
func FuehreTitelZusammen(ctx context.Context, q Datenbank, a TitelZusammenfuehrenAuftrag) (*TitelZusammenfuehrung, error) {
	// Eine Kennung kann in Großbuchstaben ankommen; Postgres sieht darin denselben Titel, der
	// Vergleich hier und die Schlüssel der Sperre müssen es auch (wie FasseAuflagenZusammen).
	a.ZielID, a.QuelleID = strings.ToLower(a.ZielID), strings.ToLower(a.QuelleID)
	if a.ZielID == a.QuelleID {
		return nil, ErrTitelZusammenfuehrenGleich
	}
	tx, err := q.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("titel zusammenführen: transaktion öffnen: %w", err)
	}
	defer db.Verwerfe(ctx, tx)

	erg, err := fuehreTitelZusammenIn(ctx, tx, a)
	if err != nil || a.NurVorschau {
		return erg, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("titel zusammenführen: commit: %w", err)
	}
	loescheCoverDateien(erg.coverDateien)
	return erg, nil
}

// fuehreTitelZusammenIn führt die Schritte in der Transaktion aus. Reihenfolge ist Schutz: erst
// die Sperre der Auflagen und beide Titel, dann wandern die Zeilen, dann fällt die Quelle, erst
// dann bekommt das Ziel ihre Felder, denn die ISBN ist eindeutig.
func fuehreTitelZusammenIn(ctx context.Context, tx pgx.Tx, a TitelZusammenfuehrenAuftrag) (*TitelZusammenfuehrung, error) {
	koepfe, err := sperreAuflagen(ctx, tx, a.ZielID, a.QuelleID)
	if err != nil {
		return nil, err
	}
	ziel, quelle, err := ladeTitelPaar(ctx, tx, a.ZielID, a.QuelleID)
	if err != nil {
		return nil, err
	}
	nachher, uebernommen, abweichend, err := fuegeTitelZusammen(ziel, quelle)
	if err != nil {
		return nil, err
	}
	g, err := verschiebeTitelZeilen(ctx, tx, ziel.id, quelle.id)
	if err != nil {
		return nil, err
	}
	erg := ergebnisDerZusammenfuehrung(ziel, quelle, g)
	erg.Uebernommen, erg.Abweichend, erg.ISBN = uebernommen, abweichend, wert(nachher.isbn)
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM buecher_exemplare e WHERE e.titel_id = $1 AND `+
		SQLExemplarImBestand, ziel.id).Scan(&erg.ExemplareDanach); err != nil {
		return nil, fmt.Errorf("titel zusammenführen: bestand zählen: %w", err)
	}
	if err := verbindeWerkMitZiel(ctx, tx, koepfe[ziel.id], koepfe[quelle.id]); err != nil {
		return nil, err
	}
	// Ein lokales Cover der Quelle fällt mit ihr, außer das Ziel übernimmt es.
	if wert(nachher.coverURL) != wert(quelle.coverURL) {
		if erg.coverDateien, err = ladeLokaleCoverNurDieserTitel(ctx, tx, []string{quelle.id}); err != nil {
			return nil, err
		}
	}
	if err := protokolliereZusammenfuehrung(ctx, tx, a.BearbeiterID, ziel, quelle, g); err != nil {
		return nil, err
	}
	if err := loescheQuelle(ctx, tx, ziel.id, quelle.id); err != nil {
		return nil, err
	}
	return erg, schreibeTitelStand(ctx, tx, nachher)
}

// ladeTitelPaar liest Ziel und Quelle, die sperreAuflagen schon gesperrt hat.
func ladeTitelPaar(ctx context.Context, tx pgx.Tx, zielID, quelleID string) (titelStand, titelStand, error) {
	ziel, err := ladeTitelStand(ctx, tx, zielID)
	if err != nil {
		return ziel, titelStand{}, err
	}
	quelle, err := ladeTitelStand(ctx, tx, quelleID)
	return ziel, quelle, err
}

// ergebnisDerZusammenfuehrung zählt, was gewandert ist.
func ergebnisDerZusammenfuehrung(ziel, quelle titelStand, g *gewanderteZeilen) *TitelZusammenfuehrung {
	return &TitelZusammenfuehrung{
		TitelID: ziel.id, Titel: ziel.titel, QuelleID: quelle.id, QuelleTitel: quelle.titel,
		QuelleISBN: wert(quelle.isbn), Exemplare: len(g.Exemplare), Vormerkungen: len(g.Vormerkungen),
		VormerkungenDoppelt: len(g.VormerkungenDoppelt), Klassensaetze: len(g.Klassensaetze),
		Bestellpositionen: len(g.Bestellpositionen), Anliegen: len(g.Anliegen),
	}
}

// protokolliereZusammenfuehrung schreibt den Rückweg an das Ziel: die Felder beider Titel vorher
// und die Kennungen aller gewanderten Zeilen. Ohne Person (bearbeiterID "") bleibt die Spalte
// leer, statt an einer ungültigen Kennung zu scheitern.
func protokolliereZusammenfuehrung(ctx context.Context, tx pgx.Tx, bearbeiterID string, ziel, quelle titelStand, g *gewanderteZeilen) error {
	kontext := "Titel zusammengeführt"
	var bearbeiter *string
	if bearbeiterID != "" {
		bearbeiter = &bearbeiterID
	}
	if err := schreibeProtokollEintrag(ctx, tx, protokollEintrag{
		Tabelle: "buecher_titel", Aktion: "UPDATE", DatensatzID: ziel.id,
		BearbeiterID: bearbeiter, Akteur: "USER", Kontext: &kontext,
		Details: map[string]any{
			"action": protokollAktionTitelZusammengefuehrt, "quelle_id": quelle.id,
			"quelle": quelle.alsProtokoll(), "ziel_vorher": ziel.alsProtokoll(), "gewandert": g,
		},
	}); err != nil {
		return fmt.Errorf("titel zusammenführen: protokoll: %w", err)
	}
	return nil
}

// loescheQuelle löscht die Quelle, nachdem alles von ihr gewandert ist, und hält die Regel der
// Auflagen: Ein Werk mit weniger als zwei Titeln fällt (ladeWerkeDerTitel, raeumeWerkeAuf).
func loescheQuelle(ctx context.Context, tx pgx.Tx, zielID, quelleID string) error {
	werke, err := ladeWerkeDerTitel(ctx, tx, []string{zielID, quelleID})
	if err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `DELETE FROM buecher_titel WHERE id = $1`, quelleID)
	if err != nil {
		return fmt.Errorf("titel zusammenführen: quelle löschen: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrTitelNichtGefunden
	}
	return raeumeWerkeAuf(ctx, tx, werke)
}

// verschiebeTitelZeilen hängt alles, was an der Quelle hängt, an das Ziel. Von zwei
// Vormerkungen desselben Lesers bleibt die weiter fortgeschrittene (abholbereit vor wartend),
// bei Gleichstand die ältere, dann die des Ziels (UNIQUE titel_id, schueler_id), wie beim
// Zusammenführen zweier Leser. Lag auch für die andere schon ein Exemplar im Abholfach, geht es
// an den Nächsten in der Warteschlange des Ziels, wie beim Löschen einer Vormerkung.
func verschiebeTitelZeilen(ctx context.Context, tx pgx.Tx, ziel, quelle string) (*gewanderteZeilen, error) {
	g := &gewanderteZeilen{}
	var freigaben []freigewordenesExemplar
	var err error
	if g.VormerkungenDoppelt, freigaben, err = loescheDoppelteVormerkungen(ctx, tx, `
		WITH paar AS (
			SELECT v.id, v.titel_id, v.schueler_id, v.erstellt_am,
			       CASE WHEN v.status = 'abholbereit' THEN 2 ELSE 1 END AS rang
			FROM vormerkungen v WHERE v.titel_id IN ($1, $2) AND v.schueler_id IS NOT NULL
		)
		DELETE FROM vormerkungen d USING paar v, paar w
		WHERE d.id = v.id AND v.schueler_id = w.schueler_id AND v.titel_id <> w.titel_id
		  AND (v.rang < w.rang
		       OR (v.rang = w.rang AND v.erstellt_am > w.erstellt_am)
		       OR (v.rang = w.rang AND v.erstellt_am = w.erstellt_am AND v.titel_id = $2))
		RETURNING d.id::text, CASE WHEN d.status = 'abholbereit' THEN d.bereitgestellt_exemplar_id::text END,
		          d.titel_id::text`, ziel, quelle); err != nil {
		return nil, fmt.Errorf("doppelte vormerkungen: %w", err)
	}
	schritte := []struct {
		was string
		sql string
		ids *[]string
	}{
		{"exemplare", `UPDATE buecher_exemplare SET titel_id = $1 WHERE titel_id = $2 RETURNING id::text`, &g.Exemplare},
		{"vormerkungen", `UPDATE vormerkungen SET titel_id = $1 WHERE titel_id = $2 RETURNING id::text`, &g.Vormerkungen},
		{"klassensätze", `UPDATE klassensatz_reservierungen SET titel_id = $1 WHERE titel_id = $2 RETURNING id::text`, &g.Klassensaetze},
		{"bestellpositionen", `UPDATE bestellungen_positionen SET titel_id = $1 WHERE titel_id = $2 RETURNING id::text`, &g.Bestellpositionen},
		{"anliegen", `UPDATE lehrer_anliegen SET titel_id = $1 WHERE titel_id = $2 RETURNING id::text`, &g.Anliegen},
		// Was das Ziel schon trägt, kommt nicht doppelt; der Rest der Quelle fällt mit ihr.
		{"schlagworte", `INSERT INTO titel_schlagworte (titel_id, schlagwort_id)
			SELECT $1, schlagwort_id FROM titel_schlagworte WHERE titel_id = $2
			ON CONFLICT DO NOTHING RETURNING schlagwort_id::text`, &g.Schlagworte},
		{"klassen", `INSERT INTO class_books (class_name, book_id)
			SELECT class_name, $1 FROM class_books WHERE book_id = $2
			ON CONFLICT DO NOTHING RETURNING class_name`, &g.Klassen},
	}
	for _, s := range schritte {
		if *s.ids, err = ladeIDs(ctx, tx, s.sql, ziel, quelle); err != nil {
			return nil, fmt.Errorf("%s verschieben: %w", s.was, err)
		}
	}
	// Jetzt stehen die Wartenden beider Titel in der Schlange des Ziels.
	for i := range freigaben {
		freigaben[i].titelID = &ziel
	}
	if err := rueckeNach(ctx, tx, freigaben, time.Now()); err != nil {
		return nil, fmt.Errorf("titel zusammenführen: %w", err)
	}
	return g, nil
}

// verbindeWerkMitZiel hängt das Ziel an das Werk der Quelle, wenn sie Auflage eines Buchs war:
// Das Ziel tritt an ihre Stelle. Hat das Ziel ein anderes Werk, werden beide eins.
func verbindeWerkMitZiel(ctx context.Context, tx pgx.Tx, ziel, quelle auflagenKopf) error {
	switch {
	case quelle.werkID == nil:
		return nil
	case ziel.werkID == nil:
		return setzeWerk(ctx, tx, *quelle.werkID, ziel.id)
	case *ziel.werkID != *quelle.werkID:
		return vereineWerke(ctx, tx, *ziel.werkID, *quelle.werkID)
	}
	return nil
}
