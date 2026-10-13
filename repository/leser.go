// Package repository stellt den Datenzugriff und die Tabellenmodelle der PostgreSQL-Datenbank bereit.
package repository

import (
	"bibliothek/db"
	"bibliothek/internal/ausweis"
	"context"
	"time"
)

// Leser repräsentiert einen LESER in der Datenbank (Tabelle `leser`).
//
// Der Typ heißt weiter Leser, weil die allermeisten Leser Schüler sind und die Listen,
// Mahnungen und Klassenansichten genau die meinen; sie lesen die Sicht `schueler`. Die
// Theke dagegen fragt nach einem LESER (FindeZuBarcode, Finde) und bekommt auch
// Kollegen — unterscheidbar an `Art`.
type Leser struct {
	// ID ist der eindeutige Primärschlüssel (UUID) des Lesers.
	ID string `json:"id"`
	// Art: chk_leser_art (Migration 123 und 153). Sie entscheidet keine Rechte,
	// sondern nur, wen der LUSD-Abgleich und der Löschjob erfassen — und was die Theke
	// am Treffer anzeigt. Leer, wo die Abfrage die Sicht `schueler` liest: dort ist
	// jede Zeile ein Schüler.
	Art string `json:"art,omitempty"`
	// BarcodeID ist der eindeutige Barcode des Ausweises.
	BarcodeID string `json:"barcode_id"`
	// Vorname ist der Vorname des Schülers.
	Vorname string `json:"vorname"`
	// Nachname ist der Nachname des Schülers.
	Nachname string `json:"nachname"`
	// Klasse ist die aktuelle Klasse bzw. Kursstufe des Schülers (z. B. "07B").
	Klasse string `json:"klasse"`
	// AbgaengerJahr ist das Jahr, in dem der Schüler voraussichtlich die Schule verlässt.
	// Steuert die DSGVO-Löschung und die Stapel-Archivierung — NICHT die Gültigkeit des
	// Ausweises. Beide Zahlen fallen auseinander: Ein Gymnasialschüler bleibt bis
	// Jahrgang 13, sein Ausweis gilt aber nur bis zum Ende der Mittelstufe.
	AbgaengerJahr int `json:"abgaenger_jahr"`
	// AusweisGueltigBis ist das Jahr, bis zu dem der Schülerausweis gilt (31.07.), aus
	// dem Bildungsgang der Klasse gerechnet (internal/ausweis). Kein Datenbankfeld,
	// sondern beim Lesen gesetzt. nil heißt: aus dieser Klassenbezeichnung nicht
	// ableitbar — dann fragt der Druckdialog nach, statt ein Datum zu erfinden.
	AusweisGueltigBis *int `json:"ausweis_gueltig_bis,omitempty"`
	// IstGesperrt ist die Sperre, die das Programm setzt: Ehemalige (Versetzung, LUSD-Import),
	// Papierkorb, Anonymisierung. Die Theke lässt dann nur die Rückgabe zu, außer beim
	// Lernmittel (service.pruefeAusleihSperren). Offene Forderungen und überfällige Medien
	// setzen sie NICHT — die zählt die Theke bei jeder Ausleihe selbst.
	IstGesperrt bool `json:"ist_gesperrt"`
	// LusdID ist die Schüler-ID aus dem hessischen LUSD-System für automatisierte Abgleiche.
	LusdID *string `json:"lusd_id,omitempty"`
	// IstAbgaenger markiert Schüler, die die Schule bereits verlassen haben.
	IstAbgaenger bool `json:"ist_abgaenger"`
	// Geburtsdatum speichert das Geburtsdatum (Format: DATE in PostgreSQL, als String im Code).
	Geburtsdatum *string `json:"geburtsdatum,omitempty"`
	// ErstelltAm ist der Erstellungszeitpunkt des Schülerdatensatzes.
	ErstelltAm time.Time `json:"erstellt_am"`
	// AktualisiertAm ist der Zeitpunkt der letzten Aktualisierung.
	AktualisiertAm time.Time `json:"aktualisiert_am"`
	// IstVonHandGesperrt ist die Sperre von Hand (PATCH /api/admin/students/{id}/lock). Sie gilt
	// auch beim Lernmittel; aufgehoben wird sie in der Akte.
	IstVonHandGesperrt bool `json:"is_manually_blocked"`
	// Sperrgrund ist der Grund der Sperre — von Hand oder vom Programm, eine Spalte für beide.
	Sperrgrund *string `json:"block_reason,omitempty"`
	// Strasse ist der Straßenname der Postanschrift (optionale Stammdaten).
	Strasse string `json:"strasse"`
	// Hausnummer ergänzt die Straße der Postanschrift.
	Hausnummer string `json:"hausnummer"`
	// Plz ist die Postleitzahl der Postanschrift.
	Plz string `json:"plz"`
	// Ort ist der Wohnort der Postanschrift.
	Ort string `json:"ort"`
	// ElternEmail ist die Kontakt-E-Mail der Erziehungsberechtigten.
	ElternEmail string `json:"eltern_email"`
	// IstAnonymisiert: anonymized_at ist gesetzt, die Zeile ist keine Person mehr. Die Theke
	// gibt ihr nichts aus, auch kein Lernmittel (service.pruefeAusleihSperren) — die Namenssuche
	// findet sie noch („Abgänger Anonymisiert-…"). Gefüllt nur über die Spaltenliste der
	// Leser-Abfragen (repository.spaltenLeser); nicht in der API.
	IstAnonymisiert bool `json:"-"`
}

