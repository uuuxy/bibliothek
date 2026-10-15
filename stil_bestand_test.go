package main

// stilBestand nennt je Paket, was docs/STIL.md noch nicht folgt. Ein Paket ohne Zeile ist
// fertig; die Zahlen sinken mit jedem umgestellten Paket und steigen nie.
var stilBestand = map[string]stilZahl{
	".":        {Fach: 1, Verb: 4, Kuerzel: 0, Datei: 0, Datum: 4},
	"inventur": {Fach: 59, Verb: 48, Kuerzel: 1, Datei: 9, Datum: 45},
}
