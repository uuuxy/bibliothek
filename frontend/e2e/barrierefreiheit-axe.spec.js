// Gate gegen Barrieren, die eine Maschine finden kann — axe-core (WCAG 2.1 A/AA) über
// die öffentlichen Seiten und jede interne Hauptansicht, dazu die drei Gerüstfragen,
// die axe nicht stellt: Sprache des Dokuments, EIN <main>, Überschrift je Seite,
// Skip-Link als erstes Fokusziel.
//
// Anlass (Prüfung 08.09.2026): Die Anwendung gilt für die Schule als öffentliche Stelle
// (HessBGG/HVBIT → EN 301 549 / WCAG 2.1 AA, für Intranet-Anwendungen seit 09/2019).
// Der erste Lauf fand unter anderem `lang="en"`, zwei verschachtelte <main>, 592
// nested-interactive-Knoten und 168 `span[aria-label]` — alles Dinge, die ein Mensch
// beim Klicken nie bemerkt.
//
// Warum je Seite ein eigener Test: Ein roter Lauf soll die SEITE nennen, nicht nur
// „irgendwo 37 Verstöße". Gemessen wird am laufenden Container (8084), wie alle
// Browser-Gates — die Fixes sitzen in Bauteilen (Modal, Tabelle, PageShell, Token), und
// ob ein Bauteil auf einer Seite wirkt, sieht man nur dort.
//
// Dieses Gate wurde vor dem ersten Fix ROT gesehen (09.09.2026, 17 von 19 Tests).
import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import { uiLogin, gehZu } from './helpers.js';

/** Öffentlich, ohne Anmeldung. */
const OEFFENTLICH = [
	['Anmeldung', '/'],
	['Katalog', '/katalog']
];

/** Jede interne Hauptansicht (Router.svelte, tabToPath) — außer dem Kollegiums-Portal,
 *  das ein Admin nicht sieht. */
const INTERN = [
	['Ausleihe', '/kiosk'],
	['Medienkatalog', '/medienkatalog'],
	['Signaturen', '/signaturen'],
	['Druck-Center', '/druck-center'],
	['Klassensätze', '/schulklassen'],
	['Schülerdatei', '/schuelerdatei'],
	['Mahnwesen', '/mahnwesen'],
	['Abgänger', '/abgaenger'],
	['Bestellungen', '/bestellungen'],
	['Inventur', '/inventur'],
	['Statistiken', '/statistiken'],
	['System-Logs', '/system-logs'],
	['Schuljahreswechsel', '/schuljahr'],
	['Benutzer & Rechte', '/berechtigungen'],
	['Einstellungen', '/einstellungen']
];

/**
 * Alle WCAG-A/AA-Verstöße einer Seite als lesbare Zeilen. Best-Practice-Regeln von axe
 * (Landmarken-Kür, Überschriften-Reihenfolge) bleiben draußen — Gate ist die Norm.
 * @param {import('@playwright/test').Page} page
 */
async function verstoesse(page) {
	const ergebnis = await new AxeBuilder({ page })
		.withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
		.analyze();
	// Bis zu drei Fundstellen je Regel: Eine allein zeigt oft nur die Seitenleiste, die
	// auf jeder Seite dieselben Treffer liefert — die seitenspezifischen stünden dahinter.
	return ergebnis.violations.map(
		(v) =>
			`${v.id} (${v.impact}) ×${v.nodes.length}: ${v.help} — z. B. ` +
			v.nodes
				.slice(0, 3)
				.map((n) => n.target.join(' ') + farben(n))
				.join(' | ')
	);
}

/**
 * Bei Kontrast-Funden: Vorder-/Hintergrundfarbe und das gemessene Verhältnis — sonst
 * steht da nur ein Selektor, und die Suche nach dem Grund beginnt von vorn.
 * @param {{ any?: { data?: any }[] }} n
 */
function farben(n) {
	const d = n.any?.[0]?.data;
	if (!d?.fgColor) return '';
	return ` [${d.fgColor} auf ${d.bgColor}, ${d.contrastRatio}:1 bei ${d.fontSize}]`;
}

/** @param {import('@playwright/test').Page} page */
async function zurRuhe(page) {
	// Kein networkidle (SSE hält die Leitung offen): Es reicht, dass das Gerüst steht
	// und ein Frame gerendert wurde.
	await page.locator('body').waitFor();
	await page.waitForTimeout(400);
}