// scanner kapselt die Scan-Schnittstelle von pgx.Row und pgx.Rows,
// um gemeinsame Helferfunktionen zum Einlesen von Zeilen zu ermöglichen.
type scanner interface {
	Scan(dest ...any) error
}

// LeserAbfragen definiert die Operationen zur Abfrage und zum Abgleich von Schülern in der Datenbank.
type LeserAbfragen interface {
	// FindeZuBarcode sucht einen LESER anhand seiner Ausweisnummer — Schüler UND
	// Kollegium. Das ist die Abfrage der Theke: Dort steht ein Mensch mit einer Karte,
	// und welcher Art er ist, entscheidet nichts über das Ausleihen (Migration 125).
	// Liefert nil, wenn die Nummer zu niemandem gehört.
	//
	// Eine Schwester über die Sicht `schueler` (GetByBarcode) gab es bis zum 21.09.2026;
	// ihr letzter Aufrufer war die Scan-Erkennung der Suchleiste, und die meint den Leser.
	FindeZuBarcode(ctx context.Context, barcode string) (*Leser, error)

	// Finde sucht einen LESER anhand seiner UUID — wie FindeZuBarcode, aber
	// für den bereits an der Theke stehenden Leser und für die Akte in der Leserdatei.
	//
	// Ein GetByID über die Sicht `schueler` gab es daneben bis zum 16.09.2026; es hatte
	// nach der Umstellung der Akte keinen Aufrufer mehr und ist zurückgebaut. Wer
	// wirklich nur Schüler meint, grenzt an der Art ein, statt eine zweite Tür zu bauen.
	Finde(ctx context.Context, id string) (*Leser, error)

	// FindeUnscharf sucht tokenweise und diakritikfrei über Vorname, Nachname
	// und Barcode-ID. Zweiter Rückgabewert ist die Gesamtzahl der Treffer vor dem
	// Limit — ohne sie kann die Oberfläche eine abgeschnittene Liste nicht als solche
	// kennzeichnen und zeigt stillschweigend nur die ersten Namen.
	FindeUnscharf(ctx context.Context, suchtext string, limit int) ([]Leser, int, error)

	// Hinweis: Der LUSD-Abgleich läuft ausschließlich über die Tür in api/leser/lusd.go und den
	// Lauf in internal/lusd (Fuehre → WendeAenderungenAn). Eine frühere
	// Massen-Pipeline (BulkSyncLUSD/GetAllLUSDStudents) wurde entfernt: ungenutzt und
	// mit latenten Fehlern (u. a. Ghost-Block bei Rückkehrern).

	// HatFoto checks if an encrypted photo exists for the student.
	HatFoto(ctx context.Context, leserID string) (bool, error)

	// ZaehleOffeneSchaeden: Anzahl und Summe der unbezahlten Forderungen des Lesers.
	ZaehleOffeneSchaeden(ctx context.Context, leserID string) (anzahl int, summe float64, err error)

	// LadeLaufendeAusleihen retrieves all books currently borrowed by the student.
	LadeLaufendeAusleihen(ctx context.Context, leserID string) ([]LaufendeAusleihe, error)

	// LadeKlassen returns a list of all active classes.
	LadeKlassen(ctx context.Context) ([]string, error)

	// LadeEtikettZeilen liefert Barcode, Name und Klasse zu den angegebenen IDs — die vier
	// Angaben, die auf einem Klebeetikett stehen. Sortiert nach Nachname, Vorname.
	LadeEtikettZeilen(ctx context.Context, ids []string) ([]LeserEtikettZeile, error)

	// LadeSchuelerListe liefert Schüler samt Ausleihzahlen, optional nach Klasse
	// und/oder Suchbegriff eingegrenzt. Die Suche läuft über dieselben SQL-Bausteine
	// wie FindeUnscharf und kennt keine 500er-Grenze.
	LadeSchuelerListe(ctx context.Context, klassen []string, suche string, sortierung LeserSortierung) ([]LeserListenZeile, error)
	// LadeLeserListe liefert ALLE Leser — Schüler und Kollegium — für die Leserdatei.
	// Dieselben Zeilen, dieselbe Suche; gelesen wird die Tabelle `leser` statt der Sicht.
	LadeLeserListe(ctx context.Context, klassen []string, suche string, sortierung LeserSortierung) ([]LeserListenZeile, error)
	// LadeEhemaligenListe liefert die Weggegangenen (ist_abgaenger = true) für den
	// Reiter „Ehemalige / Archiv" — dieselben Zeilen und dieselbe Suche wie die Aktiven.
	LadeEhemaligenListe(ctx context.Context, suche string, sortierung LeserSortierung) ([]LeserListenZeile, error)
}

