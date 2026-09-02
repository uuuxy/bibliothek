package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
)

// Zwei Datensätze, ein Mensch. Ohne Schüler-ID im LUSD-Export entsteht bei einer
// Namensänderung oder Datumskorrektur ein zweiter Datensatz (Abgänger + Neuanlage,
// siehe api/lusd_paarung.go für den Fang beim Import). Fällt das erst später auf — das
// Kind steht mit dem alten Ausweis an der Theke und gilt als gesperrter Abgänger — führt
// diese Funktion beide Zeilen zu einer zusammen. Sie ist das Sicherheitsnetz hinter der
// Vorschau-Paarung und gilt genauso für Dubletten aus Handanlage + Import.
//
// Regeln (Entscheidung Peter, 02.09.2026):
//   - Das ZIEL bleibt: dieselbe UUID, derselbe Barcode — Ausweis und Historie gelten weiter.
//   - Die QUELLE geht auf: Ausleihen, Schäden, Vormerkungen, Foto und Protokollspuren
//     wandern zum Ziel, danach wird die Zeile endgültig gelöscht (kein Papierkorb — eine
//     wiederherstellbare Hülle wäre die dritte Wahrheit über dieselbe Person).
//   - LUSD bleibt führend: Die Stammdaten kommen von dem Datensatz, den ein LUSD-Export
//     zuletzt bestätigt hat (lusd_bestaetigt_am); ein aktiver Datensatz schlägt dabei
//     einen Abgänger (fuehrend); leere Felder füllt der andere auf.
//   - Das Ziel ist danach aktiv (kein Abgänger). Eine automatische Abgänger-Sperre fällt,
//     sofern keine Vorgänge offen sind; eine manuelle Sperre bleibt.

var (
	// ErrZusammenfuehrenGleich meldet: Ziel und Quelle sind derselbe Datensatz.
	ErrZusammenfuehrenGleich = errors.New("ziel und quelle sind derselbe datensatz")
	// ErrZusammenfuehrenNichtGefunden meldet: einer der beiden fehlt oder liegt im Papierkorb.
	ErrZusammenfuehrenNichtGefunden = errors.New("schüler nicht gefunden oder gelöscht")
	// ErrZusammenfuehrenAnonymisiert meldet: ein anonymisierter Datensatz trägt keine Person mehr.
	ErrZusammenfuehrenAnonymisiert = errors.New("anonymisierte datensätze lassen sich nicht zusammenführen")
)

// ZusammenfuehrenAuftrag benennt die beiden Datensätze; AbgaengerJahr rechnet das
// Abgangsjahr aus der übernommenen Klasse (dieselbe Regel wie bei der Handanlage,
// api/student_create.go — sie gehört dem Aufrufer, nicht dieser Schicht).
type ZusammenfuehrenAuftrag struct {
	ZielID, QuelleID string
	AbgaengerJahr    func(klasse string) int
	// BearbeiterID: wer zusammenführt — steht am Rückweg-Eintrag (audit_log); leer = SYSTEM.
	BearbeiterID string
}

// ZusammenfuehrenErgebnis ist das, was die Oberfläche nach dem Zusammenführen zeigt:
// der verbliebene Datensatz und was zu ihm gewandert ist.
type ZusammenfuehrenErgebnis struct {
	ZielID        string `json:"ziel_id"`
	BarcodeID     string `json:"barcode_id"`
	Vorname       string `json:"vorname"`
	Nachname      string `json:"nachname"`
	Klasse        string `json:"klasse"`
	QuelleBarcode string `json:"quelle_barcode"`
	Ausleihen     int64  `json:"ausleihen"`
	Schaeden      int64  `json:"schaeden"`
	Vormerkungen  int64  `json:"vormerkungen"`
	// DoppelteVormerkungen: Vormerkungen der Quelle auf Titel, die das Ziel schon
	// vorgemerkt hatte — sie fallen weg (UNIQUE titel_id, schueler_id).
	DoppelteVormerkungen int64 `json:"doppelte_vormerkungen"`
}

