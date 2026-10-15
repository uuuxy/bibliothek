package db

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// Am echten Postgres prüfbar, in einem Test ohne Datenbank nicht: Den Wert leitung gibt es im
// ENUM. Fehlt ein Wert dort, ist eine Rolle im Code fertig, hat Rechte und Wege im Router, und
// trotzdem kann niemand sie bekommen; ein grüner Test in Go bewiese nichts.
func TestRolleLeitungVergebbar(t *testing.T) {
	pool := pgTestPool(t)

	inTx(t, pool, func(tx pgx.Tx) {
		erwarteErfolg(t, tx, "Benutzer mit Rolle leitung",
			`INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
			 VALUES ('L', 'Leitung', 'l@example.org', 'leitung', true)`)
	})
}

// Die Rechte einer neuen Rolle erreichen eine Anlage auf zwei Wegen, und beide müssen dasselbe
// ergeben:
//
//	frische Anlage → db/rechte_vorgabe.go (RechteVorgabe, ON CONFLICT DO NOTHING)
//	gewachsene Anlage → migrations/122_rechte_leitung.sql
//
// Der Test geht den ersten Weg und vergleicht mit derselben Ableitung, die auch die Migration
// nimmt: die Rechte des Admins ohne manage_users und manage_settings. Liefe die Migration
// auseinander, meldete die Selbstprüfung im Betrieb eine Abweichung; der Test hält die Vorgabe
// fest, mit der sie vergleicht.
func TestRechteLeitungLiegenInDerDatenbank(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	d := &Verbindung{Pool: pool}
	if err := d.SchreibeRechteVorgabe(ctx); err != nil {
		t.Fatalf("SchreibeRechteVorgabe: %v", err)
	}

	zeilen := func(rolle string) map[string]bool {
		rows, err := pool.Query(ctx,
			`SELECT permission, allowed FROM role_permissions WHERE UPPER(role) = UPPER($1)`, rolle)
		if err != nil {
			t.Fatalf("Rechte von %s lesen: %v", rolle, err)
		}
		defer rows.Close()
		out := map[string]bool{}
		for rows.Next() {
			var perm string
			var allowed bool
			if err := rows.Scan(&perm, &allowed); err != nil {
				t.Fatalf("Zeile lesen: %v", err)
			}
			out[perm] = allowed
		}
		// Ohne diese Prüfung endete eine abgebrochene Abfrage als leere Map, und die Aussage „die
		// Rechte stimmen" stützte sich auf nichts.
		if err := rows.Err(); err != nil {
			t.Fatalf("Rechte von %s lesen (Abbruch mitten in der Ergebnismenge): %v", rolle, err)
		}
		return out
	}

	admin := zeilen("ADMIN")
	leitung := zeilen("LEITUNG")
	if len(admin) == 0 {
		t.Fatal("keine ADMIN-Zeilen in role_permissions — der Seed lief nicht")
	}
	if len(leitung) == 0 {
		t.Fatal("keine LEITUNG-Zeilen in role_permissions — die Rolle wäre rechtelos")
	}

	verschlossen := map[string]bool{"manage_users": true, "manage_settings": true}
	for perm, adminDarf := range admin {
		soll := adminDarf && !verschlossen[perm]
		ist, vorhanden := leitung[perm]
		if !vorhanden {
			t.Errorf("LEITUNG/%s fehlt in role_permissions (ADMIN hat die Zeile)", perm)
			continue
		}
		if ist != soll {
			t.Errorf("LEITUNG/%s = %v, erwartet %v", perm, ist, soll)
		}
	}
}

// Migration 122 leitet die Rechte der Leitung aus den Zeilen des Admins ab, statt sie
// abzuschreiben.
//
// Eine in SQL abgeschriebene Liste wäre eine zweite Wahrheit neben db/rechte_vorgabe.go: Beim
// nächsten neuen Recht trüge es jemand an einer der beiden Stellen nach, und die Leitung bekäme es
// auf frischen Anlagen und auf gewachsenen nicht, oder umgekehrt. Das liefe still auseinander,
// weil beide Wege gültige Zeilen liefern, nur verschiedene.
//
// Geprüft werden die Anweisungen, nicht die Datei: Der Kopf der Migration erklärt die Ableitung
// und nennt dabei jedes Wort, auf das der Test sieht. Ohne das Abschneiden der Kommentare wäre der
// Test grün, sobald die Begründung dasteht, auch über einer abgeschriebenen Liste.
func TestMigration122LeitetAbStattAbzuschreiben(t *testing.T) {
	pfad := filepath.Join("..", "migrations", "122_rechte_leitung.sql")
	inhalt, err := os.ReadFile(pfad)
	if err != nil {
		t.Fatalf("%s nicht lesbar: %v", pfad, err)
	}
	sql := ohneSQLKommentare(string(inhalt))

	if strings.TrimSpace(sql) == "" {
		t.Fatal("Migration 122 besteht nur aus Kommentaren")
	}

	// Die Ableitung: ein SELECT über die ADMIN-Zeilen derselben Tabelle.
	if !strings.Contains(sql, "FROM role_permissions") {
		t.Error("Migration 122 liest role_permissions nicht — sie schreibt die Rechte " +
			"offenbar ab, statt sie aus den ADMIN-Zeilen abzuleiten")
	}
	// Die zwei Türen müssen ausdrücklich zufallen, nicht aus dem Quellwert folgen.
	for _, recht := range []string{"manage_users", "manage_settings"} {
		if !strings.Contains(sql, recht) {
			t.Errorf("Migration 122 nennt %s nicht in ihren Anweisungen — genau dieses "+
				"Recht soll der Leitung verschlossen bleiben", recht)
		}
	}
	// Bestehendes darf die Migration nicht anfassen: Wer der Leitung ein Recht bewusst
	// weggenommen hat, bekommt es durch ein Update nicht zurück.
	if !strings.Contains(sql, "ON CONFLICT") {
		t.Error("Migration 122 hat kein ON CONFLICT — ein zweiter Lauf oder eine " +
			"Anlage, die die Rechte schon hat, würde mit einem Fehler abbrechen")
	}
}

// ohneSQLKommentare schneidet Zeilenkommentare (--) ab. Das reicht, weil migrations/ keine
// Blockkommentare und keine Zeichenketten mit „--" enthält. Änderte sich das, prüfte der Test zu
// wenig Text und würde rot, nicht fälschlich grün.
func ohneSQLKommentare(sql string) string {
	var b strings.Builder
	for _, zeile := range strings.Split(sql, "\n") {
		if i := strings.Index(zeile, "--"); i >= 0 {
			zeile = zeile[:i]
		}
		b.WriteString(zeile)
		b.WriteString("\n")
	}
	return b.String()
}
