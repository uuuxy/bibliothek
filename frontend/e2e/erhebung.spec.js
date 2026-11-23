// Gate: Erhebung nach Material 3 (Elevation), gemessen im Browser.
//
// Googles Stufen in material-web (elevation/internal/_elevation.scss): zwei Schatten in der
// Rolle shadow, der erste mit 30 %, der zweite mit 15 %; level1 „0 1px 2px 0, 0 1px 3px 1px",
// level2 „0 1px 2px 0, 0 2px 6px 2px", level3 „0 1px 3px 0, 0 4px 8px 3px", level4
// „0 2px 3px 0, 0 6px 10px 4px", level5 „0 4px 4px 0, 0 8px 12px 6px". Jeder Schatten auf den
// Seiten der Navigation ist eine dieser Stufen; die geöffneten Bauteile tragen die Stufe aus
// ihrem Token (container-elevation): Dialog level3, angedockter Kalender level3, Menü des
// Auswahlfelds level2, Snackbar level3.
// Sieht nicht: Schatten an ::before und ::after, Ringe und Schatten nach innen (sie sind keine
// Erhebung), Zustände, die der Test nicht öffnet.
import { test, expect } from '@playwright/test';
import { uiLogin, gehZu } from './helpers.js';
import { beobachteDatenanfragen, warteAufInhalt } from './messhilfe.js';
import { SEITEN } from './seiten.js';

const STUFEN = {
	level1: '0px 1px 2px 0px|0px 1px 3px 1px',
	level2: '0px 1px 2px 0px|0px 2px 6px 2px',
	level3: '0px 1px 3px 0px|0px 4px 8px 3px',
	level4: '0px 2px 3px 0px|0px 6px 10px 4px',
	level5: '0px 4px 4px 0px|0px 8px 12px 6px'
};

/**
 * Liest die Schatten der Seite oder eines Elements und benennt ihre Stufe. Läuft im Browser.
 * @param {{ wahl?: string, stufen: Record<string, string> }} p
 */
function messeErhebung({ wahl, stufen }) {
	/** @param {Element} el */
	const stufe = (el) => {
		const s = getComputedStyle(el).boxShadow;
		if (s === 'none') return 'none';
		// Ein Schatten ist „rgba(…) 0px 1px 2px 0px"; Ringe (0 0 0 n) und inset sind keine Erhebung.
		const FARBE = /rgba?\([^)]*\)|color\([^)]*\)/;
		const teile = s
			.split(/,(?![^(]*\))/)
			.map((t) => t.trim())
			.filter((t) => !/\binset\b/.test(t))
			.map((t) => ({
				farbe: t.match(FARBE)?.[0] ?? '',
				mass: t.replace(FARBE, '').trim()
			}))
			.filter((t) => !/^0px 0px 0px/.test(t.mass));
		if (teile.length === 0) return 'none';
		const masse = teile.map((t) => t.mass).join('|');
		const name = Object.entries(stufen).find(([, m]) => m === masse)?.[0];
		// Deckkraft: rgba(r, g, b, a) oder color(srgb r g b / a).
		const deckkraft = teile.map((t) => {
			const a = t.farbe.match(/\/\s*([\d.]+)\)|,\s*([\d.]+)\)$/);
			return Number(a?.[1] ?? a?.[2] ?? 1);
		});
		return name && deckkraft[0] === 0.3 && deckkraft[1] === 0.15 ? name : `abseits: ${s}`;
	};
	if (wahl) {
		const el = document.querySelector(wahl);
		return el ? stufe(el) : 'fehlt';
	}
	const name = (/** @type {Element} */ el) =>
		`${el.tagName.toLowerCase()} „${(el.getAttribute('aria-label') || el.textContent || '').trim().slice(0, 30)}“`;
	const abseits = [];
	let gemessen = 0;
	for (const el of document.body.querySelectorAll('*')) {
		if (el.getBoundingClientRect().width === 0) continue;
		const s = stufe(el);
		if (s === 'none') continue;
		gemessen++;
		if (s.startsWith('abseits')) abseits.push(`${name(el)}: ${s}`);
	}
	return { abseits, gemessen };
}

test('Jeder Schatten ist eine Stufe von Google, geöffnete Bauteile tragen ihre Stufe', async ({
	page
}) => {
	test.setTimeout(300_000);
	const anfragen = beobachteDatenanfragen(page);
	const zaehle = () => page.locator('body *').count();
	/** @type {string[]} */
	const abseits = [];
	let gemessen = 0;
	await uiLogin(page);
	for (const [seite, pfad] of SEITEN) {
		await gehZu(page, pfad);
		await warteAufInhalt(anfragen, zaehle);
		const m = await page.evaluate(messeErhebung, { stufen: STUFEN });
		gemessen += m.gemessen;
		abseits.push(...m.abseits.map((a) => `${seite}: ${a}`));
	}
	/** @param {string} wahl */
	const stufeVon = (wahl) => page.evaluate(messeErhebung, { wahl, stufen: STUFEN });

	await gehZu(page, '/schuelerdatei');
	await page.getByRole('button', { name: 'Leser anlegen', exact: true }).click();
	const dialog = page.getByRole('dialog', { name: /Leser anlegen/ });
	await expect(dialog).toBeVisible();
	expect.soft(await stufeVon('[role=dialog]'), 'Dialog: md.comp.dialog level3').toBe('level3');
	await dialog.getByRole('button', { name: 'Kalender öffnen' }).click();
	await expect(page.getByRole('dialog', { name: 'Datum wählen' })).toBeVisible();
	expect
		.soft(
			await stufeVon('[role=dialog][aria-label="Datum wählen"]'),
			'Kalender: date-picker-docked level3'
		)
		.toBe('level3');
	await page.keyboard.press('Escape');
	const auswahl = dialog.getByRole('combobox').first();
	if (await auswahl.count()) {
		await auswahl.click();
		await expect(page.getByRole('listbox').first()).toBeVisible();
		expect.soft(await stufeVon('[role=listbox]'), 'Menü: md.comp.menu level2').toBe('level2');
		await page.keyboard.press('Escape');
	}
	await page.keyboard.press('Escape');

	// Snackbar: eine unbekannte Nummer an der Theke.
	await gehZu(page, '/kiosk');
	// Erst wenn die Theke steht, nimmt jemand den Scan an.
	await expect(page.locator('#omnibox-input')).toBeFocused();
	await page.keyboard.type('ZZZ0000000999');
	await page.keyboard.press('Enter');
	const meldung = page.locator('[role=alert], [role=status]').filter({ hasText: /./ }).last();
	await expect(meldung).toBeVisible();
	expect.soft(await meldung.evaluate((el) => getComputedStyle(el).boxShadow)).not.toBe('none');
	const snackbar = await meldung.evaluate((el) => (el.setAttribute('data-messung', ''), true));
	expect(snackbar).toBe(true);
	expect.soft(await stufeVon('[data-messung]'), 'Snackbar: level3').toBe('level3');

	// Im Grundzustand tragen die Seiten keinen Schatten; erhoben sind Überlagerungen und Karten,
	// die erst mit Daten oder einer Auswahl erscheinen. Gemessen wird trotzdem, für jeden neuen.
	test.info().annotations.push({ type: 'Schatten im Grundzustand', description: String(gemessen) });
	expect.soft(abseits, 'Schatten abseits von Googles Stufen (material-web, elevation)').toEqual([]);
});
