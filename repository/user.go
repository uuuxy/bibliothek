package repository

import (
	"context"
	"errors"
	"fmt"

	"bibliothek/db"

	"github.com/jackc/pgx/v5"
)

// UserRepository definiert die Datenbankoperationen zur Verwaltung von Systembenutzern (Lehrer, Admins, Helfer).
type UserRepository interface {
	// GetUsers ruft alle registrierten Systembenutzer sortiert nach Nachname und Vorname ab.
	GetUsers(ctx context.Context) ([]User, error)

	// CheckEmailExists prüft, ob eine E-Mail-Adresse bereits einem Benutzer zugeordnet ist.
	// Mit excludeID kann die ID des aktuell bearbeiteten Benutzers von der Prüfung ausgeschlossen werden.
	CheckEmailExists(ctx context.Context, email string, excludeID string) (bool, error)

	// CheckBarcodeExists prüft, ob eine Ausweisnummer bereits vergeben ist. Seit
	// Migration 125 gibt es dafür nur noch einen Ort — die Lesertabelle.
	// Mit excludeID kann die Leserzeile des aktuell bearbeiteten Kontos ausgeschlossen werden.
	CheckBarcodeExists(ctx context.Context, barcode string, excludeID string) (bool, error)

	// LeserzeileOhneKonto prüft, ob ein Kollege dieses Namens schon eine Leserzeile
	// ohne Konto hat (OFFEN.md 5.17). Ein neues Konto bekäme vom Trigger eine zweite,
	// leere Zeile; der Weg zur vorhandenen ist die Schul-E-Mail in der Akte.
	LeserzeileOhneKonto(ctx context.Context, vorname, nachname string) (bool, error)

	// CreateUser legt ein neues Konto an und gibt dessen generierte ID (UUID) zurück.
	// Die Leserzeile entsteht dabei von selbst (Trigger trg_benutzer_hat_leserzeile);
	// barcode nil oder "" lässt sie ohne Ausweisnummer.
	CreateUser(ctx context.Context, barcode *string, vorname, nachname, email, rolle string) (string, error)

	// UpdateUser aktualisiert die Daten eines bestehenden Systembenutzers.
	UpdateUser(ctx context.Context, p UpdateUserParams) error

	// GetRolleByID liefert die AKTUELL gespeicherte Rolle eines Benutzers in
	// Großschreibung. Kein Treffer ergibt ("", nil).
	//
	// Die Rolle aus dem Request-Rumpf taugt dafür nicht: Wer einen Admin-Datensatz
	// bearbeitet, schickt selbst mit, welche Rolle daraus werden soll. Ob das Ziel
	// HEUTE ein Admin ist — und die Bearbeitung damit Admin-Rechte des Aufrufers
	// verlangt — steht nur in der Datenbank.
	GetRolleByID(ctx context.Context, id string) (string, error)
}

// postgresUserRepo implementiert das UserRepository für PostgreSQL.
type postgresUserRepo struct {
	pool db.PgxPoolIface
}

// NewUserRepository erzeugt eine neue Instanz des PostgreSQL-basierten UserRepositorys.
func NewUserRepository(pool db.PgxPoolIface) UserRepository {
	return &postgresUserRepo{pool: pool}
}

// Wer ausleihen darf, entscheidet seit Migration 125 nicht mehr das Konto, sondern die
// Leserzeile: ein aktiver Leser. Die frühere Regel SQLAktiveLehrkraft („Personenart
// gesetzt und aktiv") und die Suche GetLehrerByBarcode sind damit ersatzlos gefallen —
// die Theke sucht einen LESER (StudentRepository.GetLeserByBarcode), nicht ein Konto.

