import { test, expect } from '@playwright/test';
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

/**
 * Startet das Server-Programm neu (Update, Absturz), antwortet der Proxy davor mit 502. Das Band
 * sagt dann „Keine Verbindung — Scannen geht weiter, die Buchungen folgen von selbst". Die Theke
 * legt den Scan deshalb ab wie bei einem Netzausfall, und die Minutenuhr bucht ihn nach, sobald
 * der Server wieder arbeitet (omniboxBuchung.svelte.js, nichtErreicht).
 */
test('Antwortet der Server nicht (502 vom Proxy), legt die Theke den Scan ab und bucht ihn nach', async ({
	page
}) => {
	test.setTimeout(60_000);
	const s = neuerTeil();
	const buch = `B-PX${s}`;
	legeBuecherAn(s, [buch]);
	try {
		await page.clock.install();
		await oeffneTheke(page);
		const anna = await legeSchuelerAn(page, 'Anna', s);
		await scanne(page, anna.nummer);
		await expect(page.getByRole('heading', { name: /Anna/ }).first()).toBeVisible();

		await page.route('**/api/action', (route) => route.fulfill({ status: 502, body: '' }));
		await scanne(page, buch);
		await expect.poll(() => warteschlange(page).then((q) => q.length)).toBe(1);
		expect(await verliehenAn(buch)).toBe('');

		await page.unroute('**/api/action');
		await page.clock.fastForward('01:05');
		await expect.poll(() => verliehenAn(buch), { timeout: 15_000 }).toBe(anna.id);
		await expect.poll(() => warteschlange(page)).toEqual([]);
	} finally {
		raeumeAuf(s);
	}
});
