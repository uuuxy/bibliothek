package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bibliothek/internal/testdaten"
	"bibliothek/pkg/schulzeit"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die zwei Hinweise beim Ausgeben eines Schulbuchs gegen echtes Postgres: Wer zählt zur Klasse,
// welche Ausleihe zählt (offen, Schulbuch, Frist im Schuljahr der neuen Ausleihe), und wann die
// Schwelle der Klassensatz-Übersicht erreicht ist. Das steht im SQL und an der Sicht schueler.

// Fristen in zwei Schuljahren: 2026/27 ist das Schuljahr der neuen Ausleihen, 2025/26 das der
// Bücher, die beim Büchertausch noch in der Klasse sind.
var (
	fristSchuljahrNeu = time.Date(2027, 7, 31, 23, 59, 59, 0, schulzeit.Zone())
	fristSchuljahrAlt = time.Date(2026, 7, 31, 23, 59, 59, 0, schulzeit.Zone())
)

// leiheMitFrist legt ein Exemplar von titelID an und verleiht es an schuelerID mit frist;
// zurueck legt die Ausleihe als zurückgegeben an. Geliefert wird die Kennung des Exemplars.
func leiheMitFrist(t *testing.T, pool *pgxpool.Pool, titelID, schuelerID string, frist time.Time, zurueck bool) string {
	t.Helper()
	ctx := context.Background()
	var anzahl int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM buecher_exemplare`).Scan(&anzahl); err != nil {
		t.Fatal(err)
	}
	exemplar := testdaten.LegeExemplarAn(t, pool, titelID, fmt.Sprintf("B-BIK-%03d", anzahl+1), true, "")
	rueckgabe := "NULL"
	if zurueck {
		rueckgabe = "now()"
	}
	if _, err := pool.Exec(ctx, `INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist, rueckgabe_am)
		VALUES ($1, $2, $3, `+rueckgabe+`)`, exemplar, schuelerID, frist); err != nil {
		t.Fatalf("Ausleihe anlegen: %v", err)
	}
	return exemplar
}

func TestFindeBuchFremdInKlasse(t *testing.T) {
	pool := pgTestPool(t)
	resetAuflagen(t, pool)
	ctx := context.Background()

	mathe8 := legeAuflageAn(t, pool, "Mathe 8", 2023, true)
	mathe7 := legeAuflageAn(t, pool, "Mathe 7", 2022, true)
	roman := legeAuflageAn(t, pool, "Roman", 2020, false)

	// Die 8G1 hat zehn Kinder; Ida bekommt Mathe 7, die Ausleihe, zu der der Hinweis gehört.
	ida := legeSchuelerAn(t, pool, "S-BIK-00", "Ida", "8g1")
	var kinder []string
	for i := 1; i <= 9; i++ {
		klasse := "8g1"
		if i == 9 {
			klasse = "08G1" // dieselbe Klasse in der Schreibweise des Vokabulars
		}
		kinder = append(kinder, legeSchuelerAn(t, pool, fmt.Sprintf("S-BIK-%02d", i), fmt.Sprintf("Kind%d", i), klasse))
	}
	leiheMitFrist(t, pool, mathe7, ida, fristSchuljahrNeu, false)

	pruefe := func(stufe string, soll *BuchFremdInKlasse) {
		t.Helper()
		ist, err := FindeBuchFremdInKlasse(ctx, pool, mathe7, ida, fristSchuljahrNeu)
		if err != nil {
			t.Fatalf("%s: %v", stufe, err)
		}
		if (ist == nil) != (soll == nil) || (ist != nil && *ist != *soll) {
			t.Errorf("%s: Hinweis %+v, erwartet %+v", stufe, ist, soll)
		}
	}

	// Vier weitere Kinder haben ihre Bücher fürs Schuljahr: Mit Ida sind es fünf von zehn, nicht
	// mehr als die Hälfte.
	for _, k := range kinder[:4] {
		leiheMitFrist(t, pool, mathe8, k, fristSchuljahrNeu, false)
	}
	pruefe("fünf von zehn", nil)

	// Was nicht zählt: ein Buch der Schülerbücherei, ein zurückgegebenes Schulbuch und ein
	// Schulbuch fürs alte Schuljahr. Keines macht das fünfte Kind zu einem mit Büchern.
	leiheMitFrist(t, pool, roman, kinder[4], fristSchuljahrNeu, false)
	leiheMitFrist(t, pool, mathe8, kinder[4], fristSchuljahrNeu, true)
	leiheMitFrist(t, pool, mathe8, kinder[4], fristSchuljahrAlt, false)
	pruefe("Bücherei, zurückgegeben, altes Schuljahr", nil)

	// Das sechste Kind mit Büchern, in der anderen Schreibweise der Klasse: sechs von zehn.
	leiheMitFrist(t, pool, mathe8, kinder[8], fristSchuljahrNeu, false)
	soll := &BuchFremdInKlasse{Klasse: "08G1", MitBuechern: 6, Kinder: 10}
	pruefe("sechs von zehn", soll)

	// Der Büchertausch: Drei Kinder haben Mathe 7 noch aus dem alten Schuljahr. Das ist nicht das
	// Buch, das die Klasse fürs neue hat; der Hinweis bleibt.
	for _, k := range kinder[5:8] {
		leiheMitFrist(t, pool, mathe7, k, fristSchuljahrAlt, false)
	}
	pruefe("Mathe 7 nur im alten Schuljahr", soll)

	// Wer nicht mehr zur Klasse gehört, zählt nicht, auch mit Mathe 7 fürs neue Schuljahr in der
	// Hand: ein Abgänger und ein gelöschtes Kind. Die Sicht schueler filtert beides nicht.
	gustav := legeSchuelerAn(t, pool, "S-BIK-20", "Gustav", "8g1")
	hanna := legeSchuelerAn(t, pool, "S-BIK-21", "Hanna", "8g1")
	leiheMitFrist(t, pool, mathe7, gustav, fristSchuljahrNeu, false)
	leiheMitFrist(t, pool, mathe7, hanna, fristSchuljahrNeu, false)
	if _, err := pool.Exec(ctx, `UPDATE leser SET ist_abgaenger = true WHERE id = $1`, gustav); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE leser SET deleted_at = now() WHERE id = $1`, hanna); err != nil {
		t.Fatal(err)
	}
	pruefe("Abgänger und gelöschtes Kind", soll)

	// Ein anderes Kind hat Mathe 7 in einer anderen Auflage fürs neue Schuljahr: Es ist dasselbe
	// Buch, kein Hinweis. Die andere Auflage meldet LadeAuflagenMischungInKlasse.
	mathe7Alt := legeAuflageAn(t, pool, "Mathe 7", 2016, true)
	if _, err := FasseAuflagenZusammen(ctx, pool, mathe7Alt, mathe7); err != nil {
		t.Fatal(err)
	}
	auflage := leiheMitFrist(t, pool, mathe7Alt, kinder[0], fristSchuljahrNeu, false)
	pruefe("andere Auflage desselben Buchs", nil)
	if _, err := pool.Exec(ctx, `UPDATE ausleihen SET rueckgabe_am = now() WHERE exemplar_id = $1`, auflage); err != nil {
		t.Fatal(err)
	}
	pruefe("andere Auflage zurückgegeben", soll)

	// Hat ein anderes Kind der Klasse Mathe 7 fürs neue Schuljahr, ist es kein fremdes Buch.
	leiheMitFrist(t, pool, mathe7, kinder[1], fristSchuljahrNeu, false)
	pruefe("ein anderes Kind hat das Buch", nil)
}

