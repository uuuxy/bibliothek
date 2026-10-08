package repository

// Was ein Protokolleintrag auf dem Blatt der Auskunft (Art. 15 DSGVO) nennt.
//
// Die Einträge beider Protokolle tragen ihre Angaben in details. Das Blatt nennt jede mit einer
// Bezeichnung (protokollAngaben). Nicht aufs Blatt kommen die Kennungen des Programms, das Konto,
// das gebucht hat, und Merker, die das Blatt in Worten ausschreibt (protokollOhneAngabe). Einen
// Schlüssel, den keine der beiden Listen kennt, druckt das Blatt mit seinem Namen, statt ihn
// wegzulassen. Gate: TestProtokollSchluessel_FuerDieAuskunftEingeordnet.

// ProtokollWertArt sagt, wie das Blatt den Wert eines Schlüssels schreibt.
type ProtokollWertArt int

const (
	// ProtokollText ist ein Text oder eine Zahl ohne eigene Form.
	ProtokollText ProtokollWertArt = iota
	// ProtokollZeit ist ein Tag oder ein Zeitpunkt.
	ProtokollZeit
	// ProtokollBetrag ist ein Betrag in Euro.
	ProtokollBetrag
	// ProtokollJaNein ist ein Wahrheitswert.
	ProtokollJaNein
	// ProtokollAnzahl ist eine Zahl; bei einer Liste die Zahl ihrer Einträge, nie die Einträge.
	ProtokollAnzahl
	// ProtokollRolle ist die Rolle eines Zugangskontos.
	ProtokollRolle
	// ProtokollLeserart ist die Art eines Lesers.
	ProtokollLeserart
	// ProtokollAbschnitt ist ein Objekt mit eigenen Angaben, etwa das Abbild eines Datensatzes.
	ProtokollAbschnitt
)

// ProtokollAngabe ist die Bezeichnung eines Schlüssels auf dem Blatt und die Art seines Werts.
type ProtokollAngabe struct {
	Schluessel  string
	Bezeichnung string
	Art         ProtokollWertArt
}

// protokollAngaben in der Reihenfolge, in der das Blatt sie nennt: erst worum es ging, dann die
// Person, dann Beträge und Gründe, zuletzt Zeiten und Zahlen.
var protokollAngaben = []ProtokollAngabe{
	{"titel", "Titel", ProtokollText},
	{"barcode_id", "Buchnummer", ProtokollText},
	{"ausgeliehen_am", "ausgeliehen am", ProtokollZeit},
	{"vorname", "Vorname", ProtokollText},
	{"nachname", "Nachname", ProtokollText},
	{"schuldner", "Name", ProtokollText},
	{"betrifft", "Name", ProtokollText},
	{"entleiher", "Name", ProtokollText},
	{"klasse", "Klasse", ProtokollText},
	{"art", "Art des Lesers", ProtokollLeserart},
	{"geburtsdatum", "Geburtsdatum", ProtokollZeit},
	{"schul_eintritt_am", "Schuleintritt", ProtokollZeit},
	{"strasse", "Straße", ProtokollText},
	{"hausnummer", "Hausnummer", ProtokollText},
	{"plz", "PLZ", ProtokollText},
	{"ort", "Ort", ProtokollText},
	{"eltern_email", "Eltern-E-Mail", ProtokollText},
	{"email", "E-Mail-Adresse", ProtokollText},
	{"rolle", "Rolle", ProtokollRolle},
	{"aktiv", "Konto aktiv", ProtokollJaNein},
	{"lusd_id", "LUSD-ID", ProtokollText},
	{"lusd_bestaetigt_am", "Zuletzt im LUSD-Export bestätigt", ProtokollZeit},
	{"barcode", "Ausweisnummer", ProtokollText},
	{"aufgeloest_barcode", "Ausweisnummer des zweiten Datensatzes", ProtokollText},
	{"beschreibung", "Beschreibung", ProtokollText},
	{"betrag", "Betrag", ProtokollBetrag},
	{"gesamtbetrag", "Gesamtbetrag", ProtokollBetrag},
	{"referenznummer", "Referenznummer", ProtokollText},
	{"mittel", "Mittel", ProtokollText},
	{"positionen", "Positionen", ProtokollAnzahl},
	{"status", "Status", ProtokollText},
	{"grund", "Grund", ProtokollText},
	{"reason", "Grund", ProtokollText},
	{"block_reason", "Sperrgrund", ProtokollText},
	{"ist_gesperrt", "Gesperrt", ProtokollJaNein},
	{"is_manually_blocked", "Manuell gesperrt", ProtokollJaNein},
	{"von_hand", "Sperre von Hand", ProtokollJaNein},
	{"vom_programm", "Sperre vom Programm", ProtokollJaNein},
	{"ist_abgaenger", "Abgänger", ProtokollJaNein},
	{"abgaenger_seit", "Abgänger seit", ProtokollZeit},
	{"abgaenger_jahr", "Abgangsjahr", ProtokollText},
	{"erstellt_am", "erfasst am", ProtokollZeit},
	{"storniert_am", "storniert am", ProtokollZeit},
	{"geloescht_am", "gelöscht am", ProtokollZeit},
	{"ausleihen", "umgehängte Ausleihen", ProtokollAnzahl},
	{"schaeden", "umgehängte Schadensfälle", ProtokollAnzahl},
	{"schadensfaelle", "umgehängte Schadensfälle", ProtokollAnzahl},
	{"vormerkungen", "umgehängte Vormerkungen", ProtokollAnzahl},
	{"vormerkungen_doppelt_geloescht", "doppelte Vormerkungen gelöscht", ProtokollAnzahl},
	{"bescheide", "umgehängte Bescheide", ProtokollAnzahl},
	{"foto", "Foto übernommen", ProtokollJaNein},
	{"ziel_foto_gewichen", "bisheriges Foto ersetzt", ProtokollJaNein},
	{"platzhalter_konten", "gelöschte Platzhalter-Konten", ProtokollAnzahl},
	{"konten_geloescht", "gelöschte Zugangskonten", ProtokollAnzahl},
	{"leserzeile_geloescht", "Leserdatensatz mitgelöscht", ProtokollJaNein},
	{"leserzeile_bleibt_wegen", "Leserdatensatz blieb wegen", ProtokollText},
	{"quelle", "Zweiter Datensatz", ProtokollAbschnitt},
	{"ziel_vorher", "Dieser Datensatz vor dem Zusammenführen", ProtokollAbschnitt},
	{"gewandert", "Vom zweiten Datensatz", ProtokollAbschnitt},
}

