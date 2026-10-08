// Gate gegen den fünften Icon-Button.
//
// Vor der Normierung hatten Buttons, die nur ein Symbol tragen, kein gemeinsames
// Maß: 20×20 (Cover-Vorschau, ganz ohne Polsterung), 22×22 (Titelsatz in der
// Bestellhistorie, darin ein 14-px-Symbol), 30×30 (Navigation ein-/ausklappen)
// und 38×28 (Banner schließen) — während der beschriftete Nachbar daneben längst
// auf 36 px stand. Material 3 nennt für den Icon-Button „extra small" 32 dp;
// das ist die Untergrenze, die hier gilt (siehe .icon-btn in app.css).
//
// Warum ein E2E-Test und kein Klassen-Grep: Die Fläche entsteht erst im Browser
// aus Symbolgröße + Polsterung, und die statische Inventur war nachweislich
// falsch — sie zählte dekorative Symbole und Symbole in beschrifteten Buttons
// mit und kam auf 29 Fundstellen, wo es real fünf waren. Gemessen wird deshalb
// die Bounding-Box an der laufenden Anwendung.
//
// Zwei Zustände, die man leicht übersieht und die dieser Test deshalb ausdrücklich
// herstellt: die geöffnete BESTELLUNG (die Symbole stehen in der Detailansicht, die
// nur ein Zeilenklick erreicht) und die EINGEKLAPPTE Navigation (ihr Umschalter ist
// ein anderer Button als der zum Einklappen — er war beim ersten Messen unsichtbar
// und dadurch übersehen).
import { test, expect } from '@playwright/test';
import { uiLogin, gehZu, seedBestellung, oeffneBestellungsDetail } from './helpers.js';
import { beobachteDatenanfragen, warteAufInhalt, pruefeNichtZuFruehGemessen } from './messhilfe.js';

const MIN_FLAECHE = 32; // px — Material 3 Icon-Button „extra small"

// Die Bestellung, deren Detailansicht gemessen wird, legt der Test selbst an: Er läuft
// damit auch allein und auf einer leeren Bestellhistorie.
let marke = '';
/** @type {(() => void) | undefined} */
let aufraeumen;
test.beforeEach(() => {
	({ marke, aufraeumen } = seedBestellung());
});
test.afterEach(() => {
	aufraeumen?.();
	aufraeumen = undefined;
});

const SCREENS = [
	['Bestellungen', '/bestellungen'],
	['Mahnwesen', '/mahnwesen'],
	['Leserdatei', '/schuelerdatei'],
	['Katalog', '/medienkatalog'],
	['Druck-Center', '/druck-center'],
	['Inventur', '/inventur'],
	['Abgänger', '/abgaenger'],
	['Schuljahreswechsel', '/schuljahr'],
	['Klassensätze', '/schulklassen'],
	['Mein Portal', '/kollegium-portal'],
	['Einstellungen', '/einstellungen']
];

/**
 * Sammelt jeden sichtbaren Button, dessen Inhalt praktisch nur ein Symbol ist —
 * kein Text oder höchstens eine kurze Zahl daneben (der Nachdruck-Button trägt
 * die Anzahl offener Etiketten). Beschriftete Buttons sind ausgenommen: Ihre
 * Höhe regelt die Control-Höhe, nicht diese Regel.
 *
 * Läuft im Browser, wird als Funktionsrumpf übertragen — deshalb ohne Import.
 */
const MESSEN = () => {
	const alle = [];
	for (const b of document.querySelectorAll('button, [role="button"]')) {
		if (!b.querySelector('svg')) continue;
		const r = b.getBoundingClientRect();
		if (r.width === 0 || r.height === 0) continue; // unsichtbar
		if ((b.textContent || '').trim().length > 3) continue; // beschriftet
		alle.push({
			breite: Math.round(r.width),
			hoehe: Math.round(r.height),
			label: b.getAttribute('aria-label') || b.getAttribute('title') || '(ohne Label)',
			klassen: b.getAttribute('class') || '',
			radius: Number.parseFloat(getComputedStyle(b).borderTopLeftRadius) || 0
		});
	}
	return alle;
};

/**
 * Die zu kleinen unter ihnen.
 * @param {import('@playwright/test').Page} page
 */
async function zuKleine(page) {
	const alle = await page.evaluate(MESSEN);
	return alle.filter((t) => t.breite < MIN_FLAECHE || t.hoehe < MIN_FLAECHE);
}