// TestFindeBuchFremdInKlasse_Ausnahmen: Oberstufe, Einführungsphase, eine kleine Klasse, ein Kind
// ohne Klasse und ein Kollege bekommen nie den Hinweis, auch wenn mehr als die Hälfte ihre
// Bücher hat.
func TestFindeBuchFremdInKlasse_Ausnahmen(t *testing.T) {
	pool := pgTestPool(t)
	resetAuflagen(t, pool)
	ctx := context.Background()

	mathe := legeAuflageAn(t, pool, "Mathe", 2023, true)
	physik := legeAuflageAn(t, pool, "Physik", 2023, true)

	// Je Klasse sechs Kinder mit Physik und eines mit Mathe: Ohne die Ausnahme wäre das der
	// Hinweis.
	klassenVoll := func(klasse string) string {
		t.Helper()
		for i := 0; i < 6; i++ {
			k := legeSchuelerAn(t, pool, fmt.Sprintf("S-BIK-%s-%d", klasse, i), "Kind", klasse)
			leiheMitFrist(t, pool, physik, k, fristSchuljahrNeu, false)
		}
		k := legeSchuelerAn(t, pool, "S-BIK-"+klasse+"-M", "Mia", klasse)
		leiheMitFrist(t, pool, mathe, k, fristSchuljahrNeu, false)
		return k
	}

	// Die Gegenprobe: In der 9R2 ist es der Hinweis.
	if h, err := FindeBuchFremdInKlasse(ctx, pool, mathe, klassenVoll("9R2"), fristSchuljahrNeu); err != nil || h == nil {
		t.Fatalf("9R2: %+v, %v — erwartet der Hinweis", h, err)
	}
	for _, klasse := range []string{"12T1", "ET1"} {
		if h, err := FindeBuchFremdInKlasse(ctx, pool, mathe, klassenVoll(klasse), fristSchuljahrNeu); err != nil || h != nil {
			t.Errorf("%s: %+v, %v — erwartet kein Hinweis", klasse, h, err)
		}
	}

	// Eine Restklasse mit vier Kindern, drei mit Büchern: mehr als die Hälfte, aber unter der
	// Mindestzahl der Klassensatz-Übersicht.
	klein := legeSchuelerAn(t, pool, "S-BIK-KL-M", "Mia", "10H1")
	leiheMitFrist(t, pool, mathe, klein, fristSchuljahrNeu, false)
	for i := 0; i < 2; i++ {
		k := legeSchuelerAn(t, pool, fmt.Sprintf("S-BIK-KL-%d", i), "Kind", "10H1")
		leiheMitFrist(t, pool, physik, k, fristSchuljahrNeu, false)
	}
	legeSchuelerAn(t, pool, "S-BIK-KL-9", "Kind", "10H1")
	if h, err := FindeBuchFremdInKlasse(ctx, pool, mathe, klein, fristSchuljahrNeu); err != nil || h != nil {
		t.Errorf("10H1 mit vier Kindern: %+v, %v — erwartet kein Hinweis", h, err)
	}

	ohneKlasse := legeSchuelerAn(t, pool, "S-BIK-OK", "Ole", "")
	leiheMitFrist(t, pool, mathe, ohneKlasse, fristSchuljahrNeu, false)
	if h, err := FindeBuchFremdInKlasse(ctx, pool, mathe, ohneKlasse, fristSchuljahrNeu); err != nil || h != nil {
		t.Errorf("ohne Klasse: %+v, %v — erwartet kein Hinweis", h, err)
	}

	// Ein Kollege gehört zu keiner Klasse, auch wenn im Feld eine steht.
	var kollege string
	if err := pool.QueryRow(ctx, `INSERT INTO leser (barcode_id, vorname, nachname, art, klasse)
		VALUES ('L-BIK-01', 'Lena', 'Test', 'lehrkraft', '9R2') RETURNING id`).Scan(&kollege); err != nil {
		t.Fatal(err)
	}
	if h, err := FindeBuchFremdInKlasse(ctx, pool, mathe, kollege, fristSchuljahrNeu); err != nil || h != nil {
		t.Errorf("Kollege: %+v, %v — erwartet kein Hinweis", h, err)
	}
}

