package coverquelle

import "testing"

// Der Neuaufbau nimmt Schema und Port aus der Eingabe heraus. Ohne ihn ginge die rohe
// Zeichenkette an den HTTP-Client, und ein zweiter Parser kann sie anders lesen als der
// prüfende.
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
			// Ohne Neuaufbau ginge die Anfrage an covers.openlibrary.org:2222: der Hostname
			// erlaubt, das Ziel ein anderer Dienst.
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
			"erlaubter Name als Subdomain des Angreifers",
			"https://covers.openlibrary.org.angreifer.example/x.jpg", "", false,
		},
		{
			// Alles vor dem @ ist Userinfo, der Host steht dahinter.
			"erlaubter Name als Userinfo vor fremdem Host",
			"https://covers.openlibrary.org@angreifer.example/x.jpg", "", false,
		},
		{"IP statt Hostname", "https://127.0.0.1/x.jpg", "", false},
		{"leerer String", "", "", false},
		{"kaputte URL", "https://%zz/x.jpg", "", false},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			got, ok := SichereURL(f.eingabe, BildHosts)
			if ok != f.ok {
				t.Fatalf("SichereURL(%q) ok = %v; want %v", f.eingabe, ok, f.ok)
			}
			if got != f.will {
				t.Errorf("SichereURL(%q) = %q; want %q", f.eingabe, got, f.will)
			}
		})
	}
}

// erlaubt sagt, ob SichereURL die Adresse annimmt.
func erlaubt(rohURL string, erlaubte []string) bool {
	_, ok := SichereURL(rohURL, erlaubte)
	return ok
}

// Die Liste für Metadaten ist mit Absicht enger als die für Bilder. Der Test hält den
// Unterschied fest, damit ihn niemand beim Aufräumen angleicht.
func TestMetadatenHostsSindEngerAlsBildHosts(t *testing.T) {
	// Von Google Books holt das Paket inventur nur Bilder, keine Datensätze.
	const googleBooks = "https://books.google.com/books/content?id=abc"
	if !erlaubt(googleBooks, BildHosts) {
		t.Error("books.google.com muss für Cover erlaubt sein")
	}
	if erlaubt(googleBooks, MetadatenHosts) {
		t.Error("books.google.com darf für Metadaten nicht erlaubt sein")
	}

	// Jeder Host für Metadaten darf auch Bilder liefern. Sonst wäre die engere Liste nicht
	// enger, sondern anders, und das wäre ein Versehen.
	for _, h := range MetadatenHosts {
		if !erlaubt("https://"+h+"/x", BildHosts) {
			t.Errorf("Metadaten-Host %q fehlt in BildHosts: Die Listen sind auseinandergelaufen", h)
		}
	}
}

// lobid.org steht in keiner der beiden Listen, und kein Programmcode fragt dort an. Wer die
// Quelle anschließt, trägt sie hier ein und baut den Abruf; eines allein genügt nicht.
func TestLobidIstNichtFreigegeben(t *testing.T) {
	const lobid = "https://lobid.org/resources/search?q=isbn:123"
	if erlaubt(lobid, MetadatenHosts) || erlaubt(lobid, BildHosts) {
		t.Error("lobid.org ist freigegeben, aber kein Programmcode fragt dort an")
	}
}