test('Icon-Buttons halten die Mindest-Trefferfläche', async ({ page }) => {
	test.setTimeout(180_000);
	const anfragen = beobachteDatenanfragen(page);
	const zaehle = async () => (await page.evaluate(MESSEN)).length;
	await uiLogin(page);

	/** @type {string[]} */
	const zuKlein = [];
	let untersucht = 0;
	/** @type {string[]} */
	const eckig = [];
	let symbolknoepfe = 0;

	for (const [name, pfad] of SCREENS) {
		await gehZu(page, pfad);
		let gemessen = await warteAufInhalt(anfragen, zaehle);

		// Die Symbole der Bestellhistorie (Nachdruck, Titelsatz) stehen in der Detailansicht
		// einer Bestellung. Der Helfer wartet hart auf beide: Ein übersprungener Messpunkt
		// sähe aus wie ein bestandener.
		if (pfad === '/bestellungen') {
			await oeffneBestellungsDetail(page, marke);
			gemessen = await warteAufInhalt(anfragen, zaehle);
		}

		const gefunden = await page.evaluate(MESSEN);
		await pruefeNichtZuFruehGemessen(page, anfragen, zaehle, gemessen, name);
		untersucht += gefunden.length;
		for (const t of gefunden.filter((t) => t.breite < MIN_FLAECHE || t.hoehe < MIN_FLAECHE)) {
			zuKlein.push(`${name}: "${t.label}" ist ${t.breite}×${t.hoehe} px [${t.klassen}]`);
		}
		for (const t of gefunden.filter((t) => /(^|\s)icon-btn(\s|$)/.test(t.klassen))) {
			symbolknoepfe++;
			if (t.radius < Math.min(t.breite, t.hoehe) / 2) {
				eckig.push(
					`${name}: "${t.label}" hat ${t.radius} px Rundung bei ${t.breite}×${t.hoehe} px`
				);
			}
		}
	}

	// Eingeklappte Navigation: Der Ausklapp-Umschalter existiert nur in diesem
	// Zustand und ist ein anderer Button als der zum Einklappen — beim ersten
	// Messen war er unsichtbar und dadurch übersehen.
	await page.goto('/bestellungen');
	await warteAufInhalt(anfragen, zaehle);
	await page.getByRole('button', { name: 'Navigation einklappen' }).click();
	await page.getByRole('button', { name: 'Navigation ausklappen' }).waitFor();
	for (const t of await zuKleine(page)) {
		zuKlein.push(
			`Navigation eingeklappt: "${t.label}" ist ${t.breite}×${t.hoehe} px [${t.klassen}]`
		);
	}

	// Selbstschutz: Bricht eine Navigation oder ein Selektor, misst der Test still
	// nichts mehr und wäre für immer grün. Die Untergrenze ist bewusst grob — sie
	// soll den Totalausfall fangen, nicht eine Zahl festschreiben.
	expect(
		untersucht,
		'Der Test hat fast keine Icon-Buttons gefunden — vermutlich bricht eine Navigation, ' +
			'und er misst in Wahrheit nichts mehr.'
	).toBeGreaterThan(20);

	expect(
		zuKlein,
		`Icon-Buttons unter ${MIN_FLAECHE}×${MIN_FLAECHE} px.\n` +
			`Behebung: die Klasse .icon-btn setzen (app.css, @layer components) statt eigener Polsterung.\n` +
			`Eine bewusste Ausnahme braucht ein explizites min-w-*/min-h-* an der Fundstelle UND eine\n` +
			`Begründung im Code — nicht hier im Test.\n\n` +
			zuKlein.join('\n')
	).toEqual([]);

	// Rund ist die Form des Symbolknopfs (M3, Icon buttons, Specs: Shape „Round (default)"). An
	// ihr hängt auch die Schicht beim Zeigen: Sie erbt den Radius des Knopfs.
	expect(symbolknoepfe, 'Der Test hat keinen Knopf mit .icon-btn gefunden.').toBeGreaterThan(0);
	expect(
		eckig,
		`Symbolknöpfe mit .icon-btn, die nicht rund sind (${symbolknoepfe} gemessen).\n` +
			`Behebung: Die Rundung steht in .icon-btn (styles/komponenten.css).\n\n` +
			eckig.join('\n')
	).toEqual([]);
});
