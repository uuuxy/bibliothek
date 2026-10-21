import { test, expect } from '@playwright/test';
import {
	OHNE_NETZ,
	legeBuecherAn,
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
 * Ohne Netz bringt Anna ein Buch zurück und nimmt ein neues mit, beides nach ihrem Ausweis, wie es
 * das Handbuch sagt („Scanne wie sonst: erst den Ausweis, dann die Bücher"). Am Netz nimmt
 * derselbe Scan das Buch zurück, das schon auf ihrem Konto liegt; nach dem Nachbuchen gilt
 * dasselbe, weil jeder Scan mit Person die Dauer des Besuchs trägt (besuch_seit_ms).
 */
test('Ohne Netz: Mit dem Ausweis kommt das alte Buch zurück, und das neue geht hinaus', async ({
	page,
	context
}) => {
	test.setTimeout(60_000);
	const s = neuerTeil();
	const buch = { alt: `B-ORA${s}`, neu: `B-ORN${s}` };
	legeBuecherAn(s, [buch.alt, buch.neu]);
	try {
		await oeffneTheke(page);
		const anna = await legeSchuelerAn(page, 'Anna', s);
		verleihe(buch.alt, anna.id);

		await context.setOffline(true);
		await expect(page.getByText(OHNE_NETZ)).toBeVisible();
		for (const scan of [anna.nummer, buch.alt, buch.neu]) await scanne(page, scan);
		await expect.poll(() => warteschlange(page).then((q) => q.length)).toBe(2);

		await context.setOffline(false);
		await expect.poll(() => warteschlange(page), { timeout: 15_000 }).toEqual([]);
		expect(await verliehenAn(buch.alt), 'das abgegebene Buch steht noch auf Annas Konto').toBe('');
		expect(await verliehenAn(buch.neu)).toBe(anna.id);
	} finally {
		raeumeAuf(s);
	}
});
