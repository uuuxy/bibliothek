package lusd

import (
	"context"
	"time"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// Die Bestandsdaten und ihre Schlüssel liegen in repository/lusd_bestand.go (Schichtung:
// ein Handler formuliert kein SQL). Hier stehen nur die Nachschlagekarten des Imports.

type bestandsSchueler = repository.LUSDBestandsSchueler

func waisenSchluessel(vorname, nachname string, geb *time.Time) string {
	return repository.LUSDSchluessel(vorname, nachname, geb)
}

func namensSchluessel(vorname, nachname string) string {
	return repository.LUSDNamensSchluessel(vorname, nachname)
}

// bestandsSchluessel liefert den Schlüssel, unter dem der Modus einen Bestandsschüler
// nachschlägt: LUSD-ID, Name+Geburtsdatum oder nur Name.
func bestandsSchluessel(s *bestandsSchueler, modus Modus) string {
	switch modus {
	case ModusName:
		return s.Schluessel
	case ModusNurName:
		return s.Namensschluessel
	default:
		return s.LUSDID
	}
}

func ladeBestand(ctx context.Context, tx pgx.Tx) ([]bestandsSchueler, error) {
	return repository.LadeLUSDBestand(ctx, tx)
}

// bestandsIndex sind die Nachschlagekarten eines Modus. Ein nil-Eintrag heißt MEHRDEUTIG:
// zwei Bestandszeilen teilen denselben Schlüssel (im Namensmodus etwa "Anna Müller" und
// "anna müller" mit demselben Geburtstag — der Unique-Index ist case-sensitiv). Mehr-
// deutiges wird NICHT zugeordnet: lieber melden als die falsche Person zusammenführen.
type bestandsIndex struct {
	aktiv     map[string]*bestandsSchueler // Modus-Schlüssel → aktiver Schüler
	abgaenger map[string]*bestandsSchueler // Modus-Schlüssel → Abgänger (Rückkehrer-Kandidat)
	// waisen gibt es nur im ID-Modus: Name+Geburtsdatum → aktiver Schüler OHNE echte
	// LUSD-ID (Handanlage oder Littera-Übernahme). Eine CSV-Zeile, deren ID im Bestand
	// fehlt, adoptiert ihn statt ein Duplikat anzulegen.
	waisen map[string]*bestandsSchueler
	// ohneSchluessel gibt es nur im Name+Geburtsdatum-Modus: aktive Schüler ohne
	// Geburtsdatum. Sie sind nicht über den Schlüssel abgleichbar — die Vorschau meldet
	// sie. (Im Nur-Name-Modus hat jeder Schüler einen Schlüssel.)
	ohneSchluessel []bestandsSchueler
	// ohneDatumNachName: dieselben Schüler, nach NAME nachschlagbar (nil = mehrdeutig).
	// Rückfallstufe für den Modus-Wechsel: Eine Schule, die erst mit der LANIS-Liste
	// (nur Name) importierte und später einen Export MIT Geburtsdatum bekommt, hätte sonst
	// jeden Bestandsschüler als „neu" doppelt angelegt (Prüfung 22.08.2026). Ein
	// eindeutiger Namenstreffer wird zugeordnet und bekommt das Datum nachgetragen.
	ohneDatumNachName map[string]*bestandsSchueler
	// mehrdeutigAktiv sind die aktiven Zeilen hinter nil-Einträgen in aktiv — damit
	// die Vorschau die Paare nennen kann, auch wenn keine CSV-Zeile sie trifft.
	mehrdeutigAktiv []bestandsSchueler
}

// baueIndex sortiert den Bestand in die Karten des Modus ein.
func baueIndex(bestand []bestandsSchueler, modus Modus) bestandsIndex {
	index := bestandsIndex{
		aktiv:             make(map[string]*bestandsSchueler),
		abgaenger:         make(map[string]*bestandsSchueler),
		waisen:            make(map[string]*bestandsSchueler),
		ohneDatumNachName: make(map[string]*bestandsSchueler),
	}
	for i := range bestand {
		s := &bestand[i]
		schluessel := bestandsSchluessel(s, modus)
		switch {
		case schluessel == "" && modus == ModusName && !s.IstAbgaenger:
			index.ohneSchluessel = append(index.ohneSchluessel, *s)
			trageEin(index.ohneDatumNachName, s.Namensschluessel, s)
		case schluessel == "":
			// ID-Modus ohne echte ID: Adoptions-Kandidat, sofern ein Geburtsdatum da ist.
			if !s.IstAbgaenger && s.Schluessel != "" {
				trageEin(index.waisen, s.Schluessel, s)
			}
		case s.IstAbgaenger:
			trageEin(index.abgaenger, schluessel, s)
		default:
			if vorher, belegt := index.aktiv[schluessel]; belegt {
				if vorher != nil {
					index.mehrdeutigAktiv = append(index.mehrdeutigAktiv, *vorher)
				}
				index.mehrdeutigAktiv = append(index.mehrdeutigAktiv, *s)
			}
			trageEin(index.aktiv, schluessel, s)
		}
	}
	return index
}

// trageEin setzt den Eintrag — oder markiert den Schlüssel bei Kollision als mehrdeutig.
func trageEin(m map[string]*bestandsSchueler, schluessel string, s *bestandsSchueler) {
	if _, belegt := m[schluessel]; belegt {
		m[schluessel] = nil
		return
	}
	m[schluessel] = s
}