// pgLeserAbfragen implementiert das LeserAbfragen für PostgreSQL.
type pgLeserAbfragen struct {
	db db.PgxPoolIface
}

// NewLeserAbfragen erzeugt eine neue Instanz des PostgreSQL-basierten StudentRepositorys.
func NewLeserAbfragen(db db.PgxPoolIface) LeserAbfragen {
	return &pgLeserAbfragen{db: db}
}

// leseLeserMitZusatz scannt die Standard-Spaltenliste und danach beliebige
// Zusatzspalten (z. B. count(*) OVER () für die Gesamttrefferzahl). So bleibt die
// Feldreihenfolge an genau einer Stelle — eine zweite Kopie würde beim nächsten
// Spaltenzuwachs still auseinanderlaufen.
func leseLeserMitZusatz(row scanner, zusatz ...any) (*Leser, error) {
	var s Leser
	ziele := []any{
		&s.ID, &s.BarcodeID, &s.Vorname, &s.Nachname, &s.Klasse, &s.AbgaengerJahr, &s.IstGesperrt, &s.LusdID, &s.IstAbgaenger, &s.Geburtsdatum, &s.ErstelltAm, &s.AktualisiertAm, &s.IstVonHandGesperrt, &s.Sperrgrund,
		&s.Strasse, &s.Hausnummer, &s.Plz, &s.Ort, &s.ElternEmail, &s.IstAnonymisiert,
	}
	if err := row.Scan(append(ziele, zusatz...)...); err != nil {
		return nil, err
	}
	s.AusweisGueltigBis = ausweisGueltigBis(s.Klasse)
	return &s, nil
}

// ausweisGueltigBis rechnet die Ausweisgültigkeit aus der Klasse. Kein Datenbankfeld:
// Die Regel darf sich ändern (Schulform, Zweigwechsel), ohne dass Altbestände
// nachgezogen werden müssen — und ein gespeicherter Wert wäre nach jedem
// Schuljahreswechsel still veraltet.
//
// Rückgabe nil bei einer Klassenbezeichnung, aus der sich nichts ableiten lässt. Der
// Druckdialog fragt dann nach, statt ein Datum zu erfinden.
func ausweisGueltigBis(klasse string) *int {
	jahr, ok := ausweis.GueltigBisJahr(klasse, ausweis.SchuljahrEnde(time.Now()))
	if !ok {
		return nil
	}
	return &jahr
}

// spaltenLeser ist die Spaltenliste von leseLeserMitZusatz — einmal, damit die Abfragen über
// die Sicht `schueler` und die über die Tabelle `leser` nicht auseinanderlaufen können.
const spaltenLeser = `id, coalesce(barcode_id, ''), coalesce(vorname, ''), coalesce(nachname, ''), coalesce(klasse, ''), coalesce(abgaenger_jahr, 0), coalesce(ist_gesperrt, false), lusd_id, coalesce(ist_abgaenger, false), TO_CHAR(geburtsdatum, 'YYYY-MM-DD'), erstellt_am, aktualisiert_am, coalesce(is_manually_blocked, false), block_reason, coalesce(strasse, ''), coalesce(hausnummer, ''), coalesce(plz, ''), coalesce(ort, ''), coalesce(eltern_email, ''), anonymized_at IS NOT NULL`
