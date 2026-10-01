import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix } from './helpers.js';

// Der Bestand in der Buchmaske: Er geht nur zum Server, wenn jemand das Feld geändert hat,
// zusammen mit der Zahl vom Öffnen; weicht die vom Stand ab, lehnt der Server ab. Vorher
// schrieb jede Maske ihre Zahl von vorhin zurück, und der Server sonderte aus, was ein
// anderer Platz angelegt hatte oder was bestellt war — gemeldet wurde „gespeichert".
const s = uniqueSuffix().slice(0, 6);
const kern = ('978' + String(Date.now()).slice(-9)).slice(0, 12);
const ISBN =
	kern + ((10 - ([...kern].reduce((a, d, i) => a + Number(d) * (i % 2 ? 3 : 1), 0) % 10)) % 10);
const TITEL = `Bestandsmaske ${s}`;

/** Exemplare des Titels als „im Bestand/im Zulauf/ausgesondert". */
const stand = () =>
	querySQL(`
		SELECT count(*) FILTER (WHERE NOT e.ist_ausgesondert AND e.bestellstatus IS NULL) || '/' ||
		       count(*) FILTER (WHERE NOT e.ist_ausgesondert AND e.bestellstatus IS NOT NULL) || '/' ||
		       count(*) FILTER (WHERE e.ist_ausgesondert)
		FROM buecher_exemplare e JOIN buecher_titel t ON t.id = e.titel_id WHERE t.isbn = '${ISBN}'`);
const signatur = () => querySQL(`SELECT signatur FROM buecher_titel WHERE isbn = '${ISBN}'`);

/** @param {string} nummer @param {string} [spalten] @param {string} [werte] */
const legeExemplarAn = (nummer, spalten = '', werte = '') =>
	seedSQL(`
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar${spalten})
		SELECT id, 'B-BM-${nummer}-${s}', ${spalten ? 'false' : 'true'}${werte} FROM buecher_titel WHERE isbn = '${ISBN}';`);

/** @param {import('@playwright/test').Page} page */
async function zurTitelVerwaltung(page) {
	await uiLogin(page);
	await page.goto('/medienkatalog');
	await page.getByRole('tab', { name: 'Titel-Verwaltung' }).click();
}

/** @param {import('@playwright/test').Page} page @param {string} bestand */
async function oeffneTitel(page, bestand) {
	await zurTitelVerwaltung(page);
	await page.getByRole('searchbox', { name: 'Bücher durchsuchen' }).fill(TITEL);
	await page.getByText(TITEL).first().click();
	await expect(page.getByRole('heading', { name: 'Buch bearbeiten' })).toBeVisible();
	await expect(page.locator('#buch-bestand')).toHaveValue(bestand);
}

/** @param {import('@playwright/test').Page} page */
async function speichern(page) {
	await page.getByRole('button', { name: 'Speichern' }).click();
}

