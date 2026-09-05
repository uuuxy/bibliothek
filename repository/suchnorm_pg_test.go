package repository

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// Zwillings-Pflicht: Suchnorm (Go) und suchnorm (SQL, Migration 054) müssen für jeden
// Namen dasselbe liefern — sonst findet die Schülersuche einen Menschen, den der
// LUSD-Schlüssel nicht findet (oder umgekehrt), und niemand merkt es. Der Korpus deckt
// ab, was an einer hessischen Schule wirklich in Namenslisten steht: deutsche Umlaute und
// Ersatzschreibungen, türkische, polnische, rumänische, vietnamesische, südslawische,
// skandinavische, spanische, französische Buchstaben, dazu die Sonderfälle, die
// unaccent kennt, die Unicode-Zerlegung aber nicht (ß, ø, ł, æ, œ, đ, ı, þ, ð).
func TestSuchnorm_GoUndSQLSindZwillinge(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	korpus := []string{
		"Müller", "Mueller", "MÜLLER", "Muller", "Öztürk", "Oeztuerk", "Straße", "Strasse",
		"Bauer", "Baur", "Goethe", "Gothe", "Anna-Lena", "Anna Lena", "  Anna  Lena ",
		"García", "Nguyễn", "Łukasz", "Şule", "Çelik", "Işık", "İlker", "Ștefan", "Țăranu",
		"Đorđe", "Čović", "Šimić", "Žana", "Bjørn", "Åse", "Æbelø", "Œuvre", "Þór", "Guðrún",
		"Nuñez", "João", "Élodie", "François", "Ærø", "Ångström", "Hồ Chí Minh", "Dvořák",
		"Al-Sayed", "O'Brien", "Müller-Lüdenscheidt", "Sæther", "Đặng", "Ægir", "",
	}
	for _, name := range korpus {
		var sql string
		if err := pool.QueryRow(ctx, `SELECT suchnorm($1)`, name).Scan(&sql); err != nil {
			t.Fatalf("suchnorm(%q): %v", name, err)
		}
		if got := Suchnorm(name); got != sql {
			t.Errorf("Suchnorm(%q) = %q, SQL suchnorm = %q — die Zwillinge sind auseinander", name, got, sql)
		}
	}
}

// Der Korpus oben ist eine Stichprobe. Dieser Durchlauf ist die Vollprobe über die
// lateinischen Zeichenblöcke, aus denen Schülernamen bestehen: Latin-1 Supplement,
// Latin Extended-A/B und Latin Extended Additional (Vietnamesisch). Jedes Zeichen
// einzeln durch beide Seiten — ein Buchstabe, den unaccent.rules kennt, der Go-Zwilling
// aber nicht (kein abtrennbarer Akzent, kein Sonderfall), fällt hier auf, statt erst bei
// dem einen Schüler, dessen Name ihn trägt.
func TestSuchnorm_ZeichenDurchlaufGoUndSQL(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	bloecke := [][2]rune{{0x00C0, 0x024F}, {0x1E00, 0x1EFF}}
	var abweichungen []string
	for _, b := range bloecke {
		for r := b[0]; r <= b[1]; r++ {
			name := string(r)
			var sql string
			if err := pool.QueryRow(ctx, `SELECT suchnorm($1)`, name).Scan(&sql); err != nil {
				t.Fatalf("suchnorm(%q U+%04X): %v", name, r, err)
			}
			if got := Suchnorm(name); got != sql {
				abweichungen = append(abweichungen, fmt.Sprintf("U+%04X %q: Go %q, SQL %q", r, name, got, sql))
			}
		}
	}
	if len(abweichungen) > 0 {
		t.Errorf("%d Zeichen, bei denen Go und SQL auseinanderliegen:\n%s", len(abweichungen), strings.Join(abweichungen, "\n"))
	}
}
