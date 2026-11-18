// Gate: Ecken nach Material 3, gemessen an jedem sichtbaren Element jeder Seite der Navigation.
//
// - Jede Ecke liegt auf Googles Skala, M3 (Shape, Corner radius scale): 0, 4, 8, 12, 16, 20, 28,
//   32, 48 dp oder voll. Ausnahmen nennt Google selbst: das Kästchen mit 2 px (material-web,
//   checkbox: „container-shape: 2px“) und der Strich des Reiters mit 3 px oben (primary tab:
//   „active-indicator-shape: 3px 3px 0px 0px“); er ist 3 px hoch und zählt als voll.
// - Jedes Bauteil trägt die Stufe aus seinem Token in material-web (container-shape): Textfeld,
//   Auswahlfeld und ihr Menü „corner-extra-small“ (4 px), angedockter Kalender „corner-large“
//   (16 px), Dialog „corner-extra-large“ (28 px). Eine Suchleiste ist voll („search-bar:
//   container-shape: corner-full“).
// Sieht nicht: Ecken an ::before und ::after, Seiten und Zustände, die der Test nicht öffnet.
import { test, expect } from '@playwright/test';
import { uiLogin, gehZu } from './helpers.js';
import { beobachteDatenanfragen, warteAufInhalt } from './messhilfe.js';
import { SEITEN } from './seiten.js';

/** Misst die Ecken der Seite oder eines Bereichs. Läuft im Browser. @param {string} [bereich] */
function messeEcken(bereich) {
	const SKALA = [0, 4, 8, 12, 16, 20, 28, 32, 48];
	const wurzel = bereich ? document.querySelector(bereich) : document.body;
	const ecken = (/** @type {Element} */ el) => {
		const s = getComputedStyle(el);
		return [
			s.borderTopLeftRadius,
			s.borderTopRightRadius,
			s.borderBottomRightRadius,
			s.borderBottomLeftRadius
		].map((v) => parseFloat(v) || 0);
	};
	const name = (/** @type {Element} */ el) =>
		`${el.tagName.toLowerCase()} „${(el.getAttribute('aria-label') || el.textContent || '').trim().slice(0, 30)}“`;
	const abseits = [];
	const felder = [];
	let gemessen = 0;
	for (const el of wurzel ? [wurzel, ...wurzel.querySelectorAll('*')] : []) {
		const s = getComputedStyle(el);
		const r = el.getBoundingClientRect();
		if (r.width === 0 || r.height === 0 || s.visibility !== 'visible') continue;
		const gemalt =
			s.backgroundColor !== 'rgba(0, 0, 0, 0)' ||
			parseFloat(s.borderTopWidth) > 0 ||
			s.boxShadow !== 'none' ||
			el.tagName === 'IMG';
		if (!gemalt) continue;
		gemessen++;
		const kurz = Math.min(r.width, r.height);
		for (const e of new Set(ecken(el))) {
			const voll = e >= kurz / 2 - 0.5;
			const kaestchen = e === 2 && kurz <= 20;
			if (!voll && !kaestchen && !SKALA.includes(e)) abseits.push(`${name(el)}: ${e} px`);
		}
	}
	// Das Textfeld: das Element mit Rahmen um die Eingabe, sie selbst oder ein naher Vorfahr.
	const eingaben = wurzel?.querySelectorAll(
		'input:not([type=checkbox]):not([type=radio]):not([type=range]):not([type=file]):not([type=hidden]), textarea, [role=combobox]'
	);
	for (const eingabe of eingaben ?? []) {
		if (eingabe.getBoundingClientRect().width === 0) continue;
		let feld = /** @type {Element | null} */ (eingabe);
		for (let i = 0; feld && i < 3 && parseFloat(getComputedStyle(feld).borderTopWidth) === 0; i++)
			feld = feld.parentElement;
		if (!feld || parseFloat(getComputedStyle(feld).borderTopWidth) === 0) continue;
		const r = feld.getBoundingClientRect();
		const e = ecken(feld)[0];
		const suchleiste = e >= Math.min(r.width, r.height) / 2 - 0.5 && r.height >= 40;
		felder.push({ wer: name(eingabe), ecke: e, suchleiste });
	}
	return { abseits, felder, gemessen };
}

test('Ecken nach M3 auf jeder Seite', async ({ page }) => {
	test.setTimeout(300_000);
	const anfragen = beobachteDatenanfragen(page);
	const zaehle = () => page.locator('body *').count();
	/** @type {string[]} */
	const abseits = [];
	/** @type {string[]} */
	const felder = [];
	let gemessen = 0;
	let felderGemessen = 0;
	/** @param {string} seite @param {string} [bereich] */
	const pruefe = async (seite, bereich) => {
		const m = await page.evaluate(messeEcken, bereich);
		gemessen += m.gemessen;
		abseits.push(...m.abseits.map((a) => `${seite}: ${a}`));
		for (const f of m.felder) {
			felderGemessen++;
			if (f.ecke !== 4 && !f.suchleiste) felder.push(`${seite}: ${f.wer} hat ${f.ecke} px`);
		}
	};
	for (const [seite, pfad] of [
		['Anmeldung', '/'],
		['Katalog', '/katalog']
	]) {
		await page.goto(pfad);
		await warteAufInhalt(anfragen, zaehle);
		await pruefe(seite);
	}
	await uiLogin(page);
	for (const [seite, pfad] of SEITEN) {
		await gehZu(page, pfad);
		await warteAufInhalt(anfragen, zaehle);
		await pruefe(seite);
	}

	// Geöffnete Bauteile: Dialog, darin das Auswahlfeld mit seinem Menü und der Kalender.
	const ecke = (/** @type {import('@playwright/test').Locator} */ l) =>
		l.evaluate((el) => parseFloat(getComputedStyle(el).borderTopLeftRadius));
	await gehZu(page, '/schuelerdatei');
	await page.getByRole('button', { name: 'Leser anlegen', exact: true }).click();
	const dialog = page.getByRole('dialog', { name: /Leser anlegen/ });
	await expect(dialog).toBeVisible();
	expect.soft(await ecke(dialog), 'Dialog: corner-extra-large').toBe(28);
	await pruefe('Dialog Leser anlegen', '[role=dialog]');
	await dialog.getByRole('button', { name: 'Kalender öffnen' }).click();
	const kalender = page.getByRole('dialog', { name: 'Datum wählen' });
	await expect(kalender).toBeVisible();
	expect.soft(await ecke(kalender), 'angedockter Kalender: corner-large').toBe(16);
	await page.keyboard.press('Escape');
	await expect(kalender).toBeHidden();
	const auswahl = dialog.getByRole('combobox').first();
	if (await auswahl.count()) {
		await auswahl.click();
		const liste = page.getByRole('listbox').first();
		await expect(liste).toBeVisible();
		expect.soft(await ecke(liste), 'Menü des Auswahlfelds: corner-extra-small').toBe(4);
		await page.keyboard.press('Escape');
	}

	// Selbstschutz: Bricht eine Navigation oder ein Selektor, misst der Test still nichts mehr.
	expect(gemessen, 'kaum Elemente gemessen').toBeGreaterThan(1000);
	expect(felderGemessen, 'kaum Textfelder gemessen').toBeGreaterThan(20);
	expect.soft(abseits, 'Ecke abseits der Skala von M3 (Shape, Corner radius scale)').toEqual([]);
	expect
		.soft(felder, 'Textfeld ohne 4 px: material-web, outlined-text-field „corner-extra-small“')
		.toEqual([]);
});
