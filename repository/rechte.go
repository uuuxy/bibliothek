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
