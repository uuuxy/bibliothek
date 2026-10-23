import { test, expect } from '@playwright/test';
import { seedSQL, sitzungsKennung } from './helpers.js';
import { oeffneTheke } from './ohne-netz.js';

// Die Live-Leitung endet, sobald die Anmeldung am Server nicht mehr gilt (IstSitzungGueltig in
// api/tuer/sitzung.go). Die Theke zeigt dann ohne Bedienung, was am Server gilt: den
// Sperrbildschirm oder die Anmeldung, nicht nach 25 Sekunden „Keine Verbindung“. Die Leitung
// prüft vor jedem Lebenszeichen, also spätestens nach 15 Sekunden.

/**
 * Die Zähler der Badges fragen alle 30 Sekunden und sähen das Ende der Anmeldung auch. Sie
 * antworten hier mit 0, damit allein die Leitung es meldet; was sonst mit 401 oder 423 abgewiesen
 * wird, sammelt die Liste.
 * @param {import('@playwright/test').Page} page
 */
async function nurDieLeitungMeldet(page) {
	const kontext = page.context();
	await kontext.route(/\/api\/.+\/anzahl$/, (route) => route.fulfill({ json: { anzahl: 0 } }));
	/** @type {string[]} */
	const abgewiesen = [];
	page.on('response', (antwort) => {
		const pfad = new URL(antwort.url()).pathname;
		if (![401, 423].includes(antwort.status())) return;
		if (pfad.startsWith('/api/') && !pfad.startsWith('/api/auth/')) {
			abgewiesen.push(`${antwort.status()} ${pfad}`);
		}
	});
	return abgewiesen;
}

test('zehn Stunden nach der Passworteingabe zeigt die Theke von selbst den Sperrbildschirm', async ({
	page
}) => {
	test.setTimeout(60_000);
	const abgewiesen = await nurDieLeitungMeldet(page);
	await oeffneTheke(page);

	const sid = await sitzungsKennung(page);
	seedSQL(
		`UPDATE sitzungen SET passwort_geprueft_am = NOW() - INTERVAL '10 hours 1 minute' WHERE id = '${sid}'`
	);

	const bildschirm = page.getByTestId('sperrbildschirm');
	await expect(bildschirm).toBeVisible({ timeout: 20_000 });
	await expect(bildschirm.getByRole('heading', { name: 'Gesperrt nach 10 Stunden' })).toBeVisible();
	await expect(page.getByText(/Keine Verbindung/)).toHaveCount(0);
	expect(abgewiesen).toEqual([]);
});

test('abgemeldet in einem anderen Fenster: Die Theke zeigt von selbst die Anmeldung', async ({
	page
}) => {
	test.setTimeout(60_000);
	const abgewiesen = await nurDieLeitungMeldet(page);
	await oeffneTheke(page);

	const nebenan = await page.context().newPage();
	await nebenan.goto('/');
	await nebenan.getByRole('button', { name: 'Abmelden' }).click();
	await expect(nebenan.locator('#login-email')).toBeVisible();

	await expect(page.locator('#login-email')).toBeVisible({ timeout: 20_000 });
	await expect(page.getByText('Sitzung abgelaufen — bitte neu anmelden.')).toBeVisible();
	await expect(page.getByText(/Keine Verbindung/)).toHaveCount(0);
	expect(abgewiesen).toEqual([]);
});
