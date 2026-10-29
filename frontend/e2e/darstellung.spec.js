import { test, expect } from '@playwright/test';
import { uiLogin, gehZu, seedSQL, uniqueSuffix } from './helpers.js';

// „Dunkles Design" schaltet jeder Rechner für sich, ohne Wahl gilt das System
// (stores/darstellung.svelte.js, public/darstellung.js). Gemessen wird an den Farben, die daraus
// werden; Ausdruck und Papier-Vorschauen bleiben hell.

/** @param {import('@playwright/test').Page} page */
const dunkel = (page) =>
	page.evaluate(() => document.documentElement.classList.contains('schema-dunkel'));

/**
 * Der Wert einer Farbrolle an einem Element.
 * @param {import('@playwright/test').Locator} element
 * @param {string} name
 */
const rolle = (element, name) =>
	element.evaluate((el, n) => getComputedStyle(el).getPropertyValue(n).trim(), name);

/** @param {import('@playwright/test').Page} page */
const schalter = (page) => page.getByRole('switch', { name: 'Dunkles Design' });

/**
 * Hält fest, ob die Seite dunkel stand, als das Dokument gelesen war: Erst danach laufen die
 * Module des Programms (HTML, „the end": readyState interactive vor den aufgeschobenen Skripten).
 * @param {import('@playwright/test').Page} page
 */
const merkeSchemaVorDemProgramm = (page) =>
	page.addInitScript(() => {
		document.addEventListener('readystatechange', () => {
			if (document.readyState !== 'interactive') return;
			const html = document.documentElement;
			html.dataset.schemaVorDemProgramm = String(html.classList.contains('schema-dunkel'));
		});
	});

test('Ohne Wahl folgt die Seite dem System, schon bevor das Programm läuft', async ({ page }) => {
	await merkeSchemaVorDemProgramm(page);
	for (const farbschema of /** @type {const} */ (['dark', 'light'])) {
		await page.emulateMedia({ colorScheme: farbschema });
		await page.goto('/');
		const erwartet = String(farbschema === 'dark');
		await expect(page.locator('html')).toHaveAttribute('data-schema-vor-dem-programm', erwartet);
		expect(String(await dunkel(page)), farbschema).toBe(erwartet);
	}
});

test('„Dunkles Design" gilt auf diesem Rechner auch nach dem Neuladen', async ({ page }) => {
	await merkeSchemaVorDemProgramm(page);
	await page.emulateMedia({ colorScheme: 'light' });
	await uiLogin(page);
	expect(await dunkel(page)).toBe(false);
	await expect(schalter(page)).toHaveAttribute('aria-checked', 'false');

	await schalter(page).click();
	await expect.poll(() => dunkel(page)).toBe(true);
	// Die Fläche der Seitenleiste ist surface im dunklen Schema (rollen.css).
	await expect(page.locator('aside > div').first()).toHaveCSS(
		'background-color',
		'rgb(18, 19, 22)'
	);

	await page.reload();
	await expect(page.locator('html')).toHaveAttribute('data-schema-vor-dem-programm', 'true');
	await expect(schalter(page)).toHaveAttribute('aria-checked', 'true');
});

test('Ausgeschaltet bleibt die Seite hell, auch wenn das System dunkel steht', async ({ page }) => {
	await page.emulateMedia({ colorScheme: 'dark' });
	await uiLogin(page);
	expect(await dunkel(page)).toBe(true);
	await expect(schalter(page)).toHaveAttribute('aria-checked', 'true');
	await schalter(page).click();
	await expect.poll(() => dunkel(page)).toBe(false);
	await page.reload();
	await expect(schalter(page)).toHaveAttribute('aria-checked', 'false');
	expect(await dunkel(page)).toBe(false);
});