// GetUsers fragt alle registrierten Benutzer ab.
func (r *postgresUserRepo) GetUsers(ctx context.Context) ([]User, error) {
	query := `
		SELECT b.id, coalesce(l.barcode_id, ''), b.vorname, b.nachname, b.email, b.rolle, b.aktiv,
		       b.erstellt_am, b.zugang_beantragt_am, coalesce(b.leser_id::text, '')
		FROM benutzer b
		LEFT JOIN leser l ON l.id = b.leser_id
		ORDER BY b.nachname, b.vorname
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []User{}
	for rows.Next() {
		var u User
		err := rows.Scan(&u.ID, &u.BarcodeID, &u.Vorname, &u.Nachname, &u.Email, &u.Rolle, &u.Aktiv, &u.ErstelltAm,
			&u.ZugangBeantragtAm, &u.LeserID)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

// CheckEmailExists prüft das Vorhandensein einer E-Mail-Adresse im System — in der
// Normalform der Anmeldung (LOWER, auth/handlers.go). Bis zum 10.09.2026 exakt: Zwei
// Konten, die sich nur in der Schreibweise unterschieden, gingen durch, und welches der
// Login öffnet, entschied die Speicherreihenfolge (Bestands-Durchgang). Migration 113
// hält dasselbe als Index in der Datenbank.
func (r *postgresUserRepo) CheckEmailExists(ctx context.Context, email string, excludeID string) (bool, error) {
	var exists bool
	var err error
	if excludeID == "" {
		err = r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM benutzer WHERE LOWER(email) = LOWER($1))", email).Scan(&exists)
	} else {
		err = r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM benutzer WHERE LOWER(email) = LOWER($1) AND id != $2)", email, excludeID).Scan(&exists)
	}
	return exists, err
}

// CheckBarcodeExists prüft, ob eine Ausweisnummer schon vergeben ist.
//
// Seit Migration 125 steht jede Ausweisnummer in derselben Tabelle, und der partielle
// Index uniq_schueler_barcode_active hält sie dort eindeutig. Die frühere Kreuzprüfung
// über zwei Tabellen — und der Trigger, der sie erzwang — sind mit der Zweiteilung
// gefallen. Gelöschte Leser geben ihre Nummer frei.
//
// excludeID ist wie bisher die KONTO-ID des gerade bearbeiteten Benutzers; ausgenommen
// wird seine Leserzeile. Der Aufrufer muss die Leser-ID dafür nicht kennen.
func (r *postgresUserRepo) CheckBarcodeExists(ctx context.Context, barcode string, excludeID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
		    SELECT 1 FROM leser
		     WHERE barcode_id = $1
		       AND deleted_at IS NULL
		       AND id IS DISTINCT FROM (SELECT leser_id FROM benutzer WHERE id = NULLIF($2, '')::uuid))`,
		barcode, excludeID).Scan(&exists)
	return exists, err
}

// LeserzeileOhneKonto sagt, ob für den Namen schon ein Kollege OHNE Konto in der
// Leserdatei steht — so bleibt seine Zeile nach dem Löschen des Kontos zurück (die
// Ausleihen hängen daran), und so kommt sie aus der Littera-Übernahme.
//
// Nur das Kollegium zählt (ein Schüler gleichen Namens ist eine andere Person), und nur
// Zeilen ohne Konto: Hat der Namensvetter schon eines, ist der Neue ein zweiter Mensch.
// Verglichen wird in suchnorm wie bei der Namensdublette beim Anlegen eines Lesers.
func (r *postgresUserRepo) LeserzeileOhneKonto(ctx context.Context, vorname, nachname string) (bool, error) {
	var vorhanden bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM leser l
			WHERE l.art <> 'schueler' AND l.deleted_at IS NULL
			  AND suchnorm(l.vorname) = suchnorm($1) AND suchnorm(l.nachname) = suchnorm($2)
			  AND NOT EXISTS (SELECT 1 FROM benutzer b WHERE b.leser_id = l.id))`,
		vorname, nachname).Scan(&vorhanden)
	return vorhanden, err
}

// CreateUser legt ein Konto an — und mit ihm die Leserzeile, in der der Ausweis landet.
//
// Beides in EINER Transaktion: Ein Konto ohne Leserzeile fände die Theke nicht, eine
// Leserzeile ohne Konto stünde namenlos in der Leserdatei. Die Zeile selbst legt der
// Trigger an (trg_benutzer_hat_leserzeile); hier wird nur noch der Ausweis nachgetragen,
// denn den kennt der Trigger nicht.
func (r *postgresUserRepo) CreateUser(ctx context.Context, barcode *string, vorname, nachname, email, rolle string) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer db.SafeRollback(ctx, tx)

	var userID, leserID string
	err = tx.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ($1, $2, $3, $4::benutzer_rolle, true)
		RETURNING id, leser_id::text
	`, vorname, nachname, email, rolle).Scan(&userID, &leserID)
	if err != nil {
		return "", err
	}

	if barcode != nil && *barcode != "" {
		tag, err := tx.Exec(ctx, `UPDATE leser SET barcode_id = $1 WHERE id = $2`, *barcode, leserID)
		if err != nil {
			return "", err
		}
		// 0 Zeilen hieße: Das Konto hat keine Leserzeile bekommen. Der Trigger garantiert
		// sie, aber ein stiller Erfolg wäre hier ein Konto, dessen Ausweisnummer nirgends
		// steht — und das an der Theke nicht gefunden wird.
		if tag.RowsAffected() == 0 {
			return "", fmt.Errorf("die Leserzeile des neuen Kontos fehlt (id %s)", leserID)
		}
	}
	return userID, tx.Commit(ctx)
}