test.describe('axe: keine WCAG-A/AA-Verstöße', () => {
	// Ohne laufende Übergänge messen: Mitten in einer Einblendung liest axe gemischte
	// Farben (der Monitor wechselt Folien per Fade) und meldet Kontraste, die es im
	// Ruhezustand nicht gibt. Bewegung reduzieren ist ohnehin ein Nutzerzustand, den
	// die Anwendung tragen muss (basis.css).
	test.beforeEach(({ page }) => page.emulateMedia({ reducedMotion: 'reduce' }));
	for (const [name, pfad] of OEFFENTLICH) {
		test(`öffentlich · ${name}`, async ({ page }) => {
			await page.goto(pfad);
			await zurRuhe(page);
			expect(await verstoesse(page), `${name} (${pfad})`).toEqual([]);
		});
	}
	// Der Monitor wechselt alle 15 s die Folie — und jede Folie hat eigene Farben. Ein
	// einzelner Scan sieht nur eine (lokal grün, CI rot am 09.09.2026: die Rangzahlen der
	// Folie „Beliebt" mit 1,84:1). Gescannt wird deshalb jede Folie, die sich zeigt;
	// Folien ohne Inhalt überspringt der Monitor selbst, dann endet der Wechsel früher.
	test('öffentlich · Monitor, jede Folie', async ({ page }) => {
		test.setTimeout(90_000); // zwei Folienwechsel à 15 s plus drei Scans
		await page.goto('/monitor');
		await zurRuhe(page);
		const anzeige = page.getByTestId('monitor-folie');
		const gesehen = new Set();
		/** @type {string[]} */
		const befunde = [];
		for (let i = 0; i < 3; i++) {
			const folie = (await anzeige.textContent())?.trim() ?? '';
			if (!gesehen.has(folie)) {
				gesehen.add(folie);
				for (const v of await verstoesse(page)) befunde.push(`${folie}: ${v}`);
			}
			if (gesehen.size === 3) break;
			try {
				await expect(anzeige).not.toHaveText(folie, { timeout: 20_000 });
			} catch {
				break; // nur diese eine Folie hat Inhalt — mehr gibt es nicht zu scannen
			}
			await zurRuhe(page);
		}
		expect(befunde, `Monitor, Folien: ${[...gesehen].join(', ')}`).toEqual([]);
	});
	for (const [name, pfad] of INTERN) {
		test(`intern · ${name}`, async ({ page }) => {
			await uiLogin(page);
			await gehZu(page, pfad);
			await zurRuhe(page);
			expect(await verstoesse(page), `${name} (${pfad})`).toEqual([]);
		});
	}
});

test('Das Dokument ist Deutsch: <html lang="de">', async ({ page }) => {
	await page.goto('/');
	expect(await page.locator('html').getAttribute('lang')).toBe('de');
});

test('Gerüst: ein <main>, eine Überschrift, Skip-Link als erstes Fokusziel', async ({ page }) => {
	await uiLogin(page);
	/** @type {string[]} */
	const befunde = [];
	for (const [name, pfad] of INTERN) {
		await gehZu(page, pfad);
		await zurRuhe(page);
		const mains = await page.locator('main').count();
		if (mains !== 1) befunde.push(`${name}: ${mains} <main> statt genau einem`);
		if ((await page.locator('h1').count()) === 0) befunde.push(`${name}: kein <h1>`);
	}
	// Frisch geladen sitzt der Fokus auf <body>; das erste Tab muss auf dem Skip-Link
	// landen, sonst muss eine Tastaturnutzerin die ganze Seitenleiste durchtabben.
	// Gemessen wird NICHT an der Theke (/kiosk): Dort hält das Scanfeld den Fokus, und
	// das ist gewollt — ein Scanner tippt blind (scanner-pfad). Einstellungen hat keinen
	// Autofokus.
	await gehZu(page, '/einstellungen');
	await zurRuhe(page);
	await page.keyboard.press('Tab');
	const erstesZiel = await page.evaluate(() => {
		const a = document.activeElement;
		return a ? `${a.tagName.toLowerCase()} ${a.getAttribute('href') ?? ''}`.trim() : 'nichts';
	});
	if (erstesZiel !== 'a #hauptinhalt') befunde.push(`erstes Tab landet auf „${erstesZiel}"`);
	expect(befunde).toEqual([]);
});
