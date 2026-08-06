import { execSync } from 'node:child_process';
import { writeFileSync } from 'node:fs';

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
