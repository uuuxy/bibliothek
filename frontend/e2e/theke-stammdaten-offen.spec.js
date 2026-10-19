import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix, scanneWieScanner } from './helpers.js';

// An der Theke liegt die Maske „Stammdaten bearbeiten" in der Akte des Lesers. Ein Scan lud die
// Akte neu oder wechselte den Leser, und die Eingaben waren ohne Frage weg; ebenso, wenn ein
// anderer Platz für denselben Leser buchte (Live-Leitung) oder die Schnellrückgabe eingeschaltet
// wurde. Jetzt bucht die Theke nichts, solange die Maske Ungespeichertes hält, und sagt warum;
// die Akte lädt erst neu, wenn die Maske zu ist; die Schnellrückgabe fragt vorher.
const s = uniqueSuffix().slice(0, 6).toUpperCase();
const VORNAME = `Thekoff${s}`;
const AUSWEIS = `S-TO-${s}`;
const ZWEITER = `S-TP-${s}`;
const BUCH = `B-TO-${s}`;
const BUCH_ANDERER_PLATZ = `B-TQ-${s}`;
const GRUND = /Erst die Eingaben in der Maske speichern oder verwerfen/;

/** @param {string} barcode */
const offeneAusleihen = (barcode) =>
	querySQL(
		`SELECT count(*) FROM ausleihen a JOIN buecher_exemplare e ON e.id = a.exemplar_id
		 WHERE e.barcode_id = '${barcode}' AND a.rueckgabe_am IS NULL;`
	);

/** @param {import('@playwright/test').Page} page */
async function oeffneMaskeAnDerTheke(page) {
	await uiLogin(page);
	await expect(page.getByPlaceholder('Scannen oder Namen eingeben')).toBeVisible();
	await scanneWieScanner(page, AUSWEIS);
	await page.getByRole('tab', { name: 'Stammdaten & Adresse' }).click();
	await page.getByRole('button', { name: 'Bearbeiten' }).click();
	await expect(page.locator('#ort')).toHaveValue('Friedrichsdorf');
}

/** @param {import('@playwright/test').Page} page */
const maske = (page) => page.getByRole('heading', { name: 'Schüler bearbeiten' });
/** @param {import('@playwright/test').Page} page */
const rueckfrage = (page) => page.getByRole('dialog', { name: 'Ungespeicherte Änderungen' });
/** @param {import('@playwright/test').Page} page */
const schnellrueckgabe = (page) => page.getByRole('button', { name: 'Schnellrückgabe' });

/** @param {import('@playwright/test').Page} page */
async function verwirfMaske(page) {
	await page.getByRole('button', { name: 'Zurück', exact: true }).click();
	await rueckfrage(page).getByRole('button', { name: 'Verwerfen' }).click();
	await expect(maske(page)).toHaveCount(0);
}