func TestHatBuchSchon(t *testing.T) {
	pool := pgTestPool(t)
	resetAuflagen(t, pool)
	ctx := context.Background()

	mathe8 := legeAuflageAn(t, pool, "Mathe 8", 2023, true)
	mathe8Alt := legeAuflageAn(t, pool, "Mathe 8", 2018, true)
	if _, err := FasseAuflagenZusammen(ctx, pool, mathe8Alt, mathe8); err != nil {
		t.Fatal(err)
	}
	physik := legeAuflageAn(t, pool, "Physik", 2023, true)
	ida := legeSchuelerAn(t, pool, "S-BIK-30", "Ida", "8g1")
	ben := legeSchuelerAn(t, pool, "S-BIK-31", "Ben", "8g1")

	pruefe := func(stufe, exemplar string, soll bool) {
		t.Helper()
		ist, err := HatBuchSchon(ctx, pool, exemplar, ida)
		if err != nil {
			t.Fatalf("%s: %v", stufe, err)
		}
		if ist != soll {
			t.Errorf("%s: %v, erwartet %v", stufe, ist, soll)
		}
	}

	neu := leiheMitFrist(t, pool, mathe8, ida, fristSchuljahrNeu, false)
	pruefe("nur die eine Ausleihe", neu, false)

	leiheMitFrist(t, pool, physik, ida, fristSchuljahrNeu, false)
	leiheMitFrist(t, pool, mathe8, ben, fristSchuljahrNeu, false)
	pruefe("anderes Buch, anderes Kind", neu, false)

	leiheMitFrist(t, pool, mathe8, ida, fristSchuljahrAlt, true)
	pruefe("zurückgegebenes Exemplar", neu, false)

	alt := leiheMitFrist(t, pool, mathe8Alt, ida, fristSchuljahrAlt, false)
	pruefe("andere Auflage, altes Schuljahr", neu, true)
	if _, err := pool.Exec(ctx, `UPDATE ausleihen SET rueckgabe_am = now() WHERE exemplar_id = $1`, alt); err != nil {
		t.Fatal(err)
	}
	pruefe("andere Auflage zurückgegeben", neu, false)

	leiheMitFrist(t, pool, mathe8, ida, fristSchuljahrNeu, false)
	pruefe("zweites Exemplar derselben Auflage", neu, true)
}
