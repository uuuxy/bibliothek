import { test, expect } from '@playwright/test';
import { uiLogin, scanneWieScanner, ADMIN_EMAIL, ADMIN_PASSWORD } from './helpers.js';

// Ein Handscanner tippt in das Feld, das den Fokus hat, und am Sperrbildschirm ist das
// das Passwortfeld. Der Scan ging als Passwort zum Server und zählte als Fehlversuch; nach
// fünf Scans war das Konto an diesem Rechner 15 Minuten gesperrt, auch für das richtige
// Passwort. Gescannt wird hier wie an der Theke: blind getippt, ohne Klick ins Feld.
// Der lokale Stack nimmt jedes Passwort an — ohne die Erkennung schlösse der Scan auf.

/** @param {import('@playwright/test').Page} page @param {string} pfad */
function zaehlePost(page, pfad) {
	/** @type {string[]} */
	const versuche = [];
	page.on('request', (r) => {
		if (new URL(r.url()).pathname === pfad && r.method() === 'POST') versuche.push(pfad);
	});
	return versuche;
}

test('Sperrbildschirm: ein Scan geht nicht als Passwort zum Server und drückt keinen Knopf', async ({
	page
}) => {
	const versuche = zaehlePost(page, '/api/auth/entsperren');
	await page.clock.install();
	// Die Fristen kommen nach der Anmeldung vom Server und stellen die Uhr neu; erst danach
	// läuft die Viertelstunde, die hier vorgespult wird.
	const fristen = page.waitForResponse(
		(r) => new URL(r.url()).pathname === '/api/einstellungen/sitzung'
	);
	await uiLogin(page);
	await fristen;
	await expect(page.getByPlaceholder(/scannen/i).first()).toBeVisible();
	await page.clock.fastForward('15:10');
	const sperre = page.getByTestId('sperrbildschirm');
	await expect(sperre).toBeVisible();
	await expect(page.locator('#sperre-passwort')).toBeFocused();

	// Sechs Scans — einer mehr, als der Server an Fehlversuchen zulässt.
	for (const nummer of ['B-00123', '5896800039556', 'A-104711', 'B-00124', 'B-00125', 'B-00126']) {
		await scanneWieScanner(page, nummer);
		await expect(sperre.getByRole('alert')).toContainText('Scan erkannt');
		await expect(page.locator('#sperre-passwort')).toHaveValue('');
	}
	expect(versuche, 'kein Scan darf als Passwort hinausgehen').toEqual([]);

	// Steht der Fokus auf „Abmelden …", meldet das Enter des Scanners nicht ab.
	await sperre.getByRole('button', { name: 'Abmelden und als andere Person anmelden' }).focus();
	await scanneWieScanner(page, 'B-00127');
	await expect(sperre).toBeVisible();
	await expect(page.locator('#login-email')).toHaveCount(0);

	// Von Hand getippt geht die Sperre auf, und der Hinweis ist weg.
	await page.locator('#sperre-passwort').focus();
	await page.keyboard.type(ADMIN_PASSWORD, { delay: 90 });
	await page.keyboard.press('Enter');
	await expect(sperre).toHaveCount(0);
	expect(versuche).toEqual(['/api/auth/entsperren']);
	await expect(page.getByRole('button', { name: 'Abmelden', exact: true })).toBeVisible();
});

test('Anmeldung: ein Scan wird nicht abgeschickt, und die Felder stehen wie davor', async ({
	page
}) => {
	const versuche = zaehlePost(page, '/login');
	await page.goto('/');
	// Die Maske setzt den Fokus selbst ins E-Mail-Feld; dort landet ein Scan zuerst.
	await expect(page.locator('#login-email')).toBeFocused();
	await page.locator('#login-email').fill(ADMIN_EMAIL);
	await scanneWieScanner(page, '5896800039556');
	const hinweis = page.getByText('Scan erkannt: Bitte erst anmelden, dann scannen.');
	await expect(hinweis).toBeVisible();
	await expect(page.locator('#login-email')).toHaveValue(ADMIN_EMAIL);

	await page.locator('#login-password').focus();
	await scanneWieScanner(page, 'B-00123');
	await expect(page.locator('#login-password')).toHaveValue('');
	expect(versuche).toEqual([]);
	await expect(page.locator('#login-email')).toBeVisible();

	await page.keyboard.type(ADMIN_PASSWORD, { delay: 90 });
	await page.keyboard.press('Enter');
	await page.getByRole('button', { name: 'Abmelden', exact: true }).waitFor();
	expect(versuche).toEqual(['/login']);
});
