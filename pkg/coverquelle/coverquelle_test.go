package coverquelle

import "testing"

// Was der Neuaufbau ÜBER die reine Hostnamen-Prüfung hinaus leistet: Er nimmt Schema
// und Port aus der Eingabe heraus. Ohne ihn geht genau der rohe String los, den ein
// zweiter Parser anders lesen kann als der prüfende.
func TestSichereURL(t *testing.T) {
	faelle := []struct {
		name    string
		eingabe string
		will    string
		ok      bool
	}{
		{
			"erlaubter Host bleibt inhaltlich unverändert",
			"https://covers.openlibrary.org/b/isbn/9783161484100-M.jpg",
			"https://covers.openlibrary.org/b/isbn/9783161484100-M.jpg", true,
		},
		{
			"http wird auf https gehoben, Query bleibt",
			"http://books.google.com/books/content?id=abc&printsec=frontcover",
			"https://books.google.com/books/content?id=abc&printsec=frontcover", true,
		},
		{
			// Der Port ist der greifbarste Gewinn: Ohne Neuaufbau ginge die Anfrage an
			// covers.openlibrary.org:2222 — Hostname erlaubt, Ziel ein anderer Dienst.
			"abweichender Port wird verworfen",
			"https://covers.openlibrary.org:2222/b/isbn/123-M.jpg",
			"https://covers.openlibrary.org/b/isbn/123-M.jpg", true,
		},
		{
			"leerer Pfad wird zu /",
			"https://openlibrary.org",
			"https://openlibrary.org/", true,
		},
		{"fremder Host", "https://angreifer.example/x.jpg", "", false},
		{
			"Allowlist-Name als Subdomain des Angreifers",
			"https://covers.openlibrary.org.angreifer.example/x.jpg", "", false,
		},
		{
			// Klassiker: Alles vor dem @ ist Userinfo, der echte Host steht dahinter.
			"Allowlist-Name als Userinfo vor fremdem Host",
			"https://covers.openlibrary.org@angreifer.example/x.jpg", "", false,
		},
		{"IP statt Hostname", "https://127.0.0.1/x.jpg", "", false},
		{"leerer String", "", "", false},
		{"kaputte URL", "https://%zz/x.jpg", "", false},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			got, ok := SichereURL(f.eingabe, CoverHosts)
			if ok != f.ok {
				t.Fatalf("SichereURL(%q) ok = %v; want %v", f.eingabe, ok, f.ok)
			}
			if got != f.will {
				t.Errorf("SichereURL(%q) = %q; want %q", f.eingabe, got, f.will)
			}
		})
	}
}

// Die Metadaten-Liste ist bewusst enger als die Cover-Liste. Der Test hält den
// Unterschied fest, damit er nicht beim nächsten Aufräumen "vereinheitlicht" wird.
func TestMetadatenHostsSindEngerAlsCoverHosts(t *testing.T) {
	// Von Google Books holt das Inventur-Modul nur Bilder, keine Datensätze.
	const googleBooks = "https://books.google.com/books/content?id=abc"
	if !IstErlaubt(googleBooks, CoverHosts) {
		t.Error("books.google.com muss für Cover erlaubt sein")
	}
	if IstErlaubt(googleBooks, MetadatenHosts) {
		t.Error("books.google.com darf für Metadaten NICHT erlaubt sein")
	}

	// Jeder Metadaten-Host muss auch Cover liefern dürfen — sonst ist die engere Liste
	// nicht enger, sondern schlicht anders, und das wäre ein Versehen.
	for _, h := range MetadatenHosts {
		if !IstErlaubt("https://"+h+"/x", CoverHosts) {
			t.Errorf("Metadaten-Host %q fehlt in CoverHosts — die Listen sind auseinandergelaufen", h)
		}
	}
}

// lobid.org steht in KEINER der beiden Listen. Das ist kein Versehen, sondern der
// dokumentierte Ist-Zustand: inventur.sucheLobid hat keinen Aufrufer, und selbst mit
// einem käme die Anfrage hier nicht durch. Wer die Quelle aktivieren will, muss beides
// anfassen — dieser Test macht sichtbar, dass ein Eintrag hier allein nicht reicht.
func TestLobidIstNichtFreigegeben(t *testing.T) {
	const lobid = "https://lobid.org/resources/search?q=isbn:123"
	if IstErlaubt(lobid, MetadatenHosts) || IstErlaubt(lobid, CoverHosts) {
		t.Error("lobid.org ist freigegeben — dann gehört auch der tote Aufrufpfad sucheLobid angeschlossen")
	}
}
