package repository

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// LitteraHerkunftPraefix ist die Marke, die der Littera-Altbestandsimport in
// schueler.lusd_id schreibt ('littera:<Buchungsnummer>', internal/littera). Sie ist
// KEINE LUSD-ID: Für den Landesabgleich gilt ein solcher Schüler als ID-los — er muss
// per Name+Geburtsdatum zugeordnet werden und darf NICHT als „LUSD-verwaltet" gelten,
// sonst stünde er bei jedem Import als Abgänger da (seine Marke steht nie in der CSV)
// und würde anonymisiert. Bis 21.08.2026 war genau das der Fall.
const LitteraHerkunftPraefix = "littera:"

// HatEchteLusdID sagt, ob ein lusd_id-Wert eine LUSD-Kennung ist (gesetzt und keine
// Littera-Herkunftsmarke).
func HatEchteLusdID(lusdID *string) bool {
	return lusdID != nil && *lusdID != "" && !strings.HasPrefix(*lusdID, LitteraHerkunftPraefix)
}

// LusdSchluessel normalisiert Vorname, Nachname und Geburtsdatum zu einem
// vergleichbaren Schlüssel (kleingeschrieben, Datum als YYYY-MM-DD). Leerer String,
// wenn kein Geburtsdatum vorliegt — ohne Datum ist kein sicherer Abgleich möglich.
func LusdSchluessel(vorname, nachname string, geb *time.Time) string {
	if geb == nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(vorname)) + "\x1f" +
		strings.ToLower(strings.TrimSpace(nachname)) + "\x1f" +
		geb.Format("2006-01-02")
}

// LusdNamensSchluessel ist der Nur-Name-Schlüssel (Vorname + Nachname, kleingeschrieben,
// getrimmt) — die letzte Stufe, wenn der Export weder ID noch Geburtsdatum trägt.
func LusdNamensSchluessel(vorname, nachname string) string {
	return strings.ToLower(strings.TrimSpace(vorname)) + "\x1f" + strings.ToLower(strings.TrimSpace(nachname))
}

// LusdBestandsSchueler ist eine nicht gelöschte schueler-Zeile, so wie der LUSD-Import
// sie sieht. LusdID ist "" ohne ECHTE Kennung (NULL oder Littera-Marke), Schluessel der
// Name+Geburtsdatum-Schlüssel ("" ohne Datum), Namensschluessel der Nur-Name-Schlüssel,
// LusdBestaetigt spiegelt lusd_bestaetigt_am IS NOT NULL (Migration 084).
type LusdBestandsSchueler struct {
	ID, Klasse, Vorname, Nachname        string
	LusdID, Schluessel, Namensschluessel string
	IstAbgaenger, LusdBestaetigt         bool
}

// LadeLusdBestand liest ALLE nicht soft-gelöschten Schüler — aktive UND Abgänger. Beide
// braucht der Import: Aktive für Klassenwechsel und Abgänger-Erkennung, Abgänger für
// Rückkehrer. deleted_at IS NULL ist zwingend: Eine Papierkorb-Zeile darf nie matchen,
// sonst würde der aktive Schüler nie neu angelegt. Läuft in der Import-Transaktion.
func LadeLusdBestand(ctx context.Context, tx pgx.Tx) ([]LusdBestandsSchueler, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, klasse, vorname, nachname, lusd_id, geburtsdatum, ist_abgaenger,
		       lusd_bestaetigt_am IS NOT NULL
		FROM schueler WHERE deleted_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bestand []LusdBestandsSchueler
	for rows.Next() {
		var s LusdBestandsSchueler
		var lusdID *string
		var geb *time.Time
		if err := rows.Scan(&s.ID, &s.Klasse, &s.Vorname, &s.Nachname, &lusdID, &geb, &s.IstAbgaenger, &s.LusdBestaetigt); err != nil {
			return nil, err
		}
		if HatEchteLusdID(lusdID) {
			s.LusdID = *lusdID
		}
		s.Schluessel = LusdSchluessel(s.Vorname, s.Nachname, geb)
		s.Namensschluessel = LusdNamensSchluessel(s.Vorname, s.Nachname)
		bestand = append(bestand, s)
	}
	return bestand, rows.Err()
}
