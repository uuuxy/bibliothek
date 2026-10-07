import { test, expect } from '@playwright/test';
import {
	uiLogin,
	seedSQL,
	querySQL,
	uniqueSuffix,
	scanneWieScanner,
	menuepunkt
} from './helpers.js';

// Ein Handscanner wartet nicht auf die Antwort des Servers. Kommt die Antwort auf einen Scan
// spät (langsamer Server, WLAN zwischen den Regalen), treffen die nächsten Scans ein, solange
// die Anfrage läuft. Jeder davon zählt: Was die Inventur nicht bucht, sondert der Abschluss als
// Verlust aus. Gescannt wird blind, ohne Klick ins Feld, und bewiesen an der Datenbank.
test('Inventur: Scans während einer laufenden Anfrage werden gezählt', async ({ page }) => {
	await uiLogin(page);
	const suffix = uniqueSuffix();
	const signatur = `E2E-INVR-${suffix}`;
	const barcodes = ['A', 'B', 'C'].map((b) => `B-INVR${b}-${suffix}`);
	const erfasst = () =>
		querySQL(`
			SELECT count(*) FROM inventur_erfassungen e
			JOIN buecher_exemplare x ON x.id = e.exemplar_id
			WHERE x.barcode_id IN (${barcodes.map((b) => `'${b}'`).join(', ')})`);

	try {
		seedSQL(`
			WITH t AS (
				INSERT INTO buecher_titel (titel, signatur)
				VALUES ('E2E-Scanreihe-${suffix}', '${signatur}') RETURNING id
			)
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
			SELECT id, b, true FROM t, unnest(ARRAY[${barcodes.map((b) => `'${b}'`).join(', ')}]) AS b;
		`);

		await menuepunkt(page, 'Inventur').click();
		await page.getByRole('button', { name: 'Neue Bestandsprüfung starten' }).click();
		await page.getByText('Nur bestimmte Signatur').click();
		await page.getByLabel('Signatur auswählen').fill(signatur);
		await page.getByRole('button', { name: 'Inventur Starten' }).click();
		const feld = page.getByPlaceholder('Barcode scannen...');
		await expect(feld).toBeFocused();

		// Die Antwort auf den ersten Scan kommt nach anderthalb Sekunden.
		let ersterScan = true;
		await page.route('**/api/inventur/scan', async (route) => {
			if (ersterScan) {
				ersterScan = false;
				await new Promise((weiter) => setTimeout(weiter, 1500));
			}
			await route.continue();
		});

		for (const barcode of barcodes) {
			await scanneWieScanner(page, barcode);
		}

		await expect
			.poll(erfasst, { message: 'jeder der drei Scans steht in der Inventur', timeout: 10_000 })
			.toBe('3');
		await expect(feld, 'der nächste Scan landet wieder im Feld').toBeFocused();
		await expect(feld).toHaveValue('');
	} finally {
		seedSQL(`
			DELETE FROM inventur_sessions WHERE scope_signatur = '${signatur}';
			DELETE FROM buecher_exemplare WHERE barcode_id IN (${barcodes.map((b) => `'${b}'`).join(', ')});
			DELETE FROM buecher_titel WHERE titel = 'E2E-Scanreihe-${suffix}';
		`);
	}
});