// UpdateUserParams bündelt die aktualisierbaren Felder eines Benutzers.
type UpdateUserParams struct {
	ID       string
	Barcode  *string
	Vorname  string
	Nachname string
	Email    string
	Rolle    string
	Aktiv    bool
}

// ErrBenutzerNichtGefunden meldet eine unbekannte Benutzer-ID beim Ändern/Löschen — 0 Zeilen sind
// hier ein Fehler, kein Erfolg (Phantom-Erfolg-Sweep 31.08.2026: der Handler schrieb
// sonst einen USER_UPDATE-Audit-Eintrag über eine Änderung, die nie stattfand).
var ErrBenutzerNichtGefunden = errors.New("benutzer nicht gefunden")

// UpdateUser ändert ein Konto — und führt die Leserzeile mit.
//
// Name und Ausweis stehen für die Theke in der Leserzeile, für die Anmeldung am Konto.
// Liefen sie auseinander, hieße dieselbe Person an der Theke anders als in der
// Benutzerverwaltung, und der Ausweis, den jemand hier einträgt, bliebe wirkungslos.
// Deshalb ein Schreibvorgang über beide Tabellen, in EINER Transaktion.
func (r *postgresUserRepo) UpdateUser(ctx context.Context, p UpdateUserParams) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.SafeRollback(ctx, tx)

	var leserID *string
	err = tx.QueryRow(ctx, `
		UPDATE benutzer
		SET vorname = $1, nachname = $2, email = $3, rolle = $4::benutzer_rolle, aktiv = $5,
		    -- Freischalten erledigt den Antrag; ein späteres Deaktivieren soll nicht
		    -- wieder wie ein Antrag aussehen (Migration 086).
		    zugang_beantragt_am = CASE WHEN $5 THEN NULL ELSE zugang_beantragt_am END,
		    aktualisiert_am = CURRENT_TIMESTAMP
		WHERE id = $6
		RETURNING leser_id::text
	`, p.Vorname, p.Nachname, p.Email, p.Rolle, p.Aktiv, p.ID).Scan(&leserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrBenutzerNichtGefunden
	}
	if err != nil {
		return err
	}

	if leserID != nil {
		tag, err := tx.Exec(ctx, `
			UPDATE leser SET vorname = $1, nachname = $2, barcode_id = NULLIF($3::text, '')
			WHERE id = $4
		`, p.Vorname, p.Nachname, barcodeText(p.Barcode), *leserID)
		if err != nil {
			return err
		}
		// Ein stiller Erfolg hieße: Der Name wurde am Konto geändert, an der Leserzeile
		// nicht — dieselbe Person hieße an der Theke anders als in der Verwaltung.
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("die Leserzeile %s zu Konto %s fehlt", *leserID, p.ID)
		}
	}
	return tx.Commit(ctx)
}

// barcodeText macht aus dem optionalen Ausweis einen Text: nil und "" heißen beide
// „kein Ausweis" und werden in der Leserzeile zu NULL. Ein leerer String wäre dort eine
// vergebene Nummer und würde die Eindeutigkeit gegen den nächsten Leser ohne Ausweis
// verletzen.
func barcodeText(b *string) string {
	if b == nil {
		return ""
	}
	return *b
}

// GetRolleByID liest die gespeicherte Rolle eines Benutzers.
//
// rolle::text, denn die Spalte ist das ENUM benutzer_rolle.
func (r *postgresUserRepo) GetRolleByID(ctx context.Context, id string) (string, error) {
	var rolle string
	err := r.pool.QueryRow(ctx, `SELECT UPPER(rolle::text) FROM benutzer WHERE id = $1`, id).Scan(&rolle)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("konnte die Rolle des Benutzers nicht lesen: %w", err)
	}
	return rolle, nil
}
