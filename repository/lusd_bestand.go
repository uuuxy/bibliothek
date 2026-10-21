package repository

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// litteraHerkunftPraefix ist die Marke, die der Littera-Altbestandsimport in schueler.lusd_id
// schreibt ('littera:<Buchungsnummer>', internal/littera). Sie ist keine LUSD-ID: Für den
// Landesabgleich gilt ein solcher Schüler als ID-los. Er muss per Name und Geburtsdatum
// zugeordnet werden und darf nicht als „LUSD-verwaltet" gelten, sonst stünde er bei jedem
// Import als Abgänger da (seine Marke steht nie in der CSV) und würde anonymisiert.
const litteraHerkunftPraefix = "littera:"

// hatEchteLUSDID sagt, ob ein lusd_id-Wert eine LUSD-Kennung ist (gesetzt und keine
// Littera-Herkunftsmarke).
func hatEchteLUSDID(lusdID *string) bool {
	return lusdID != nil && *lusdID != "" && !strings.HasPrefix(*lusdID, litteraHerkunftPraefix)
}

// LUSDSchluessel normalisiert Vorname, Nachname und Geburtsdatum zu einem
// vergleichbaren Schlüssel (Namen in der Normalform suchnorm, Datum als YYYY-MM-DD).
// Leerer String, wenn kein Geburtsdatum vorliegt — ohne Datum ist kein sicherer
// Abgleich möglich.
func LUSDSchluessel(vorname, nachname string, geb *time.Time) string {
	if geb == nil {
		return ""
	}
	return NamensteilNorm(vorname) + "\x1f" + NamensteilNorm(nachname) + "\x1f" + geb.Format("2006-01-02")
}

// LUSDNamensSchluessel ist der Nur-Name-Schlüssel (Vorname + Nachname in der Normalform
// suchnorm) — die letzte Stufe, wenn der Export weder ID noch Geburtsdatum trägt.
func LUSDNamensSchluessel(vorname, nachname string) string {
	return NamensteilNorm(vorname) + "\x1f" + NamensteilNorm(nachname)
}

// NamensteilNorm ist die eine Regel hinter beiden Schlüsseln: Ränder und mehrfache Leerzeichen
// weg, dann die Normalform suchnorm. Mit lower und trim allein fände der Export „Anna Mueller"
// die von Hand angelegte „Anna Müller" nie: Neuanlage plus „nicht im Export" statt Zuordnung.
// Gefahrlos ist die weitere Regel, weil der Import einen doppelt belegten Schlüssel als
// mehrdeutig markiert und dann niemanden anfasst (internal/lusd/bestand.go, trageEin): Ein
// Schlüssel, der zwei Menschen zusammenwirft, führt nie in eine falsche Zuordnung. Bindestriche
// bleiben stehen: „Anna-Lena" und „Anna" sind verschiedene Vornamen; die Umbenennungs-Paarung
// (internal/lusd/paarung.go) baut darauf ihre weitere Regel mit menschlicher Bestätigung.
func NamensteilNorm(s string) string {
	return suchnorm(strings.Join(strings.Fields(s), " "))
}

// LUSDBestandsSchueler ist eine nicht gelöschte schueler-Zeile, so wie der LUSD-Import sie
// sieht. LUSDID ist "" ohne echte Kennung (NULL oder Littera-Marke), Schluessel der Schlüssel
// aus Name und Geburtsdatum ("" ohne Datum), Namensschluessel der Schlüssel nur aus dem Namen,
// LUSDBestaetigt spiegelt lusd_bestaetigt_am IS NOT NULL.
//
// Geburtsdatum, EintrittAm, Strasse und PLZ trägt die Zeile roh mit: Die Umbenennungs-Paarung
// (internal/lusd/paarung.go) vergleicht damit einen Abgänger mit einem vermeintlichen
// Neuzugang; gleiches Geburtsdatum, gleicher Schuleintritt und gleiche Anschrift sind die
// Signale, die eine Namensänderung überstehen. Anonymisiert sagt, ob die Zeile ihre
// Personendaten schon verloren hat; sie taugt dann für keine Paarung.
type LUSDBestandsSchueler struct {
	ID, Klasse, Vorname, Nachname        string
	LUSDID, Schluessel, Namensschluessel string
	Strasse, PLZ                         string
	Geburtsdatum, EintrittAm             *time.Time
	IstAbgaenger, LUSDBestaetigt         bool
	Anonymisiert                         bool
}

// LadeLUSDBestand liest alle Schüler außerhalb des Papierkorbs, aktive und Abgänger. Beide
// braucht der Import: Aktive für Klassenwechsel und das Erkennen der Abgänger, Abgänger für
// Rückkehrer. deleted_at IS NULL ist zwingend: Eine Zeile im Papierkorb darf nie treffen, sonst
// würde der aktive Schüler nie neu angelegt. Läuft in der Transaktion des Imports.
//
// art = 'schueler' ist ebenso zwingend: Die Tabelle führt alle Leser, und durch die LUSD kommen
// ausschließlich Schüler. Ohne die Einschränkung stünde jeder Kollege im Bestand, den der
// Export nicht kennt, also jeder, und der Abgleich machte ihn zum Abgänger: gesperrt, und nach
// der Karenz Name, Adresse und Geburtsdatum geleert. Gemerkt hätte es niemand, bis jemand einen
// Namen sucht.
//
// Der Scan in *string unten stolperte an einem Kollegen ohnehin: Er hat keine Klasse, und die
// Spalte ist bei ihm NULL (repository/leser_lusd_schutz_pg_test.go zeigt beides). Das ist ein
// Glücksfall, kein Schutz. Die zweite Schranke steht deshalb in der Datenbank:
// chk_leser_nur_schueler_werden_abgaenger.
func LadeLUSDBestand(ctx context.Context, tx pgx.Tx) ([]LUSDBestandsSchueler, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, klasse, vorname, nachname, lusd_id, geburtsdatum, ist_abgaenger,
		       lusd_bestaetigt_am IS NOT NULL, schul_eintritt_am,
		       COALESCE(strasse, ''), COALESCE(plz, ''), anonymized_at IS NOT NULL
		FROM schueler WHERE deleted_at IS NULL AND art = 'schueler'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bestand []LUSDBestandsSchueler
	for rows.Next() {
		var s LUSDBestandsSchueler
		var lusdID *string
		if err := rows.Scan(&s.ID, &s.Klasse, &s.Vorname, &s.Nachname, &lusdID, &s.Geburtsdatum, &s.IstAbgaenger,
			&s.LUSDBestaetigt, &s.EintrittAm, &s.Strasse, &s.PLZ, &s.Anonymisiert); err != nil {
			return nil, err
		}
		if hatEchteLUSDID(lusdID) {
			s.LUSDID = *lusdID
		}
		s.Schluessel = LUSDSchluessel(s.Vorname, s.Nachname, s.Geburtsdatum)
		s.Namensschluessel = LUSDNamensSchluessel(s.Vorname, s.Nachname)
		bestand = append(bestand, s)
	}
	return bestand, rows.Err()
}
