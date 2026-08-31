import { test, expect } from '@playwright/test';
import { uiLogin } from './helpers.js';

// Nebenbefund vom 25.08.2026 (Feld-Migration, Register): Die Bestell-Reiter brachen
// bei 1280 px in zwei Zeilen um. Beim Nachmessen am 31.08. war der Fall durch den
// Wegzug des siebten Reiters („Nachdrucken" → Druck-Center) nicht mehr reproduzierbar
// — die BAUFORM (kein nowrap, kein Scroll) hätte aber bei der nächsten längeren
// Beschriftung wieder umbrochen. M3 sieht für zu schmale Leisten scrollbare Tabs vor.
//
// Gemessen wird im Browser (statische Inventur lügt), bei 860 px: Dort brach die
// alte Leiste sicher um (rot gesehen), mit Scroll-Verhalten bleibt sie einzeilig.
// Kalibrierung 31.08.: einzeilige Reiter sind 32–33 px hoch, ein intern
// umgebrochener 52 px — Schwelle 45.
async function misstEineZeile(page, name) {
	const leiste = page.getByRole('tablist', { name });
	await expect(leiste).toBeVisible();
	const tabs = leiste.getByRole('tab');
	const n = await tabs.count();
	expect(n, 'Reiterzahl').toBeGreaterThanOrEqual(5);

	const kanten = [];
	const hoehen = [];
	for (let i = 0; i < n; i++) {
		const box = await tabs.nth(i).boundingBox();
		if (!box) throw new Error(`Reiter ${i} ohne boundingBox`);
		kanten.push(Math.round(box.y));
		hoehen.push(Math.round(box.height));
	}
	expect(new Set(kanten).size, `Reiter-Oberkanten: ${kanten.join(', ')}`).toBe(1);
	for (const h of hoehen) {
		expect(h, `Reiterhöhen: ${hoehen.join(', ')} — >45 heißt interner Umbruch`).toBeLessThan(45);
	}
}

test('Bestell-Reiter: eine Zeile bei 1280 px (historischer Befund)', async ({ page }) => {
	await page.setViewportSize({ width: 1280, height: 800 });
	await uiLogin(page);
	await page.getByTitle('Bestellungen').click();
	await misstEineZeile(page, 'Bereiche des Bestellwesens');
});

test('Bestell-Reiter: eine Zeile auch bei 860 px — scrollen statt umbrechen', async ({ page }) => {
	await page.setViewportSize({ width: 860, height: 800 });
	await uiLogin(page);
	await page.getByTitle('Bestellungen').click();
	await misstEineZeile(page, 'Bereiche des Bestellwesens');
});
