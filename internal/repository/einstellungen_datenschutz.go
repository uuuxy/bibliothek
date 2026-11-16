package repository

import "strconv"

// Fristen des Datenschutzes und der Sitzung.
//
// Alle Werte sind Zeiger wie OeffentlicheAdresse: nil heißt „diese Kategorie kennt das Feld
// nicht" (der gespeicherte Wert bleibt), ein gesetzter Zeiger heißt „so speichern", auch 0.
// Die 0 ist hier ein Wert (aus) und kein fehlender: Ohne Zeiger schaltete jedes Speichern
// einer anderen Kategorie die Befristung ab.

const (
	// LesehistorieTageVorgabe sind die Tage nach der Rückgabe, nach denen eine Ausleihe der
	// Schülerbücherei (Freihand, Medien, Geräte) vom Leser getrennt wird: einer. Der nächtliche
	// Lauf trennt sie, sobald ein voller Tag vergangen ist (KulanzJob). So empfiehlt es Hessen:
	// Das Muster „Schulbibliothek" des HBDI verlangt, dass die Daten „unverzüglich gelöscht
	// [werden], sofern sie für die Zwecke … nicht mehr notwendig sind"; das Muster der LAG
	// Schulbibliotheken in Hessen nennt für die Ausleihdaten „nach Rückgabe des Mediums
	// Löschung beim Ausleiher" und rät, keine historischen Verleihdaten zu speichern („Verbot
	// der Möglichkeit, ein ‚Leseprofil' eines Lesers zu erstellen"). Nachfragen zu einem später
	// bemerkten Schaden sind damit nach der Rückgabe nicht mehr möglich; eine Ausleihe mit
	// offener Forderung bleibt zugeordnet.
	LesehistorieTageVorgabe = 1
	// LesehistorieLernmittelTageVorgabe ist die Frist für Lernmittel: zwei Jahre nach der
	// Rückgabe. Lernmittel bleiben länger zuordenbar, weil die Bestandskartei Ausleihe und
	// Rücklauf nachweisen muss (HKM-Leitfaden LMF 11.3) und Schadensersatz
	// öffentlich-rechtlich über die Schulaufsicht läuft (12.3 bis 12.7); das zieht sich über
	// das folgende Schuljahr.
	LesehistorieLernmittelTageVorgabe = 730
	// AnliegenTageVorgabe sind die Tage nach der Erledigung, nach denen ein Wunsch oder eine
	// Meldung aus dem Portal des Kollegiums gelöscht wird. Die Zeile trägt den Freitext der
	// Lehrkraft, ihre Klasse und über den Fremdschlüssel ihren Namen; erledigt ist ihr Zweck
	// erreicht. Ein Jahr deckt den Blick zurück über ein Schuljahr („war der Titel schon
	// einmal gewünscht?"). 0 schaltet ab.
	AnliegenTageVorgabe = 365
	// ThekeLeerenMinutenVorgabe ist die Zeit ohne Eingabe, nach der die Theke den geladenen
	// Leser fallen lässt, damit der nächste an der Theke nicht den vorigen sieht.
	ThekeLeerenMinutenVorgabe = 5
	// SperreMinutenVorgabe ist die Zeit ohne Eingabe, nach der der Sperrbildschirm kommt.
	SperreMinutenVorgabe = 15
)

func zeiger(n int) *int { return &n }

// setzeDatenschutzVorgaben füllt die Vorgaben der Fristen.
func setzeDatenschutzVorgaben(e *SystemEinstellungen) {
	e.LesehistorieTage = zeiger(LesehistorieTageVorgabe)
	e.LesehistorieLernmittelTage = zeiger(LesehistorieLernmittelTageVorgabe)
	e.AnliegenTage = zeiger(AnliegenTageVorgabe)
	e.ProtokollAufbewahrungMonate = zeiger(ProtokollAufbewahrungMonateVorgabe)
	e.ThekeLeerenMinuten = zeiger(ThekeLeerenMinutenVorgabe)
	e.SperreMinuten = zeiger(SperreMinutenVorgabe)
	e.AbgaengerKarenzTage = zeiger(AbgaengerKarenzTageVorgabe)
}

// setzeIntZeiger übernimmt eine gespeicherte Zahl in ein Zeigerfeld. Ein unlesbarer Wert lässt
// die Vorgabe stehen, ein negativer zählt als 0 (aus).
func setzeIntZeiger(wert *string, ziel **int) {
	if wert == nil || *wert == "" {
		return
	}
	n, err := strconv.Atoi(*wert)
	if err != nil {
		return
	}
	if n < 0 {
		n = 0
	}
	*ziel = zeiger(n)
}

// setzeIntZeigerMitErsatz arbeitet wie setzeIntZeiger, macht aus einem negativen Wert aber den
// Ersatz statt der 0: für Schlüssel, bei denen 0 der schärfste Wert ist.
func setzeIntZeigerMitErsatz(wert *string, ziel **int, ersatz int) {
	if wert == nil || *wert == "" {
		return
	}
	n, err := strconv.Atoi(*wert)
	if err != nil {
		return
	}
	if n < 0 {
		n = ersatz
	}
	*ziel = zeiger(n)
}

// setzeDatenschutzEinstellung überträgt eine Zeile in ihr Feld und meldet, ob der Schlüssel zu
// diesen Fristen gehört.
func setzeDatenschutzEinstellung(e *SystemEinstellungen, schluessel string, wert *string) bool {
	switch schluessel {
	case "lesehistorie_tage":
		setzeIntZeiger(wert, &e.LesehistorieTage)
	case "lesehistorie_lernmittel_tage":
		setzeIntZeiger(wert, &e.LesehistorieLernmittelTage)
	case "anliegen_tage":
		setzeIntZeiger(wert, &e.AnliegenTage)
	case ProtokollAufbewahrungSchluessel:
		setzeIntZeiger(wert, &e.ProtokollAufbewahrungMonate)
	case "theke_leeren_minuten":
		setzeIntZeiger(wert, &e.ThekeLeerenMinuten)
	case "sperre_minuten":
		setzeIntZeiger(wert, &e.SperreMinuten)
	case AbgaengerKarenzSchluessel:
		// Ein negativer Wert wird zur Vorgabe, nie zur 0: 0 heißt sofort anonymisieren.
		setzeIntZeigerMitErsatz(wert, &e.AbgaengerKarenzTage, AbgaengerKarenzTageVorgabe)
	default:
		return false
	}
	return true
}

// TageOderVorgabe liefert den Wert eines Zeigerfelds oder die Vorgabe, wenn es nil ist.
func TageOderVorgabe(v *int, vorgabe int) int {
	if v == nil {
		return vorgabe
	}
	return *v
}

// AbgaengerKarenzTageOderVorgabe liefert die Karenz vor der Anonymisierung aus den
// Einstellungen oder die Vorgabe, wenn sie nicht lesbar sind. Import, nächtlicher Lauf und
// Selbstprüfung haben so denselben Rückfall.
func AbgaengerKarenzTageOderVorgabe(e *SystemEinstellungen) int {
	if e == nil {
		return AbgaengerKarenzTageVorgabe
	}
	return TageOderVorgabe(e.AbgaengerKarenzTage, AbgaengerKarenzTageVorgabe)
}
