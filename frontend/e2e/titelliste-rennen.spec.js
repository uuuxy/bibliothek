import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix } from './helpers.js';

// Die Titel-Verwaltung lädt beim Öffnen die ganze Titelliste; an einem großen Bestand dauert
// das. Was in dieser Zeit geschieht, darf die späte Antwort nicht ersetzen: ein Suchergebnis
// (die kleine Abfrage ist vor der ersten fertig) und einen Titel, der inzwischen gespeichert
// wurde. Mit wenigen Titeln ist die erste Abfrage schneller als jede Eingabe, deshalb wird sie
// hier verzögert oder festgehalten.
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

// Die Antwort des laufenden Abrufs kennt den eben gespeicherten Titel nicht. Ersetzte sie die
// Liste, verschwände er, bis die Seite neu lädt.
test('ein neuer Titel bleibt stehen, wenn die ganze Liste erst danach ankommt', async ({
	page
}) => {
	const neu = `E2E Späte Liste ${uniqueSuffix()}`;
	/** @type {(wert?: unknown) => void} */
	let gibFrei = () => {};
	const freigabe = new Promise((fertig) => (gibFrei = fertig));
	/** @type {(wert?: unknown) => void} */
	let ganzeListeDa = () => {};
	const ganzeListe = new Promise((fertig) => (ganzeListeDa = fertig));
	let gehalten = 0;
	try {
		await uiLogin(page);
		// Die erste Antwort der Liste wird festgehalten, bis der neue Titel gespeichert ist.
		await page.route(/\/api\/books(\?.*)?$/, async (route) => {
			if (route.request().method() !== 'GET' || gehalten > 0) return route.continue();
			gehalten++;
			const antwort = await route.fetch();
			await freigabe;
			await route.fulfill({ response: antwort });
			ganzeListeDa();
		});
		await page.goto('/medienkatalog');
		await page.getByRole('tab', { name: 'Titel-Verwaltung' }).click();
		await page.getByRole('button', { name: 'Neues Buch' }).first().click();
		await page.locator('#buch-titel').fill(neu);
		await page.locator('#buch-signatur').fill('BIB Spae');
		const gespeichert = page.waitForResponse(
			(r) => r.request().method() === 'POST' && /\/api\/books$/.test(r.url())
		);
		await page.getByRole('button', { name: 'Speichern' }).click();
		expect((await gespeichert).status(), 'Der neue Titel wurde nicht gespeichert').toBe(201);
		expect(gehalten, 'Die Liste war schon da: Es gab keine späte Antwort').toBe(1);
		// Gespeichert steht der Titel oben in der Liste, noch bevor sie ganz geladen ist.
		await expect(page.getByText(neu).first()).toBeVisible({ timeout: 15_000 });

		gibFrei();
		await ganzeListe;
		// Die Seite hat die Antwort verarbeitet, wenn die Liste auch den Titel von oben zählt.
		await expect
			.poll(
				async () => {
					const kopf = await page.getByRole('heading', { name: /Bücher \(\d+\)/ }).textContent();
					return Number(kopf?.match(/\((\d+)\)/)?.[1] ?? 0);
				},
				{
					timeout: 15_000,
					message:
						'Nach der späten Antwort zählt die Liste nicht beide Titel, den vorhandenen und den neuen.'
				}
			)
			.toBeGreaterThanOrEqual(2);
		await expect(
			page.getByText(neu).first(),
			'Der neue Titel steht nicht mehr in der Liste: Die späte Antwort des ersten Abrufs hat ihn ersetzt.'
		).toBeVisible();
	} finally {
		gibFrei();
		seedSQL(`
			DELETE FROM buecher_exemplare WHERE titel_id IN (SELECT id FROM buecher_titel WHERE titel = '${neu}');
			DELETE FROM buecher_titel WHERE titel = '${neu}';`);
	}
});
