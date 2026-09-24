package api

import (
	"fmt"
	"strings"

	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"

	"github.com/jung-kurt/gofpdf"
)

// dsgvoKontoAbschnitt druckt das Zugangskonto samt Klassenleitungen, Anfragen und den
// Einträgen des Verwaltungsprotokolls über das Konto (repository/dsgvo_konto.go). Seit dem
// 24.09.2026 Teil der Auskunft, weil es sie seither für jeden Leser gibt (OFFEN.md 5.19).
// Bei Schülern zeigt in der Regel kein Konto auf den Leser; dann steht das hier in einem Satz.
func dsgvoKontoAbschnitt(p *gofpdf.Fpdf, tr func(string) string, k *repository.DsgvoZugangskonto) {
	dsgvoAbschnitt(p, tr, "10. Zugangskonto")
	if k == nil {
		dsgvoHinweis(p, tr, "Auf diesen Leserdatensatz zeigt kein Zugangskonto.")
		return
	}
	dsgvoZeile(p, tr, "Interne ID", k.ID)
	dsgvoZeile(p, tr, "E-Mail-Adresse", k.Email)
	dsgvoZeile(p, tr, "Vorname", k.Vorname)
	dsgvoZeile(p, tr, "Nachname", k.Nachname)
	dsgvoZeile(p, tr, "Rolle", dsgvoRolle(k.Rolle))
	dsgvoZeile(p, tr, "Aktiv", dsgvoJaNein(k.Aktiv))
	dsgvoZeile(p, tr, "Zugang beantragt am", dsgvoZeitPtr(k.ZugangBeantragtAm))
	dsgvoZeile(p, tr, "Angelegt am", dsgvoZeit(k.ErstelltAm))
	dsgvoZeile(p, tr, "Zuletzt geändert", dsgvoZeit(k.AktualisiertAm))
	dsgvoZeile(p, tr, "Klassenleitung", strings.Join(k.Klassenleitungen, ", "))

	dsgvoUnterabschnitt(p, tr, fmt.Sprintf("Wünsche, Meldungen und Reservierungen im Kollegiums-Portal (%d)", len(k.Anfragen)))
	if len(k.Anfragen) == 0 {
		dsgvoLeer(p, tr)
	}
	for _, a := range k.Anfragen {
		teile := []string{dsgvoAnfrageArt(a.Art)}
		if a.ISBN != "" {
			teile = append(teile, "ISBN: "+a.ISBN)
		}
		if a.Klasse != "" {
			teile = append(teile, "Klasse: "+a.Klasse)
		}
		if a.Anzahl > 0 {
			teile = append(teile, fmt.Sprintf("Anzahl: %d", a.Anzahl))
		}
		teile = append(teile, "gestellt: "+dsgvoDatum(a.ErstelltAm))
		switch {
		case a.ErledigtAm != nil:
			teile = append(teile, "erledigt: "+dsgvoDatum(*a.ErledigtAm))
		case a.Erledigt:
			teile = append(teile, "erledigt")
		default:
			teile = append(teile, "offen")
		}
		if a.Kommentar != "" {
			teile = append(teile, "Kommentar: "+a.Kommentar)
		}
		if a.ErledigtNotiz != "" {
			teile = append(teile, "Notiz der Bibliothek: "+a.ErledigtNotiz)
		}
		dsgvoEintragTitel(p, tr, a.Titel)
		dsgvoEintragZeile(p, tr, strings.Join(teile, " · "))
	}

	dsgvoUnterabschnitt(p, tr, fmt.Sprintf("Einträge im Verwaltungsprotokoll zu diesem Konto (%d)", len(k.Ereignisse)))
	if len(k.Ereignisse) == 0 {
		dsgvoLeer(p, tr)
	}
	p.SetFont("Arial", "", 8)
	for _, e := range k.Ereignisse {
		p.MultiCell(0, 5, tr(dsgvoZeit(e.Zeitpunkt)+" — "+dsgvoKontoAktion(e.Aktion)), "", "L", false)
	}

	dsgvoUnterabschnitt(p, tr, fmt.Sprintf("Vorgänge, die diese Person im System bearbeitet hat (%d)", len(k.EigeneVorgaenge)))
	dsgvoHinweis(p, tr, "Ohne Angaben zu anderen Personen (Art. 15 Abs. 4 DSGVO). Eine Buchung kann zweimal "+
		"stehen: als Ausleihvorgang, solange die bearbeitende Person daran gespeichert ist (bis 14 Tage nach "+
		"der Rückgabe), und als Protokolleintrag.")
	if len(k.EigeneVorgaenge) == 0 {
		dsgvoLeer(p, tr)
	}
	p.SetFont("Arial", "", 8)
	for _, z := range dsgvoVorgaengeJeTag(k.EigeneVorgaenge) {
		p.MultiCell(0, 5, tr(fmt.Sprintf("%s · %s (%d): %s", z.tag, z.handlung, len(z.zeiten),
			strings.Join(z.zeiten, ", "))), "", "L", false)
	}
}