// zusammenfuehrenZeile sind die Felder, die beim Zusammenführen entschieden werden.
type zusammenfuehrenZeile struct {
	id, barcode, vorname, nachname, klasse             string
	geburtsdatum, eintritt                             *time.Time
	strasse, hausnummer, plz, ort, elternEmail, lusdID *string
	bestaetigtAm                                       *time.Time
	anonymisiert                                       bool
	gesperrt, manuellGesperrt, abgaenger               bool
	sperrgrund                                         *string
	abgaengerSeit                                      *time.Time
	abgaengerJahr                                      int
}

func ladeZusammenfuehrenZeile(ctx context.Context, tx pgx.Tx, id string) (*zusammenfuehrenZeile, error) {
	z := &zusammenfuehrenZeile{}
	err := tx.QueryRow(ctx, `
		SELECT id, barcode_id, vorname, nachname, klasse, geburtsdatum, schul_eintritt_am,
		       strasse, hausnummer, plz, ort, eltern_email, lusd_id, lusd_bestaetigt_am,
		       anonymized_at IS NOT NULL,
		       ist_gesperrt, COALESCE(is_manually_blocked, false), ist_abgaenger, block_reason,
		       abgaenger_seit, abgaenger_jahr
		FROM schueler WHERE id = $1 AND deleted_at IS NULL FOR UPDATE`, id).Scan(
		&z.id, &z.barcode, &z.vorname, &z.nachname, &z.klasse, &z.geburtsdatum, &z.eintritt,
		&z.strasse, &z.hausnummer, &z.plz, &z.ort, &z.elternEmail, &z.lusdID, &z.bestaetigtAm,
		&z.anonymisiert, &z.gesperrt, &z.manuellGesperrt, &z.abgaenger, &z.sperrgrund,
		&z.abgaengerSeit, &z.abgaengerJahr)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrZusammenfuehrenNichtGefunden
	}
	if err != nil {
		return nil, err
	}
	if z.anonymisiert {
		return nil, ErrZusammenfuehrenAnonymisiert
	}
	return z, nil
}

// fuehrend sagt, wessen Stammdaten gelten. Erst der Status: Ein AKTIVER Datensatz
// schlägt einen Abgänger — der Abgänger ist per Definition der Stand, den die LUSD
// nicht mehr kennt (typisch: Kind umbenannt, an der Theke von Hand neu angelegt, alter
// Datensatz wegen des Ausweises behalten). Sind beide aktiv oder beide Abgänger, gilt
// die jüngere LUSD-Bestätigung; ohne Bestätigung oder bei Gleichstand bleibt das Ziel
// führend — es ist der Datensatz, den der Admin bewusst behalten will.
func fuehrend(ziel, quelle *zusammenfuehrenZeile) (*zusammenfuehrenZeile, *zusammenfuehrenZeile) {
	if ziel.abgaenger != quelle.abgaenger {
		if quelle.abgaenger {
			return ziel, quelle
		}
		return quelle, ziel
	}
	if quelle.bestaetigtAm != nil && (ziel.bestaetigtAm == nil || quelle.bestaetigtAm.After(*ziel.bestaetigtAm)) {
		return quelle, ziel
	}
	return ziel, quelle
}

func erster(a, b *string) *string {
	if a != nil && strings.TrimSpace(*a) != "" {
		return a
	}
	if b != nil && strings.TrimSpace(*b) != "" {
		return b
	}
	return nil
}

func ersteZeit(a, b *time.Time) *time.Time {
	if a != nil {
		return a
	}
	return b
}

