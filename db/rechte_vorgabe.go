package db

import (
	"context"
	"fmt"
)

const sqlFehlendesRecht = `
		INSERT INTO role_permissions (role, permission, allowed)
		VALUES ($1, $2, $3)
		ON CONFLICT (role, permission) DO NOTHING
	`

// RechteEintrag ist eine Zeile der Rechte-Vorgabe.
type RechteEintrag struct {
	Rolle   string
	Recht   string
	Erlaubt bool
}

// RechteVorgabe ist die eine Quelle der Rechte, mit denen eine Anlage beginnt:
// SchreibeRechteVorgabe schreibt fehlende Zeilen, und die Selbstprüfung
// (internal/bereitschaft/rechte.go) gleicht die Tabelle damit ab. Den Abgleich braucht es, weil
// ON CONFLICT DO NOTHING Bestehendes nie anfasst: Eine geänderte Vorgabe erreicht eine
// bestehende Anlage nicht von selbst.
//
// Die Rolle eines Kontos steht klein in benutzer.rolle (ENUM); die Prüfung der Rechte verbindet
// beide Schreibweisen mit UPPER() (repository/rechte.go). role_permissions sagt also, was eine
// Rolle darf, nicht, wer sie hat.
var RechteVorgabe = []RechteEintrag{
	// Admin
	{"ADMIN", "perform_actions", true},
	{"ADMIN", "view_students", true},
	{"ADMIN", "edit_students", true},
	{"ADMIN", "create_students", true},
	{"ADMIN", "delete_students", true},
	{"ADMIN", "import_students", true},
	{"ADMIN", "upload_photos", true},
	{"ADMIN", "view_books", true},
	{"ADMIN", "edit_books", true},
	{"ADMIN", "delete_books", true},
	{"ADMIN", "inventory_scan", true},
	{"ADMIN", "manage_inventory", true},
	{"ADMIN", "view_orders", true},
	{"ADMIN", "create_orders", true},
	{"ADMIN", "create_reservations", true},
	{"ADMIN", "manage_vormerkungen", true},
	{"ADMIN", "view_graduates", true},
	{"ADMIN", "view_stats", true},
	{"ADMIN", "audit_logs", true},
	// audit_details öffnet die Auskunft an der Theke (api/system/theken_auskunft.go), den einen
	// Leseweg in audit_log.details. Es ist ein eigenes Recht neben audit_logs, weil das
	// Protokoll keine Namen zeigt (Stufe 0) und die Auskunft schon (Stufe 2). Ein neues Recht
	// erreicht eine bestehende Anlage über SchreibeRechteVorgabe: Die fehlende Zeile kommt dazu.
	{"ADMIN", "audit_details", true},
	{"ADMIN", "manage_users", true},
	{"ADMIN", "manage_settings", true},
	{"ADMIN", "manage_students_admin", true},
	{"ADMIN", "merge_students", true},

	// Leitung: alle Rechte außer „Benutzer & Rechte" und „Einstellungen".
	//
	// Die Liste ist die des Admins ohne zwei Türen, in derselben Reihenfolge, damit ein neues
	// Recht beim Nachtragen nicht nur bei einer der beiden Rollen landet. Die Prüfung dafür steht
	// in api/benutzer/rolle_leitung_test.go: Sie leitet das Soll aus den Zeilen des Admins ab und
	// meldet jede fehlende oder überzählige Zeile.
	//
	// Warum die Leitung kein manage_users bekommt, obwohl sie die Bibliothek führt: Mit
	// diesem Recht kann man die E-Mail-Adresse eines Kontos ändern, und die Anmeldung
	// erkennt eine Person allein an ihrer E-Mail. Die Rechtevergabe ist damit der Weg zu
	// jedem Konto der Anlage — das bleibt beim Admin. Der Admin kann es ihr erteilen.
	{"LEITUNG", "perform_actions", true},
	{"LEITUNG", "view_students", true},
	{"LEITUNG", "edit_students", true},
	{"LEITUNG", "create_students", true},
	{"LEITUNG", "delete_students", true},
	{"LEITUNG", "import_students", true},
	{"LEITUNG", "upload_photos", true},
	{"LEITUNG", "view_books", true},
	{"LEITUNG", "edit_books", true},
	{"LEITUNG", "delete_books", true},
	{"LEITUNG", "inventory_scan", true},
	{"LEITUNG", "manage_inventory", true},
	{"LEITUNG", "view_orders", true},
	{"LEITUNG", "create_orders", true},
	{"LEITUNG", "create_reservations", true},
	{"LEITUNG", "manage_vormerkungen", true},
	{"LEITUNG", "view_graduates", true},
	{"LEITUNG", "view_stats", true},
	{"LEITUNG", "audit_logs", true},
	{"LEITUNG", "audit_details", true},
	{"LEITUNG", "manage_users", false},
	{"LEITUNG", "manage_settings", false},
	{"LEITUNG", "manage_students_admin", true},
	{"LEITUNG", "merge_students", true},

	// Mitarbeiter
	{"MITARBEITER", "perform_actions", true},
	{"MITARBEITER", "view_students", true},
	{"MITARBEITER", "edit_students", true},
	{"MITARBEITER", "create_students", true},
	{"MITARBEITER", "delete_students", true},
	{"MITARBEITER", "import_students", true},
	{"MITARBEITER", "upload_photos", true},
	{"MITARBEITER", "view_books", true},
	{"MITARBEITER", "edit_books", true},
	{"MITARBEITER", "delete_books", true},
	{"MITARBEITER", "inventory_scan", true},
	{"MITARBEITER", "manage_inventory", true},
	{"MITARBEITER", "view_orders", true},
	{"MITARBEITER", "create_orders", true},
	{"MITARBEITER", "create_reservations", true},
	{"MITARBEITER", "manage_vormerkungen", true},
	{"MITARBEITER", "view_graduates", true},
	{"MITARBEITER", "view_stats", true},
	{"MITARBEITER", "audit_logs", false},
	{"MITARBEITER", "audit_details", false},
	{"MITARBEITER", "manage_users", false},
	{"MITARBEITER", "manage_settings", false},
	{"MITARBEITER", "manage_students_admin", false},
	{"MITARBEITER", "merge_students", false},

	// Kollegium: Eine Lehrkraft meldet sich an, um im Portal einen Klassensatz zu reservieren;
	// dafür reicht create_reservations, das keine Personendaten öffnet. Jedes weitere Recht
	// öffnete über das Menü (frontend/src/lib/menue.js) Seiten wie Medienkatalog, Druck-Center
	// oder Schülerdatei, perform_actions dazu Ausleihe und Rückgabe (/api/action).
	{"KOLLEGIUM", "create_reservations", true},
	{"KOLLEGIUM", "perform_actions", false},
	{"KOLLEGIUM", "view_students", false},
	{"KOLLEGIUM", "edit_students", false},
	{"KOLLEGIUM", "create_students", false},
	{"KOLLEGIUM", "delete_students", false},
	{"KOLLEGIUM", "import_students", false},
	{"KOLLEGIUM", "upload_photos", false},
	{"KOLLEGIUM", "view_books", false},
	{"KOLLEGIUM", "edit_books", false},
	{"KOLLEGIUM", "delete_books", false},
	{"KOLLEGIUM", "inventory_scan", false},
	{"KOLLEGIUM", "view_orders", false},
	{"KOLLEGIUM", "create_orders", false},
	{"KOLLEGIUM", "manage_vormerkungen", false},
	{"KOLLEGIUM", "view_graduates", false},
	{"KOLLEGIUM", "view_stats", false},
	{"KOLLEGIUM", "audit_logs", false},
	{"KOLLEGIUM", "audit_details", false},
	{"KOLLEGIUM", "manage_users", false},
	{"KOLLEGIUM", "manage_settings", false},
	{"KOLLEGIUM", "manage_students_admin", false},
	{"KOLLEGIUM", "merge_students", false},

	// Helfer: leiht an der Theke aus und nimmt zurück (perform_actions: /api/action, /scan,
	// /search, /events). view_students bleibt zu: Es öffnete Schülerlisten, Profile, das
	// Mahnwesen und den Druck vieler Mahnungen auf einmal, der die Mahnstufe erhöht.
	{"HELFER", "perform_actions", true},
	{"HELFER", "view_students", false},
	{"HELFER", "edit_students", false},
	{"HELFER", "create_students", false},
	{"HELFER", "delete_students", false},
	{"HELFER", "import_students", false},
	{"HELFER", "upload_photos", false},
	// Den Katalog sieht der Helfer: An der Theke ist er die erste Anlaufstelle für „Habt ihr
	// Band 3 von Gregs Tagebuch noch da?" und müsste sonst jedes Mal an die Leitung abgeben. Das
	// Recht liest nur und öffnet keine Personendaten; die stecken hinter view_students.
	{"HELFER", "view_books", true},
	{"HELFER", "edit_books", false},
	{"HELFER", "delete_books", false},
	{"HELFER", "inventory_scan", false},
	{"HELFER", "view_orders", false},
	{"HELFER", "create_orders", false},
	{"HELFER", "create_reservations", false},
	// Die Vormerkungen kann der Admin dem Helfer in der Matrix öffnen; ab Werk sind sie zu. Die
	// Warteliste nennt Name und Klasse, dieselben Felder, die der Helfer beim Scan des Ausweises
	// sieht, aber verknüpft mit dem, was jemand lesen will
	// (bewertung/sicherheitsbefund-vormerkungen.md). Das Paar steht in RechteOptional; die
	// Selbstprüfung meldet den Haken nicht als Abweichung.
	{"HELFER", "manage_vormerkungen", false},
	{"HELFER", "view_graduates", false},
	{"HELFER", "view_stats", false},
	{"HELFER", "audit_logs", false},
	{"HELFER", "audit_details", false},
	{"HELFER", "manage_users", false},
	{"HELFER", "manage_settings", false},
	{"HELFER", "manage_students_admin", false},
	{"HELFER", "merge_students", false},
}