// dsgvoVorgangsZeile fasst die Vorgänge eines Tages mit derselben Handlung zusammen.
type dsgvoVorgangsZeile struct {
	tag, handlung string
	zeiten        []string
}

// dsgvoVorgaengeJeTag fasst die selbst bearbeiteten Vorgänge je Tag und Handlung zusammen und
// behält jede Uhrzeit. Bei einer Bibliothekskraft sind es Tausende Vorgänge; eine Zeile je
// Vorgang wären Hunderte Seiten. Die Reihenfolge folgt der Abfrage (neueste zuerst); innerhalb
// eines Tages stehen die Handlungen in der Reihenfolge, in der sie zuerst vorkommen.
func dsgvoVorgaengeJeTag(vorgaenge []repository.DsgvoEigenerVorgang) []dsgvoVorgangsZeile {
	var zeilen []dsgvoVorgangsZeile
	index := map[[2]string]int{}
	for _, v := range vorgaenge {
		tag, zeit := "ohne gespeicherten Zeitpunkt", "—"
		if v.Zeitpunkt != nil {
			lokal := v.Zeitpunkt.In(schulzeit.Zone())
			tag, zeit = lokal.Format(dsgvoDatumFormat), lokal.Format("15:04")
		}
		if v.IPAdresse != nil && *v.IPAdresse != "" {
			zeit += " (IP " + *v.IPAdresse + ")"
		}
		schluessel := [2]string{tag, v.Handlung}
		i, gibtEs := index[schluessel]
		if !gibtEs {
			i = len(zeilen)
			index[schluessel] = i
			zeilen = append(zeilen, dsgvoVorgangsZeile{tag: tag, handlung: v.Handlung})
		}
		zeilen[i].zeiten = append(zeilen[i].zeiten, zeit)
	}
	return zeilen
}

// dsgvoUnterabschnitt ist eine Zwischenüberschrift innerhalb eines Abschnitts.
func dsgvoUnterabschnitt(p *gofpdf.Fpdf, tr func(string) string, titel string) {
	p.Ln(3)
	p.SetFont("Arial", "B", 10)
	p.MultiCell(0, 6, tr(titel), "", "L", false)
	p.Ln(1)
}

// dsgvoHinweis ist ein Satz in der Form von „Keine Einträge vorhanden.".
func dsgvoHinweis(p *gofpdf.Fpdf, tr func(string) string, satz string) {
	p.SetFont("Arial", "I", 9)
	p.SetTextColor(120, 120, 120)
	p.MultiCell(0, 5, tr(satz), "", "L", false)
	p.SetTextColor(0, 0, 0)
}

// dsgvoRolle schreibt die Rolle eines Kontos mit denselben Wörtern wie die
// Benutzerverwaltung (UserManagementEditModal.svelte). Ein unbekannter Wert bleibt stehen.
func dsgvoRolle(rolle string) string {
	switch rolle {
	case "kollegium":
		return "Kollegium (keine weitere Rolle)"
	case "helfer":
		return "Helfer"
	case "mitarbeiter":
		return "Mitarbeiter"
	case "leitung":
		return "Leitung"
	case "admin":
		return "Administrator"
	}
	return rolle
}

// dsgvoAnfrageArt schreibt die Art einer Anfrage wie das Kollegiums-Portal
// (AnliegenWidget.svelte). Ein unbekannter Wert bleibt stehen.
func dsgvoAnfrageArt(art string) string {
	switch art {
	case "wunsch":
		return "Wunsch"
	case "meldung":
		return "Meldung"
	case "klassensatz":
		return "Klassensatz-Reservierung"
	}
	return art
}

// dsgvoKontoAktion schreibt die Aktion eines Kontoereignisses aus (Schreiber:
// api/user_admin_mutations.go, auth/selbstanmeldung.go). Ein unbekannter Wert bleibt stehen.
func dsgvoKontoAktion(aktion string) string {
	switch aktion {
	case "USER_CREATE":
		return "Konto angelegt"
	case "USER_UPDATE":
		return "Konto geändert"
	case "SELBSTANMELDUNG":
		return "Zugang über die eigene Anmeldung beantragt"
	}
	return aktion
}
