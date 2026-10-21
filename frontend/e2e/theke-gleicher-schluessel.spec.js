import { test, expect } from '@playwright/test';
import { csrfToken } from './helpers.js';
import {
	ausleihen,
	legeBuecherAn,
	legeSchuelerAn,
	neuerTeil,
	oeffneTheke,
	raeumeAuf,
	verliehenAn
} from './ohne-netz.js';

/**
 * Derselbe Scan kann den Server mehrmals erreichen: zwei Theken-Tabs, ein doppelter Scan, ein
 * Nachbuchen, das einen abgebrochenen Online-Versand wiederholt. Der Schlüssel des Scans macht
 * daraus eine Buchung: Die erste Anfrage reserviert ihn, jede weitere wartet auf ihre Antwort
 * oder meldet, dass er gerade gebucht wird (api/theke/eingabe.go). Ohne das läge das Buch nach
 * der ersten Anfrage beim Kind, und die nächste nähme es als Rückgabe zurück.
 */
test('Fünf gleichzeitige Anfragen mit einem Schlüssel buchen eine Ausleihe', async ({ page }) => {
	await oeffneTheke(page);
	const s = neuerTeil();
	try {
		const anna = await legeSchuelerAn(page, 'Anna', s);
		const buch = `B-GS${s}`;
		legeBuecherAn(s, [buch]);

		const token = await csrfToken(page);
		const rumpf = { query: buch, active_leser_id: anna.id, idempotency_key: crypto.randomUUID() };
		const antworten = await Promise.all(
			Array.from({ length: 5 }, () =>
				page.request.post('/api/action', { data: rumpf, headers: { 'X-CSRF-Token': token } })
			)
		);
		const ergebnisse = await Promise.all(
			antworten.map(async (a) =>
				a.ok() ? `${a.status()} ${(await a.json()).type}` : `${a.status()}`
			)
		);

		// Jede Antwort ist dieselbe Ausleihe oder „wird gerade gebucht" (409), keine eine Rückgabe.
		for (const e of ergebnisse) expect(['200 ausleihe', '409'], ergebnisse.join(', ')).toContain(e);
		expect(ergebnisse).toContain('200 ausleihe');
		expect(ausleihen(buch)).toBe('1/1');
		expect(verliehenAn(buch)).toBe(anna.id);
	} finally {
		raeumeAuf(s);
	}
});
