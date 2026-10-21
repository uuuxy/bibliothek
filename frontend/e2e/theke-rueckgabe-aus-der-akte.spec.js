import { test, expect } from '@playwright/test';
import { seedSQL } from './helpers.js';
import {
	legeBuecherAn,
	legeSchuelerAn,
	neuerTeil,
	oeffneTheke,
	raeumeAuf,
	scanne,
	verleihe,
	verliehenAn
} from './ohne-netz.js';

/**
 * „Buch zurückgeben" in der Akte an der Theke gibt zurück und leiht nie aus. Die Akte lädt nicht
 * nach, wenn ein anderer Platz das Buch zurücknimmt; ein Klick darauf ging mit dem geladenen Leser
 * zum Server, und der lieh das freie Buch ihm aus. Dasselbe beim zweiten Klick, wenn der erste
 * schon gebucht war. Die Rückgabe trägt deshalb keine Person (leserFuerVersand).
 */
test('„Buch zurückgeben" aus einer Akte, die das Buch noch zeigt, leiht es nicht wieder aus', async ({
	page
}) => {
	test.setTimeout(60_000);
	const s = neuerTeil();
	const buch = `B-AK${s}`;
	legeBuecherAn(s, [buch]);
	try {
		await oeffneTheke(page);
		const anna = await legeSchuelerAn(page, 'Anna', s);
		verleihe(buch, anna.id);
		await scanne(page, anna.nummer);
		const zeile = page.getByRole('row').filter({ hasText: buch });
		await expect(zeile).toBeVisible();

		// Ein anderer Platz nimmt das Buch zurück; diese Akte zeigt es weiter.
		seedSQL(`UPDATE ausleihen SET rueckgabe_am = NOW()
			WHERE rueckgabe_am IS NULL AND exemplar_id = (SELECT id FROM buecher_exemplare WHERE barcode_id = '${buch}');`);
		await zeile.getByRole('button', { name: 'Buch zurückgeben' }).click();

		await expect(page.getByText(/nicht ausgeliehen/)).toBeVisible();
		expect(await verliehenAn(buch), 'das freie Buch wurde Anna ausgeliehen').toBe('');
	} finally {
		raeumeAuf(s);
	}
});
