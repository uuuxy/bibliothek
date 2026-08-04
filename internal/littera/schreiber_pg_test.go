package littera

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// bestand baut einen kleinen Altbestand: n Titel mit je einem Exemplar.
func bestand(titel ...Titel) *Altbestand {
	ab := &Altbestand{
		Verlage:     map[string]string{"1": "Klett"},
		Medienarten: map[string]string{"1": "Buch"},
		Signaturen:  map[string]string{},
	}
	for _, t := range titel {
		ab.Titel = append(ab.Titel, t)
		ab.Exemplare = append(ab.Exemplare, Exemplar{
			ID: "E" + t.ID, Exemplarnummer: "N" + t.ID, TitelID: t.ID,
			Zugangsdatum: "05/03/07", Signatur: "Ea 1 / Xyz",
		})
		ab.Signaturen[t.ID] = "Ea 1 / Xyz"
	}
	return ab
}

func titel(id, name, isbn string) Titel {
	return Titel{ID: id, Haupttitel: name, ISBN: isbn, VerlagID: "1", MedienartID: "1", Erscheinungsjahr: 2007}
}

// Gültige ISBN-13 (Prüfziffer stimmt) — mit einer ungültigen liefe der Testfall an der
// Kollision vorbei, weil KlaereISBN sie vorher verwirft.
const (
	isbnA = "9783161484100"
	isbnB = "9780306406157"
)

// TestBestandUeberlebtEinzelnenZeilenfehler ist der Kern: ein kollidierender Datensatz in
// der Mitte darf die anderen nicht mitreißen — und die übrigen müssen COMMITTET sein.
//
// Der Fall, den das absichert: Postgres versetzt die Transaktion beim ersten Fehler in
// den Abbruchzustand (25P02); ohne Savepoint scheitert danach jedes Statement und der
// COMMIT wird zum ROLLBACK. Das Protokoll meldete dann brav „übersprungen" für Zeilen,
// die in Wahrheit alle verloren waren.
func TestBestandUeberlebtEinzelnenZeilenfehler(t *testing.T) {
	pool := pgTestPool(t)
	leereAlles(t, pool)
	s, protokoll := testSchreiber(t, pool, nil)

	ab := bestand(titel("1", "Erster", ""), titel("2", "Kollidiert", ""), titel("3", "Dritter", ""))
	erzwingeZeilenfehler(t, pool, "buecher_titel", "titel", "Kollidiert")

	bericht, err := s.SchreibeBestand(context.Background(), ab)
	if err != nil {
		t.Fatalf("ein Datensatzfehler darf den Lauf nicht abbrechen: %v", err)
	}

	if bericht.Titel != 2 || bericht.Uebersprungen != 1 {
		t.Errorf("2 übernommen / 1 übersprungen erwartet, gemeldet: %d / %d",
			bericht.Titel, bericht.Uebersprungen)
	}
	// Die Gegenprobe an der Datenbank: die Meldung oben ist wertlos, wenn der COMMIT nicht
	// durchkam. Genau das war der Zustand, den der Savepoint behebt.
	if n := zaehle(t, pool, `SELECT count(*) FROM buecher_titel`); n != 2 {
		t.Errorf("2 Titel müssen committet in der Datenbank stehen, gefunden: %d", n)
	}
	if n := zaehle(t, pool, `SELECT count(*) FROM buecher_titel WHERE titel = 'Kollidiert'`); n != 0 {
		t.Errorf("der kollidierende Titel darf nicht dastehen, gefunden: %d", n)
	}
	if !bericht.AbgleichOK {
		t.Error("der Abgleich mit der Datenbank muss stimmen")
	}
	if text := protokoll(); !strings.Contains(text, "FEHLER") || !strings.Contains(text, "23514") {
		t.Errorf("das Protokoll benennt die Ursache nicht:\n%s", text)
	}
}

// TestTitelKommtGanzOderGarNicht sichert die Wahl der atomaren Einheit ab: buecher_titel.stock
// trägt die Stückzahl. Ein Titel mit stock=3 und nur zwei Exemplaren wäre ein stiller
// Bestandsfehler, den niemand je bemerkt.
func TestTitelKommtGanzOderGarNicht(t *testing.T) {
	pool := pgTestPool(t)
	leereAlles(t, pool)
	s, _ := testSchreiber(t, pool, nil)

	ab := bestand(titel("7", "Dreibändiges Werk", ""))
	ab.Exemplare = append(ab.Exemplare,
		Exemplar{ID: "E7b", Exemplarnummer: "STOLPERT", TitelID: "7"},
		Exemplar{ID: "E7c", Exemplarnummer: "N7c", TitelID: "7"})
	// Das ZWEITE von drei Exemplaren scheitert — der Titel selbst steht dann schon.
	erzwingeZeilenfehler(t, pool, "buecher_exemplare", "barcode_id", "STOLPERT")

	bericht, err := s.SchreibeBestand(context.Background(), ab)
	if err != nil {
		t.Fatalf("ein Exemplarfehler ist ein Datensatzfehler, kein Abbruch: %v", err)
	}
	if bericht.Titel != 0 || bericht.Exemplare != 0 || bericht.Uebersprungen != 1 {
		t.Errorf("der Titel muss vollständig zurückgenommen werden, gemeldet: %+v", bericht)
	}
	if n := zaehle(t, pool, `SELECT count(*) FROM buecher_titel`); n != 0 {
		t.Errorf("kein halb befüllter Titel erlaubt, gefunden: %d", n)
	}
}

