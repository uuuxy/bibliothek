package api

import "fmt"

// dsgvoVerarbeitungsangabenKollegium sind die Pflichtangaben nach Art. 15 Abs. 1 DSGVO für
// eine Lehrkraft oder LiV. Grundlage ist das Verzeichnis von Verarbeitungstätigkeiten
// (docs/datenschutz/vvt_entwurf.md, Tätigkeit 3: Benutzerkonten und Protokollierung des
// Personals). Die Angaben für Schüler (Lernmittelfreiheit, Eltern, LUSD, Abgang und Karenz)
// treffen auf einen Kollegen nicht zu; bis zum 24.09.2026 gab es seine Auskunft gar nicht
// (OFFEN.md 5.19).
//
// Jede Aussage ist am Code nachgesehen (24.09.2026):
//   - Klassenleitung: klassen_lehrer_mapping speist die Mahnliste der Klasse und den Versand
//     der Abgänger-Kontoauszüge (api/graduates_mail.go).
//   - Namen am Anliegen und an der Reservierung sieht das Bibliothekspersonal
//     (repository/anliegen_repo.go, reservation_repo.go); die Kontenliste verlangt
//     manage_users, das Protokoll audit_logs (api/routes_system.go).
//   - Trennen der Ausleihen: PredikatLesehistorieAusleihen fragt nicht nach der Art.
//   - Klassensatz-Reservierungen löscht kein Job; ein gelöschter Kollege bleibt im
//     Papierkorb, weil PredikatAnonymisierung nur art = 'schueler' nimmt.
//
// Die Fristen kommen aus denselben Einstellungen wie bei den Jobs (dsgvoFristen), damit eine
// geänderte Frist hier nicht als Werksvorgabe stehen bleibt.
func dsgvoVerarbeitungsangabenKollegium(f dsgvoFristWerte) DsgvoVerarbeitungsangaben {
	const abgeschaltet = "ohne Frist (Befristung in dieser Installation abgeschaltet)"
	nachRueckgabe := func(tage int) string {
		if tage <= 0 {
			return abgeschaltet
		}
		return fmt.Sprintf("%d Tage nach Rückgabe", tage)
	}
	anliegen := abgeschaltet
	if f.anliegenTage > 0 {
		anliegen = fmt.Sprintf("%d Tage nach der Erledigung", f.anliegenTage)
	}
	return DsgvoVerarbeitungsangaben{
		Zwecke: []string{
			"Zugangskontrolle: Anmeldung mit der dienstlichen E-Mail-Adresse und Rolle im System",
			"Nachvollziehbarkeit von Änderungen an Schülerdaten und Einstellungen (Rechenschaftspflicht nach Art. 5 Abs. 2 DSGVO)",
			"Wünsche, Meldungen und Klassensatz-Reservierungen im Kollegiums-Portal",
			"Versand an die Klassenleitung: Liste der überfälligen Medien der Klasse und Kontoauszüge der Abgänger",
			"Ausleihe von Büchern und Medien an die Person selbst",
		},
		Rechtsgrundlage: "Zugangskonto, Protokolle und Anfragen: Art. 6 Abs. 1 lit. e DSGVO i. V. m. § 83 HSchG; für Beschäftigte § 23 HDSIG (Datenverarbeitung im Beschäftigungsverhältnis). " +
			"Eigene Ausleihen: dieselben Grundlagen wie bei der Ausleihe an Schülerinnen und Schüler. Maßgeblich ist das Verzeichnis von Verarbeitungstätigkeiten der Schule.",
		Empfaenger: "Keine Übermittlung an Dritte. Das Bibliothekspersonal der Schule sieht die eigenen Ausleihen sowie Wünsche, Meldungen und Reservierungen mit dem Namen der anfragenden Person; " +
			"die Liste der Zugangskonten und die Protokolle sehen nur Personen, denen die Schule deren Verwaltung übertragen hat. Helfer an der Theke sehen nur Name, Ausweisnummer und Sperrstatus.",
		Speicherdauer: "Ausleihvorgänge bleiben der Person zugeordnet: Schülerbücherei " + nachRueckgabe(f.lesehistorieTage) + ", Lernmittel " + nachRueckgabe(f.lernmittelTage) + "; danach automatisch getrennt. " +
			"Bearbeitende Person einer Ausleihe nach 14 Tagen entfernt. Erledigte Wünsche und Meldungen: " + anliegen + "; Klassensatz-Reservierungen werden nicht automatisch gelöscht. " +
			"Protokolle " + fmt.Sprintf("%d", f.auditMonate) + " Monate. Leserdatensatz und Zugangskonto bis zum Ausscheiden; gelöscht werden sie von Hand durch die Schule, eine automatische Frist gibt es nicht, auch nicht für einen gelöschten Leserdatensatz im Papierkorb. " +
			"Verschlüsselte Backups 14 Tage.",
		Herkunft:          "Anlage durch die Bibliothek oder die Verwaltung der Zugangskonten, die eigene Anmeldung mit der dienstlichen E-Mail-Adresse oder die Übernahme aus dem bisherigen Bibliotheksprogramm; Protokolleinträge entstehen bei der Arbeit im System",
		Betroffenenrechte: dsgvoBetroffenenrechte,
	}
}
