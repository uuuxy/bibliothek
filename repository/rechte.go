package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// RechtDerRolle sagt, ob role_permissions der Rolle das Recht erteilt. Ohne Zeile gilt es als
// nicht erteilt. Ein Fehler der Datenbank kommt als Fehler zurück: Der Aufrufer muss ihn von
// einem fehlenden Recht unterscheiden können, sonst würde aus einem Aussetzer ein Verbot.
func RechtDerRolle(ctx context.Context, db DBQueryer, rolle, recht string) (bool, error) {
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

// ErlaubteRechteJeRolle liefert je Rolle, klein geschrieben, die Rechte, die role_permissions
// erteilt: dieselbe Tabelle, aus der Anmeldung und Rechteprüfung lesen.
func ErlaubteRechteJeRolle(ctx context.Context, db DBQueryer) (map[string][]string, error) {
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

// RollenRecht ist eine Zeile der Rechte-Matrix.
type RollenRecht struct {
	Rolle, Recht string
	Erlaubt      bool
}

// ListeRollenRechte liefert die ganze Rechte-Matrix, nach Rolle und Recht geordnet.
func ListeRollenRechte(ctx context.Context, db DBQueryer) ([]RollenRecht, error) {
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
		if err := rows.Scan(&re.Rolle, &re.Recht, &re.Erlaubt); err == nil {
			rechte = append(rechte, re)
		}
	}
	return rechte, rows.Err()
}

// SetzeRollenRecht erteilt oder entzieht der Rolle das Recht und nennt die Zahl der getroffenen
// Zeilen. Null heißt: Diese Paarung aus Rolle und Recht gibt es nicht; der Aufrufer darf das
// nicht als Erfolg melden.
func SetzeRollenRecht(ctx context.Context, db DBQueryer, rolle, recht string, erlaubt bool) (int64, error) {
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
