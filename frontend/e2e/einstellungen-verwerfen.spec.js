import { test, expect } from '@playwright/test';
import {
	uiLogin,
	gehZu,
	menuepunkt,
	einstellungsKategorie,
	uniqueSuffix,
	seedSQL,
	querySQL
} from './helpers.js';

// Eine Kategorie der Einstellungen verliert beim Wechsel der Kategorie oder der Seite, was nicht
// gespeichert ist. Vorher fragt sie wie die Buchmaske (buchmaske-verwerfen.spec.js); ohne Eingabe
// fragt sie nicht. Zwei Formulare in einer Kategorie (Postausgang und Vorlagen unter „Mail")
// zählen beide.
const s = uniqueSuffix().slice(0, 6);

const frist = () =>
	querySQL(`SELECT wert FROM system_einstellungen WHERE schluessel = 'frist_buch_tage'`);
let FRIST_VORHER = '21';
// Die Einstellungen sind global, und die Suite teilt sich eine Datenbank.
test.beforeAll(() => {
	FRIST_VORHER = frist() || '21';
});
test.afterAll(() => {
	seedSQL(
		`UPDATE system_einstellungen SET wert = '${FRIST_VORHER}' WHERE schluessel = 'frist_buch_tage'`
	);
});

/** @param {import('@playwright/test').Page} page */
const rueckfrage = (page) => page.getByRole('dialog', { name: 'Ungespeicherte Änderungen' });

/** @param {import('@playwright/test').Page} page @param {string} titel */
const kategorieOffen = (page, titel) =>
	expect(page.getByRole('heading', { level: 2, name: titel, exact: true })).toBeVisible();

test('Kategorie wechseln fragt vor dem Verwerfen; „Bleiben" lässt die Eingabe stehen', async ({
	page
}) => {
	await uiLogin(page);
	await gehZu(page, '/einstellungen');
	await einstellungsKategorie(page, 'Schule').click();
	await kategorieOffen(page, 'Schule');
	const name = page.getByLabel('Name der Schule');
	const vorher = await name.inputValue();

	// Ohne Eingabe wechselt die Kategorie ohne Frage.
	await einstellungsKategorie(page, 'Ausleihe & Fristen').click();
	await kategorieOffen(page, 'Ausleihe & Fristen');
	await expect(rueckfrage(page)).toHaveCount(0);

	await einstellungsKategorie(page, 'Schule').click();
	await name.fill(`Probe ${s}`);
	await einstellungsKategorie(page, 'Ausleihe & Fristen').click();
	await expect(rueckfrage(page)).toBeVisible();
	await rueckfrage(page).getByRole('button', { name: 'Bleiben' }).click();
	await expect(rueckfrage(page)).toHaveCount(0);
	await kategorieOffen(page, 'Schule');
	await expect(name).toHaveValue(`Probe ${s}`);

	await einstellungsKategorie(page, 'Ausleihe & Fristen').click();
	await rueckfrage(page).getByRole('button', { name: 'Verwerfen' }).click();
	await kategorieOffen(page, 'Ausleihe & Fristen');
	await einstellungsKategorie(page, 'Schule').click();
	await expect(name, 'gespeichert ist nichts').toHaveValue(vorher);
});

test('Eine geänderte Mail-Vorlage hält den Wechsel der Seite an', async ({ page }) => {
	await uiLogin(page);
	await gehZu(page, '/einstellungen');
	await einstellungsKategorie(page, 'Mail').click();
	await kategorieOffen(page, 'Mail');
	const betreff = page.getByLabel('Betreff');
	await betreff.fill(`Betreff ${s}`);

	await menuepunkt(page, 'Ausleihe').click();
	await expect(rueckfrage(page)).toBeVisible();
	await rueckfrage(page).getByRole('button', { name: 'Bleiben' }).click();
	await expect(page).toHaveURL(/\/einstellungen$/);
	await expect(betreff).toHaveValue(`Betreff ${s}`);

	await menuepunkt(page, 'Ausleihe').click();
	await rueckfrage(page).getByRole('button', { name: 'Verwerfen' }).click();
	await expect(page).toHaveURL(/\/kiosk$/);

	// Das zweite Formular derselben Kategorie hält den Wechsel ebenso an.
	await gehZu(page, '/einstellungen');
	await einstellungsKategorie(page, 'Mail').click();
	await kategorieOffen(page, 'Mail');
	await page.getByLabel('SMTP Host').fill(`smtp-${s}.example`);
	await menuepunkt(page, 'Ausleihe').click();
	await expect(rueckfrage(page)).toBeVisible();
	await rueckfrage(page).getByRole('button', { name: 'Verwerfen' }).click();
	await expect(page).toHaveURL(/\/kiosk$/);
});

// Speichern baut die Kategorie aus dem gespeicherten Stand neu auf (KategorieDetail). Danach ist
// nichts mehr ungespeichert, und erst die nächste Eingabe hält den Wechsel wieder an.
test('Nach dem Speichern fragt die Kategorie erst bei der nächsten Eingabe wieder', async ({
	page
}) => {
	await uiLogin(page);
	await gehZu(page, '/einstellungen');
	await einstellungsKategorie(page, 'Ausleihe & Fristen').click();
	const tage = page.getByLabel('Tage / Buch');
	const gespeichert = String(Number(FRIST_VORHER) + 1);
	await tage.fill(gespeichert);
	// Die Marke am alten Feld: Ist sie weg, steht die Kategorie neu aufgebaut da.
	await tage.evaluate((el) => el.setAttribute('data-vor-dem-speichern', ''));
	await page.getByRole('button', { name: 'Ausleihe & Fristen speichern' }).click();
	await expect(
		page.locator('[data-vor-dem-speichern]'),
		'Speichern baut die Kategorie aus dem gespeicherten Stand neu auf'
	).toHaveCount(0);
	expect(frist()).toBe(gespeichert);

	await einstellungsKategorie(page, 'Schule').click();
	await kategorieOffen(page, 'Schule');
	await expect(rueckfrage(page)).toHaveCount(0);

	await einstellungsKategorie(page, 'Ausleihe & Fristen').click();
	await expect(tage).toHaveValue(gespeichert);
	await tage.fill(String(Number(FRIST_VORHER) + 2));
	await einstellungsKategorie(page, 'Schule').click();
	await expect(rueckfrage(page)).toBeVisible();
	await rueckfrage(page).getByRole('button', { name: 'Verwerfen' }).click();
	await kategorieOffen(page, 'Schule');
});