// protokollOhneAngabe: Schlüssel, deren Wert nicht aufs Blatt kommt, mit Grund. Eine Liste und
// keine Map: Eine Map mit dem Schlüssel schueler_id hält die Ratsche der Tilgung für einen
// Protokolleintrag.
var protokollOhneAngabe = []struct{ Schluessel, Grund string }{
	{"schueler_id", "Kennung des Lesers; sie steht als „Interne ID\" bei den Stammdaten"},
	{"ziel_id", "Kennung des Zugangskontos; sie steht als „Interne ID\" beim Konto"},
	{"id", "Kennung der Zeile im Abbild eines Datensatzes"},
	{"aufgeloest_id", "Kennung der zusammengeführten, danach gelöschten Leserzeile"},
	{"exemplar_id", "Kennung des Exemplars; das Blatt nennt Titel und Buchnummer"},
	{"titel_id", "Kennung des Titels; das Blatt nennt den Titel"},
	{"ausleihe_id", "Kennung der Ausleihe"},
	{"schadensfall_id", "Kennung der Forderung"},
	{"bescheid_id", "Kennung des Bescheids; das Blatt nennt die Referenznummer"},
	{"bearbeiter_id", "das Konto, das gebucht hat (Art. 15 Abs. 4 DSGVO, EuGH C-579/21)"},
	{"benutzer_id", "das Konto, das gebucht hat (Art. 15 Abs. 4 DSGVO, EuGH C-579/21)"},
	{"action", "Merker des Vorgangs; das Blatt schreibt ihn in Worten"},
	{"anlass", "Merker des Anlasses; das Blatt schreibt ihn in Worten"},
	{"tabelle", "Name der Tabelle; das Blatt schreibt den Vorgang in Worten"},
	{"zeitpunkt", "Zeitpunkt der Buchung; er steht als Zeitpunkt des Eintrags in derselben Zeile"},
}

var protokollAngabeJeSchluessel = func() map[string]ProtokollAngabe {
	je := make(map[string]ProtokollAngabe, len(protokollAngaben))
	for _, a := range protokollAngaben {
		je[a.Schluessel] = a
	}
	return je
}()

// ProtokollAngaben liefert die Angaben in der Reihenfolge des Blatts.
func ProtokollAngaben() []ProtokollAngabe {
	return protokollAngaben
}

// protokollBezeichnungJeAktion: wo derselbe Schlüssel je Vorgang etwas anderes nennt. Beim
// Aufheben einer Sperre ist grund der Grund der Sperre, nicht der des Aufhebens, und die zwei
// Wahrheitswerte sagen, woher die aufgehobene Sperre kam.
var protokollBezeichnungJeAktion = map[string]map[string]string{
	"LESER_GESPERRT": {"grund": "Sperrgrund"},
	"LESER_ENTSPERRT": {
		"grund":        "Grund der aufgehobenen Sperre",
		"von_hand":     "Sperre war von Hand gesetzt",
		"vom_programm": "Sperre kam vom Programm",
	},
}

// ProtokollAngabeZu liefert Bezeichnung und Art eines Schlüssels im Eintrag mit dieser Aktion.
// amLeser sagt, dass der Eintrag die Leserzeile selbst beschreibt: Dort ist barcode_id die
// Ausweisnummer, in den Spuren eines Buchs die Nummer des Exemplars.
func ProtokollAngabeZu(schluessel string, amLeser bool, aktion string) (ProtokollAngabe, bool) {
	a, bekannt := protokollAngabeJeSchluessel[schluessel]
	if !bekannt {
		return a, false
	}
	if amLeser && schluessel == "barcode_id" {
		a.Bezeichnung = "Ausweisnummer"
	}
	if bezeichnung, eigen := protokollBezeichnungJeAktion[aktion][schluessel]; eigen {
		a.Bezeichnung = bezeichnung
	}
	return a, true
}

var protokollOhneAngabeJeSchluessel = func() map[string]bool {
	je := make(map[string]bool, len(protokollOhneAngabe))
	for _, o := range protokollOhneAngabe {
		je[o.Schluessel] = true
	}
	return je
}()

// ProtokollOhneAngabe sagt, ob der Wert eines Schlüssels nicht aufs Blatt kommt.
func ProtokollOhneAngabe(schluessel string) bool {
	return protokollOhneAngabeJeSchluessel[schluessel]
}
