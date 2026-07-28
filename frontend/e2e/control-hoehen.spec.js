// Gate gegen die neunte Feldhöhe.
//
// Vor der Normierung standen auf den Hauptbildschirmen SECHZEHN verschiedene
// Feldhöhen zwischen 22 und 62 px, während die Buttons längst geschlossen auf 36 px
// lagen — neben jedem Button saß ein Feld auf einer eigenen Grundlinie.
//
// Warum ein E2E-Test und kein Klassen-Grep: Die Höhe entsteht erst im Browser aus
// Zeilenhöhe + Padding + Rahmen, und sie steckt oft in einer Variablen oder in einem
// Ausdruck (class={inputClass}). Statisch war die Inventur nachweislich falsch —
// gemessen wird deshalb offsetHeight an der laufenden Anwendung.
import { test, expect } from '@playwright/test';
import { uiLogin } from './helpers.js';

const CONTROL_HOEHE = 36; // identisch zu Button size="md" (h-9)

const SCREENS = [
	['Einstellungen', '/einstellungen'],
	['Inventur', '/inventur'],
	['Schülerdatei', '/schuelerdatei'],
	['Schulklassen', '/schulklassen'],
	['Bestellungen', '/bestellungen'],
	['Katalog', '/katalog'],
	['Abgänger', '/abgaenger'],
	['Statistiken', '/statistiken'],
	['Mahnwesen', '/mahnwesen'],
	['Lehrer-Portal', '/lehrer-portal'],
	['System-Logs', '/system-logs'],
	['LMF-Aktionen', '/lmf-aktionen'],
	['Druck-Center', '/druck-center'],
	['Kiosk', '/kiosk']
];

/**
 * Bewusste Ausnahmen — jede trägt an ihrer Fundstelle ein explizites h-auto/h-full.
 * Kein Feld darf hier landen, weil „es sonst rot ist": Die Ausnahme muss eine
 * Bedeutung haben, die die Control-Höhe nicht ausdrücken kann.
 */
const AUSNAHMEN = [
	{
		kennung: 'Titel, Autor oder ISBN eingeben …',
		grund: 'Katalog-Hero: das Suchfeld IST der Bildschirm, nicht ein Bedienelement darin'
	},
	{
		kennung: 'Titel, Autor oder ISBN suchen …',
		grund: 'Lehrer-Portal-Hero, gleiche Rolle wie die Katalog-Suche'
	},
	{
		kennung: 'omnibox-input',
		grund: 'füllt die 48-px-Scan-Pille (h-full); die Pille ist das Bedienelement, nicht das Feld'
	}
];

const MESSEN = () =>
	[...document.querySelectorAll('input, select')]
		.filter((el) => {
			const t = (el.getAttribute('type') || 'text').toLowerCase();
			if (
				['checkbox', 'radio', 'hidden', 'file', 'range', 'color', 'submit', 'button'].includes(t)
			) {
				return false;
			}
			const r = el.getBoundingClientRect();
			return r.width > 0 && r.height > 0;
		})
		.map((el) => ({
			hoehe: Math.round(el.getBoundingClientRect().height),
			kennung:
				el.id || el.getAttribute('placeholder') || el.getAttribute('aria-label') || '(namenlos)',
			tag: el.tagName.toLowerCase()
		}));

test('Alle Eingabefelder stehen auf der 36-px-Grundlinie', async ({ page }) => {
	await uiLogin(page);

	/** @type {string[]} */
	const abweichler = [];
	let geprueft = 0;

	for (const [name, pfad] of SCREENS) {
		await page.goto(pfad);
		await page.waitForTimeout(1000);

		for (const feld of await page.evaluate(MESSEN)) {
			if (AUSNAHMEN.some((a) => a.kennung === feld.kennung)) continue;
			geprueft++;
			if (feld.hoehe !== CONTROL_HOEHE) {
				abweichler.push(`${name}: <${feld.tag}> „${feld.kennung}" = ${feld.hoehe} px`);
			}
		}
	}

	expect(geprueft).toBeGreaterThan(15); // Schutz vor einem stillen Nulllauf
	expect(abweichler, `Felder abseits der ${CONTROL_HOEHE}-px-Grundlinie`).toEqual([]);
});

// Die Grundlinie ist nur etwas wert, wenn Feld und Button sie TEILEN. Diesen
// Vergleich prüft der Test oben nicht: Er würde auch grün bleiben, wenn beide
// gemeinsam auf 40 px wanderten.
test('Feld und Button stehen in derselben Werkzeugleiste auf einer Linie', async ({ page }) => {
	await uiLogin(page);
	await page.goto('/mahnwesen');

	const feld = page.getByPlaceholder('Schüler oder Klasse suchen …');
	// Zugänglicher Name = aria-label, nicht die Beschriftung „Alle anmahnen".
	const button = page.getByRole('button', { name: /Mahnlauf konfigurieren/ });
	await feld.waitFor();
	await button.waitFor();

	const feldBox = await feld.boundingBox();
	const buttonBox = await button.boundingBox();

	expect(Math.round(feldBox.height)).toBe(CONTROL_HOEHE);
	expect(Math.round(buttonBox.height)).toBe(CONTROL_HOEHE);
});
