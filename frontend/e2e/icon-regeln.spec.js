// Gate: Die Regeln von Material 3 für Icons, gemessen an jedem sichtbaren Icon jeder Seite der
// Navigation, hell und dunkel, dazu die öffentlichen Seiten und der Flur-Monitor (immer dunkel).
//
// - Grade nach dem Grund, M3 (Icons, Grade): „the default grade for a dark icon on a light
//   background is 0, and -25 for a light icon on a dark background“. Hell und dunkel entscheidet
//   die gemessene Helligkeit von Icon und Grund, keine Klasse: So meldet der Test auch eine
//   kräftige Fläche, die styles/icons.css nicht kennt.
// - Ein Icon in einer Textzeile, M3 (Icons, Using Material Symbols with typography): „Use the
//   same size for your Material Symbols and text“ und „Shift down the baseline of symbols to
//   approximately 11.5% of the text size“ (Icon imText).
// - Ein Icon ragt nicht aus seiner Hülle; die Hüllen in Bauteilen sieht nur der Browser.
// Vorab: Die Schrift trägt die Achsen GRAD und FILL. Ohne sie bliebe jede Einstellung wirkungslos,
// und die Messung der Grade sähe nur die Angabe im Stil.
import { test, expect } from '@playwright/test';
import { uiLogin, gehZu } from './helpers.js';
import { beobachteDatenanfragen, warteAufInhalt, zeigeMonitorMitPlantag } from './messhilfe.js';
import { SEITEN } from './seiten.js';

const OEFFENTLICH = [
	['Anmeldung', '/'],
	['Katalog', '/katalog']
];

