import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix } from './helpers.js';

// Die Titel-Verwaltung lädt beim Öffnen die ganze Titelliste. Wer sofort sucht, stößt eine
// zweite, kleine Abfrage an, und die ist vor der ersten fertig. Kam die ganze Liste danach an,
// stand sie unter dem Suchwort: Das Suchfeld zeigte den Titel, die Liste alle. Mit wenigen
// Titeln ist die erste Abfrage schneller als das Tippen, deshalb fiel es erst an einem großen
// Bestand auf — hier wird sie verzögert.
const s = uniqueSuffix().slice(0, 8);
const TITEL = `E2E Suchrennen ${s}`;

test.beforeAll(() => {
	seedSQL(`
		WITH t AS (
			INSERT INTO buecher_titel (titel, autor) VALUES ('${TITEL}', 'E2E') RETURNING id
		)
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
		SELECT id, 'B-SR-${s}', true FROM t;`);
});
test.afterAll(() => {
	seedSQL(`
		DELETE FROM buecher_exemplare WHERE barcode_id = 'B-SR-${s}';
		DELETE FROM buecher_titel WHERE titel = '${TITEL}';`);
});

test('ein Suchergebnis bleibt stehen, wenn die ganze Liste erst danach ankommt', async ({
	page
}) => {
	/** @type {(wert?: unknown) => void} */
	let ganzeListeDa = () => {};
	const ganzeListe = new Promise((fertig) => (ganzeListeDa = fertig));
	await page.route(/\/api\/books$/, async (route) => {
		if (route.request().method() !== 'GET') return route.continue();
		await new Promise((r) => setTimeout(r, 1500));
		await route.continue();
		ganzeListeDa();
	});

	await uiLogin(page);
	await page.goto('/medienkatalog');
	await page.getByRole('tab', { name: 'Titel-Verwaltung' }).click();
	await page.getByRole('searchbox', { name: 'Bücher durchsuchen' }).fill(TITEL);
	await expect(page.getByRole('heading', { name: 'Bücher (1)' })).toBeVisible();

	// Erst jetzt kommt die ganze Liste an. Sie gehört zu keinem Suchwort mehr.
	await ganzeListe;
	await page.waitForTimeout(1000);
	await expect(page.getByRole('heading', { name: 'Bücher (1)' })).toBeVisible();
	await expect(page.getByText(TITEL).first()).toBeVisible();
});
