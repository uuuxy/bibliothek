import { test, expect } from '@playwright/test';
import {
	uiLogin,
	seedSQL,
	querySQL,
	uniqueSuffix,
	scanneWieScanner,
	menuepunkt
} from './helpers.js';

/** @param {string[]} barcodes */
const liste = (barcodes) => barcodes.map((b) => `'${b}'`).join(', ');

/** @param {string[]} barcodes Wie viele dieser Exemplare stehen in der Inventur? */
const erfasst = (barcodes) =>
	querySQL(`
		SELECT count(*) FROM inventur_erfassungen e
		JOIN buecher_exemplare x ON x.id = e.exemplar_id
		WHERE x.barcode_id IN (${liste(barcodes)})`);

/**
 * Legt einen Titel mit diesen Exemplaren an und startet eine Inventur über seine Signatur.
 * @param {import('@playwright/test').Page} page
 * @param {string} suffix
 * @param {string[]} barcodes
 */
async function starteInventur(page, suffix, barcodes) {
	seedSQL(`
		WITH t AS (
			INSERT INTO buecher_titel (titel, signatur)
			VALUES ('E2E-Scanreihe-${suffix}', 'E2E-INVR-${suffix}') RETURNING id
		)
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
		SELECT id, b, true FROM t, unnest(ARRAY[${liste(barcodes)}]) AS b;
	`);
	await menuepunkt(page, 'Inventur').click();
	await page.getByRole('button', { name: 'Neue Inventur' }).click();
	await page.getByText('Nur bestimmte Signatur').click();
	await page.getByLabel('Signatur auswählen').fill(`E2E-INVR-${suffix}`);
	await page.getByRole('button', { name: 'Inventur starten' }).click();
	const feld = page.getByPlaceholder('Barcode scannen...');
	await expect(feld).toBeFocused();
	return feld;
}

/** @param {string} suffix @param {string[]} barcodes */
function raeumeAuf(suffix, barcodes) {
	seedSQL(`
		DELETE FROM inventur_sessions WHERE scope_signatur = 'E2E-INVR-${suffix}';
		DELETE FROM buecher_exemplare WHERE barcode_id IN (${liste(barcodes)});
		DELETE FROM buecher_titel WHERE titel = 'E2E-Scanreihe-${suffix}';
	`);
}

// Ein Handscanner wartet nicht auf die Antwort des Servers. Kommt die Antwort auf einen Scan
// spät (langsamer Server, WLAN zwischen den Regalen), treffen die nächsten Scans ein, solange
// die Anfrage läuft. Jeder davon zählt: Was die Inventur nicht bucht, sondert der Abschluss als
// Verlust aus. Gescannt wird blind, ohne Klick ins Feld, und bewiesen an der Datenbank.
test('Inventur: Scans während einer laufenden Anfrage werden gezählt', async ({ page }) => {
	await uiLogin(page);
	const suffix = uniqueSuffix();
	const barcodes = ['A', 'B', 'C'].map((b) => `B-INVR${b}-${suffix}`);

	try {
		const feld = await starteInventur(page, suffix, barcodes);

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
			.poll(() => erfasst(barcodes), {
				message: 'jeder der drei Scans steht in der Inventur',
				timeout: 10_000
			})
			.toBe('3');
		await expect(feld, 'der nächste Scan landet wieder im Feld').toBeFocused();
		await expect(feld).toHaveValue('');
	} finally {
		raeumeAuf(suffix, barcodes);
	}
});

// Ein Handscanner mit amerikanischer Tastatur tippt an einem Rechner mit deutscher den
// Bindestrich als ß (FACHKONZEPT §1). Die Inventur zählt das Exemplar unter der Nummer, die
// die Theke nachschlägt; sonst sonderte der Abschluss ein Buch aus, das im Regal steht.
test('Inventur: ein Scanner mit amerikanischer Tastatur zählt das Etikett', async ({ page }) => {
	await uiLogin(page);
	const suffix = uniqueSuffix();
	const barcodes = [`B-8${String(Date.now()).slice(-8)}`];

	try {
		await starteInventur(page, suffix, barcodes);
		await scanneWieScanner(page, barcodes[0], { deutscherRechner: true });

		await expect
			.poll(() => erfasst(barcodes), { message: 'das Etikett steht in der Inventur' })
			.toBe('1');
	} finally {
		raeumeAuf(suffix, barcodes);
	}
});