/** Misst jedes sichtbare Icon der Seite. Läuft im Browser, deshalb ohne Bezug nach außen. */
function messeIcons() {
	const leinwand = document.createElement('canvas').getContext('2d', { willReadFrequently: true });
	/** Jede Farbschreibweise (auch oklab aus color-mix) als [r, g, b, a] über die Leinwand. */
	const farbe = (/** @type {string} */ wert) => {
		if (!leinwand) return [0, 0, 0, 0];
		leinwand.clearRect(0, 0, 1, 1);
		leinwand.fillStyle = '#000';
		leinwand.fillStyle = wert;
		leinwand.fillRect(0, 0, 1, 1);
		const [r, g, b, a] = leinwand.getImageData(0, 0, 1, 1).data;
		return [r, g, b, a / 255];
	};
	const kanal = (/** @type {number} */ c) =>
		c / 255 <= 0.04045 ? c / 255 / 12.92 : ((c / 255 + 0.055) / 1.055) ** 2.4;
	const helligkeit = (/** @type {number[]} */ [r, g, b]) =>
		0.2126 * kanal(r) + 0.7152 * kanal(g) + 0.0722 * kanal(b);

	/** Der gezeichnete Grund hinter einem Element; null hinter einem Bild oder Verlauf. */
	const grund = (/** @type {Element | null} */ el) => {
		/** @type {number[][]} */
		const schichten = [];
		for (let e = el; e; e = e.parentElement) {
			const stil = getComputedStyle(e);
			if (stil.backgroundImage !== 'none') return null;
			const f = farbe(stil.backgroundColor);
			if (f[3] > 0) schichten.push(f);
			if (f[3] >= 0.999) break;
		}
		let [r, g, b] = [255, 255, 255];
		for (const [sr, sg, sb, a] of schichten.reverse()) {
			[r, g, b] = [sr * a + r * (1 - a), sg * a + g * (1 - a), sb * a + b * (1 - a)];
		}
		return [r, g, b];
	};

	return [...document.querySelectorAll('.icon')].flatMap((icon) => {
		const stil = getComputedStyle(icon);
		const rahmen = icon.getBoundingClientRect();
		if (rahmen.width === 0 || stil.visibility !== 'visible') return [];
		const eltern = /** @type {HTMLElement} */ (icon.parentElement);
		const elternStil = getComputedStyle(eltern);
		const benannt = icon.closest('[aria-label], button, a, label, [role]');
		const wer =
			`${icon.getAttribute('data-icon')} in ` +
			(benannt?.getAttribute('aria-label') ||
				benannt?.textContent?.trim().slice(0, 40) ||
				eltern.tagName);
		const hinten = grund(eltern);
		const gradIst = Number(/"GRAD"\s+(-?[\d.]+)/.exec(stil.fontVariationSettings)?.[1] ?? 0);
		const gradSoll = hinten && helligkeit(farbe(stil.color)) > helligkeit(hinten) ? -25 : 0;

		const imFluss = !/flex|grid/.test(elternStil.display);
		const textDaneben = [...eltern.childNodes].some(
			(n) => n.nodeType === Node.TEXT_NODE && n.textContent?.trim()
		);
		const schrift = parseFloat(elternStil.fontSize);
		const tiefer = parseFloat(stil.verticalAlign);
		const imText =
			imFluss && textDaneben
				? Math.abs(parseFloat(stil.fontSize) - schrift) <= 0.5 &&
					Math.abs(tiefer + 0.115 * schrift) <= 0.6
				: true;

		const h = eltern.getBoundingClientRect();
		const ragt =
			!/^(inline|contents)$/.test(elternStil.display) &&
			(rahmen.left < h.left - 0.5 ||
				rahmen.right > h.right + 0.5 ||
				rahmen.top < h.top - 0.5 ||
				rahmen.bottom > h.bottom + 0.5);
		return [
			{ wer, gradIst, gradSoll, unsicher: !hinten, imText, ragt, inZeile: imFluss && textDaneben }
		];
	});
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {string} seite
 * @param {{ grade: string[], text: string[], ragt: string[], zahl: number, hellAufDunkel: number, imText: number }} befunde
 */
async function pruefe(page, seite, befunde) {
	await page.mouse.move(0, 0);
	for (const i of await page.evaluate(messeIcons)) {
		befunde.zahl++;
		if (i.gradSoll === -25) befunde.hellAufDunkel++;
		if (i.inZeile) befunde.imText++;
		if (!i.unsicher && i.gradIst !== i.gradSoll)
			befunde.grade.push(`${seite}: ${i.wer} hat Grade ${i.gradIst}, Soll ${i.gradSoll}`);
		if (!i.imText) befunde.text.push(`${seite}: ${i.wer}`);
		if (i.ragt) befunde.ragt.push(`${seite}: ${i.wer}`);
	}
}

/** @param {ReturnType<typeof neueBefunde>} b @param {string} wo */
function bewerte(b, wo) {
	expect
		.soft(
			b.grade,
			`Grade passt nicht zum Grund (${wo}).\nBehebung: kräftige Fläche in styles/icons.css nennen.\n`
		)
		.toEqual([]);
	expect.soft(b.text, `Icon in einer Textzeile ohne imText (${wo}).\n`).toEqual([]);
	expect.soft(b.ragt, `Icon ragt aus seiner Hülle (${wo}).\n`).toEqual([]);
}

const neueBefunde = () => ({
	grade: /** @type {string[]} */ ([]),
	text: /** @type {string[]} */ ([]),
	ragt: /** @type {string[]} */ ([]),
	zahl: 0,
	hellAufDunkel: 0,
	imText: 0
});

test('Die Icon-Schrift zeichnet mit GRAD und FILL sichtbar anders', async ({ page }) => {
	await page.goto('/');
	// Jedes Zeichen an eigener Stelle: Übereinander zeigte jedes Bildschirmfoto das oberste.
	await page.evaluate(async () => {
		const achsen = { g0: "'GRAD' 0", g25: "'GRAD' -25", f0: "'FILL' 0", f1: "'FILL' 1" };
		Object.entries(achsen).forEach(([id, wert], i) => {
			const s = document.createElement('span');
			Object.assign(s, { id, className: 'icon' });
			s.dataset.icon = 'delete';
			s.style.cssText = `font-size:48px;font-variation-settings:${wert};position:fixed;top:0;left:${i * 64}px;z-index:9999;background:#fff;color:#000`;
			document.body.append(s);
		});
		await document.fonts.load("48px 'Material Symbols Outlined'");
	});
	const bild = (/** @type {string} */ id) => page.locator(`#${id}`).screenshot();
	expect((await bild('g0')).equals(await bild('g25')), 'GRAD ohne Wirkung').toBe(false);
	expect((await bild('f0')).equals(await bild('f1')), 'FILL ohne Wirkung').toBe(false);
});

for (const schema of /** @type {const} */ (['light', 'dark'])) {
	test.describe(`Icons nach M3, ${schema}`, () => {
		test.use({ colorScheme: schema });
		test(`jede Seite, ${schema}`, async ({ page }) => {
			test.setTimeout(300_000);
			await page.emulateMedia({ reducedMotion: 'reduce' });
			const anfragen = beobachteDatenanfragen(page);
			const zaehle = () => page.locator('.icon').count();
			const b = neueBefunde();
			for (const [seite, pfad] of OEFFENTLICH) {
				await page.goto(pfad);
				await warteAufInhalt(anfragen, zaehle);
				await pruefe(page, seite, b);
			}
			await uiLogin(page);
			const dunkel = await page.evaluate(() =>
				document.documentElement.classList.contains('schema-dunkel')
			);
			expect(dunkel, 'Farbschema').toBe(schema === 'dark');
			for (const [seite, pfad] of SEITEN) {
				await gehZu(page, pfad);
				await warteAufInhalt(anfragen, zaehle);
				await pruefe(page, seite, b);
			}
			// Selbstschutz: Bricht eine Navigation oder ein Selektor, misst der Test still nichts mehr.
			expect(b.zahl, 'kaum Icons gemessen').toBeGreaterThan(150);
			// Hell und dunkel muss die Messung je einmal unterscheiden: im dunklen Schema fast jedes
			// Icon, im hellen die Icons auf kräftigen Flächen.
			expect(b.hellAufDunkel, 'kein helles Icon auf dunklem Grund gesehen').toBeGreaterThan(0);
			bewerte(b, schema);
		});
	});
}

test('Icons nach M3 auf dem Flur-Monitor, jede Folie', async ({ page }) => {
	test.setTimeout(120_000);
	await page.emulateMedia({ reducedMotion: 'reduce' });
	// Icons in einer Textzeile stehen nur auf den Folien mit Büchern. Eine davon bringt der Test
	// selbst mit, statt von den Büchern anderer Specs zu leben.
	await zeigeMonitorMitPlantag(page, {
		buch_des_monats: {
			id: 'e2e-monitor',
			titel: 'Monitor-Probe',
			autor: '',
			cover_url: '',
			isbn: ''
		}
	});
	await page.goto('/monitor');
	const anzeige = page.getByTestId('monitor-folie');
	await anzeige.waitFor();
	const b = neueBefunde();
	const gesehen = new Set();
	for (let i = 0; i < 4; i++) {
		const folie = (await anzeige.textContent())?.trim() ?? '';
		if (gesehen.has(folie)) break;
		gesehen.add(folie);
		await pruefe(page, `Monitor ${folie.slice(0, 30)}`, b);
		try {
			await expect(anzeige).not.toHaveText(folie, { timeout: 20_000 });
		} catch {
			break; // nur diese Folie hat Inhalt
		}
		// Der Text ist gewechselt, gemessen wird erst, wenn die neue Folie gezeichnet ist.
		await page.evaluate(
			() => new Promise((fertig) => requestAnimationFrame(() => requestAnimationFrame(fertig)))
		);
	}
	expect(b.imText, 'kein Icon in einer Textzeile gesehen: Folien ohne Inhalt?').toBeGreaterThan(0);
	expect([...gesehen], 'die Folie „Heute" des Plantags wurde gemessen').toContain(
		'Heute Büchertausch'
	);
	bewerte(b, 'Monitor');
});
