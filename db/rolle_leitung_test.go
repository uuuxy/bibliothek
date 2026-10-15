package db

import "testing"

// Die Leitung beginnt mit allen Rechten außer „Benutzer & Rechte" und „Einstellungen".
//
// Der Test prüft die Paarung, keine abgeschriebene Liste: Er leitet das Soll aus den Zeilen des
// Admins in derselben Vorgabe ab. Eine von Hand gepflegte Liste liefe beim nächsten neuen Recht
// still auseinander; die Leitung bekäme es nicht, und der Test bliebe grün.
//
// Er braucht keine Datenbank: Den Abgleich mit der Tabelle machen die Selbstprüfung und
// rolle_leitung_pg_test.go, und die hängen an TEST_DATABASE_URL. Dieser Test läuft immer.
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
		switch e.Rolle {
		case "ADMIN":
			admin[e.Recht] = e.Erlaubt
		case "LEITUNG":
			leitung[e.Recht] = e.Erlaubt
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

// Die Rechte der Leitung sind ein Startwert, kein Soll: Was nicht passt, nimmt der Admin in der
// Matrix weg.
//
// Die Selbstprüfung (internal/bereitschaft/rechte.go) meldet deshalb einen abweichenden Wert
// nicht, sonst stünde jede Anlage, die die Rolle angepasst hat, dauerhaft mit einer Warnung da,
// und an Dauerwarnungen gewöhnt man sich (aus demselben Grund steht dort
// MITARBEITER/manage_settings). Ob die Zeile da ist, prüft sie weiter: Ein fehlendes Recht
// erreicht eine bestehende Anlage nie von selbst.
func TestRechteDerLeitungSindEinStartwert(t *testing.T) {
	var leitung int
	for _, e := range RechteVorgabe {
		if e.Rolle != "LEITUNG" {
			continue
		}
		leitung++
		if !RechteOptional[e.Rolle+"/"+e.Recht] {
			t.Errorf("LEITUNG/%s fehlt in RechteOptional — nimmt der Admin dieses Recht "+
				"weg, meldet die Selbstprüfung dauerhaft eine Abweichung", e.Recht)
		}
	}
	if leitung == 0 {
		t.Fatal("keine LEITUNG-Zeilen in der Vorgabe — der Test prüft nichts")
	}
}
