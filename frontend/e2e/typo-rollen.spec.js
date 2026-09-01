import { test, expect } from '@playwright/test';
import { uiLogin } from './helpers.js';

// Gate aus dem M3-Typografie-Audit vom 25.08.2026 (Bericht aufgelöst, Historie in git log):
// Lesetext liegt auf der Skala, nicht eine Stufe darunter. Gemessen wird im
// Browser mit getComputedStyle — die statische Inventur hat bei Schriftgrößen
// zweimal gelogen (Klassen in Variablen, Vererbung).
//
// Regeln (aus den M3-Rollen, Herleitung im Audit):
//   - Zellen-Text (td):   >= 14 px (body-medium) — Klasse, Barcode, Datum, Status
//     sind das, was man beim Arbeiten ABLIEST, kein Beiwerk.
//   - Tabellenkopf (th):  >= 12 px
//   - Knopf-Text:         >= 14 px (M3 kennt keinen 12-px-Knopf)
//   - h2/h3:              >= 16 px
//   - Schriftgewicht:     <= 700 überall (800/900 gibt es in M3 nicht)
// Ausnahme: die PILLE (Chip/Badge/Tag) — erkannt an der gemessenen GESTALT
// (getönte Fläche + Eckenradius), nicht an Markern im Markup: Marker in den
// geduldeten Dateien hätten die Dateigrößen-Ratsche gerissen, und eine getönte
// runde Fläche IST die Chip-Bauform. Kurzinhalt (<= 4 Zeichen, Zähler-Badge)
// darf 11 px (label-small), Chip-Text 12 px (M3-dense). Der Cover-Platzhalter
// der Theke (Zeichnung, laut Audit keine UI) fällt als getönte Fläche mit
// Radius ebenfalls unter diese Regel.
const MESSE_TYPO = () => {
	/** @type {string[]} */
	const verstoesse = [];
	const wurzel = document.querySelector('main') ?? document.body;
	const walker = document.createTreeWalker(wurzel, NodeFilter.SHOW_TEXT);
	let knoten;
	let geprueft = 0;
	while ((knoten = walker.nextNode())) {
		const text = (knoten.textContent ?? '').trim();
		if (!text) continue;
		const el = knoten.parentElement;
		if (!el || el.closest('svg, script, style')) continue;
		const rect = el.getBoundingClientRect();
		if (rect.width === 0 || rect.height === 0) continue;
		const stil = getComputedStyle(el);
		if (stil.visibility === 'hidden' || stil.display === 'none') continue;
		geprueft++;

		const groesse = parseFloat(stil.fontSize);
		const gewicht = parseInt(stil.fontWeight, 10);
		const kurz = text.length > 40 ? `${text.slice(0, 40)}…` : text;

		if (gewicht > 700) {
			verstoesse.push(`Gewicht ${gewicht} (M3: max 500/700): "${kurz}"`);
		}

		// Pille: getönte Fläche + Radius, gesucht vom Textknoten aufwärts bis zur
		// Zelle/zum Knopf. Zähler (<= 4 Zeichen) sind Badges, sonst Chip-Text.
		let pille = null;
		for (let p = el; p; p = p.parentElement) {
			// Grenze VOR der Signatur: der M3-Knopf ist selbst eine getönte Pille —
			// sein Text bleibt trotzdem Knopf-Text (>= 14), kein Chip.
			if (p.matches('td, th, button, main')) break;
			const ps = getComputedStyle(p);
			const radius = parseFloat(ps.borderRadius) || 0;
			const bg = ps.backgroundColor;
			if (radius >= 4 && bg && bg !== 'transparent' && !bg.startsWith('rgba(0, 0, 0, 0)')) {
				pille = p;
				break;
			}
		}

		let mindest = 0;
		let rolle = '';
		if (pille) {
			mindest = text.length <= 4 ? 11 : 12;
			rolle = text.length <= 4 ? 'Badge' : 'Chip';
		} else if (el.closest('td')) {
			mindest = 14;
			rolle = 'td';
		} else if (el.closest('th')) {
			mindest = 12;
			rolle = 'th';
		} else if (el.closest('h2, h3')) {
			mindest = 16;
			rolle = 'h2/h3';
		} else if (el.closest('button')) {
			mindest = 14;
			rolle = 'Knopf';
		}
		if (mindest && groesse < mindest) {
			verstoesse.push(`${rolle} ${groesse}px < ${mindest}: "${kurz}"`);
		}
	}
	return { verstoesse: [...new Set(verstoesse)].slice(0, 20), geprueft };
};

/** Stabiler Baum wie in kontrast.spec.js — kein networkidle (SSE). */
async function warteAufStabilenBaum(page) {
	let vorherige = -1;
	await expect
		.poll(
			async () => {
				const jetzt = await page.evaluate(() => document.querySelectorAll('main *').length);
				const stabil = jetzt === vorherige;
				vorherige = jetzt;
				return stabil;
			},
			{ timeout: 10_000, intervals: [100, 150, 200, 300] }
		)
		.toBe(true);
}

test('Lesetext liegt auf der M3-Skala (td 14, th 12, Knopf 14, h2/h3 16, Gewicht <= 700)', async ({
	page
}) => {
	await uiLogin(page);

	/** @type {{ name: string, oeffne: () => Promise<void> }[]} */
	const ansichten = [
		{ name: 'Schülerdatei', oeffne: () => page.getByTitle('Schülerdatei').click() },
		{ name: 'Mahnwesen', oeffne: () => page.getByTitle('Mahnwesen').click() },
		{
			name: 'Medienkatalog › Titel-Verwaltung',
			oeffne: async () => {
				await page.getByTitle('Medienkatalog').click();
				await page.getByRole('tab', { name: 'Titel-Verwaltung' }).click();
			}
		},
		{
			name: 'Bestellungen › Bestellhistorie',
			oeffne: async () => {
				await page.getByTitle('Bestellungen').click();
				await page.getByRole('tab', { name: /Bestellhistorie/ }).click();
			}
		},
		{
			name: 'Druck-Center › Fehlende Etiketten',
			oeffne: async () => {
				await page.getByTitle('Druck-Center').click();
				await page.getByRole('tab', { name: /Fehlende Etiketten/ }).click();
			}
		},
		{ name: 'System-Logs', oeffne: () => page.goto('/system-logs') }
	];

	const alle = [];
	let geprueftGesamt = 0;
	for (const a of ansichten) {
		await a.oeffne();
		await page.locator('main').first().waitFor();
		await warteAufStabilenBaum(page);
		const { verstoesse, geprueft } = await page.evaluate(MESSE_TYPO);
		geprueftGesamt += geprueft;
		for (const v of verstoesse) alle.push(`[${a.name}] ${v}`);
	}

	// Nulllauf-Wächter: Wenn kaum Text gemessen wurde, prüft der Test nichts.
	expect(geprueftGesamt, 'gemessene Textknoten').toBeGreaterThan(300);
	expect(alle, `${alle.length}+ Typografie-Verstöße:\n  ${alle.join('\n  ')}`).toEqual([]);
});
