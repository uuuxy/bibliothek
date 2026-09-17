import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix, gehZu } from './helpers.js';

// Punkt 5 des Protokolls vom 16.09.2026: „Schülerdatei ohne Sortierung und Filter". Der
// Filter steht seit dem 17.09.2026 früh, das hier ist die andere Hälfte.
//
// Gemessen wird AM DRAHT, nicht am Ergebnis allein: Eine Sortierung im Browser würde die
// sichtbare Reihenfolge genauso ändern — aber sie säße hinter der Kappung bei 500 Zeilen
// und ordnete dann die ersten 500 der Kartei-Reihenfolge um statt der ersten 500 der
// gewählten Spalte. Das sieht richtig aus und ist es nicht. Deshalb prüft dieser Test,
// dass die Anfrage mit `sortierung=` zum Server geht.

test('Leserdatei: Klick auf einen Spaltenkopf sortiert am Server', async ({ page }) => {
	const s = String(uniqueSuffix()).slice(-6);
	// Klasse und Name laufen gegenläufig: Nach Klasse steht Zyska vorn, nach Name Abele.
	seedSQL(`
		INSERT INTO leser (barcode_id, vorname, nachname, klasse, art, abgaenger_jahr)
		VALUES ('SRT1-${s}', 'Anna',  'Abele${s}', '09G',  'schueler', 2031),
		       ('SRT2-${s}', 'Bernd', 'Zyska${s}', '05F1', 'schueler', 2031);
	`);

	await uiLogin(page);
	await gehZu(page, '/schuelerdatei');

	// Auf die beiden Zeilen eingrenzen, damit die Reihenfolge ablesbar bleibt.
	await page.getByRole('searchbox').first().fill(s);
	await expect(page.getByRole('cell', { name: new RegExp(`Abele${s}`) }).first()).toBeVisible();

	/** @type {string[]} */
	const anfragen = [];
	page.on('request', (r) => anfragen.push(r.url()));

	const nameKopf = page.getByRole('button', { name: /Nach Name .*sortieren/ });
	await nameKopf.click();

	// 1. Die Anfrage trägt die Sortierung — sie ist am Server gelaufen.
	await expect
		.poll(() => anfragen.filter((u) => u.includes('sortierung=name&richtung=auf')).length)
		.toBeGreaterThan(0);

	// 2. Der Kopf meldet den Zustand, den ein Screenreader liest.
	const nameZelle = page.getByRole('columnheader', { name: /Name/ }).first();
	await expect(nameZelle).toHaveAttribute('aria-sort', 'ascending');

	// 3. Und die Reihenfolge stimmt: Abele vor Zyska.
	const namen = async () =>
		(await page.locator('tbody tr').allInnerTexts()).map((z) => z.replace(/\s+/g, ' '));
	await expect
		.poll(async () => {
			const zeilen = await namen();
			return zeilen.findIndex((z) => z.includes(`Abele${s}`)) <
				zeilen.findIndex((z) => z.includes(`Zyska${s}`))
				? 'Abele zuerst'
				: 'Zyska zuerst';
		})
		.toBe('Abele zuerst');

	// Zweiter Klick dreht die Richtung — und sagt es auch.
	await page.getByRole('button', { name: /Nach Name absteigend sortieren/ }).click();
	await expect
		.poll(() => anfragen.filter((u) => u.includes('sortierung=name&richtung=ab')).length)
		.toBeGreaterThan(0);
	await expect(nameZelle).toHaveAttribute('aria-sort', 'descending');
	await expect
		.poll(async () => {
			const zeilen = await namen();
			return zeilen.findIndex((z) => z.includes(`Zyska${s}`)) <
				zeilen.findIndex((z) => z.includes(`Abele${s}`))
				? 'Zyska zuerst'
				: 'Abele zuerst';
		})
		.toBe('Zyska zuerst');

	seedSQL(`DELETE FROM leser WHERE barcode_id LIKE 'SRT%-${s}';`);
});