// RechteOptional nennt die Paare aus Rolle und Recht, deren Wert der Admin entscheidet: Die
// Vorgabe ist nur der Startwert, der Haken in der Matrix der vorgesehene Gebrauch. Die
// Selbstprüfung (pruefeRechteVorgabe) verlangt die Zeile, vergleicht aber ihren Wert nicht;
// sonst stünde jede Anlage, die das Recht nutzt, dauerhaft mit einer Warnung da, und an
// Dauerwarnungen gewöhnt man sich.
var RechteOptional = mitLeitungAlsStartwert(map[string]bool{
	"HELFER/manage_vormerkungen": true,
	// Diese beiden Rechte sind dafür da, ans Sekretariat (Rolle MITARBEITER) abgegeben zu
	// werden; stünden sie nicht hier, wäre jede Abgabe eine Dauerwarnung.
	"MITARBEITER/manage_settings":       true,
	"MITARBEITER/manage_students_admin": true,
	// Das Zusammenführen zweier Datensätze ebenso.
	"MITARBEITER/merge_students": true,
})

// mitLeitungAlsStartwert nimmt jedes Paar der Leitung dazu. Bei den anderen Rollen ist ein
// abweichender Wert die Ausnahme und steht einzeln oben; bei der Leitung ist er die Regel: Sie
// beginnt mit allen Rechten außer „Benutzer & Rechte" und „Einstellungen", und was nicht passt,
// nimmt der Admin in der Matrix weg.
//
// Die Paare kommen aus RechteVorgabe statt aus einer zweiten Liste, die beim nächsten neuen
// Recht auseinanderliefe (Prüfung: api/benutzer/rolle_leitung_test.go). Dass jede Zeile da ist,
// prüft die Selbstprüfung weiter.
func mitLeitungAlsStartwert(m map[string]bool) map[string]bool {
	for _, e := range RechteVorgabe {
		if e.Rolle == "LEITUNG" {
			m[e.Rolle+"/"+e.Recht] = true
		}
	}
	return m
}

