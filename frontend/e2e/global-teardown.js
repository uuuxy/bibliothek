import { execSync } from 'node:child_process';
import { readFileSync, rmSync } from 'node:fs';
import { MERKZETTEL } from './global-setup.js';

// Nach dem Lauf: Testlieferanten abräumen und den Hauptlieferanten wiederherstellen.
//
// Warum das Abräumen hier steht und nicht in den Specs: Mehrere Flows legen Lieferanten mit
// eindeutigen Namen an (E2E-…, ZZZ-…) und keiner räumte auf. Nach einigen Wochen standen
// 224 Testlieferanten gegen 3 echte — und deren lange Namen haben die Lieferantentabelle so
// weit aufgebläht, dass die Spalte "Aktionen" aus dem Fenster geschoben wurde. Der
// Bearbeiten-Knopf war unerreichbar, ohne dass irgendetwas fehlschlug.
//
// Nach dem GESAMTEN Lauf, nicht nach jedem Test: Manche Flows bauen aufeinander auf, ein
// afterEach würde ihnen den Zustand unter den Füßen wegziehen.
//
// Seit dem 08.08.2026 gehen die TESTBESTELLUNGEN mit, und das ist eine Korrektur:
// Hier stand „Bestellungen an diese Lieferanten bleiben als Beleg erhalten". Für eine
// echte Bestellung, deren Lieferant gelöscht wurde, stimmt das — sie ist ein Beleg.
// Eine E2E-Bestellung belegt nichts. Sie blieb trotzdem liegen, jeder Lauf legte neue
// nach, und in der Bestellhistorie summierte sich das zu „Gesamtausgaben", die niemand
// je ausgegeben hat. Peter beim Blick auf die Zahl: „wie kommt denn da oben die zahl
// 310933 zusammen?" — der Löwenanteil war ein Lasttest, der Rest genau dieser Bodensatz.
//
// Erkannt am lieferant_name in der Bestellung selbst, nicht über den Fremdschlüssel:
// Der Name steht denormalisiert in bestellungen_verlauf, die Zuordnung überlebt also
// auch das ON DELETE SET NULL. Die Positionen hängen per ON DELETE CASCADE daran.
//
// Dasselbe für die Klassensatz-Reservierungen: Zwei Specs legen welche an, beide
// benennen ihren Titel „E2E …". Ohne Aufräumen wuchs der rote Zähler im Menü mit jedem
// Lauf — zuletzt stand dort 223.
export default function globalTeardown() {
	const container = process.env.E2E_DB_CONTAINER || 'bibliothek-db-local';

	let vorher = '';
	try {
		vorher = readFileSync(MERKZETTEL, 'utf8').trim();
		rmSync(MERKZETTEL, { force: true });
	} catch {
		// Kein Merkzettel — dann gab es beim Start keinen Hauptlieferanten oder das Setup
		// kam nicht durch. Aufgeräumt wird trotzdem.
	}

	// Erst räumen, dann setzen: Solange ein Testlieferant die Rolle noch hält, würde das
	// Zurücksetzen am Teil-Index scheitern (dieselbe Reihenfolge wie setzeHauptlieferant).
	//
	// Die Muster sind bewusst eng an dem, was die Specs selbst schreiben. Ein weiter
	// gefasstes Aufräumen hat hier schon einmal den Hauptlieferanten mitgenommen — was
	// niemandem auffiel, weil nichts fehlschlug, nur der Bestätigungs-Link ausblieb.
	// ALLES IN EINER TRANSAKTION. Ohne sie lief jede Anweisung fuer sich, und die
	// Ruecksetzung besteht aus ZWEI: erst allen die Rolle nehmen, dann dem Richtigen
	// geben. Scheitert die zweite — ungueltige ID, Lieferant inzwischen geloescht,
	// Datenbank kurz weg —, dann hat die erste bereits zugeschlagen und niemand traegt
	// die Rolle mehr. Genau das ist am 08.08.2026 zweimal passiert.
	//
	// Es faellt nicht auf, weil der Teardown seine Fehler bewusst schluckt (ein
	// misslungenes Aufraeumen darf einen gruenen Lauf nicht rot faerben). Die Warnung
	// steht dann in der Konsole, das Ergebnis ist gruen, und die Anlage verschickt
	// Bestellungen ohne Bestaetigungs-Link. Mit BEGIN/COMMIT gilt: entweder die Rolle
	// wandert zurueck, oder sie bleibt, wo sie ist — kein Zustand dazwischen.
	const sql =
		`BEGIN;
		 DELETE FROM bestellungen_verlauf
		  WHERE lieferant_name LIKE 'E2E%' OR lieferant_name LIKE 'ZZZ%';
		 DELETE FROM klassensatz_reservierungen r
		  WHERE r.notiz LIKE 'E2E%'
		     OR EXISTS (SELECT 1 FROM buecher_titel t
		                 WHERE t.id = r.titel_id AND t.titel LIKE 'E2E %');
		 DELETE FROM lieferanten WHERE name LIKE 'E2E%' OR name LIKE 'ZZZ%';` +
		(vorher
			? `UPDATE lieferanten SET ist_hauptlieferant = false WHERE ist_hauptlieferant;
			   UPDATE lieferanten SET ist_hauptlieferant = true WHERE id = '${vorher}';`
			: '') +
		`COMMIT;`;

	try {
		execSync(`docker exec -i ${container} psql -U postgres -d bibliothek -v ON_ERROR_STOP=1`, {
			input: sql,
			stdio: ['pipe', 'ignore', 'pipe']
		});
	} catch (err) {
		// Kein harter Abbruch: Ein fehlgeschlagenes Aufräumen darf einen grünen Lauf nicht
		// rot färben — es sagt nichts über die geprüfte Anwendung aus.
		console.warn('E2E-Teardown: nicht abgeräumt —', String(err));
	}
}
