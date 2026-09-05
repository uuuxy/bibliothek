package repository

import (
	"context"
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