// TestStockEntsprichtDenExemplaren: die Zahl in buecher_titel.stock ist der Bestand, den
// die Oberfläche anzeigt. Weicht sie von den tatsächlich geschriebenen Exemplaren ab, sucht
// später jemand Bücher, die es nie gab.
func TestStockEntsprichtDenExemplaren(t *testing.T) {
	pool := pgTestPool(t)
	leereAlles(t, pool)
	s, _ := testSchreiber(t, pool, nil)

	ab := bestand(titel("9", "Klassensatz", ""))
	for i := range 4 {
		ab.Exemplare = append(ab.Exemplare, Exemplar{
			ID: "E9-" + string(rune('a'+i)), Exemplarnummer: "N9-" + string(rune('a'+i)), TitelID: "9"})
	}

	if _, err := s.SchreibeBestand(context.Background(), ab); err != nil {
		t.Fatalf("SchreibeBestand: %v", err)
	}
	var stock, exemplare int
	err := pool.QueryRow(context.Background(), `
		SELECT bt.stock, (SELECT count(*) FROM buecher_exemplare WHERE titel_id = bt.id)
		FROM buecher_titel bt`).Scan(&stock, &exemplare)
	if err != nil {
		t.Fatalf("Abfrage: %v", err)
	}
	if stock != 5 || exemplare != 5 {
		t.Errorf("stock=5 und 5 Exemplare erwartet, gefunden: stock=%d, %d Exemplare", stock, exemplare)
	}
}

// TestDoppelteISBNKostetDieISBNNichtDasBuch: buecher_titel.isbn ist UNIQUE, und im
// Altbestand tragen 1.100 Titel eine doppelt vergebene ISBN. Ohne Abwertung wären das
// 1.100 verlorene Titel samt Exemplaren.
func TestDoppelteISBNKostetDieISBNNichtDasBuch(t *testing.T) {
	pool := pgTestPool(t)
	leereAlles(t, pool)
	s, protokoll := testSchreiber(t, pool, nil)

	ab := bestand(titel("1", "Erstausgabe", isbnA), titel("2", "Nachdruck", isbnA))
	bericht, err := s.SchreibeBestand(context.Background(), ab)
	if err != nil {
		t.Fatalf("SchreibeBestand: %v", err)
	}
	if bericht.Titel != 2 || bericht.Uebersprungen != 0 {
		t.Fatalf("beide Titel müssen ankommen, gemeldet: %+v", bericht)
	}
	if n := zaehle(t, pool, `SELECT count(*) FROM buecher_titel WHERE isbn IS NOT NULL`); n != 1 {
		t.Errorf("genau ein Titel darf die ISBN tragen, gefunden: %d", n)
	}
	if text := protokoll(); !strings.Contains(text, "WARNUNG") || !strings.Contains(text, "doppelte ISBN") {
		t.Errorf("die Abwertung muss als WARNUNG im Protokoll stehen:\n%s", text)
	}
}

// TestVorhandeneISBNKollidiertNicht: läuft der Import in eine Datenbank, in der schon
// Titel stehen (etwa aus dem MAB2-Import), wäre eine dort vergebene ISBN sonst ein 23505
// — und der Littera-Titel ginge samt Exemplaren verloren.
func TestVorhandeneISBNKollidiertNicht(t *testing.T) {
	pool := pgTestPool(t)
	leereAlles(t, pool)
	s, _ := testSchreiber(t, pool, nil)

	if _, err := pool.Exec(context.Background(),
		`INSERT INTO buecher_titel (titel, isbn) VALUES ('Aus dem Bestand', $1)`, isbnB); err != nil {
		t.Fatalf("Vorbedingung: %v", err)
	}

	bericht, err := s.SchreibeBestand(context.Background(), bestand(titel("1", "Littera-Fassung", isbnB)))
	if err != nil {
		t.Fatalf("SchreibeBestand: %v", err)
	}
	if bericht.Titel != 1 || bericht.Uebersprungen != 0 {
		t.Errorf("der Titel muss ohne ISBN ankommen statt verloren zu gehen, gemeldet: %+v", bericht)
	}
	if n := zaehle(t, pool,
		`SELECT count(*) FROM buecher_titel WHERE titel = 'Littera-Fassung' AND isbn IS NULL`); n != 1 {
		t.Errorf("der Littera-Titel soll ohne ISBN dastehen, gefunden: %d", n)
	}
}

