package bereitschaft

import (
	"fmt"
	"sort"
	"strings"

	"bibliothek/db"
)

// pruefeAdminKonten macht sichtbar, WER Vollzugriff hat und die Alarm-Mails erhält.
//
// Anlass (16.08.2026 abends): Die erste Kritisch-Mail ging an ein Admin-Konto, von
// dem der Betreiber nichts wusste — auf Prod lagen VIER aktive Admins, drei davon
// ohne bekannte Herkunft (die Konten-Anlage war bis dahin unauditiert). Eine Liste
// auf dieser Seite hätte das Wochen früher gezeigt. Bewusst Stufe OK, solange
// überhaupt Admins existieren: Mehrere Admins können völlig richtig sein — die
// Seite URTEILT nicht, sie zeigt. Wer hier einen unbekannten Namen liest, handelt.
func pruefeAdminKonten(l Lage) Befund {
	b := Befund{Bereich: "Admin-Konten"}
	switch {
	case l.AdminKonten == nil:
		b.Stufe = StufeWarnung
		b.Befund = "Die Admin-Konten konnten nicht gelesen werden."
		b.Folge = "Unbekannt, wer Vollzugriff hat und die Alarm-Mails erhält."
		b.Abhilfe = abhilfeDBNeuLaden
	case len(l.AdminKonten) == 0:
		b.Stufe = StufeKritisch
		b.Befund = "Kein aktives Admin-Konto vorhanden."
		b.Folge = "Niemand kann das System verwalten, und die Kritisch-Alarme erreichen niemanden."
		b.Abhilfe = "Über die Datenbank bzw. INITIAL_ADMIN_EMAIL ein Admin-Konto herstellen."
	default:
		b.Stufe = StufeOK
		ziel := "Die Kritisch-Alarme gehen an alle diese Konten."
		if strings.TrimSpace(l.AlarmEmpfaenger) != "" {
			ziel = "Die Kritisch-Alarme gehen an die konfigurierten Empfänger: " + l.AlarmEmpfaenger + "."
		}
		b.Befund = fmt.Sprintf("%d Admin-Konto/-Konten mit Vollzugriff: %s. %s",
			len(l.AdminKonten), strings.Join(l.AdminKonten, "; "), ziel)
	}
	return b
}

// pruefeRechteVorgabe vergleicht die Live-Rechte mit der Code-Vorgabe (db.RechteVorgabe).
//
// Der Seed schreibt nur FEHLENDE Zeilen (ON CONFLICT DO NOTHING) — ändert sich die
// Vorgabe im Code, erreicht sie eine bestehende Anlage nie von selbst. Genau so sah
// das Kollegium wochenlang nur einen Teil seiner Menüpunkte, und niemand merkte es.
//
// Eine Abweichung ist bewusst nur eine WARNUNG, kein Fehler: role_permissions ist
// über die Oberfläche einstellbar, die Abweichung kann eine Admin-Entscheidung sein.
// Die Prüfung urteilt nicht darüber — sie macht die Abweichung sichtbar, damit ein
// Mensch entscheidet. Repariert wird hier nichts.
func pruefeRechteVorgabe(l Lage) Befund {
	b := Befund{Bereich: "Rechte-Vorgabe"}
	if l.RechteLive == nil {
		b.Stufe = StufeWarnung
		b.Befund = "Die Live-Rechte (role_permissions) konnten nicht gelesen werden."
		b.Folge = "Ob Menü und API der aktuellen Vorgabe folgen, ist unbekannt."
		b.Abhilfe = abhilfeDBNeuLaden
		return b
	}

	abweichungen := rechteAbweichungen(l.RechteLive)
	if len(abweichungen) == 0 {
		b.Stufe = StufeOK
		b.Befund = fmt.Sprintf("Live-Rechte decken sich mit der Code-Vorgabe (%d Zeilen).", len(db.RechteVorgabe))
		return b
	}

	// Map-Reihenfolge ist Zufall — sortieren, damit Anzeige und Tests stabil sind.
	sort.Strings(abweichungen)
	const zeigeMax = 6
	gezeigt := abweichungen
	if len(gezeigt) > zeigeMax {
		gezeigt = append(append([]string{}, gezeigt[:zeigeMax]...),
			fmt.Sprintf("… und %d weitere", len(abweichungen)-zeigeMax))
	}
	b.Stufe = StufeWarnung
	b.Befund = fmt.Sprintf("%d Abweichung(en) von der Code-Vorgabe: %s.",
		len(abweichungen), strings.Join(gezeigt, "; "))
	b.Folge = "Menü und API richten sich nach der Live-Tabelle, nicht nach dem Code. " +
		"Nach einer Code-Änderung bleibt die alte Einstellung bestehen — eine Rolle sieht " +
		"dann mehr oder weniger, als die aktuelle Vorgabe vorsieht, ohne dass es auffällt."
	b.Abhilfe = "Jede Zeile prüfen. Bewusste Admin-Entscheidung: so lassen (die Warnung " +
		"dokumentiert sie). Drift nach Code-Änderung: unter Benutzer & Rechte → Rollen & Rechte angleichen."
	return b
}

// rechteAbweichungen vergleicht die Live-Rechte in beiden Richtungen mit der Vorgabe und
// nennt jede Abweichung; die Reihenfolge ist die der Maps und damit zufällig.
func rechteAbweichungen(live map[string]map[string]bool) []string {
	var abweichungen []string
	vorgabe := map[string]map[string]bool{}
	for _, v := range db.RechteVorgabe {
		if vorgabe[v.Role] == nil {
			vorgabe[v.Role] = map[string]bool{}
		}
		vorgabe[v.Role][v.Permission] = v.Allowed
		zeile, bekannt := live[v.Role]
		liveWert, vorhanden := false, false
		if bekannt {
			liveWert, vorhanden = zeile[v.Permission]
		}
		switch {
		case !vorhanden:
			abweichungen = append(abweichungen,
				fmt.Sprintf("%s/%s fehlt live (Vorgabe: %s)", v.Role, v.Permission, anAus(v.Allowed)))
		// Optionale Paare (db.RechteOptional): Der Haken in der Rechte-Matrix ist
		// dort der vorgesehene Gebrauch, kein Drift — nur die Existenz zählt.
		case db.RechteOptional[v.Role+"/"+v.Permission]:
		case liveWert != v.Allowed:
			abweichungen = append(abweichungen,
				fmt.Sprintf("%s/%s live %s, Vorgabe %s", v.Role, v.Permission, anAus(liveWert), anAus(v.Allowed)))
		}
	}
	// Gegenrichtung: Live-Zeilen, deren Rolle oder Recht die Vorgabe nicht kennt —
	// Reste alter Vokabulare (z. B. eine umbenannte Rolle) oder Tippfehler im Editor.
	for rolle, zeile := range live {
		for recht, erlaubt := range zeile {
			if _, kennt := vorgabe[rolle][recht]; !kennt {
				abweichungen = append(abweichungen,
					fmt.Sprintf("%s/%s live %s, in der Vorgabe unbekannt", rolle, recht, anAus(erlaubt)))
			}
		}
	}
	return abweichungen
}

func anAus(erlaubt bool) string {
	if erlaubt {
		return "an"
	}
	return "aus"
}
