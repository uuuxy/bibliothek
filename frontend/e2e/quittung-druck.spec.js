import { test, expect } from '@playwright/test';
import { uiLogin, gehZu, seedSQL, uniqueSuffix } from './helpers.js';

// Die Ausleih-Quittung erscheint, wenn die offene Leserakte über den Browser gedruckt wird; einen
// eigenen Knopf hat sie nicht (styles/druck-ausweise.css). Auf dem Papier steht nur sie: Was am
// Bildschirm außerhalb eines .no-print-Bereichs steht, druckt unter ihr mit, auch die offene
// Maske „Stammdaten bearbeiten".

/**
 * Ob die Quittung im Druck steht, und jeder Text, der dort außerhalb von ihr zu sehen ist.
 * @param {import('@playwright/test').Page} page
 */
async function druckseite(page) {
	await page.emulateMedia({ media: 'print' });
	const lage = await page.evaluate(() => {
		const sichtbar = (/** @type {Element} */ e) => {
			const r = e.getBoundingClientRect();
			// Unsichtbare Beschriftungen für Screenreader messen 1 × 1 px.
			return r.width > 1 && r.height > 1 && getComputedStyle(e).visibility !== 'hidden';
		};
		const quittung = document.querySelector('.print-receipt-section');
		const fremd = [];
		for (const e of document.querySelectorAll('body *')) {
			if (e.children.length > 0 || !e.textContent?.trim()) continue;
			if (sichtbar(e) && !quittung?.contains(e)) fremd.push(e.textContent.trim().slice(0, 40));
		}
		return { quittung: !!quittung && sichtbar(quittung), fremd };
	});
	await page.emulateMedia({ media: 'screen' });
	return lage;
}

test.describe('Ausleih-Quittung: der Druck der Leserakte trägt nur die Quittung', () => {
	const s = uniqueSuffix();
	const ausweis = `E2E-QD-${s}`;
	const vorname = `Quittung${s}`;

	test.beforeAll(() => {
		seedSQL(`
			INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
			VALUES ('${ausweis}', '${vorname}', 'Druckprobe', '07a', EXTRACT(YEAR FROM CURRENT_DATE)::int + 3);
		`);
	});

	test.afterAll(() => {
		seedSQL(`DELETE FROM schueler WHERE barcode_id = '${ausweis}';`);
	});

	test('mit offener Akte und mit offener Maske „Stammdaten bearbeiten"', async ({ page }) => {
		await uiLogin(page);
		await gehZu(page, '/schuelerdatei');
		await page.getByRole('searchbox', { name: 'Leser suchen' }).fill(vorname);
		await page.getByRole('button', { name: new RegExp(`Profil von ${vorname} `) }).click();
		await page.getByRole('tab', { name: 'Stammdaten & Adresse' }).click();

		const akte = await druckseite(page);
		expect(akte.quittung, 'Akte: Die Quittung steht nicht im Druck.').toBe(true);
		expect(akte.fremd, 'Akte: Neben der Quittung druckt Text der Oberfläche mit.').toEqual([]);

		await page.getByRole('button', { name: 'Bearbeiten' }).click();
		await page.locator('#ort').waitFor();
		const maske = await druckseite(page);
		expect(maske.quittung, 'Maske: Die Quittung steht nicht im Druck.').toBe(true);
		expect(maske.fremd, 'Maske: Neben der Quittung druckt Text der Oberfläche mit.').toEqual([]);
	});
});
