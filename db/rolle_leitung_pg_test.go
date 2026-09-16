package db

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// TestRolleLeitungVergebbar prüft am ECHTEN Postgres, was ein Unit-Test nicht kann: dass
// der ENUM-Wert existiert. Genau das fehlte der Rolle Helfer bis Migration 042 — sie war
// im Code fertig gebaut, geseedet und im Router verzweigt, aber niemandem zuweisbar,
// weil das ENUM sie nicht kannte. Ein Grün in Go beweist hier also nichts.
func TestRolleLeitungVergebbar(t *testing.T) {
	pool := pgTestPool(t)

	inTx(t, pool, func(tx pgx.Tx) {
		erwarteErfolg(t, tx, "Benutzer mit Rolle leitung",
			`INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
			 VALUES ('L', 'Leitung', 'l@example.org', 'leitung', true)`)
	})
}

// TestRechteLeitungLiegenInDerDatenbank prüft den Weg, den die Rechte einer neuen Rolle
// auf eine BESTEHENDE Anlage nehmen. Es gibt zwei, und beide müssen dasselbe Ergebnis
// liefern:
//
//	frische Anlage → db/seed.go (RechteVorgabe, ON CONFLICT DO NOTHING)
//	gewachsene Anlage → migrations/122_rechte_leitung.sql
//
// Der Test fährt den Seed-Weg und vergleicht das Ergebnis gegen dieselbe Ableitung, die
// auch die Migration verwendet: ADMIN minus manage_users und manage_settings. Läuft die
// SQL-Datei aus 122 auseinander, meldet die Selbstprüfung im Betrieb eine Abweichung —
// und dieser Test hält die Vorgabe fest, gegen die sie vergleicht.
func TestRechteLeitungLiegenInDerDatenbank(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()

	d := &Database{Pool: pool}
	if err := d.InitPermissions(ctx); err != nil {
		t.Fatalf("InitPermissions: %v", err)
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
		// Ohne diese Prüfung endet eine abgebrochene Abfrage als LEERE Map — und die
		// Behauptung „die Rechte stimmen" stützte sich dann auf nichts.
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

// TestMigration122LeitetAbStattAbzuschreiben liest die Migration und verlangt, dass sie
// ihre Rechteliste aus den ADMIN-Zeilen ABLEITET.
//
// Der Grund ist eine Bugklasse, nicht Geschmack: Eine in SQL abgeschriebene Liste ist
// eine zweite Wahrheit neben db/seed.go. Beim nächsten neuen Recht trägt es jemand an
// einer der beiden Stellen nach — die Leitung bekommt es auf frischen Anlagen und auf
// gewachsenen nicht, oder umgekehrt. Auseinanderlaufen würde still passieren: Beide
// Wege liefern gültige Zeilen, nur verschiedene.
//
// Geprüft wird auf den ANWEISUNGEN, nicht auf der Datei: Der Kopf der Migration erklärt
// genau diese Ableitung und nennt dabei jedes Wort, auf das der Test sieht. Ohne das
// Abschneiden der Kommentare wäre das Gate grün, sobald die Begründung dasteht — auch
// wenn darunter eine abgeschriebene Liste folgt.
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

// ohneSQLKommentare schneidet Zeilenkommentare (--) weg. Absichtlich schlicht: Es gibt in
// migrations/ keine Blockkommentare und keine Zeichenketten mit „--" darin. Sollte das
// einmal nicht mehr stimmen, ist ein zu grob geschnittenes SQL für ein Gate der
// harmlosere Fehler — es prüft dann zu wenig Text und wird rot, nicht falsch grün.
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
