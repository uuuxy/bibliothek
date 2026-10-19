import { test, expect } from '@playwright/test';
import { uiLogin, uniqueSuffix, ADMIN_EMAIL, ADMIN_PASSWORD, menuepunkt } from './helpers.js';

// Abmelden verwirft, was in einer Maske ungespeichert steht. Vorher fragt die Anwendung, in der
// Seitenleiste wie am Sperrbildschirm („Abmelden und als andere Person anmelden"); dort fragt der
// Bildschirm selbst, weil hinter ihm keine Rückfrage der Anwendung zu sehen ist.
const s = uniqueSuffix().slice(0, 6);
const TITEL = `Abmeldeprobe ${s}`;

/** @param {import('@playwright/test').Page} page */
const rueckfrage = (page) => page.getByRole('dialog', { name: 'Ungespeicherte Änderungen' });

/** Anmelden mit gestellter Uhr; die Fristen der Sperre sind geladen.
 * @param {import('@playwright/test').Page} page */
async function anmeldenMitUhr(page) {
	await page.clock.install();
	const fristen = page.waitForResponse(
		(r) => new URL(r.url()).pathname === '/api/einstellungen/sitzung'
	);
	await uiLogin(page);
	await fristen;
}

/** @param {import('@playwright/test').Page} page */
async function neuesBuchMitEingabe(page) {
	await menuepunkt(page, 'Medienkatalog').click();
	await page.getByRole('tab', { name: 'Titel-Verwaltung' }).click();
	await page.getByRole('button', { name: 'Neues Buch' }).first().click();
	await page.locator('#buch-titel').fill(TITEL);
}

/** @param {import('@playwright/test').Page} page */
async function sperre(page) {
	await page.clock.fastForward('15:10');
	await expect(page.getByTestId('sperrbildschirm')).toBeVisible();
}

/** @param {import('@playwright/test').Page} page */
const abmeldenAmSperrbildschirm = (page) =>
	page.getByRole('button', { name: 'Abmelden und als andere Person anmelden' });

test('Abmelden in der Seitenleiste fragt; die nächste Anmeldung beginnt an der Theke', async ({
	page
}) => {
	await uiLogin(page);
	await neuesBuchMitEingabe(page);

	await page.getByRole('button', { name: 'Abmelden' }).click();
	await rueckfrage(page).getByRole('button', { name: 'Bleiben' }).click();
	await expect(page.locator('#buch-titel')).toHaveValue(TITEL);

	await page.getByRole('button', { name: 'Abmelden' }).click();
	await rueckfrage(page).getByRole('button', { name: 'Verwerfen und abmelden' }).click();
	await expect(page.locator('#login-email')).toBeVisible();

	// Ohne Neuladen wieder anmelden: Der verworfene Bildschirm hält den Wechsel zur Theke nicht an.
	// Erst fokussieren, dann füllen, wie uiLogin: Die Bindungen stehen kurz nach dem Aufbau.
	for (const [feld, wert] of [
		['#login-email', ADMIN_EMAIL],
		['#login-password', ADMIN_PASSWORD]
	]) {
		await page.locator(feld).click();
		await page.locator(feld).fill(wert);
		await expect(page.locator(feld)).toHaveValue(wert);
	}
	await page.getByRole('button', { name: 'Anmelden' }).click();
	await expect(page.getByPlaceholder('Scannen oder Namen eingeben')).toBeVisible();
	await expect(rueckfrage(page)).toHaveCount(0);
});

test('der Sperrbildschirm fragt vor dem Abmelden, wenn dahinter etwas ungespeichert ist', async ({
	page
}) => {
	await anmeldenMitUhr(page);
	await neuesBuchMitEingabe(page);
	await sperre(page);

	await abmeldenAmSperrbildschirm(page).click();
	await expect(rueckfrage(page)).toBeVisible();
	await expect(rueckfrage(page).getByRole('button', { name: 'Bleiben' })).toBeFocused();
	await rueckfrage(page).getByRole('button', { name: 'Bleiben' }).click();
	await expect(page.getByTestId('sperrbildschirm')).toBeVisible();

	await page.locator('#sperre-passwort').fill(ADMIN_PASSWORD);
	await page.getByRole('button', { name: 'Entsperren' }).click();
	// Erst wenn die Sperre weg ist, läuft ihre Uhr wieder (#schliesseAuf). Wer vorher vorspult,
	// spult ins Leere: Die Antwort des Servers plant die 15 Minuten danach von vorn.
	await expect(page.getByTestId('sperrbildschirm')).toHaveCount(0);
	await expect(page.locator('#buch-titel'), 'die Eingabe steht noch').toHaveValue(TITEL);

	await sperre(page);
	await abmeldenAmSperrbildschirm(page).click();
	await rueckfrage(page).getByRole('button', { name: 'Verwerfen und abmelden' }).click();
	await expect(page.locator('#login-email')).toBeVisible();
});

test('ohne Ungespeichertes meldet der Sperrbildschirm ohne Frage ab', async ({ page }) => {
	await anmeldenMitUhr(page);
	await sperre(page);

	await abmeldenAmSperrbildschirm(page).click();
	await expect(page.locator('#login-email')).toBeVisible();
	await expect(rueckfrage(page)).toHaveCount(0);
});
