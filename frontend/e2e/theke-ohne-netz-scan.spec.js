import { test, expect } from '@playwright/test';
import {
	OHNE_NETZ,
	legeBuecherAn,
	legeLehrkraftAn,
	legeSchuelerAn,
	neuerTeil,
	oeffneTheke,
	raeumeAuf,
	scanne,
	verleihe,
	verliehenAn,
	warteschlange
} from './ohne-netz.js';

/**
 * Erreicht ein Scan den Server nicht, legt die Theke ihn ab, wie er beim Scannen gemeint war:
 * mit der Person, die da geladen war, und mit der Absicht, die der Knopf vorgab. Nachgebucht
 * wird, sobald die Verbindung zurück ist. Belegt wird an der Warteschlange im Browser und an
 * den Ausleihen in der Datenbank (docs/FACHKONZEPT.md 18.4).
 */

/** @type {string[]} */
const angelegt = [];
test.afterEach(() => {
	for (const s of angelegt.splice(0)) raeumeAuf(s);
});

test('Bleibt die Antwort aus, geht der Scan mit dem Schüler vom Scan in die Warteschlange, auch nach Escape', async ({
	page,
	context
}) => {
	// Die Frist von apiFetch für eine Buchung ist 10 s; die Uhr wird vorgespult, nicht abgewartet.
	await page.clock.install();
	await oeffneTheke(page);
	const s = neuerTeil();
	angelegt.push(s);
	const anna = await legeSchuelerAn(page, 'Anna', s);
	const buch = `B-ON1${s}`;
	legeBuecherAn(s, [buch]);

	await scanne(page, anna.nummer);
	await expect(page.getByText(anna.nachname).first()).toBeVisible();

	/** @type {import('@playwright/test').Route | null} */
	let gehalten = null;
	await page.route('**/api/action', (route) => {
		gehalten = route;
	});
	await scanne(page, buch);
	await expect
		.poll(() => gehalten !== null, { message: 'die Buchung ging nicht hinaus' })
		.toBe(true);

	await page.keyboard.press('Escape');
	await expect(page.getByText(anna.nachname)).toHaveCount(0);
	await page.clock.fastForward(11_000);

	await expect
		.poll(() => warteschlange(page))
		.toEqual([expect.objectContaining({ art: 'ausleihe', barcode: buch, leser_id: anna.id })]);
	expect(verliehenAn(buch), 'die gehaltene Anfrage hat den Server nie erreicht').toBe('');

	await page.unroute('**/api/action');
	await /** @type {import('@playwright/test').Route} */ (gehalten).abort().catch(() => {});
	// Die Rückkehr der Verbindung stößt das Nachbuchen an.
	await context.setOffline(true);
	await expect(page.getByText(/Sobald die Verbindung zurück ist/)).toBeVisible();
	await context.setOffline(false);
	await expect.poll(() => verliehenAn(buch), { timeout: 10_000 }).toBe(anna.id);
	await expect.poll(() => warteschlange(page)).toEqual([]);
});

test('Ohne Netz ist „Buch zurückgeben" in der Akte eine Rückgabe, keine Ausleihe', async ({
	page,
	context
}) => {
	await oeffneTheke(page);
	const s = neuerTeil();
	angelegt.push(s);
	const anna = await legeSchuelerAn(page, 'Anna', s);
	const buch = `B-ON2${s}`;
	legeBuecherAn(s, [buch]);
	verleihe(buch, anna.id);

	await scanne(page, anna.nummer);
	const zeile = page.getByRole('row', { name: new RegExp(`E2E-OhneNetz ${buch}`) });
	await expect(zeile).toBeVisible();

	await context.setOffline(true);
	await expect(page.getByText(OHNE_NETZ)).toBeVisible();
	await zeile.getByRole('button', { name: 'Buch zurückgeben' }).click();

	// Mit geladenem Schüler wäre ein Scan ohne Vorgabe eine Ausleihe.
	await expect
		.poll(() => warteschlange(page))
		.toEqual([expect.objectContaining({ art: 'rueckgabe', barcode: buch, leser_id: anna.id })]);
	expect(verliehenAn(buch)).toBe(anna.id);

	await context.setOffline(false);
	await expect.poll(() => verliehenAn(buch), { timeout: 10_000 }).toBe('');
	await expect.poll(() => warteschlange(page)).toEqual([]);
});

test('Ohne Netz geht ein Buch an die geladene Lehrkraft', async ({ page, context }) => {
	await oeffneTheke(page);
	const s = neuerTeil();
	angelegt.push(s);
	const lena = await legeLehrkraftAn(page, s);
	const buch = `B-ON3${s}`;
	legeBuecherAn(s, [buch]);

	await scanne(page, lena.nummer);
	await expect(page.getByText(lena.nachname).first()).toBeVisible();

	await context.setOffline(true);
	await expect(page.getByText(OHNE_NETZ)).toBeVisible();
	await scanne(page, buch);
	await expect(page.getByText(`Offline: Aktion für „${buch}“ gespeichert.`)).toBeVisible();
	await expect
		.poll(() => warteschlange(page))
		.toEqual([expect.objectContaining({ art: 'ausleihe', barcode: buch, leser_id: lena.id })]);

	await context.setOffline(false);
	await expect.poll(() => verliehenAn(buch), { timeout: 10_000 }).toBe(lena.id);
	await expect.poll(() => warteschlange(page)).toEqual([]);
});

test('Lässt der Browser nichts ablegen, meldet die Theke den Scan als NICHT gespeichert', async ({
	page,
	context
}) => {
	// So sieht es aus, wenn ein Browser Website-Daten sperrt: Jede Datenbank lässt sich nicht öffnen.
	await page.addInitScript(() => {
		IDBFactory.prototype.open = () => {
			throw new DOMException('Website-Daten gesperrt', 'SecurityError');
		};
	});
	await oeffneTheke(page);
	const s = neuerTeil();
	angelegt.push(s);
	const buch = `B-ON4${s}`;
	legeBuecherAn(s, [buch]);
	await expect(page.getByText(/Warteschlange nicht lesbar/)).toBeVisible();

	await context.setOffline(true);
	await scanne(page, buch);
	await expect(
		page.getByText(`NICHT gespeichert: „${buch}“ konnte nicht auf diesem Rechner abgelegt werden`)
	).toBeVisible();
	await expect(page.getByText(`Offline: Aktion für „${buch}“ gespeichert.`)).toHaveCount(0);

	await context.setOffline(false);
	expect(verliehenAn(buch)).toBe('');
});
