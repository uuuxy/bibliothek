package repository

// anmeldung.go — was die Anmeldung über ein Konto liest und schreibt. Was ein fehlendes Konto
// bedeutet (Selbstanmeldung, keine Sitzung), entscheidet auth/; die Lesefunktionen reichen
// pgx.ErrNoRows deshalb durch.

import "context"

// AnmeldeKonto ist das Konto, wie die Anmeldung es zu einer Adresse findet.
type AnmeldeKonto struct {
	ID            string
	Ausweisnummer string
	Rolle         string
	Vorname       string
	Nachname      string
	Aktiv         bool
	Email         string
	// ZugangBeantragt: Das Konto stammt aus der Selbstanmeldung und wartet auf die
	// Freischaltung, oder es wurde nach ihr freigeschaltet.
	ZugangBeantragt bool
}

// LiesAnmeldeKonto liest das Konto zu einer Adresse, ohne Rücksicht auf Groß- und
// Kleinschreibung. Die Ausweisnummer steht an der Leserzeile des Kontos; ein Konto ohne
// Leserzeile wird trotzdem gefunden und trägt dann keine.
func LiesAnmeldeKonto(ctx context.Context, db DBQueryer, email string) (AnmeldeKonto, error) {
	var k AnmeldeKonto
	query := `
		SELECT b.id, coalesce(l.barcode_id, ''), b.rolle, b.vorname, b.nachname, b.aktiv, b.email,
		       b.zugang_beantragt_am IS NOT NULL
		FROM benutzer b
		LEFT JOIN leser l ON l.id = b.leser_id
		WHERE LOWER(b.email) = LOWER($1)
		LIMIT 1
	`
	err := db.QueryRow(ctx, query, email).Scan(&k.ID, &k.Ausweisnummer, &k.Rolle, &k.Vorname, &k.Nachname, &k.Aktiv, &k.Email, &k.ZugangBeantragt)
	return k, err
}

// LegeZugangsanfrageAn legt ein inaktives Konto der Rolle Kollegium an und sagt, ob dabei eine
// Zeile entstand. Gibt es die Adresse schon, entsteht keine, und das ist kein Fehler: Zwei
// gleichzeitige Versuche derselben Person scheitern so nicht an der eindeutigen Adresse.
func LegeZugangsanfrageAn(ctx context.Context, db DBQueryer, vorname, nachname, email string) (bool, error) {
	tag, err := db.Exec(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv, zugang_beantragt_am)
		VALUES ($1, $2, LOWER($3), 'kollegium', false, CURRENT_TIMESTAMP)
		ON CONFLICT DO NOTHING
	`, vorname, nachname, email)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// SitzungsKonto ist der Stand eines Kontos, wie ihn eine laufende Sitzung abfragt.
type SitzungsKonto struct {
	Rolle    string
	Vorname  string
	Nachname string
	Aktiv    bool
	Email    string
}

// LiesSitzungsKonto liest Rolle, Namen, Adresse und Aktiv-Status eines Kontos. Eine Sitzung
// fragt die Datenbank und nicht ihr Token: Rolle und Status können sich seit der Anmeldung
// geändert haben.
func LiesSitzungsKonto(ctx context.Context, db DBQueryer, kontoID string) (SitzungsKonto, error) {
	var k SitzungsKonto
	err := db.QueryRow(ctx, `
			SELECT rolle, vorname, nachname, aktiv, email
			FROM benutzer
			WHERE id = $1
			LIMIT 1
		`, kontoID).Scan(&k.Rolle, &k.Vorname, &k.Nachname, &k.Aktiv, &k.Email)
	return k, err
}

// KontoAktivUndRolle liest, ob das Konto aktiv ist, und seine gespeicherte Rolle: die Prüfung
// bei jeder Anfrage.
func KontoAktivUndRolle(ctx context.Context, db DBQueryer, kontoID string) (aktiv bool, rolle string, err error) {
	err = db.QueryRow(ctx, `SELECT aktiv, rolle FROM benutzer WHERE id = $1`, kontoID).Scan(&aktiv, &rolle)
	return aktiv, rolle, err
}
