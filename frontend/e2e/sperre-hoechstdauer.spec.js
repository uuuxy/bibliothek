import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix, ADMIN_PASSWORD, menuepunkt } from './helpers.js';
import {
	legeBuecherAn,
	legeSchuelerAn,
	neuerTeil,
	oeffneTheke,
	raeumeAuf,
	scanne,
	verliehenAn,
	warteschlange
} from './ohne-netz.js';

// Zehn Stunden nach der letzten Passworteingabe sperrt der Server die Anmeldung (OFFEN 5.96). Wer
// in dem Moment eine Maske offen hat, verliert nichts: Statt abzumelden, kommt der
// Sperrbildschirm mit dem Grund, das Passwort schließt auf, die Eingabe steht noch da, und
// Speichern geht danach.
const s = uniqueSuffix().slice(0, 6);
const TITEL = `Hoechstdauer ${s}`;

/** Die Kennung der Anmeldung dieses Browsers, aus dem Token im Cookie.
 * @param {import('@playwright/test').Page} page */
async function sitzungsKennung(page) {
	const cookie = (await page.context().cookies()).find((c) => c.name === 'session_token');
	if (!cookie) throw new Error('kein Cookie der Anmeldung');
	const nutzlast = JSON.parse(Buffer.from(cookie.value.split('.')[1], 'base64url').toString());
	expect(nutzlast.sid).toMatch(/^[0-9a-f-]{36}$/);
	return nutzlast.sid;
}

test.afterAll(() => {
	seedSQL(`
		DELETE FROM buecher_exemplare WHERE titel_id IN (SELECT id FROM buecher_titel WHERE titel = '${TITEL}');
		DELETE FROM buecher_titel WHERE titel = '${TITEL}';`);
});

test('zehn Stunden nach der Passworteingabe kommt der Sperrbildschirm; die Eingabe überlebt, Speichern geht danach', async ({
	page
}) => {
	await uiLogin(page);
	await menuepunkt(page, 'Medienkatalog').click();
	await page.getByRole('tab', { name: 'Titel-Verwaltung' }).click();
	await page.getByRole('button', { name: 'Neues Buch' }).first().click();
	await page.locator('#buch-titel').fill(TITEL);
	await page.locator('#buch-signatur').fill('Hoe 1');

	// Nur diese Anmeldung altert: Andere Tests laufen mit demselben Konto daneben.
	const sid = await sitzungsKennung(page);
	seedSQL(
		`UPDATE sitzungen SET passwort_geprueft_am = NOW() - INTERVAL '10 hours 1 minute' WHERE id = '${sid}'`
	);

	await page.getByRole('button', { name: 'Speichern' }).click();
	const bildschirm = page.getByTestId('sperrbildschirm');
	await expect(bildschirm).toBeVisible();
	await expect(bildschirm.getByRole('heading', { name: 'Gesperrt nach 10 Stunden' })).toBeVisible();
	await expect(bildschirm.getByText('Was nicht gespeichert ist, bleibt erhalten.')).toBeVisible();
	expect(querySQL(`SELECT count(*) FROM buecher_titel WHERE titel = '${TITEL}'`)).toBe('0');

	await page.locator('#sperre-passwort').fill(ADMIN_PASSWORD);
	await page.getByRole('button', { name: 'Entsperren' }).click();
	await expect(bildschirm).toHaveCount(0);
	await expect(page.locator('#buch-titel')).toHaveValue(TITEL);
	await expect(page.locator('#buch-signatur')).toHaveValue('Hoe 1');
	expect(
		querySQL(
			`SELECT passwort_geprueft_am > NOW() - INTERVAL '1 minute' FROM sitzungen WHERE id = '${sid}'`
		)
	).toBe('t');

	await page.getByRole('button', { name: 'Speichern' }).click();
	await expect(page.getByText('Buch erfolgreich gespeichert!')).toBeVisible();
	expect(querySQL(`SELECT count(*) FROM buecher_titel WHERE titel = '${TITEL}'`)).toBe('1');
});

// An der Theke trifft die Grenze einen Scan: Die Sperre leert die Theke, während die Antwort
// unterwegs ist. Das Buch geht mit der Person vom Scan in die Warteschlange und wird nach dem
// Aufschließen nachgebucht (omniboxBuchung.svelte.js, nichtErreicht); sonst verließe es die
// Bücherei ungebucht.
test('ein Scan, der auf die Grenze trifft, wird nach dem Aufschließen gebucht', async ({
	page
}) => {
	test.setTimeout(60_000);
	const teil = neuerTeil();
	const buch = `B-HD${teil}`;
	legeBuecherAn(teil, [buch]);
	try {
		await page.clock.install();
		await oeffneTheke(page);
		const anna = await legeSchuelerAn(page, 'Anna', teil);
		await scanne(page, anna.nummer);
		await expect(page.getByRole('heading', { name: /Anna/ }).first()).toBeVisible();

		const sid = await sitzungsKennung(page);
		seedSQL(
			`UPDATE sitzungen SET passwort_geprueft_am = NOW() - INTERVAL '10 hours 1 minute' WHERE id = '${sid}'`
		);
		await scanne(page, buch);
		await expect(page.getByTestId('sperrbildschirm')).toBeVisible();
		await expect.poll(() => warteschlange(page).then((q) => q.length)).toBe(1);
		expect(await verliehenAn(buch)).toBe('');

		await page.locator('#sperre-passwort').fill(ADMIN_PASSWORD);
		await page.getByRole('button', { name: 'Entsperren' }).click();
		await expect(page.getByTestId('sperrbildschirm')).toHaveCount(0);
		await page.clock.fastForward('01:05');
		await expect.poll(() => verliehenAn(buch), { timeout: 15_000 }).toBe(anna.id);
		await expect.poll(() => warteschlange(page)).toEqual([]);
	} finally {
		raeumeAuf(teil);
	}
});
