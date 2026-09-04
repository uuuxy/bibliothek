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

// Die M3-Masse des Reiter-Bauteils, im Browser gemessen statt am Klassennamen gelesen.
//
// Anlass (04.09.2026): Beim Umbau der Bestell-Leiste vom Handbau auf ui/Reiter.svelte
// FIEL das Schriftgewicht von 500 auf 400 — das gemeinsame Bauteil war leichter als die
// Kopien, die es ersetzen soll. Aufgefallen ist das erst beim Nachmessen: Die Klasse
// heisst `font-medium`, und wer den Namen liest, denkt an M3s `weight-medium`. In diesem
// Haus zeigt sie aber auf 400, weil theme-mass.css die Skala umbiegt (medium→400,
// semibold/bold→500). Ein Klassen-Grep hätte hier das Gegenteil bestätigt.
//
// Soll aus der M3-Token-Spezifikation (material-web v0.192):
//   primary-navigation-tab   label = title-small → weight-medium (500), 14 px
//                            active-indicator-height 3px, shape (3px 3px 0 0)
test('Reiter tragen die M3-Masse: Label 500/14px, Indikator 3 px oben gerundet', async ({
	page
}) => {
	await uiLogin(page);
	await page.getByTitle('Bestellungen').click();
	const leiste = page.getByRole('tablist', { name: 'Bereiche des Bestellwesens' });
	await expect(leiste).toBeVisible();

	const gemessen = await leiste.evaluate((el) => {
		const tabs = [...el.querySelectorAll('[role="tab"]')];
		const aktiv = tabs.find((t) => t.getAttribute('aria-selected') === 'true');
		const ind = aktiv?.querySelector('span[class*="absolute"]');
		return {
			gewichte: [...new Set(tabs.map((t) => getComputedStyle(t).fontWeight))],
			groessen: [...new Set(tabs.map((t) => getComputedStyle(t).fontSize))],
			indHoehe: ind ? getComputedStyle(ind).height : null,
			indRadius: ind ? getComputedStyle(ind).borderTopLeftRadius : null
		};
	});

	expect(
		gemessen.gewichte,
		`Reiter-Gewichte: ${gemessen.gewichte.join(', ')} — M3 title-small ist 500`
	).toEqual(['500']);
	expect(gemessen.groessen, `Reiter-Groessen: ${gemessen.groessen.join(', ')}`).toEqual(['14px']);
	expect(gemessen.indHoehe, 'Indikatorhöhe — M3 primary-navigation-tab: 3px').toBe('3px');
	expect(
		gemessen.indRadius,
		'Indikator oben gerundet (3px), nicht voll gerundet — M3 shape (3px 3px 0 0)'
	).toBe('3px');
});
