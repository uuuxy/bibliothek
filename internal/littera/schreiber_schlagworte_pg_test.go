package littera

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Littera-Schlagworte kommen mit der Übernahme (docs/OFFEN.md 4.20, Stufe 2, entschieden am
// 30.09.2026), über den einen Schreibpfad der Anwendung, und das Fach kommt aus ihnen, wo die
// Signatur keins nennt — wie beim Katalogisat-Import. Am echten Postgres: Was der Bericht
// meldet, muss in titel_schlagworte stehen.
func TestSchreibeSchlagworte_AlleWoerterUeberDenSchreibpfad(t *testing.T) {
	pool := pgTestPool(t)
	leereAlles(t, pool)
	s, protokoll := testSchreiber(t, pool, nil)
	ctx := context.Background()

	ab := bestand(titel("1", "Die Republik von Weimar", ""), titel("2", "Regenwald", ""),
		titel("3", "Pflanzen und Umwelt", ""), titel("4", "Biologie heute 7", ""))
	ab.Signaturen["4"] = "LMF Bio 7"
	viele := make([]string, repository.SchlagworteJeTitelMax+1)
	for i := range viele {
		viele[i] = fmt.Sprintf("Pflanze %03d", i)
	}
	ab.Schlagworte = SchlagwortQuelle{JeTitel: map[string][]string{
		"1":  {"Geschichte", " Weimarer   Republik ", ""},
		"2":  {"Brasilien", "brasilien", "Regenwald", strings.Repeat("x", repository.SchlagwortMaxZeichen+1)},
		"3":  viele,
		"4":  {"Geschichte", "Schulbuch"},
		"99": {"Titel fehlt"},
	}}

	bestandBericht, err := s.SchreibeBestand(ctx, ab)
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.SchreibeSchlagworte(ctx, ab, bestandBericht)
	if err != nil {
		t.Fatal(err)
	}

	for id, will := range map[string][]string{
		"1": {"Geschichte", "Weimarer Republik"},
		"2": {"Brasilien", "Regenwald"},
		"4": {"Geschichte", "Schulbuch"},
	} {
		if got := schlagworteAm(t, pool, bestandBericht.TitelIDs[id]); !slices.Equal(got, will) {
			t.Errorf("Titel %s trägt %q, erwartet %q", id, got, will)
		}
	}
	if got := len(schlagworteAm(t, pool, bestandBericht.TitelIDs["3"])); got != repository.SchlagworteJeTitelMax {
		t.Errorf("Titel 3 trägt %d Schlagworte, erwartet die Grenze %d", got, repository.SchlagworteJeTitelMax)
	}
	wollZuordnungen := 2 + 2 + repository.SchlagworteJeTitelMax + 2
	if !b.AbgleichOK || b.Zuordnungen != wollZuordnungen || b.IstZuordnungen != wollZuordnungen || b.Titel != 4 {
		t.Errorf("Bericht: Abgleich %v, %d gemeldet / %d gezählt an %d Titeln — erwartet %d an 4",
			b.AbgleichOK, b.Zuordnungen, b.IstZuordnungen, b.Titel, wollZuordnungen)
	}
	// Geschichte, Weimarer Republik, Brasilien, Regenwald, Schulbuch und die 300 Pflanzen.
	if b.IstWoerter != 5+repository.SchlagworteJeTitelMax {
		t.Errorf("%d Wörter angelegt, erwartet %d — „brasilien“ ist dasselbe Wort wie „Brasilien“",
			b.IstWoerter, 5+repository.SchlagworteJeTitelMax)
	}
	if b.Weggelassen != 2 || b.Gekuerzt != 1 || b.OhneTitel != 1 || b.Uebersprungen != 0 {
		t.Errorf("weggelassen %d, gekürzt %d, Titel fehlt %d, Fehler %d — erwartet 2, 1, 1, 0",
			b.Weggelassen, b.Gekuerzt, b.OhneTitel, b.Uebersprungen)
	}
	log := protokoll()
	for _, grund := range []string{"Schlagwort leer – weggelassen", "Schlagwort länger als 80 Zeichen – weggelassen",
		"mehr als 300 Schlagworte – die ersten 300 übernommen"} {
		if !strings.Contains(log, grund) {
			t.Errorf("Protokoll nennt „%s“ nicht:\n%s", grund, log)
		}
	}

	// Das Fach: Titel 1 aus den Schlagworten (Geschichte), Titel 4 aus der Signatur (Biologie),
	// obwohl seine Schlagworte Geschichte nennen; Titel 2 und 3 nennen keins.
	for id, will := range map[string]string{"1": "Geschichte", "2": "", "3": "", "4": "Biologie"} {
		var fach string
		if err := pool.QueryRow(ctx, `SELECT coalesce(subject, '') FROM buecher_titel WHERE id = $1`,
			bestandBericht.TitelIDs[id]).Scan(&fach); err != nil {
			t.Fatal(err)
		}
		if fach != will {
			t.Errorf("Titel %s: Fach %q, erwartet %q", id, fach, will)
		}
	}
	if bestandBericht.FachAusSchlagworten != 1 || bestandBericht.FachAusSignatur != 1 {
		t.Errorf("Fach aus den Schlagworten %d, aus der Signatur %d — erwartet 1 und 1",
			bestandBericht.FachAusSchlagworten, bestandBericht.FachAusSignatur)
	}
}

func schlagworteAm(t *testing.T, pool *pgxpool.Pool, titelID string) []string {
	t.Helper()
	w, err := repository.SchlagworteDesTitels(context.Background(), pool, titelID)
	if err != nil {
		t.Fatal(err)
	}
	return w
}
