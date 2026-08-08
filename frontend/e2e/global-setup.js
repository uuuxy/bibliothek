import { execSync } from 'node:child_process';
import { existsSync, writeFileSync } from 'node:fs';

// Merkt sich vor dem Lauf, wer Hauptlieferant ist.
//
// Die Rolle gehört höchstens EINEM (Migration 066). Mehrere Flows legen sich deshalb
// zwangsläufig einen eigenen Hauptlieferanten an und nehmen sie damit dem echten weg —
// und der globalTeardown räumt die Testlieferanten danach ab. Ohne diese Sicherung stünde
// die Anlage nach jedem E2E-Lauf ganz ohne Hauptlieferanten da: Bestellungen gingen ohne
// Bestelllink raus, die Vorauswahl im Bestellformular wäre leer, und gemerkt hätte es
// niemand. Genau das ist beim Einbau passiert und erst am Bildschirm aufgefallen.
//
// Bewusst hier und nicht in den einzelnen Specs: Ein Spec kann nur seinen eigenen Schaden
// zurücknehmen. Die Frage „wer hatte die Rolle, bevor irgendetwas lief" beantwortet nur,
// wer vor allen Specs dran ist.
export const MERKZETTEL = '.e2e-hauptlieferant';

export default function globalSetup() {
	// Ein bereits liegender Merkzettel heisst: Es laeuft schon eine Suite. Zwei Laeufe
	// gleichzeitig teilen sich EINE Datenbank — die Konfiguration steht deshalb auf
	// workers: 1 und fullyParallel: false. Zwei Prozesse hebeln das aus, und der Schaden
	// ist still: Der zweite Lauf ueberschreibt den Merkzettel, der erste Teardown loescht
	// ihn, der zweite findet keinen mehr und liest das als „es gab keinen
	// Hauptlieferanten". Er raeumt die Testlieferanten trotzdem ab und stellt nichts
	// zurueck. Danach steht die Anlage ohne Hauptlieferanten da: Bestellungen gehen ohne
	// Bestaetigungs-Link raus, und gemerkt haette es niemand.
	//
	// Genau das ist am 08.08.2026 passiert, weil ich zwei Suiten parallel gestartet habe.
	// Nebenbei fielen dabei drei Tests an drei unabhaengigen Stellen aus — die sahen aus
	// wie flaky und waren Zustandskollision. Lieber hier hart abbrechen als eine halbe
	// Stunde die falsche Spur verfolgen.
	if (existsSync(MERKZETTEL)) {
		throw new Error(
			`E2E-Setup: ${MERKZETTEL} existiert bereits — es laeuft schon eine Suite gegen dieselbe ` +
				`Datenbank. Zwei parallele Laeufe verfaelschen Ergebnisse UND nehmen der Anlage den ` +
				`Hauptlieferanten. Erst den laufenden Lauf abwarten. Ist keiner mehr aktiv, wurde er ` +
				`abgebrochen — dann die Datei loeschen und den Hauptlieferanten pruefen.`
		);
	}

	const container = process.env.E2E_DB_CONTAINER || 'bibliothek-db-local';
	try {
		const id = execSync(
			`docker exec -i ${container} psql -U postgres -d bibliothek -tA -v ON_ERROR_STOP=1`,
			{ input: `SELECT coalesce(max(id::text), '') FROM lieferanten WHERE ist_hauptlieferant;` }
		)
			.toString()
			.trim();
		writeFileSync(MERKZETTEL, id);
	} catch (err) {
		console.warn('E2E-Setup: Hauptlieferant nicht gemerkt —', String(err));
	}
}
