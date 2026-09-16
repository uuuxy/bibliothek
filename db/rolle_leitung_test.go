package db

import "testing"

// Die Rolle Leitung (15.09.2026): „Die Leitung startet mit allen Rechten außer
// ‚Benutzer & Rechte' und ‚Einstellungen'."
//
// Dieser Test prüft die PAARUNG, nicht eine abgeschriebene Liste: Er leitet das Soll aus
// den ADMIN-Zeilen derselben Vorgabe ab. Eine handgepflegte Liste würde beim nächsten
// neuen Recht still auseinanderlaufen — die Leitung bekäme es nicht, und niemand merkte
// es, weil der Test weiter grün wäre. Genau so verlor das Kollegium 2026 wochenlang
// Menüpunkte (siehe Kommentar an RechteVorgabe).
//
// Bewusst OHNE Datenbank: Der Abgleich gegen die Live-Tabelle steckt in der
// Selbstprüfung und in rolle_leitung_pg_test.go, und die hängen an TEST_DATABASE_URL.
// Ein Gate, das sich ohne seine Umgebung still abschaltet, meldet grün und hat nichts
// geprüft (siehe helfer_permissions_pg_test.go).
func TestRechteVorgabeLeitung(t *testing.T) {
	// Die zwei Türen, die der Leitung verschlossen bleiben. „Systempflege" ist kein
	// eigenes Recht: Was nur der Admin braucht, heißt manage_users (Benutzer & Rechte)
	// und manage_settings (Einstellungen).
	verschlossen := map[string]bool{
		"manage_users":    true,
		"manage_settings": true,
	}

	admin := map[string]bool{}
	leitung := map[string]bool{}
	for _, e := range RechteVorgabe {
		switch e.Role {
		case "ADMIN":
			admin[e.Permission] = e.Allowed
		case "LEITUNG":
			leitung[e.Permission] = e.Allowed
		}
	}

	if len(leitung) == 0 {
		t.Fatal("RechteVorgabe kennt keine Zeile für LEITUNG — die Rolle wäre rechtelos, " +
			"denn der Seed schreibt nur, was hier steht")
	}

	for perm, adminDarf := range admin {
		soll := adminDarf && !verschlossen[perm]
		ist, vorhanden := leitung[perm]
		if !vorhanden {
			t.Errorf("LEITUNG/%s fehlt in der Vorgabe (ADMIN hat die Zeile) — "+
				"eine fehlende Zeile erreicht eine Bestandsanlage nie von selbst", perm)
			continue
		}
		if ist != soll {
			t.Errorf("LEITUNG/%s = %v, erwartet %v (ADMIN = %v, verschlossen = %v)",
				perm, ist, soll, adminDarf, verschlossen[perm])
		}
	}

	for perm := range leitung {
		if _, vorhanden := admin[perm]; !vorhanden {
			t.Errorf("LEITUNG/%s steht in der Vorgabe, ADMIN nicht — die Leitung ist "+
				"ADMIN minus zwei Türen, nicht eine eigene Rechteliste", perm)
		}
	}

	// Die beiden Türen ausdrücklich, damit der Zweck der Rolle im Test lesbar bleibt
	// und nicht nur aus der Ableitung oben folgt.
	for perm := range verschlossen {
		if leitung[perm] {
			t.Errorf("LEITUNG/%s ist true — genau diese Tür soll der Leitung verschlossen bleiben", perm)
		}
	}
}

// Die Rechte der Leitung sind ein STARTWERT, kein Soll. Absprache vom 15.09.2026: „Die Leitung
// startet mit allen Rechten außer Benutzer & Rechte und Einstellungen; was nicht passt,
// nimmt der Admin im Rechte-Editor weg."
//
// Daraus folgt für die Selbstprüfung (api/betriebsbereitschaft.go): Sie darf einen
// abweichenden LIVE-Wert nicht als Drift melden, sonst steht jede Anlage, die die Rolle
// einmal angepasst hat, dauerhaft mit einer Warnung da — und Dauerwarnungen erziehen zum
// Wegsehen (derselbe Grund, aus dem MITARBEITER/manage_settings dort steht). Die EXISTENZ
// der Zeile prüft sie weiter: Ein fehlendes Recht erreicht eine Bestandsanlage nie von
// selbst, das bleibt ein Befund.
func TestRechteDerLeitungSindEinStartwert(t *testing.T) {
	var leitung int
	for _, e := range RechteVorgabe {
		if e.Role != "LEITUNG" {
			continue
		}
		leitung++
		if !RechteOptional[e.Role+"/"+e.Permission] {
			t.Errorf("LEITUNG/%s fehlt in RechteOptional — nimmt der Admin dieses Recht "+
				"weg, meldet die Selbstprüfung dauerhaft eine Abweichung", e.Permission)
		}
	}
	if leitung == 0 {
		t.Fatal("keine LEITUNG-Zeilen in der Vorgabe — der Test prüft nichts")
	}
}
