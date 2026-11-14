package repository

// anmeldung.go — was die Anmeldung über ein Konto liest und schreibt. Was ein fehlendes Konto
// bedeutet (Selbstanmeldung, keine Sitzung), entscheidet auth/; die Lesefunktionen reichen
// pgx.ErrNoRows deshalb durch.

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// Die Werte der Spalte benutzer.anmeldeverfahren: welcher Server das Passwort eines Kontos prüft.
const (
	// AnmeldeverfahrenIMAP prüft am Mailserver der Schule; die Vorgabe jedes Kontos.
	AnmeldeverfahrenIMAP = "imap"
	// AnmeldeverfahrenLDAP prüft am Verzeichnisdienst, über die LDAP-Kennung des Kontos.
	AnmeldeverfahrenLDAP = "ldap"
)

// AnmeldeKonto ist das Konto, wie die Anmeldung es zu einer Adresse findet.
type AnmeldeKonto struct {
	ID       string
	Rolle    string
	Vorname  string
	Nachname string
	Aktiv    bool
	Email    string
	// ZugangBeantragt: Das Konto stammt aus der Selbstanmeldung und wartet auf die
	// Freischaltung, oder es wurde nach ihr freigeschaltet.
	ZugangBeantragt bool
	// Anmeldeverfahren nennt den Server, der das Passwort prüft (AnmeldeverfahrenIMAP oder
	// AnmeldeverfahrenLDAP).
	Anmeldeverfahren string
	// LDAPKennung ist der Name des Kontos im Verzeichnisdienst; leer bei IMAP.
	LDAPKennung string
}

// ErrAnmeldungMehrdeutig heißt: Die Eingabe der Anmeldung trifft zwei Konten, eines über die
// E-Mail und eines über die LDAP-Kennung. Angemeldet wird dann keines; welches gemeint war,
// lässt sich nicht entscheiden.
var ErrAnmeldungMehrdeutig = errors.New("eingabe der anmeldung trifft mehr als ein konto")

// LadeAnmeldeKonto liest das Konto zu einer Eingabe der Anmeldung, ohne Rücksicht auf Groß- und
// Kleinschreibung: über die E-Mail, wie eingegeben oder um die Domain der Schule ergänzt
// (email), oder bei einem LDAP-Konto über seine Kennung. Ohne Treffer pgx.ErrNoRows, bei zwei
// Konten ErrAnmeldungMehrdeutig.
func LadeAnmeldeKonto(ctx context.Context, db Datenbank, eingabe, email string) (AnmeldeKonto, error) {
	rows, err := db.Query(ctx, `
		SELECT b.id, b.rolle, b.vorname, b.nachname, b.aktiv, b.email,
		       b.zugang_beantragt_am IS NOT NULL, b.anmeldeverfahren, coalesce(b.ldap_kennung, '')
		FROM benutzer b
		WHERE LOWER(b.email) IN (LOWER($1), LOWER($2))
		   OR (b.anmeldeverfahren = 'ldap' AND LOWER(b.ldap_kennung) = LOWER($1))
		LIMIT 2
	`, eingabe, email)
	if err != nil {
		return AnmeldeKonto{}, err
	}
	defer rows.Close()

	var konten []AnmeldeKonto
	for rows.Next() {
		var k AnmeldeKonto
		if err := rows.Scan(&k.ID, &k.Rolle, &k.Vorname, &k.Nachname, &k.Aktiv, &k.Email,
			&k.ZugangBeantragt, &k.Anmeldeverfahren, &k.LDAPKennung); err != nil {
			return AnmeldeKonto{}, err
		}
		konten = append(konten, k)
	}
	if err := rows.Err(); err != nil {
		return AnmeldeKonto{}, err
	}
	switch len(konten) {
	case 0:
		return AnmeldeKonto{}, pgx.ErrNoRows
	case 1:
		return konten[0], nil
	default:
		return AnmeldeKonto{}, ErrAnmeldungMehrdeutig
	}
}

// LadeAnmeldeverfahren liest, welcher Server das Passwort eines Kontos prüft, und für LDAP seine
// Kennung: was das Aufschließen der Sperre braucht.
func LadeAnmeldeverfahren(ctx context.Context, db Datenbank, kontoID string) (verfahren, kennung string, err error) {
	err = db.QueryRow(ctx, `
		SELECT anmeldeverfahren, coalesce(ldap_kennung, '') FROM benutzer WHERE id = $1
	`, kontoID).Scan(&verfahren, &kennung)
	return verfahren, kennung, err
}

// LegeZugangsanfrageAn legt ein inaktives Konto der Rolle Kollegium an und sagt, ob dabei eine
// Zeile entstand. Gibt es die Adresse schon, entsteht keine, und das ist kein Fehler: Zwei
// gleichzeitige Versuche derselben Person scheitern so nicht an der eindeutigen Adresse.
func LegeZugangsanfrageAn(ctx context.Context, db Datenbank, vorname, nachname, email string) (bool, error) {
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

// LadeSitzungsKonto liest Rolle, Namen, Adresse und Aktiv-Status eines Kontos. Eine Sitzung
// fragt die Datenbank und nicht ihr Token: Rolle und Status können sich seit der Anmeldung
// geändert haben.
func LadeSitzungsKonto(ctx context.Context, db Datenbank, kontoID string) (SitzungsKonto, error) {
	var k SitzungsKonto
	err := db.QueryRow(ctx, `
			SELECT rolle, vorname, nachname, aktiv, email
			FROM benutzer
			WHERE id = $1
			LIMIT 1
		`, kontoID).Scan(&k.Rolle, &k.Vorname, &k.Nachname, &k.Aktiv, &k.Email)
	return k, err
}

// LadeKontoAktivUndRolle liest, ob das Konto aktiv ist, und seine gespeicherte Rolle: die Prüfung
// bei jeder Anfrage.
func LadeKontoAktivUndRolle(ctx context.Context, db Datenbank, kontoID string) (aktiv bool, rolle string, err error) {
	err = db.QueryRow(ctx, `SELECT aktiv, rolle FROM benutzer WHERE id = $1`, kontoID).Scan(&aktiv, &rolle)
	return aktiv, rolle, err
}
