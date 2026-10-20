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

/** Die Sperre bei angehaltener Uhr: Ihre Zeitgeber laufen erst mit runFor oder resume.
 * @param {import('@playwright/test').Page} page */
async function sperreBeiAngehaltenerUhr(page) {
	const jetzt = await page.evaluate(() => Date.now());
	await page.clock.pauseAt(jetzt + (15 * 60 + 10) * 1000);
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
	await sperreBeiAngehaltenerUhr(page);

	// Die Rückfrage steht offen, bevor ein Zeitgeber der Sperre gelaufen ist. Laufen sie danach,
	// behält sie den Fokus (Material 3, Dialogs: „focus should automatically land on the first
	// interactive element within the dialog").
	await abmeldenAmSperrbildschirm(page).click();
	await expect(rueckfrage(page)).toBeVisible();
	const bleiben = rueckfrage(page).getByRole('button', { name: 'Bleiben' });
	await expect(bleiben).toBeFocused();
	await page.clock.runFor(1000);
	await expect(bleiben).toBeFocused();
	await page.clock.resume();
	await bleiben.click();
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

test('eine Ablehnung des Passworts nimmt der offenen Rückfrage den Fokus nicht', async ({
	page
}) => {
	await anmeldenMitUhr(page);
	await neuesBuchMitEingabe(page);
	await sperreBeiAngehaltenerUhr(page);
	// Die Antwort kommt erst, wenn die Rückfrage offen ist, und zählt am Server keinen Fehlversuch.
	/** @type {(wert?: unknown) => void} */
	let frei = () => {};
	const freigabe = new Promise((fertig) => (frei = fertig));
	await page.route('**/api/auth/entsperren', async (route) => {
		await freigabe;
		await route.fulfill({ status: 429 });
	});
	const passwort = page.locator('#sperre-passwort');

	await passwort.fill('falsch');
	await page.getByRole('button', { name: 'Entsperren' }).click();
	await expect(passwort, 'die Prüfung läuft').toBeDisabled();
	await abmeldenAmSperrbildschirm(page).click();
	const bleiben = rueckfrage(page).getByRole('button', { name: 'Bleiben' });
	await expect(bleiben).toBeFocused();
	frei();
	await expect(passwort, 'die Prüfung ist vorbei').toBeEnabled();
	await page.clock.runFor(1000);
	await expect(bleiben).toBeFocused();

	// Ohne offene Rückfrage steht der Cursor nach der Ablehnung wieder im Feld.
	await page.clock.resume();
	await bleiben.click();
	await passwort.fill('falsch');
	await page.getByRole('button', { name: 'Entsperren' }).click();
	await expect(passwort).toBeFocused();
});

test('ohne Ungespeichertes meldet der Sperrbildschirm ohne Frage ab', async ({ page }) => {
	await anmeldenMitUhr(page);
	await sperre(page);

	await abmeldenAmSperrbildschirm(page).click();
	await expect(page.locator('#login-email')).toBeVisible();
	await expect(rueckfrage(page)).toHaveCount(0);
});