test.describe.serial('Buchmaske: Bestand', () => {
	test.beforeAll(() => {
		seedSQL(`
			INSERT INTO buecher_titel (titel, autor, isbn, signatur, medientyp, cover_url)
			VALUES ('${TITEL}', 'E2E', '${ISBN}', 'Bes 1', 'Buch', '/covers/e2e-dummy.jpg');`);
		legeExemplarAn('1');
	});
	test.afterAll(() => {
		seedSQL(`
			DELETE FROM buecher_exemplare WHERE titel_id IN (SELECT id FROM buecher_titel WHERE isbn = '${ISBN}');
			DELETE FROM buecher_titel WHERE isbn = '${ISBN}';`);
	});

	test('zwei Plätze: wer nur die Signatur speichert, lässt den Bestand des anderen stehen', async ({
		browser
	}) => {
		const platz1 = await (await browser.newContext()).newPage();
		const platz2 = await (await browser.newContext()).newPage();
		await oeffneTitel(platz1, '1');
		await oeffneTitel(platz2, '1');

		await platz2.locator('#buch-bestand').fill('6');
		await speichern(platz2);
		await expect(platz2.getByText('Buch erfolgreich gespeichert!')).toBeVisible();
		expect(stand()).toBe('6/0/0');

		await platz1.locator('#buch-signatur').fill('Bes 2');
		await speichern(platz1);
		await expect(platz1.getByText('Buch erfolgreich gespeichert!')).toBeVisible();
		expect(stand()).toBe('6/0/0');
		expect(signatur()).toBe('Bes 2');
	});

	test('inzwischen geändert: abgelehnt, das Feld zeigt den neuen Stand, die Maske bleibt', async ({
		page
	}) => {
		await oeffneTitel(page, '6');
		legeExemplarAn('7');
		await page.locator('#buch-signatur').fill('Bes 3');
		await page.locator('#buch-bestand').fill('2');
		await speichern(page);
		const frage = page.getByRole('dialog').filter({ hasText: 'Bestand von 6 auf 2 verringern?' });
		await frage.getByRole('button', { name: 'Verringern' }).click();

		await expect(page.getByText(/jetzt 7 statt 6/)).toBeVisible();
		await expect(page.locator('#buch-bestand')).toHaveValue('7');
		await expect(page.locator('#buch-signatur')).toHaveValue('Bes 3');
		await expect(page.getByRole('heading', { name: 'Buch bearbeiten' })).toBeVisible();
		expect(stand()).toBe('7/0/0');
		expect(signatur()).toBe('Bes 2');
	});

	test('die Rückfrage kommt auch, wenn der Titel nicht in der geladenen Liste steht', async ({
		page
	}) => {
		await zurTitelVerwaltung(page);
		await page.getByRole('searchbox', { name: 'Bücher durchsuchen' }).fill(`kein-treffer-${s}`);
		await expect(page.getByRole('heading', { name: 'Bücher (0)' })).toBeVisible();
		await page.getByRole('button', { name: 'Neues Buch' }).first().click();
		await page.locator('#buch-isbn').fill(ISBN);
		await page.locator('#buch-isbn').press('Tab');
		await page
			.getByRole('dialog')
			.filter({ hasText: 'Vorhandenen Titel öffnen?' })
			.getByRole('button', { name: 'Titel öffnen' })
			.click();
		await expect(page.getByRole('heading', { name: 'Buch bearbeiten' })).toBeVisible();
		await expect(page.locator('#buch-bestand')).toHaveValue('7');

		await page.locator('#buch-bestand').fill('5');
		await speichern(page);
		const frage = page.getByRole('dialog').filter({ hasText: 'Bestand von 7 auf 5 verringern?' });
		await expect(frage).toContainText('2 Exemplare werden ausgesondert');
		await frage.getByRole('button', { name: 'Abbrechen' }).click();
		expect(stand()).toBe('7/0/0');

		await speichern(page);
		await frage.getByRole('button', { name: 'Verringern' }).click();
		await expect(page.getByText('Buch erfolgreich gespeichert!')).toBeVisible();
		expect(stand()).toBe('5/0/2');
	});

	// Die Antwort auf das Speichern ist der gespeicherte Titel samt Bestand. Aus den gesendeten
	// Angaben stand in der Zeile nach jedem Speichern eine 0, bis die Liste neu geladen wurde.
	test('die Zeile der Titelliste zeigt nach dem Speichern den Bestand', async ({ page }) => {
		await oeffneTitel(page, '5');
		await page.locator('#buch-signatur').fill('Bes 5');
		await speichern(page);
		await expect(page.getByText('Buch erfolgreich gespeichert!')).toBeVisible();
		const zellen = page.locator('tr', { hasText: TITEL }).first().locator('td');
		await expect(zellen.nth((await zellen.count()) - 2)).toHaveText('5');
	});

	test('ein geleertes Feld lässt den Bestand, wie er ist, und sagt es', async ({ page }) => {
		await oeffneTitel(page, '5');
		await page.locator('#buch-bestand').fill('');
		await expect(page.locator('#buch-bestand-hinweis')).toHaveText(
			'Ohne Zahl bleibt der Bestand, wie er ist.'
		);
		await speichern(page);
		await expect(page.getByText('Buch erfolgreich gespeichert!')).toBeVisible();
		expect(stand()).toBe('5/0/2');
	});

	test('bestellte Exemplare bleiben beim Speichern unberührt', async ({ page }) => {
		legeExemplarAn('Z1', ', bestellstatus', `, 'bestellt'`);
		legeExemplarAn('Z2', ', bestellstatus', `, 'im_zulauf'`);
		await oeffneTitel(page, '5');
		await page.locator('#buch-signatur').fill('Bes 4');
		await speichern(page);
		await expect(page.getByText('Buch erfolgreich gespeichert!')).toBeVisible();
		expect(stand()).toBe('5/2/2');
		expect(signatur()).toBe('Bes 4');
	});

	// „Exemplar löschen" sondert aus. Die Maske zählte danach selbst eins herunter, auch bei
	// einem bestellten Exemplar, das im Bestand nie mitzählte — und das nächste Speichern
	// sonderte dafür ein Exemplar aus dem Regal aus.
	test('ein bestelltes Exemplar in der Maske gelöscht: die Zahl im Feld bleibt', async ({
		page
	}) => {
		await oeffneTitel(page, '5');
		await page
			.locator('div')
			.filter({ hasText: new RegExp(`^B-BM-Z1-${s}`) })
			.getByRole('button', { name: 'Exemplar löschen' })
			.click();
		await page.getByRole('dialog').getByRole('button', { name: 'Löschen' }).click();
		await expect(page.getByText('Exemplar erfolgreich gelöscht')).toBeVisible();
		await expect(page.locator('#buch-bestand')).toHaveValue('5');
		expect(stand()).toBe('5/1/3');

		await speichern(page);
		await expect(page.getByText('Buch erfolgreich gespeichert!')).toBeVisible();
		expect(stand()).toBe('5/1/3');
	});
});
