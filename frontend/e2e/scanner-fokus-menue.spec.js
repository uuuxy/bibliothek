import { test, expect } from '@playwright/test';
import { uiLogin, apiPost, seedSQL, querySQL, uniqueSuffix } from './helpers.js';

// Nach einem Wechsel zur Ausleihe muss der nächste Scan im Scanfeld landen.
//
// Nachgestellt am 14.09.2026: Ein Klick auf den Menüpunkt „Ausleihe" zog den Fokus auf den
// Knopf in der Seitenleiste. Die Omnibox fokussiert das Scanfeld nur bei einem
// Zustandswechsel neu (und nur, solange kein Schüler geladen ist), einen globalen
// Tastenfang gibt es nicht. Der folgende Scan lief ohne Meldung ins Leere, und das Enter
// des Scanners löste den Menüknopf erneut aus. kiosk-scannerfokus.spec.js vermeidet genau
// diesen Klick.
//
// Getippt wird wie dort blind über page.keyboard — nie mit fill() oder einem Klick ins
// Feld, sonst prüft der Test nichts.

/**
 * Legt einen Schüler und ein ausleihbares Buch an.
 * @param {import('@playwright/test').Page} page
 * @param {string} name Teil des Nachnamens und des Buch-Barcodes
 */
async function seedSchuelerUndBuch(page, name) {
	const suffix = uniqueSuffix();
	const created = await apiPost(page, '/api/schueler', {
		geburtsdatum: '2012-06-15',
		vorname: 'E2E',
		nachname: `${name}-${suffix}`,
		klasse: '7A',
		barcode_id: `S-${suffix}`
	});
	expect(created.ok(), `Schüler-Seeding: ${created.status()}`).toBeTruthy();
	seedSQL(`
        WITH t AS (
            INSERT INTO buecher_titel (titel) VALUES ('E2E-${name}-${suffix}') RETURNING id
        )
        INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
        SELECT id, 'B-${name}-${suffix}', true FROM t;
    `);
	return { schueler: `S-${suffix}`, nachname: `${name}-${suffix}`, buch: `B-${name}-${suffix}` };
}

/**
 * Tippt blind ins Dokument und schließt mit Enter ab — wie ein Handscanner.
 * @param {import('@playwright/test').Page} page
 * @param {string} code
 */
async function scanne(page, code) {
	await page.keyboard.type(code, { delay: 5 });
	await page.keyboard.press('Enter');
}

/**
 * Wartet, bis das Scanfeld den Fokus hat. Pollend wie in kiosk-scannerfokus.spec.js: Die
 * Rückgabe des Fokus läuft erst nach dem Rendern der Ausleihe. Ohne diese Probe tippte der
 * Test schneller los, als ein Mensch nach einem Mausklick zum Scanner greift.
 * @param {import('@playwright/test').Page} page
 * @param {string} wann
 */
async function erwarteFokusImScanfeld(page, wann) {
	await expect
		.poll(() => page.evaluate(() => document.activeElement?.id ?? ''), {
			message: `Fokus ${wann} nicht im Scanfeld — ein Handscanner tippt danach ins Nichts`,
			timeout: 5000
		})
		.toBe('omnibox-input');
}

/**
 * Wartet, bis das Buch als offene Ausleihe in der Datenbank steht.
 * @param {string} buch
 */
async function erwarteAusgeliehen(buch) {
	await expect
		.poll(
			() =>
				querySQL(
					`SELECT count(*) FROM ausleihen a
                     JOIN buecher_exemplare e ON e.id = a.exemplar_id
                     WHERE a.rueckgabe_am IS NULL AND e.barcode_id = '${buch}';`
				),
			{ message: 'Das Buch wurde nicht verbucht — der Scan ist ins Leere gelaufen' }
		)
		.toBe('1');
}

test('Handscanner: nach einem Klick auf „Ausleihe" landet der Scan im Scanfeld', async ({
	page
}) => {
	await uiLogin(page);
	const { schueler, nachname, buch } = await seedSchuelerUndBuch(page, 'Menue');

	await expect(page.getByPlaceholder(/scannen/i).first()).toBeVisible();
	await page.getByRole('button', { name: 'Ausleihe', exact: true }).click();
	await erwarteFokusImScanfeld(page, 'nach dem Klick auf „Ausleihe"');

	await scanne(page, schueler);
	await expect(
		page.getByText(nachname).first(),
		'Der Schüler-Scan nach dem Menüklick ist ins Leere gelaufen'
	).toBeVisible({ timeout: 5000 });
	// Wie in kiosk-scannerfokus.spec.js: Der Schüler-Scan nimmt dem Feld den Fokus und gibt ihn
	// erst nach dem Rendern zurück (in der vollen Suite am 14.09.2026 sonst ein Buchscan ohne
	// seine ersten Zeichen).
	await erwarteFokusImScanfeld(page, 'nach dem Schüler-Scan');

	await scanne(page, buch);
	await erwarteAusgeliehen(buch);
});

test('Handscanner: mit geladenem Schüler aus einem anderen Bereich zurück zur Ausleihe', async ({
	page
}) => {
	await uiLogin(page);
	const { schueler, nachname, buch } = await seedSchuelerUndBuch(page, 'Zurueck');

	await expect(page.getByPlaceholder(/scannen/i).first()).toBeVisible();
	await scanne(page, schueler);
	await expect(page.getByText(nachname).first()).toBeVisible();

	const anderesZiel = page.getByRole('button', { name: 'Signaturen', exact: true });
	await anderesZiel.click();
	await expect(anderesZiel).toHaveAttribute('aria-current', 'page');

	await page.getByRole('button', { name: 'Ausleihe', exact: true }).click();
	// Der Schüler bleibt geladen — sonst würde der Buchscan aus einem anderen Grund scheitern.
	await expect(page.getByText(nachname).first()).toBeVisible();
	await erwarteFokusImScanfeld(page, 'nach der Rückkehr zur Ausleihe');

	await scanne(page, buch);
	await erwarteAusgeliehen(buch);
});
