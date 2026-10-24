package inventur

import "bibliothek/repository"

// titelEingabe repräsentiert die erwartete JSON-Struktur für das Erstellen oder Aktualisieren eines Buches.
// Beim Aktualisieren zählen nur die Felder, die der Rumpf nennt (leseAenderung): Ein Feld, das
// fehlt, bleibt am Titel, wie es ist.
type titelEingabe struct {
	ISBN          string `json:"isbn"`
	Fach          string `json:"subject"`
	Schulzweig    string `json:"track"`
	IstLernmittel bool   `json:"istLernmittel"`
	// Zeiger, nicht int: "nicht mitgeschickt" muss sich von "null" unterscheiden lassen. Beim
	// Ändern gleicht gleicheBestandAb die Exemplare an diese Zahl an; eine fehlende 0 sonderte sonst
	// den ganzen Bestand aus, im Rückfallzweig auch ausgeliehene Exemplare. `Number(undefined)` im
	// Formular wird zu NaN und in JSON zu null; genau das ist der Weg dorthin.
	Bestand *int `json:"stock"`
	// BestandGesehen: die Zahl, die die Maske beim Öffnen geladen hat. Beim Aktualisieren
	// gilt „stock" nur, wenn sie dem Stand am Server entspricht (setzeBestand) — sonst
	// schriebe eine länger offene Maske den Bestand von vorhin zurück.
	BestandGesehen *int    `json:"stockGesehen"`
	Titel          string  `json:"title"`
	Autor          string  `json:"author"`
	CoverURL       string  `json:"coverUrl"`
	ZaehlDatum     *string `json:"lastCounted"`
	Medientyp      string  `json:"medientyp"`
	JahrgangVon    int     `json:"jahrgangVon"`
	JahrgangBis    int     `json:"jahrgangBis"`
	Mehrjahresband bool    `json:"mehrjahresband"`
	Untertitel     string  `json:"untertitel"`
	// Auflage: Auflagenbezeichnung des Titels. Ohne dieses Feld käme der
	// Wert aus der Maske nie am Repository an — die Tür wäre gebaut und nicht verdrahtet.
	Auflage string `json:"auflage"`
	// Listenpreis aus der Maske. Zeiger aus demselben Grund wie oben:
	// Ein leeres Feld ist „nicht erfasst", eine getippte 0 ist eine Aussage.
	Listenpreis             *float64       `json:"listenpreis"`
	Verlag                  string         `json:"verlag"`
	Erscheinungsjahr        int            `json:"erscheinungsjahr"`
	Signatur                string         `json:"signatur"`
	ErweiterteEigenschaften map[string]any `json:"erweiterteEigenschaften"`
	// Schlagworte: Zeiger aus demselben Grund wie Bestand — „nicht mitgeschickt" (nil)
	// lässt die vorhandenen stehen, `[]` entfernt alle. Ein Aufrufer, der das Feld nicht
	// kennt (Scanner-Neuanlage, ältere Clients), löscht so nichts.
	Schlagworte *[]string `json:"schlagworte"`
	// AnderesMedium gilt nur beim Anlegen ohne ISBN: Die Maske hat nach dem gleichnamigen
	// Titel gefragt, und die Antwort war ein anderes Heft, ein anderer Band.
	AnderesMedium bool `json:"anderesMedium"`
}

// schlagworteAusEingabe prüft und normalisiert die Schlagworte einer Eingabe vor dem
// Schreiben, damit ein Verstoß als 400 mit Text zurückgeht (die Regel selbst steht in
// repository.NormalisiereSchlagworte). nil bleibt nil: nichts gesagt, nichts geändert.
func schlagworteAusEingabe(eingabe *[]string) ([]string, error) {
	if eingabe == nil {
		return nil, nil
	}
	return repository.NormalisiereSchlagworte(*eingabe)
}
