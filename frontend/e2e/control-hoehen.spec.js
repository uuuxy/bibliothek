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
import { uiLogin, gehZu, einstellungsKategorie } from './helpers.js';

const CONTROL_HOEHE = 36; // identisch zu Button size="md" (h-9)

const SCREENS = [
	['Einstellungen', '/einstellungen'],
	['Inventur', '/inventur'],
	['Schülerdatei', '/schuelerdatei'],
	['Klassensätze', '/schulklassen'],
	['Bestellungen', '/bestellungen'],
	['Katalog', '/medienkatalog'],
	['Abgänger', '/abgaenger'],
	['Statistiken', '/statistiken'],
	['Mahnwesen', '/mahnwesen'],
	['Mein Portal', '/kollegium-portal'],
	['System-Logs', '/system-logs'],
	['Druck-Center', '/druck-center'],
	['Kiosk', '/kiosk']
];

/**
 * Bewusste Ausnahmen — jede trägt an ihrer Fundstelle ein explizites h-auto/h-full.
 * Kein Feld darf hier landen, weil „es sonst rot ist": Die Ausnahme muss eine
 * Bedeutung haben, die die Control-Höhe nicht ausdrücken kann.
 */
const AUSNAHMEN = [
	// Seit dem 10.08.2026 tragen alle Suchpillen eine id, und MESSEN bevorzugt die id vor
	// dem Platzhalter. Die beiden frueheren Eintraege standen auf Platzhaltertexten und
	// waren damit tot, sobald sich die Beschriftung aenderte — genau das passierte beim
	// Zusammenlegen auf components/ui/Suchpille.svelte.
	{
		kennung: 'opac-suchfeld',
		grund: 'Katalog-Hero: das Suchfeld IST der Bildschirm, nicht ein Bedienelement darin'
	},
	{
		kennung: 'portal-suchfeld',
		grund: 'Kollegiums-Portal-Hero, gleiche Rolle wie die Katalog-Suche'
	},
	{
		kennung: 'inventur-scan',
		grund: 'Inventur-Scanpille: dieselbe 48-px-Suchpille wie die Omnibox (h-full im Container)'
	},
	{
		kennung: 'omnibox-input',
		grund: 'füllt die 48-px-Scan-Pille (h-full); die Pille ist das Bedienelement, nicht das Feld'
	},
	{
		kennung: 'katalog-suchfeld',
		grund:
			'Medienkatalog-Suchpille: dieselbe Bauart wie die Omnibox — der Container ist h-12 und ' +
			'trägt Rahmen, Fläche und Fokus, das Feld selbst h-full und nichts. Fiel bis zur ' +
			'Auflösung der /katalog-Pfadkollision nie auf, weil dieser Test den öffentlichen OPAC maß.'
	},
	{
		kennung: 'global-suchfeld',
		grund:
			'Globale Suchleiste der Verwaltung (03.09.2026): dieselbe 48-px-Suchpille (components/ui/' +
			'Suchpille.svelte) auf jeder Verwaltungsseite — gemessen von suchpille-einheitlich.spec.js.'
	}
];

/**
 * Wartet, bis die Feldliste eines Bildschirms zur Ruhe gekommen ist.
 *
 * Ersetzt ein blindes waitForTimeout(1000): zu kurz auf einem langsamen Rechner, zu
 * lang auf einem schnellen — vierzehnmal hintereinander.
 *
 * Bewusst NICHT „warte auf mindestens ein Feld": MESSEN erfasst nur input und select,
 * und Bildschirme wie Kiosk oder Statistiken haben davon womöglich gar keines. Eine
 * solche Bedingung liefe dort in den Timeout und machte aus einem grünen Test einen
 * roten. Stattdessen wird auf einen STABILEN Messwert gewartet — zwei gleiche Messungen
 * hintereinander —, was die Null einschließt.
 * @param {import('@playwright/test').Page} page
 */
async function warteAufStabileFelder(page) {
	await page.waitForLoadState('domcontentloaded');
	let vorherige = -1;
	await expect
		.poll(
			async () => {
				const jetzt = (await page.evaluate(MESSEN)).length;
				const stabil = jetzt === vorherige;
				vorherige = jetzt;
				return stabil;
			},
			{ timeout: 10_000, intervals: [100, 150, 200, 300] }
		)
		.toBe(true);
}

// `[role=combobox]` steht hier seit der Ablösung der nativen <select>: Ohne den
// Zusatz hätte die Umstellung die Messmenge still verkleinert — der Test wäre grün
// geblieben, gerade WEIL er die neuen Auswahlfelder nicht mehr gesehen hätte.
const MESSEN = () =>
	[...document.querySelectorAll('input, select, [role="combobox"]')]
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

	/** Misst alles, was gerade sichtbar ist, und schreibt Abweichler mit. */
	async function messen(/** @type {string} */ name) {
		await warteAufStabileFelder(page);
		for (const feld of await page.evaluate(MESSEN)) {
			if (AUSNAHMEN.some((a) => a.kennung === feld.kennung)) continue;
			geprueft++;
			if (feld.hoehe !== CONTROL_HOEHE) {
				abweichler.push(`${name}: <${feld.tag}> „${feld.kennung}" = ${feld.hoehe} px`);
			}
		}
	}

	for (const [name, pfad] of SCREENS) {
		await gehZu(page, pfad);
		await messen(name);

		// Die Einstellungen zeigen seit dem 23.08.2026 EINE Kategorie statt sieben
		// Abschnitten untereinander. Ein einziger Aufruf misst dort seither fünf Felder
		// statt zwanzig — die Zahl unten fiel prompt unter den Nulllauf-Wächter. Statt
		// die Schwelle zu senken (das hätte die Aussage verkleinert, nicht das Problem
		// gelöst) läuft der Test die Kategorien ab.
		if (pfad === '/einstellungen') {
			// „LMF-Aktionen" steht hier, weil die Route /lmf-aktionen (bis 24.08.2026 in
			// SCREENS) zur Einstellungs-Kategorie wurde — ihre Felder bleiben so vermessen.
			for (const k of ['Ausleihe & Fristen', 'Datenschutz & Sitzung', 'Mail', 'LMF-Aktionen']) {
				await einstellungsKategorie(page, k).click();
				await messen(`Einstellungen → ${k}`);
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
	// Gemessen wird „Daten neu laden" und NICHT „Alle anmahnen": Letzterer steht in
	// MahnwesenAktionen hinter {#if countAlle > 0} und erscheint nur, wenn gerade
	// überfällige Ausleihen in der Datenbank stehen. Diese Zeile lag zuvor auf
	// „Alle anmahnen" und lief 30 s in einen Timeout, sobald der Lauf mit frischer
	// Datenbank startete — die Überfälligkeit legt erst mahnwesen.spec.js an, und die
	// Datei sortiert hinter dieser hier. Der Test war damit von der Reihenfolge der
	// Testdateien abhängig, ohne es zu sagen. „Daten neu laden" steht unbedingt in
	// derselben Leiste; für eine Höhenmessung ist der Knopf ohnehin austauschbar.
	const button = page.getByRole('button', { name: 'Daten neu laden' }).first();
	await feld.waitFor();
	await button.waitFor();

	const feldBox = await feld.boundingBox();
	const buttonBox = await button.boundingBox();

	expect(Math.round(feldBox.height)).toBe(CONTROL_HOEHE);
	expect(Math.round(buttonBox.height)).toBe(CONTROL_HOEHE);
});
