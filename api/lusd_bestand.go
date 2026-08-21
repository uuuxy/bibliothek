package api

import (
	"context"
	"time"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// Die Bestandsdaten und ihre Schlüssel liegen in repository/lusd_bestand.go (Schichtung:
// ein Handler formuliert kein SQL). Hier stehen nur die Nachschlagekarten des Imports.

const litteraHerkunftPraefix = repository.LitteraHerkunftPraefix

type lusdBestandsSchueler = repository.LusdBestandsSchueler

func waisenSchluessel(vorname, nachname string, geb *time.Time) string {
	return repository.LusdSchluessel(vorname, nachname, geb)
}

func namensSchluessel(vorname, nachname string) string {
	return repository.LusdNamensSchluessel(vorname, nachname)
}

// bestandsSchluessel liefert den Schlüssel, unter dem der Modus einen Bestandsschüler
// nachschlägt: LUSD-ID, Name+Geburtsdatum oder nur Name.
func bestandsSchluessel(s *lusdBestandsSchueler, modus lusdModus) string {
	switch modus {
	case lusdModusName:
		return s.Schluessel
	case lusdModusNurName:
		return s.Namensschluessel
	default:
		return s.LusdID
	}
}

func ladeLusdBestand(ctx context.Context, tx pgx.Tx) ([]lusdBestandsSchueler, error) {
	return repository.LadeLusdBestand(ctx, tx)
}

// lusdIndex sind die Nachschlagekarten eines Modus. Ein nil-Eintrag heißt MEHRDEUTIG:
// zwei Bestandszeilen teilen denselben Schlüssel (im Namensmodus etwa "Anna Müller" und
// "anna müller" mit demselben Geburtstag — der Unique-Index ist case-sensitiv). Mehr-
// deutiges wird NICHT zugeordnet: lieber melden als die falsche Person zusammenführen.
type lusdIndex struct {
	aktiv     map[string]*lusdBestandsSchueler // Modus-Schlüssel → aktiver Schüler
	abgaenger map[string]*lusdBestandsSchueler // Modus-Schlüssel → Abgänger (Rückkehrer-Kandidat)
	// waisen gibt es nur im ID-Modus: Name+Geburtsdatum → aktiver Schüler OHNE echte
	// LUSD-ID (Handanlage oder Littera-Übernahme). Eine CSV-Zeile, deren ID im Bestand
	// fehlt, adoptiert ihn statt ein Duplikat anzulegen.
	waisen map[string]*lusdBestandsSchueler
	// ohneSchluessel gibt es nur im Name+Geburtsdatum-Modus: aktive Schüler ohne
	// Geburtsdatum. Sie sind nicht abgleichbar und bleiben unangetastet — die Vorschau
	// meldet sie. (Im Nur-Name-Modus hat jeder Schüler einen Schlüssel.)
	ohneSchluessel []lusdBestandsSchueler
	// mehrdeutigAktiv sind die aktiven Zeilen hinter nil-Einträgen in aktiv — damit
	// die Vorschau die Paare nennen kann, auch wenn keine CSV-Zeile sie trifft.
	mehrdeutigAktiv []lusdBestandsSchueler
}

// baueLusdIndex sortiert den Bestand in die Karten des Modus ein.
func baueLusdIndex(bestand []lusdBestandsSchueler, modus lusdModus) lusdIndex {
	idx := lusdIndex{
		aktiv:     make(map[string]*lusdBestandsSchueler),
		abgaenger: make(map[string]*lusdBestandsSchueler),
		waisen:    make(map[string]*lusdBestandsSchueler),
	}
	for i := range bestand {
		s := &bestand[i]
		key := bestandsSchluessel(s, modus)
		switch {
		case key == "" && modus == lusdModusName && !s.IstAbgaenger:
			idx.ohneSchluessel = append(idx.ohneSchluessel, *s)
		case key == "":
			// ID-Modus ohne echte ID: Adoptions-Kandidat, sofern ein Geburtsdatum da ist.
			if !s.IstAbgaenger && s.Schluessel != "" {
				trageEin(idx.waisen, s.Schluessel, s)
			}
		case s.IstAbgaenger:
			trageEin(idx.abgaenger, key, s)
		default:
			if vorher, belegt := idx.aktiv[key]; belegt {
				if vorher != nil {
					idx.mehrdeutigAktiv = append(idx.mehrdeutigAktiv, *vorher)
				}
				idx.mehrdeutigAktiv = append(idx.mehrdeutigAktiv, *s)
			}
			trageEin(idx.aktiv, key, s)
		}
	}
	return idx
}

// trageEin setzt den Eintrag — oder markiert den Schlüssel bei Kollision als mehrdeutig.
func trageEin(m map[string]*lusdBestandsSchueler, key string, s *lusdBestandsSchueler) {
	if _, belegt := m[key]; belegt {
		m[key] = nil
		return
	}
	m[key] = s
}