// TestBarcodeAusLitteraBleibtErhalten: die Etiketten kleben schon auf 61.520 Büchern.
// Würde der Import neue Nummern vergeben, wäre der gesamte Bestand bis zur Neubeklebung
// nicht scannbar.
func TestBarcodeAusLitteraBleibtErhalten(t *testing.T) {
	pool := pgTestPool(t)
	leereAlles(t, pool)
	s, _ := testSchreiber(t, pool, nil)

	if _, err := s.SchreibeBestand(context.Background(), bestand(titel("1", "Ein Buch", ""))); err != nil {
		t.Fatalf("SchreibeBestand: %v", err)
	}
	var barcode string
	if err := pool.QueryRow(context.Background(),
		`SELECT barcode_id FROM buecher_exemplare`).Scan(&barcode); err != nil {
		t.Fatalf("Abfrage: %v", err)
	}
	if barcode != "N1" {
		t.Errorf("die Littera-Exemplarnummer erwartet, gespeichert: %q", barcode)
	}
}

// TestBarcodeNeuZiehtAusDerSequenz: die Alternative muss aus barcode_seq bedient werden —
// derselben Sequenz, aus der die laufende Anwendung ihre Barcodes zieht. Ein eigener
// Zähler im Werkzeug wäre am Tag nach dem Import eingeholt und kollidierte.
func TestBarcodeNeuZiehtAusDerSequenz(t *testing.T) {
	pool := pgTestPool(t)
	leereAlles(t, pool)
	s, _ := testSchreiber(t, pool, func(o *Optionen) { o.Barcodes = BarcodeNeu })

	if _, err := s.SchreibeBestand(context.Background(), bestand(titel("1", "Ein Buch", ""))); err != nil {
		t.Fatalf("SchreibeBestand: %v", err)
	}
	var barcode string
	if err := pool.QueryRow(context.Background(),
		`SELECT barcode_id FROM buecher_exemplare`).Scan(&barcode); err != nil {
		t.Fatalf("Abfrage: %v", err)
	}
	if !strings.HasPrefix(barcode, "B-") {
		t.Fatalf("B-XXXXX aus der Sequenz erwartet, gespeichert: %q", barcode)
	}
	// Und die eigentliche Zusicherung: die Anwendung darf danach keine belegte Nummer ziehen.
	var naechster string
	if err := pool.QueryRow(context.Background(),
		`SELECT 'B-' || LPAD(nextval('barcode_seq')::TEXT, 5, '0')`).Scan(&naechster); err != nil {
		t.Fatalf("Sequenz: %v", err)
	}
	if naechster == barcode {
		t.Error("die Sequenz gibt dieselbe Nummer erneut aus — sie wurde nicht mitgeführt")
	}
}

// TestDoppelterBarcodeWeichtAus: 13 Exemplare des Altbestands teilen sich 5 Nummern.
// Der Titel darf daran nicht scheitern — bei einem Klassensatz wären das 354 Bücher
// wegen einer doppelten Nummer.
func TestDoppelterBarcodeWeichtAus(t *testing.T) {
	pool := pgTestPool(t)
	leereAlles(t, pool)
	s, protokoll := testSchreiber(t, pool, nil)

	ab := bestand(titel("1", "Ein Buch", ""))
	ab.Exemplare = append(ab.Exemplare,
		Exemplar{ID: "E1b", Exemplarnummer: "N1", TitelID: "1"}) // dieselbe Nummer wie E1

	bericht, err := s.SchreibeBestand(context.Background(), ab)
	if err != nil {
		t.Fatalf("SchreibeBestand: %v", err)
	}
	if bericht.Exemplare != 2 || bericht.Uebersprungen != 0 {
		t.Errorf("beide Exemplare müssen ankommen, gemeldet: %+v", bericht)
	}
	if n := zaehle(t, pool, `SELECT count(*) FROM buecher_exemplare WHERE barcode_id ~ '^B-'`); n != 1 {
		t.Errorf("genau ein Exemplar bekommt eine Ersatznummer, gefunden: %d", n)
	}
	if text := protokoll(); !strings.Contains(text, "Barcode bereits vergeben") {
		t.Errorf("die Ersatzvergabe muss protokolliert werden:\n%s", text)
	}
}

