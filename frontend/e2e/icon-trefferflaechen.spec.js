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
import { uiLogin, gehZu } from './helpers.js';

const MIN_FLAECHE = 32; // px — Material 3 Icon-Button „extra small"

const SCREENS = [
	['Bestellungen', '/bestellungen'],
	['Mahnwesen', '/mahnwesen'],
	['Schülerdatei', '/schuelerdatei'],
	['Katalog', '/medienkatalog'],
	['Druck-Center', '/druck-center'],
	['Inventur', '/inventur'],
	['Abgänger', '/abgaenger'],
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
			klassen: b.getAttribute('class') || ''
		});
	}
	return alle;
};

/**
 * Wartet, bis die Icon-Button-Liste eines Bildschirms zur Ruhe gekommen ist.
 *
 * Kein networkidle: Die App hält über den Livesync eine SSE-Verbindung offen, das
 * Netz wird nie ruhig — der Test lief damit in den Timeout. Und kein blindes
 * waitForTimeout: zu kurz auf einem langsamen Rechner, zu lang auf einem schnellen.
 * Gewartet wird wie in control-hoehen.spec.js auf einen STABILEN Messwert; das
 * schließt Bildschirme ohne einen einzigen Icon-Button ein.
 * @param {import('@playwright/test').Page} page
 */
async function warteAufStabileButtons(page) {
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
	await uiLogin(page);

	/** @type {string[]} */
	const zuKlein = [];
	let untersucht = 0;

	for (const [name, pfad] of SCREENS) {
		await gehZu(page, pfad);
		await warteAufStabileButtons(page);

		// Die Symbole der Bestellhistorie stecken in einer zugeklappten Zeile.
		//
		// Bewusst OHNE if-visible-Wächter: Ein solcher Wächter hat diesen Test in der
		// ersten Fassung still übersprungen — der Reiter war im Moment der Prüfung noch
		// nicht da, die Zeile wurde nie aufgeklappt, und der Test lief grün, obwohl ein
		// 22-px-Button auf dem Bildschirm stand. Ein übersprungener Messpunkt sieht aus
		// wie ein bestandener. Deshalb hier harte Erwartungen: Findet der Test seinen
		// Messpunkt nicht, ist er rot und nicht grün.
		if (pfad === '/bestellungen') {
			await page.getByRole('tab', { name: 'Bestellhistorie' }).click();
			const zeilen = page.locator('tbody tr');
			await zeilen.first().waitFor();
			await zeilen.first().click();
			// Seit dem 08.08.2026 fuehrt die Zeile in die Detailansicht, statt aufzuklappen —
			// die Symbole (Nachdruck, Titelsatz) stehen dort an den Positionen. Gewartet wird
			// auf deren Ueberschrift: Bleibt sie aus, ist der Messpunkt weg und der Test rot.
			await page.getByRole('heading', { name: 'Bestellte Titel' }).waitFor();
			await warteAufStabileButtons(page);
		}

		const gefunden = await page.evaluate(MESSEN);
		untersucht += gefunden.length;
		for (const t of gefunden.filter((t) => t.breite < MIN_FLAECHE || t.hoehe < MIN_FLAECHE)) {
			zuKlein.push(`${name}: "${t.label}" ist ${t.breite}×${t.hoehe} px [${t.klassen}]`);
		}
	}

	// Eingeklappte Navigation: Der Ausklapp-Umschalter existiert nur in diesem
	// Zustand und ist ein anderer Button als der zum Einklappen — beim ersten
	// Messen war er unsichtbar und dadurch übersehen.
	await page.goto('/bestellungen');
	await warteAufStabileButtons(page);
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
});
