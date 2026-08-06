import { execSync } from 'node:child_process';

// Nach dem Lauf die Testlieferanten abräumen.
//
// Warum das hier steht und nicht in den Specs: Mehrere Flows legen Lieferanten mit
// eindeutigen Namen an (E2E-Buchhandlung-…, E2E-Bekleb-…, ZZZ-Standard-…) und keiner
// räumte auf. Nach einigen Wochen standen 224 Testlieferanten gegen 3 echte — und deren
// lange Namen haben die Lieferantentabelle so weit aufgebläht, dass die Spalte "Aktionen"
// aus dem Fenster geschoben wurde. Der Bearbeiten-Knopf war unerreichbar, ohne dass
// irgendetwas fehlschlug.
//
// Nach dem GESAMTEN Lauf, nicht nach jedem Test: Manche Flows bauen aufeinander auf,
// ein afterEach würde ihnen den Zustand unter den Füßen wegziehen.
//
// Bestellungen an diese Lieferanten bleiben als Beleg erhalten — lieferant_id ist
// ON DELETE SET NULL (Migration 037), die Historie verliert nur die Verknüpfung.
export default function globalTeardown() {
	const container = process.env.E2E_DB_CONTAINER || 'bibliothek-db-local';
	try {
		execSync(`docker exec -i ${container} psql -U postgres -d bibliothek -v ON_ERROR_STOP=1`, {
			input: `DELETE FROM lieferanten WHERE name LIKE 'E2E%' OR name LIKE 'ZZZ%';`,
			stdio: ['pipe', 'ignore', 'pipe']
		});
	} catch (err) {
		// Kein harter Abbruch: Ein fehlgeschlagenes Aufräumen darf einen grünen Lauf nicht
		// rot färben — es sagt nichts über die geprüfte Anwendung aus.
		console.warn('E2E-Teardown: Testlieferanten nicht abgeräumt —', String(err));
	}
}