// sqlVererbeAufgeteilteRechte gibt manage_settings und manage_students_admin den Wert, den die
// Rolle für manage_users hat, bevor die Vorgabe fehlende Zeilen auffüllt. Beide gingen aus
// manage_users hervor; ohne das Erbe verlöre eine Rolle, die es hatte, beim Update still
// Einstellungen, Versetzung, Auskunft und Papierkorb. Steht die Zeile schon, geschieht nichts.
const sqlVererbeAufgeteilteRechte = `
	INSERT INTO role_permissions (role, permission, allowed)
	SELECT rp.role, neu.permission, rp.allowed
	FROM role_permissions rp
	CROSS JOIN (VALUES ('manage_settings'), ('manage_students_admin')) AS neu(permission)
	WHERE rp.permission = 'manage_users'
	ON CONFLICT (role, permission) DO NOTHING
`

// sqlVererbeZusammenfuehren gibt merge_students den Wert von manage_students_admin, aus dem das
// Recht hervorging: Eine Anlage, die das Sonderrecht ans Sekretariat abgegeben hat, behält so
// den Knopf zum Zusammenführen zweier Datensätze.
const sqlVererbeZusammenfuehren = `
	INSERT INTO role_permissions (role, permission, allowed)
	SELECT rp.role, 'merge_students', rp.allowed
	FROM role_permissions rp
	WHERE rp.permission = 'manage_students_admin'
	ON CONFLICT (role, permission) DO NOTHING
`

// SchreibeRechteVorgabe schreibt die Zeilen aus RechteVorgabe, die in role_permissions fehlen;
// bestehende fasst sie nicht an. Vorher erben zwei Rechte den Wert des Rechts, aus dem sie
// hervorgingen. Der Start schreibt nur Daten, das Schema kommt aus den Migrationen
// (kein_ddl_im_schreibpfad_test.go).
func (v *Verbindung) SchreibeRechteVorgabe(ctx context.Context) error {
	if _, err := v.Pool.Exec(ctx, sqlVererbeAufgeteilteRechte); err != nil {
		return fmt.Errorf("vererbung manage_users → manage_settings/manage_students_admin: %w", err)
	}
	if _, err := v.Pool.Exec(ctx, sqlVererbeZusammenfuehren); err != nil {
		return fmt.Errorf("vererbung manage_students_admin → merge_students: %w", err)
	}
	for _, e := range RechteVorgabe {
		_, err := v.Pool.Exec(ctx, sqlFehlendesRecht, e.Rolle, e.Recht, e.Erlaubt)
		if err != nil {
			return fmt.Errorf("failed to seed permission default (%s, %s): %w", e.Rolle, e.Recht, err)
		}
	}
	return nil
}
