package main

// stilBestand nennt je Paket, was docs/STIL.md noch nicht folgt. Ein Paket ohne Zeile ist
// fertig; die Zahlen sinken mit jedem umgestellten Paket und steigen nie.
var stilBestand = map[string]stilZahl{
	".":                      {Fach: 1, Verb: 4, Kuerzel: 0, Datei: 0, Datum: 4},
	"api":                    {Fach: 14, Verb: 15, Kuerzel: 0, Datei: 4, Datum: 25},
	"api/katalog":            {Fach: 68, Verb: 43, Kuerzel: 2, Datei: 15, Datum: 17},
	"api/leser":              {Fach: 36, Verb: 22, Kuerzel: 12, Datei: 12, Datum: 23},
	"apierrors":              {Fach: 3, Verb: 5, Kuerzel: 0, Datei: 0, Datum: 2},
	"auth":                   {Fach: 30, Verb: 10, Kuerzel: 0, Datei: 3, Datum: 13},
	"cmd/littera-altbestand": {Fach: 0, Verb: 0, Kuerzel: 0, Datei: 0, Datum: 1},
	"cmd/littera-import":     {Fach: 0, Verb: 0, Kuerzel: 0, Datei: 0, Datum: 1},
	"cmd/migrate":            {Fach: 1, Verb: 10, Kuerzel: 0, Datei: 0, Datum: 0},
	"cmd/seed":               {Fach: 2, Verb: 4, Kuerzel: 0, Datei: 0, Datum: 1},
	"cmd/stresstest":         {Fach: 4, Verb: 2, Kuerzel: 0, Datei: 0, Datum: 0},
	"db":                     {Fach: 5, Verb: 5, Kuerzel: 0, Datei: 0, Datum: 13},
	"inventur":               {Fach: 59, Verb: 48, Kuerzel: 1, Datei: 9, Datum: 45},
	"jobs":                   {Fach: 4, Verb: 16, Kuerzel: 0, Datei: 1, Datum: 25},
	"mailservice":            {Fach: 3, Verb: 2, Kuerzel: 0, Datei: 0, Datum: 5},
	"pdf":                    {Fach: 16, Verb: 26, Kuerzel: 3, Datei: 0, Datum: 8},
	"sse":                    {Fach: 1, Verb: 0, Kuerzel: 0, Datei: 0, Datum: 0},
}