test.describe.serial('Theke: Scans bei offener Stammdaten-Maske', () => {
	test.beforeAll(() => {
		seedSQL(`
			INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr, geburtsdatum,
				strasse, hausnummer, plz, ort, eltern_email)
			VALUES ('${VORNAME}', 'E2E', '07G', '${AUSWEIS}', 2031, '2013-05-06',
				'Altweg', '1', '61381', 'Friedrichsdorf', 'alt@example.org'),
				('Zweit${s}', 'E2E', '07G', '${ZWEITER}', 2031, '2013-07-08',
				'Altweg', '2', '61381', 'Friedrichsdorf', 'zweit@example.org');
			WITH t AS (
				INSERT INTO buecher_titel (titel, ist_lernmittel)
				VALUES ('E2E-Maske offen ${s}', true), ('E2E-Anderer Platz ${s}', true)
				RETURNING id, titel
			)
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
			SELECT id, CASE WHEN titel LIKE 'E2E-Maske%' THEN '${BUCH}' ELSE '${BUCH_ANDERER_PLATZ}' END,
				true FROM t;`);
	});
	test.afterAll(() => {
		seedSQL(`
			DELETE FROM ausleihen WHERE exemplar_id IN (
				SELECT id FROM buecher_exemplare WHERE barcode_id IN ('${BUCH}', '${BUCH_ANDERER_PLATZ}'));
			DELETE FROM buecher_titel WHERE titel LIKE 'E2E-% ${s}';
			DELETE FROM leser WHERE barcode_id IN ('${AUSWEIS}', '${ZWEITER}');`);
	});

	test('ein Scan bei Ungespeichertem bucht nicht und sagt warum', async ({ page }) => {
		await oeffneMaskeAnDerTheke(page);
		await page.locator('#ort').fill('Bad Homburg');
		await maske(page).click();

		await scanneWieScanner(page, BUCH);
		await expect(page.getByText(GRUND)).toBeVisible();
		await expect(maske(page)).toBeVisible();
		await expect(page.locator('#ort')).toHaveValue('Bad Homburg');
		expect(offeneAusleihen(BUCH), 'nichts verliehen').toBe('0');

		// Auch der Ausweis eines anderen Lesers wechselt nicht.
		await maske(page).click();
		await scanneWieScanner(page, ZWEITER);
		await expect(page.getByText(GRUND)).toBeVisible();
		await expect(page.locator('#ort')).toHaveValue('Bad Homburg');
		await expect(page.getByText(`${VORNAME} E2E · ${AUSWEIS}`)).toBeVisible();

		await verwirfMaske(page);
		await scanneWieScanner(page, BUCH);
		await expect.poll(() => offeneAusleihen(BUCH), { message: 'nach dem Verwerfen' }).toBe('1');
	});

	test('ein anderer Platz bucht für denselben Leser: die Maske behält die Eingabe', async ({
		page,
		browser
	}) => {
		await oeffneMaskeAnDerTheke(page);
		await page.locator('#ort').fill('Bad Homburg');

		const kontext = await browser.newContext();
		const andererPlatz = await kontext.newPage();
		await uiLogin(andererPlatz);
		await expect(andererPlatz.getByPlaceholder('Scannen oder Namen eingeben')).toBeVisible();
		await scanneWieScanner(andererPlatz, AUSWEIS);
		await expect(andererPlatz.getByPlaceholder('Buch-Barcode (B-) scannen')).toBeVisible();
		await scanneWieScanner(andererPlatz, BUCH_ANDERER_PLATZ);
		await expect.poll(() => offeneAusleihen(BUCH_ANDERER_PLATZ)).toBe('1');
		await kontext.close();

		// Die Meldung des anderen Platzes kommt über die Live-Leitung in Millisekunden an.
		await page.waitForTimeout(1500);
		await expect(maske(page)).toBeVisible();
		await expect(page.locator('#ort')).toHaveValue('Bad Homburg');

		// Nach dem Schließen holt die Akte das Neuladen nach.
		await verwirfMaske(page);
		await page.getByRole('tab', { name: /Ausleihen & Vormerkungen/ }).click();
		await expect(page.getByRole('table').getByText(`E2E-Anderer Platz ${s}`)).toBeVisible();
	});

	test('die Schnellrückgabe fragt, bevor sie die Akte schließt', async ({ page }) => {
		await oeffneMaskeAnDerTheke(page);
		await page.locator('#ort').fill('Bad Homburg');

		await schnellrueckgabe(page).click();
		await rueckfrage(page).getByRole('button', { name: 'Bleiben' }).click();
		await expect(schnellrueckgabe(page)).toHaveAttribute('aria-pressed', 'false');
		await expect(page.locator('#ort')).toHaveValue('Bad Homburg');

		await schnellrueckgabe(page).click();
		await rueckfrage(page).getByRole('button', { name: 'Verwerfen und weiter' }).click();
		await expect(schnellrueckgabe(page)).toHaveAttribute('aria-pressed', 'true');
		await expect(maske(page)).toHaveCount(0);
		await schnellrueckgabe(page).click();
	});

	test('ohne Ungespeichertes geht der Scan durch, und die Maske bleibt offen', async ({ page }) => {
		await oeffneMaskeAnDerTheke(page);
		await maske(page).click();
		// Das Buch aus dem ersten Fall geht zurück: ein Scan, der wirklich bucht.
		await scanneWieScanner(page, BUCH);
		await expect.poll(() => offeneAusleihen(BUCH)).toBe('0');
		await expect(maske(page)).toBeVisible();
		await expect(page.locator('#ort')).toHaveValue('Friedrichsdorf');
	});
});