test('Ein zweites Fenster dieses Browsers zieht mit', async ({ page, context }) => {
	await page.emulateMedia({ colorScheme: 'light' });
	await uiLogin(page);
	const zweites = await context.newPage();
	await zweites.emulateMedia({ colorScheme: 'light' });
	await zweites.goto('/katalog');
	await schalter(page).click();
	await expect.poll(() => dunkel(zweites)).toBe(true);
	await zweites.close();
});

test('Ohne Speicher im Browser gilt die Wahl für dieses Fenster', async ({ page }) => {
	/** @type {string[]} */
	const fehler = [];
	page.on('pageerror', (e) => fehler.push(e.message));
	await page.addInitScript(() => {
		Object.defineProperty(window, 'localStorage', {
			get() {
				throw new DOMException('Website-Daten gesperrt', 'SecurityError');
			}
		});
	});
	await page.emulateMedia({ colorScheme: 'light' });
	await uiLogin(page);
	await schalter(page).click();
	await expect.poll(() => dunkel(page)).toBe(true);
	expect(fehler).toEqual([]);
});

test('Der Ausdruck nimmt das helle Schema', async ({ page }) => {
	await page.emulateMedia({ colorScheme: 'dark' });
	await uiLogin(page);
	const html = page.locator('html');
	expect(await rolle(html, '--color-on-surface')).toBe('#e3e2e6');
	await page.emulateMedia({ colorScheme: 'dark', media: 'print' });
	expect(await rolle(html, '--color-on-surface')).toBe('#1a1c1e');
});

test('Etikettenbogen und Ausweiskarte bleiben im dunklen Schema hell', async ({ page }) => {
	const s = uniqueSuffix();
	seedSQL(`
        WITH t AS (
            INSERT INTO buecher_titel (isbn, titel, autor, verlag)
            VALUES ('978d${s}', 'E2E Papier ${s}', 'Papier Autor', 'Papierverlag') RETURNING id
        )
        INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
        SELECT id, 'B-PAP-${s}', true FROM t;
    `);
	try {
		await page.emulateMedia({ colorScheme: 'dark' });
		await uiLogin(page);
		await gehZu(page, '/druck-center');

		await page.getByRole('tab', { name: 'Schülerausweise' }).click();
		const karte = page.locator('[data-ausweis-vorschau]');
		await expect(karte).toBeVisible();
		expect(await rolle(karte, '--color-on-surface-variant')).toBe('#42474e');

		await page.getByText('Buch-Etiketten', { exact: true }).first().click();
		await page.getByRole('searchbox', { name: 'Buchtitel im Katalog suchen' }).fill(`Papier ${s}`);
		await page.getByRole('button', { name: new RegExp(`E2E Papier ${s}`) }).click();
		// Der Etikettenbogen ist ein Bild des Druck-PDFs: Sein Papier bleibt weiß, auch im dunklen
		// Schema. Gelesen wird ein Bildpunkt am Rand des Blatts, oberhalb der ersten Etikettenreihe,
		// so wie er auf dem Bildschirm steht: Ein Filter, der das Blatt abdunkelte, fiele auf.
		const blatt = page.getByTestId('etiketten-blatt');
		await expect(blatt).toBeVisible();
		const foto = (await blatt.screenshot()).toString('base64');
		const papier = await page.evaluate(async (daten) => {
			const bild = new Image();
			bild.src = `data:image/png;base64,${daten}`;
			await bild.decode();
			const leinwand = document.createElement('canvas');
			leinwand.width = bild.naturalWidth;
			leinwand.height = bild.naturalHeight;
			const kontext = /** @type {CanvasRenderingContext2D} */ (leinwand.getContext('2d'));
			kontext.drawImage(bild, 0, 0);
			return Array.from(kontext.getImageData(10, 10, 1, 1).data);
		}, foto);
		expect(papier).toEqual([255, 255, 255, 255]);
		expect(await dunkel(page)).toBe(true);
	} finally {
		seedSQL(`DELETE FROM buecher_titel WHERE isbn = '978d${s}';`);
	}
});