// ZusammenfuehrenSchueler führt quelle in ziel auf — in einer Transaktion, mit beiden
// Zeilen gesperrt. Reihenfolge ist Schutz: Erst wandern die Fremdschlüssel, dann fällt
// die Quelle, ERST DANN bekommt das Ziel die Stammdaten — sonst kollidierten Name +
// Geburtsdatum bzw. lusd_id am partiellen Unique-Index, solange die Quelle noch lebt.
func ZusammenfuehrenSchueler(ctx context.Context, pool db.PgxPoolIface, a ZusammenfuehrenAuftrag) (*ZusammenfuehrenErgebnis, error) {
	if a.ZielID == a.QuelleID {
		return nil, ErrZusammenfuehrenGleich
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer db.SafeRollback(ctx, tx)

	// Beide Zeilen in fester Reihenfolge sperren (kleinere UUID zuerst), damit zwei
	// gleichzeitige Zusammenführungen derselben Paare sich nicht gegenseitig verklemmen.
	erste, zweite := a.ZielID, a.QuelleID
	if zweite < erste {
		erste, zweite = zweite, erste
	}
	zeilen := map[string]*zusammenfuehrenZeile{}
	for _, id := range []string{erste, zweite} {
		z, err := ladeZusammenfuehrenZeile(ctx, tx, id)
		if err != nil {
			return nil, err
		}
		zeilen[id] = z
	}
	ziel, quelle := zeilen[a.ZielID], zeilen[a.QuelleID]

	erg := &ZusammenfuehrenErgebnis{ZielID: ziel.id, BarcodeID: ziel.barcode, QuelleBarcode: quelle.barcode}
	gewandert, err := verschiebeVorgaenge(ctx, tx, ziel.id, quelle.id, erg)
	if err != nil {
		return nil, err
	}
	if err := schreibeRueckwegEintrag(ctx, tx, a.BearbeiterID, ziel, quelle, gewandert); err != nil {
		return nil, err
	}
	tag, err := tx.Exec(ctx, `DELETE FROM schueler WHERE id = $1`, quelle.id)
	if err != nil {
		return nil, fmt.Errorf("quelle löschen: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return nil, ErrZusammenfuehrenNichtGefunden
	}

	f, o := fuehrend(ziel, quelle)
	erg.Vorname, erg.Nachname, erg.Klasse = f.vorname, f.nachname, f.klasse
	if err := schreibeZusammengefuehrtesZiel(ctx, tx, ziel.id, f, o, a.AbgaengerJahr(f.klasse), quelle); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return erg, nil
}

// gewanderteVorgaenge sind die Zeilen, die beim Zusammenführen den Schüler gewechselt
// haben — der Rückweg-Eintrag hält sie fest, damit sich ein falsches Paar von Hand
// wieder trennen lässt.
type gewanderteVorgaenge struct {
	Ausleihen, Schadensfaelle, Vormerkungen, VormerkungenDoppelt []string
	Foto                                                         bool
}

// verschiebeVorgaenge hängt alles, was an der Quelle hängt, an das Ziel: Vorgänge (FK),
// das Foto (nur wenn das Ziel keines hat) und die Protokollspuren, über die die
// Art.-15-Auskunft und die DSGVO-Tilgung den Schüler finden (details->>'schueler_id',
// datensatz_id) — dieselben Schlüssel wie in SpurTilgungen. Eine Vormerkung, die das
// Ziel auf denselben Titel schon hat, fällt weg (UNIQUE titel_id, schueler_id).
func verschiebeVorgaenge(ctx context.Context, tx pgx.Tx, ziel, quelle string, erg *ZusammenfuehrenErgebnis) (*gewanderteVorgaenge, error) {
	g := &gewanderteVorgaenge{}
	var err error
	if g.VormerkungenDoppelt, err = idsAus(ctx, tx, `DELETE FROM vormerkungen q WHERE q.schueler_id = $2
		AND EXISTS (SELECT 1 FROM vormerkungen z WHERE z.schueler_id = $1 AND z.titel_id = q.titel_id)
		RETURNING q.id`, ziel, quelle); err != nil {
		return nil, fmt.Errorf("doppelte vormerkungen: %w", err)
	}
	erg.DoppelteVormerkungen = int64(len(g.VormerkungenDoppelt))
	if g.Ausleihen, err = idsAus(ctx, tx, `UPDATE ausleihen SET schueler_id = $1 WHERE schueler_id = $2 RETURNING id`, ziel, quelle); err != nil {
		return nil, fmt.Errorf("ausleihen verschieben: %w", err)
	}
	if g.Schadensfaelle, err = idsAus(ctx, tx, `UPDATE schadensfaelle SET schueler_id = $1 WHERE schueler_id = $2 RETURNING id`, ziel, quelle); err != nil {
		return nil, fmt.Errorf("schadensfälle verschieben: %w", err)
	}
	if g.Vormerkungen, err = idsAus(ctx, tx, `UPDATE vormerkungen SET schueler_id = $1 WHERE schueler_id = $2 RETURNING id`, ziel, quelle); err != nil {
		return nil, fmt.Errorf("vormerkungen verschieben: %w", err)
	}
	erg.Ausleihen, erg.Schaeden, erg.Vormerkungen = int64(len(g.Ausleihen)), int64(len(g.Schadensfaelle)), int64(len(g.Vormerkungen))
	fotos, err := idsAus(ctx, tx, `UPDATE schueler_fotos SET schueler_id = $1 WHERE schueler_id = $2
		AND NOT EXISTS (SELECT 1 FROM schueler_fotos WHERE schueler_id = $1) RETURNING schueler_id`, ziel, quelle)
	if err != nil {
		return nil, fmt.Errorf("foto verschieben: %w", err)
	}
	g.Foto = len(fotos) == 1
	for _, sql := range []string{
		`UPDATE audit_log SET details = jsonb_set(details, '{schueler_id}', to_jsonb($1::text))
			WHERE tabelle = 'ausleihen' AND details->>'schueler_id' = $2`,
		`UPDATE audit_log SET datensatz_id = $1::uuid WHERE tabelle = 'schueler' AND datensatz_id = $2::uuid`,
		`UPDATE audit_logs SET details = jsonb_set(details, '{schueler_id}', to_jsonb($1::text))
			WHERE details->>'schueler_id' = $2`,
	} {
		if _, err := tx.Exec(ctx, sql, ziel, quelle); err != nil {
			return nil, fmt.Errorf("protokollspuren umschlüsseln: %w", err)
		}
	}
	return g, nil
}

// idsAus führt ein Statement mit RETURNING aus und sammelt die Kennungen.
func idsAus(ctx context.Context, tx pgx.Tx, sql string, args ...any) ([]string, error) {
	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// schreibeRueckwegEintrag hält in audit_log (tabelle='schueler', datensatz_id=Ziel)
// fest, was das Zusammenführen unumkehrbar macht: die vollständigen Stammdaten der
// Quelle, den Stand des Ziels davor und die Kennungen aller gewanderten Zeilen. Ohne
// diesen Eintrag wäre ein falsch bestätigtes Paar nicht mehr zu trennen (Raster-
// frage 10, Rückweg). Der Eintrag entsteht in derselben Transaktion — kein Fenster,
// in dem die Quelle weg ist und die Spur fehlt. Lebenszyklus: Wird das Ziel später
// anonymisiert, ersetzt SpurTilgungen dieses Objekt als Ganzes — die Klardaten der
// Quelle leben nicht länger als die des Ziels.
func schreibeRueckwegEintrag(ctx context.Context, tx pgx.Tx, bearbeiterID string, ziel, quelle *zusammenfuehrenZeile, g *gewanderteVorgaenge) error {
	details, err := json.Marshal(map[string]any{
		"action":             "zusammenfuehren",
		"aufgeloest_id":      quelle.id,
		"aufgeloest_barcode": quelle.barcode,
		"quelle":             stammdatenSnapshot(quelle),
		"ziel_vorher":        stammdatenSnapshot(ziel),
		"gewandert": map[string]any{
			"ausleihen": g.Ausleihen, "schadensfaelle": g.Schadensfaelle,
			"vormerkungen": g.Vormerkungen, "vormerkungen_doppelt_geloescht": g.VormerkungenDoppelt,
			"foto": g.Foto,
		},
	})
	if err != nil {
		return fmt.Errorf("rückweg-eintrag: %w", err)
	}
	var bearbeiter *string
	akteur := "SYSTEM"
	if bearbeiterID != "" {
		bearbeiter, akteur = &bearbeiterID, "USER"
	}
	tag, err := tx.Exec(ctx, `INSERT INTO audit_log (tabelle, aktion, datensatz_id, bearbeiter_id, akteur, details)
		VALUES ('schueler', 'ZUSAMMENGEFUEHRT', $1::uuid, $2, $3, $4::jsonb)`, ziel.id, bearbeiter, akteur, string(details))
	if err != nil {
		return fmt.Errorf("rückweg-eintrag schreiben: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return errors.New("rückweg-eintrag nicht geschrieben")
	}
	return nil
}

func stammdatenSnapshot(z *zusammenfuehrenZeile) map[string]any {
	datum := func(t *time.Time) any {
		if t == nil {
			return nil
		}
		return t.Format("2006-01-02")
	}
	zeit := func(t *time.Time) any {
		if t == nil {
			return nil
		}
		return t.Format(time.RFC3339)
	}
	return map[string]any{
		"id": z.id, "barcode_id": z.barcode, "vorname": z.vorname, "nachname": z.nachname, "klasse": z.klasse,
		"geburtsdatum": datum(z.geburtsdatum), "schul_eintritt_am": datum(z.eintritt),
		"strasse": z.strasse, "hausnummer": z.hausnummer, "plz": z.plz, "ort": z.ort, "eltern_email": z.elternEmail,
		"lusd_id": z.lusdID, "lusd_bestaetigt_am": zeit(z.bestaetigtAm),
		"ist_gesperrt": z.gesperrt, "is_manually_blocked": z.manuellGesperrt, "block_reason": z.sperrgrund,
		"ist_abgaenger": z.abgaenger, "abgaenger_seit": zeit(z.abgaengerSeit), "abgaenger_jahr": z.abgaengerJahr,
	}
}

// schreibeZusammengefuehrtesZiel setzt die Stammdaten des führenden Datensatzes (f) auf
// das Ziel, füllt Lücken aus dem anderen (o) und macht das Ziel wieder aktiv. Die CASE-
// Ausdrücke für die Sperre lesen die ALTEN Werte (Postgres wertet die rechte Seite vor
// der Zuweisung aus) — dieselbe Bauart wie der Rückkehrer-Pfad in api/lusd_apply.go.
//
// Eine MANUELLE Sperre der Quelle (Ausweis gestohlen, Hausverbot …) gehört zur Person,
// nicht zum Datensatz: Sie geht auf das Ziel über — mit ihrem Grund, sofern das Ziel
// nicht selbst schon manuell gesperrt ist (dann bleibt dessen Grund).
func schreibeZusammengefuehrtesZiel(ctx context.Context, tx pgx.Tx, zielID string, f, o *zusammenfuehrenZeile, abgaengerJahr int, quelle *zusammenfuehrenZeile) error {
	tag, err := tx.Exec(ctx, `
		UPDATE schueler SET
			vorname = $2, nachname = $3, klasse = $4,
			geburtsdatum = $5, schul_eintritt_am = $6,
			strasse = $7, hausnummer = $8, plz = $9, ort = $10, eltern_email = $11,
			lusd_id = $12, lusd_bestaetigt_am = $13,
			abgaenger_jahr = $14,
			ist_abgaenger = false, abgaenger_seit = NULL,
			is_manually_blocked = COALESCE(is_manually_blocked, false) OR $15,
			ist_gesperrt = CASE
				WHEN $15 THEN true
				WHEN `+SQLAbgaengerSperreAutomatisch+`
				     AND NOT EXISTS (SELECT 1 FROM ausleihen WHERE schueler_id = $1 AND rueckgabe_am IS NULL)
				     AND NOT EXISTS (SELECT 1 FROM schadensfaelle WHERE schueler_id = $1 AND ist_bezahlt = false)
				THEN false ELSE ist_gesperrt END,
			block_reason = CASE
				WHEN $15 AND NOT COALESCE(is_manually_blocked, false) THEN $16
				WHEN $15 THEN block_reason
				WHEN `+SQLAbgaengerSperreAutomatisch+`
				     AND NOT EXISTS (SELECT 1 FROM ausleihen WHERE schueler_id = $1 AND rueckgabe_am IS NULL)
				     AND NOT EXISTS (SELECT 1 FROM schadensfaelle WHERE schueler_id = $1 AND ist_bezahlt = false)
				THEN NULL
				WHEN `+SQLAbgaengerSperreAutomatisch+` THEN 'Sperre wegen offener Vorgänge'
				ELSE block_reason END,
			aktualisiert_am = NOW()
		WHERE id = $1`,
		zielID, f.vorname, f.nachname, f.klasse,
		ersteZeit(f.geburtsdatum, o.geburtsdatum), ersteZeit(f.eintritt, o.eintritt),
		erster(f.strasse, o.strasse), erster(f.hausnummer, o.hausnummer), erster(f.plz, o.plz),
		erster(f.ort, o.ort), erster(f.elternEmail, o.elternEmail),
		erster(f.lusdID, o.lusdID), ersteZeit(f.bestaetigtAm, o.bestaetigtAm), abgaengerJahr,
		quelle.manuellGesperrt, quelle.sperrgrund)
	if err != nil {
		return fmt.Errorf("ziel schreiben: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrZusammenfuehrenNichtGefunden
	}
	return nil
}

// ZusammenfuehrenKandidat ist ein Treffer der Kandidatensuche: bewusst auch Abgänger und
// Gesperrte, denn genau die sucht man beim Zusammenführen — die Aktivliste der
// Schülerdatei blendet sie aus.
type ZusammenfuehrenKandidat struct {
	ID            string  `json:"id"`
	BarcodeID     string  `json:"barcode_id"`
	Vorname       string  `json:"vorname"`
	Nachname      string  `json:"nachname"`
	Klasse        string  `json:"klasse"`
	Geburtsdatum  *string `json:"geburtsdatum,omitempty"`
	IstAbgaenger  bool    `json:"ist_abgaenger"`
	IstGesperrt   bool    `json:"ist_gesperrt"`
	OffeneBuecher int64   `json:"offene_buecher"`
}

// SucheZusammenfuehrenKandidaten sucht mit denselben Bausteinen wie Theke und
// Schülerdatei (suchnorm, Tokens) über ALLE nicht gelöschten, nicht anonymisierten
// Schüler — außer dem Datensatz, von dem aus gesucht wird.
func SucheZusammenfuehrenKandidaten(ctx context.Context, pool db.PgxPoolIface, ausserID, suche string, limit int) ([]ZusammenfuehrenKandidat, error) {
	tokens := suchTokens(suche)
	if len(tokens) == 0 {
		return []ZusammenfuehrenKandidat{}, nil
	}
	rows, err := pool.Query(ctx, SchuelerSuchCTE+`
		SELECT s.id, s.barcode_id, s.vorname, s.nachname, s.klasse, TO_CHAR(s.geburtsdatum, 'YYYY-MM-DD'),
		       s.ist_abgaenger, s.ist_gesperrt,
		       (SELECT count(*) FROM ausleihen a WHERE a.schueler_id = s.id AND a.rueckgabe_am IS NULL)
		FROM schueler s
		WHERE s.deleted_at IS NULL AND s.anonymized_at IS NULL AND s.id <> $4
		  AND `+SchuelerSuchBedingung(true)+`
		ORDER BY `+SchuelerSuchRang+`, s.nachname, s.vorname
		LIMIT $3`, tokens, tokens[0], limit, ausserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ZusammenfuehrenKandidat{}
	for rows.Next() {
		var k ZusammenfuehrenKandidat
		if err := rows.Scan(&k.ID, &k.BarcodeID, &k.Vorname, &k.Nachname, &k.Klasse, &k.Geburtsdatum,
			&k.IstAbgaenger, &k.IstGesperrt, &k.OffeneBuecher); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}
