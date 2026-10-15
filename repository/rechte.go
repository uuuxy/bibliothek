package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// HatRolleRecht sagt, ob role_permissions der Rolle das Recht erteilt. Ohne Zeile gilt es als
// nicht erteilt. Ein Fehler der Datenbank kommt als Fehler zurück: Der Aufrufer muss ihn von
// einem fehlenden Recht unterscheiden können, sonst würde aus einem Aussetzer ein Verbot.
func HatRolleRecht(ctx context.Context, db Datenbank, rolle, recht string) (bool, error) {
	var allowed bool
	query := `
		SELECT allowed
		FROM role_permissions
		WHERE UPPER(role) = UPPER($1) AND permission = $2
	`
	err := db.QueryRow(ctx, query, rolle, recht).Scan(&allowed)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return allowed, nil
}

// LadeErlaubteRechteJeRolle liefert je Rolle, klein geschrieben, die Rechte, die role_permissions
// erteilt: dieselbe Tabelle, aus der Anmeldung und Rechteprüfung lesen.
func LadeErlaubteRechteJeRolle(ctx context.Context, db Datenbank) (map[string][]string, error) {
	rows, err := db.Query(ctx, `
		SELECT lower(role), permission
		FROM role_permissions
		WHERE allowed = true
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rechte := map[string][]string{}
	for rows.Next() {
		var rolle, recht string
		if err := rows.Scan(&rolle, &recht); err != nil {
			return nil, err
		}
		rechte[rolle] = append(rechte[rolle], recht)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return rechte, nil
}

// LadeErlaubteRechteDerRolle liefert die Rechte, die role_permissions einer Rolle erteilt; die
// Schreibweise der Rolle spielt keine Rolle. Ohne erteiltes Recht ist die Liste leer, nicht nil:
// Die Anmeldung schickt sie so an den Browser.
func LadeErlaubteRechteDerRolle(ctx context.Context, db Datenbank, rolle string) ([]string, error) {
	rechte := []string{}
	rows, err := db.Query(ctx, `
		SELECT permission
		FROM role_permissions
		WHERE UPPER(role) = UPPER($1) AND allowed = true
	`, rolle)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var recht string
		if err := rows.Scan(&recht); err != nil {
			return nil, err
		}
		rechte = append(rechte, recht)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return rechte, nil
}

// RollenRecht ist eine Zeile der Rechte-Matrix.
type RollenRecht struct {
	Rolle, Recht string
	Erlaubt      bool
}

// LadeRollenRechte liefert die ganze Rechte-Matrix, nach Rolle und Recht geordnet.
func LadeRollenRechte(ctx context.Context, db Datenbank) ([]RollenRecht, error) {
	query := `
			SELECT role::text, permission, allowed 
			FROM role_permissions 
			ORDER BY role, permission
		`
	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rechte := []RollenRecht{}
	for rows.Next() {
		var re RollenRecht
		if err := rows.Scan(&re.Rolle, &re.Recht, &re.Erlaubt); err != nil {
			return nil, zeileUnlesbar(err)
		}
		rechte = append(rechte, re)
	}
	return rechte, rows.Err()
}

// SetzeRollenRecht erteilt oder entzieht der Rolle das Recht und nennt die Zahl der getroffenen
// Zeilen. Null heißt: Diese Paarung aus Rolle und Recht gibt es nicht; der Aufrufer darf das
// nicht als Erfolg melden.
func SetzeRollenRecht(ctx context.Context, db Datenbank, rolle, recht string, erlaubt bool) (int64, error) {
	query := `
			UPDATE role_permissions
			SET allowed = $1
			WHERE UPPER(role) = UPPER($2) AND permission = $3
		`
	tag, err := db.Exec(ctx, query, erlaubt, rolle, recht)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