// TestSignaturLandetAmTitel: buecher_titel.signatur ist seit Migration 060 die einzige
// Quelle des Inventur-Scopes, und der schneidet als Präfix am LEERZEICHEN. Steht dort
// nichts, findet keine Inventur ihr Regal.
func TestSignaturLandetAmTitel(t *testing.T) {
	pool := pgTestPool(t)
	leereAlles(t, pool)
	s, _ := testSchreiber(t, pool, nil)

	ab := bestand(titel("1", "Ein Buch", ""))
	ab.Signaturen["1"] = "LMF Deu 7 / Bie"

	if _, err := s.SchreibeBestand(context.Background(), ab); err != nil {
		t.Fatalf("SchreibeBestand: %v", err)
	}
	var sig string
	if err := pool.QueryRow(context.Background(), `SELECT signatur FROM buecher_titel`).Scan(&sig); err != nil {
		t.Fatalf("Abfrage: %v", err)
	}
	if sig != "LMF Deu 7 / Bie" {
		t.Errorf("Signatur erwartet, gespeichert: %q", sig)
	}
	// Die Zusicherung, auf die es ankommt: das Regal findet sein Buch über das Präfix.
	if n := zaehle(t, pool,
		`SELECT count(*) FROM buecher_titel WHERE signatur = $1 OR signatur LIKE $1 || ' %'`,
		"LMF Deu 7"); n != 1 {
		t.Errorf("der Präfix-Scope 'LMF Deu 7' muss den Titel treffen, gefunden: %d", n)
	}
}

// TestWiederholterLaufWirdAbgelehnt: es gibt keinen natürlichen Schlüssel, an dem Postgres
// einen zweiten Import erkennen könnte. Ohne diese Sperre stünde der Bestand danach
// doppelt in der Datenbank.
func TestWiederholterLaufWirdAbgelehnt(t *testing.T) {
	pool := pgTestPool(t)
	leereAlles(t, pool)
	s, _ := testSchreiber(t, pool, nil)
	ctx := context.Background()

	if err := s.PruefeZielbestand(ctx); err != nil {
		t.Fatalf("die leere Datenbank muss durchgehen: %v", err)
	}
	if _, err := s.SchreibeBestand(ctx, bestand(titel("1", "Ein Buch", ""))); err != nil {
		t.Fatalf("SchreibeBestand: %v", err)
	}
	err := s.PruefeZielbestand(ctx)
	if err == nil {
		t.Fatal("nach einem Lauf muss der zweite abgelehnt werden")
	}
	if !strings.Contains(err.Error(), "früheren Littera-Import") {
		t.Errorf("die Meldung muss die Ursache benennen: %v", err)
	}
}

// erzwingeZeilenfehler lässt Postgres bei einem bestimmten Spaltenwert eine
// Integritätsverletzung werfen.
//
// Nötig, weil die Vorläufe des Schreibers (Barcodes, ISBN) genau die Kollisionen abfangen,
// die man sonst zum Provozieren nähme — sie kommen gar nicht erst bis zum INSERT. Was hier
// geprüft werden soll, liegt aber dahinter: dass ein Fehler MITTEN in der Transaktion nur
// diesen einen Datensatz kostet. SQLSTATE 23514 ist bewusst gewählt: Es ist ein
// Zeilenfehler im Sinne von uebernahme.IstZeilenfehler, wie ihn jeder CHECK und jeder
// UNIQUE-Index liefert, den eine spätere Migration hinzufügen mag.
func erzwingeZeilenfehler(t *testing.T, pool *pgxpool.Pool, tabelle, spalte, wert string) {
	t.Helper()
	ctx := context.Background()
	name := "test_zeilenfehler_" + tabelle
	ddl := fmt.Sprintf(`
		CREATE OR REPLACE FUNCTION %[1]s() RETURNS trigger AS $$
		BEGIN
			IF NEW.%[2]s = %[3]s THEN
				RAISE EXCEPTION 'Testfehler auf %[2]s' USING ERRCODE = '23514';
			END IF;
			RETURN NEW;
		END; $$ LANGUAGE plpgsql;
		CREATE TRIGGER %[1]s BEFORE INSERT ON %[4]s
		FOR EACH ROW EXECUTE FUNCTION %[1]s();`,
		name, spalte, quoteLiteral(wert), tabelle)
	if _, err := pool.Exec(ctx, ddl); err != nil {
		t.Fatalf("Trigger anlegen: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, fmt.Sprintf( //nolint:errcheck // Testaufräumen
			`DROP TRIGGER IF EXISTS %[1]s ON %[2]s; DROP FUNCTION IF EXISTS %[1]s();`, name, tabelle))
	})
}

func quoteLiteral(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
