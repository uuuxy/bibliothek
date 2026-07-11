package repository

import (
	"context"
	"strings"

	"bibliothek/db"
)

// UeberfaelligesMedium repräsentiert ein einzelnes Buch- oder Medienexemplar, das die Rückgabefrist überschritten hat.
type UeberfaelligesMedium struct {
	// AusleiheID ist die UUID des Ausleihvorgangs.
	AusleiheID string `json:"ausleihe_id"`
	// Titel ist der Haupttitel des überfälligen Mediums.
	Titel string `json:"titel"`
	// Autor ist der Autor des Werks.
	Autor string `json:"autor"`
	// ISBN ist die ISBN des Buchs.
	ISBN string `json:"isbn"`
	// CoverURL verweist optional auf das Coverbild des Buchs.
	CoverURL string `json:"cover_url,omitempty"`
	// FaelligAm ist das formatierte Fälligkeitsdatum (z. B. "20.06.2026").
	FaelligAm string `json:"faellig_am"`
	// TageUeberfaellig speichert die Anzahl der Tage, die das Medium bereits überfällig ist.
	TageUeberfaellig int `json:"tage_ueberfaellig"`
}

// UeberfaelligerSchueler fasst alle überfälligen Medien eines konkreten Schülers zusammen.
type UeberfaelligerSchueler struct {
	// SchuelerID ist die UUID des betroffenen Schülers.
	SchuelerID string `json:"schueler_id"`
	// Name ist der vollständige Name des Schülers.
	Name string `json:"name"`
	// Klasse ist die aktuelle Schulklasse des Schülers.
	Klasse string `json:"klasse"`
	// MaxTage ist die maximale Anzahl an überfälligen Tagen unter allen Medien.
	MaxTage int `json:"max_tage"`
	// Medien listet alle überfälligen Buchexemplare auf, die auf diesen Schüler entfallen.
	Medien []UeberfaelligesMedium `json:"medien"`
}

// MahnwesenKlasse gruppiert überfällige Schüler und Ausleihen nach ihren Schulklassen für die Mahnwesen-Übersicht.
type MahnwesenKlasse struct {
	// Klasse ist das Klassenkürzel (z. B. "09A").
	Klasse string `json:"klasse"`
	// LehrerEmail ist die E-Mail-Adresse der Klassenleitung (für automatische Benachrichtigungen).
	LehrerEmail string `json:"lehrer_email"`
	// Schueler enthält die Liste aller Schüler dieser Klasse mit überfälligen Büchern.
	Schueler []UeberfaelligerSchueler `json:"schueler"`
}

// MahnwesenRepository stellt Abfragemethoden zur Auswertung von Fristüberschreitungen und Mahnstufen zur Verfügung.
type MahnwesenRepository struct {
	db db.PgxPoolIface
}

// NewMahnwesenRepository erzeugt eine neue Instanz des MahnwesenRepositorys.
func NewMahnwesenRepository(pool db.PgxPoolIface) *MahnwesenRepository {
	return &MahnwesenRepository{db: pool}
}

// CheckFerienAktiv prüft, ob das heutige Datum in einen eingetragenen Ferien- oder Schließzeitraum fällt.
// Ist dies der Fall, können automatische Mahnungen systemseitig pausiert werden.
func (repo *MahnwesenRepository) CheckFerienAktiv(ctx context.Context) (bool, string, error) {
	q := `
		SELECT bezeichnung 
		FROM ferien_schliesszeiten 
		WHERE CURRENT_DATE >= start_datum AND CURRENT_DATE <= end_datum 
		LIMIT 1
	`
	var bezeichnung string
	err := repo.db.QueryRow(ctx, q).Scan(&bezeichnung)
	if err != nil {
		// pgx bzw. Standardfehler abfangen, wenn kein Zeitraum aktiv ist
		if err.Error() == "no rows in result set" || strings.Contains(err.Error(), "no rows") {
			return false, "", nil
		}
		return false, "", err
	}
	return true, bezeichnung, nil
}

// CountReturnsToday queries the database for loans successfully returned today.
func (repo *MahnwesenRepository) CountReturnsToday(ctx context.Context) (int, error) {
	var count int
	err := repo.db.QueryRow(ctx, `
		SELECT count(*) FROM ausleihen
		WHERE rueckgabe_am IS NOT NULL
		  AND DATE(rueckgabe_am) = CURRENT_DATE
	`).Scan(&count)
	return count, err
}
