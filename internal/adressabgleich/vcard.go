package adressabgleich

import (
	"bufio"
	"bytes"
	"strings"
)

// Eintrag ist ein Kontakt des Adressbuchs: Name aus dem Feld N, das Kürzel aus dessen fünftem
// Teil (so pflegt die Schule das Kollegium im globalen Adressbuch) und alle Adressen.
type Eintrag struct {
	Nachname string
	Vorname  string
	Kuerzel  string
	Adressen []string
}

// LeseAdressbuch zerlegt einen vCard-Export (RFC 6350, Roundcube schreibt Fassung 3.0) in
// Einträge. Was es nicht versteht, übergeht es: Ein Eintrag ohne Namen oder ohne Adresse findet
// später kein Konto.
func LeseAdressbuch(vcf []byte) []Eintrag {
	var eintraege []Eintrag
	var aktuell *Eintrag
	for _, zeile := range entfalteZeilen(vcf) {
		name, wert, ok := zerlegeZeile(zeile)
		if !ok {
			continue
		}
		switch {
		case name == "BEGIN" && strings.EqualFold(wert, "VCARD"):
			aktuell = &Eintrag{}
		case name == "END" && strings.EqualFold(wert, "VCARD"):
			if aktuell != nil {
				eintraege = append(eintraege, *aktuell)
			}
			aktuell = nil
		case aktuell == nil:
		case name == "N":
			teile := teileWert(wert)
			aktuell.Nachname, aktuell.Vorname, aktuell.Kuerzel = teil(teile, 0), teil(teile, 1), teil(teile, 4)
		case name == "EMAIL":
			if adresse := strings.TrimSpace(entschaerfe(wert)); adresse != "" {
				aktuell.Adressen = append(aktuell.Adressen, adresse)
			}
		}
	}
	return eintraege
}

// entfalteZeilen setzt gefaltete Zeilen zusammen: Eine Zeile, die mit Leerzeichen oder Tab
// beginnt, setzt die vorige fort (RFC 6350, 3.2).
func entfalteZeilen(vcf []byte) []string {
	var zeilen []string
	scanner := bufio.NewScanner(bytes.NewReader(vcf))
	scanner.Buffer(make([]byte, 64*1024), hoechstGroesse)
	for scanner.Scan() {
		zeile := strings.TrimSuffix(scanner.Text(), "\r")
		if (strings.HasPrefix(zeile, " ") || strings.HasPrefix(zeile, "\t")) && len(zeilen) > 0 {
			zeilen[len(zeilen)-1] += zeile[1:]
			continue
		}
		zeilen = append(zeilen, zeile)
	}
	return zeilen
}

// zerlegeZeile trennt eine Zeile in den Namen der Eigenschaft (groß, ohne Gruppe wie „item1." und
// ohne Parameter) und den Wert. Ein Doppelpunkt in Anführungszeichen gehört zu einem Parameter.
func zerlegeZeile(zeile string) (name, wert string, ok bool) {
	inAnfuehrung := false
	for i, z := range zeile {
		switch {
		case z == '"':
			inAnfuehrung = !inAnfuehrung
		case z == ':' && !inAnfuehrung:
			kopf := zeile[:i]
			if j := strings.IndexByte(kopf, ';'); j >= 0 {
				kopf = kopf[:j]
			}
			if j := strings.LastIndexByte(kopf, '.'); j >= 0 {
				kopf = kopf[j+1:]
			}
			return strings.ToUpper(strings.TrimSpace(kopf)), zeile[i+1:], true
		}
	}
	return "", "", false
}

// teileWert trennt einen Wert aus Teilen (wie N) an jedem Semikolon, vor dem kein Backslash steht,
// und entschärft jeden Teil.
func teileWert(wert string) []string {
	var teile []string
	var aktuell strings.Builder
	maskiert := false
	for _, z := range wert {
		switch {
		case maskiert:
			aktuell.WriteRune('\\')
			aktuell.WriteRune(z)
			maskiert = false
		case z == '\\':
			maskiert = true
		case z == ';':
			teile = append(teile, entschaerfe(aktuell.String()))
			aktuell.Reset()
		default:
			aktuell.WriteRune(z)
		}
	}
	return append(teile, entschaerfe(aktuell.String()))
}

// entschaerfe löst die Maskierung eines Werts auf (\\, \;, \, und \n).
func entschaerfe(wert string) string {
	if !strings.Contains(wert, `\`) {
		return wert
	}
	var b strings.Builder
	maskiert := false
	for _, z := range wert {
		switch {
		case maskiert && (z == 'n' || z == 'N'):
			b.WriteRune('\n')
			maskiert = false
		case maskiert:
			b.WriteRune(z)
			maskiert = false
		case z == '\\':
			maskiert = true
		default:
			b.WriteRune(z)
		}
	}
	return b.String()
}

func teil(teile []string, i int) string {
	if i < len(teile) {
		return strings.TrimSpace(teile[i])
	}
	return ""
}
