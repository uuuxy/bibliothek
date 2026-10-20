import { test, expect } from '@playwright/test';
import { uiLogin, gehZu, seedSQL, uniqueSuffix, einstellungsKategorie } from './helpers.js';

// Material 3, Text fields: „It's strongly recommended to show an error icon when the text field
// is in the error state." ui/Feld setzt das Symbol rechts ins Feld, vor nachlaufende Knöpfe. Hier
// wird gemessen: Es liegt im Feld und im Innenabstand, also über keinem Text und keinem Symbol
// des Browsers (Kalender am Datumsfeld); Knöpfe im Feld bleiben stehen; wer nach dem Fehler
// tippt, bleibt im Feld.

test.use({ viewport: { width: 1280, height: 800 } });

const s = uniqueSuffix().slice(0, 6);

/** @param {import('@playwright/test').Locator} feld */
async function miss(feld) {
	return feld.evaluate((el) => {
		const kasten = (/** @type {Element} */ e) => {
			const b = e.getBoundingClientRect();
			return { links: b.left, rechts: b.right, oben: b.top, unten: b.bottom };
		};
		const zeichen = el.parentElement?.querySelector('svg.lucide-circle-alert');
		return {
			feld: kasten(el),
			symbol: zeichen ? kasten(zeichen) : null,
			innenRechts: Number.parseFloat(getComputedStyle(el).paddingRight)
		};
	});
}

/** @param {Awaited<ReturnType<typeof miss>>} m @param {string} wo */
function symbolImFeld(m, wo) {
	const { feld, symbol, innenRechts } = m;
	if (!symbol) throw new Error(`${wo}: kein Fehlersymbol im Feld.`);
	expect(feld.rechts - symbol.rechts, `${wo}: Abstand zum rechten Rand`).toBeCloseTo(12, 0);
	expect(symbol.links, `${wo}: Das Symbol liegt über dem Text.`).toBeGreaterThanOrEqual(
		feld.rechts - innenRechts
	);
	const mitte = (/** @type {{ oben: number, unten: number }} */ k) => (k.oben + k.unten) / 2;
	expect(Math.abs(mitte(symbol) - mitte(feld)), `${wo}: nicht mittig`).toBeLessThan(1);
}

test('Signaturen: Symbol im Formular und in der Tabelle; wer tippt, bleibt im Feld', async ({
	page
}) => {
	seedSQL(
		`INSERT INTO systematik_kategorien (kuerzel, bezeichnung) VALUES ('F${s}', 'Fehlersymbol ${s}');`
	);
	try {
		await uiLogin(page);
		await gehZu(page, '/signaturen');
		const kuerzel = page.locator('#sys-kuerzel');
		await page.getByRole('button', { name: 'Anlegen' }).click();
		await expect(kuerzel).toHaveAttribute('aria-invalid', 'true');
		symbolImFeld(await miss(kuerzel), 'Kürzel');
		symbolImFeld(await miss(page.locator('#sys-bezeichnung')), 'Bezeichnung');

		// Der Fehler geht mit dem ersten Zeichen; das Feld wird dabei nicht neu gebaut.
		await expect(kuerzel).toBeFocused();
		await page.keyboard.type('Fs');
		await expect(kuerzel).not.toHaveAttribute('aria-invalid');
		await expect(kuerzel).toHaveValue('Fs');
		await expect(kuerzel).toBeFocused();
		expect((await miss(kuerzel)).symbol).toBeNull();

		// In der Tabelle ist das Feld 96 px breit; die Zelle ist breiter.
		const zeile = page.getByRole('row').filter({ hasText: `Fehlersymbol ${s}` });
		await zeile.getByRole('button', { name: 'Ändern' }).click();
		const tabellenKuerzel = page.locator('#sys-aendern-kuerzel');
		await tabellenKuerzel.fill('');
		await page.getByRole('button', { name: 'Sichern' }).click();
		await expect(tabellenKuerzel).toHaveAttribute('aria-invalid', 'true');
		symbolImFeld(await miss(tabellenKuerzel), 'Kürzel in der Tabelle');
	} finally {
		seedSQL(`DELETE FROM systematik_kategorien WHERE kuerzel = 'F${s}';`);
	}
});

test('Buchmaske: Das Symbol steht vor den Knöpfen der ISBN, die Knöpfe bleiben stehen', async ({
	page
}) => {
	await uiLogin(page);
	await gehZu(page, '/medienkatalog');
	await page.getByRole('tab', { name: 'Titel-Verwaltung' }).click();
	await page.getByRole('button', { name: 'Neues Buch' }).first().click();
	const isbn = page.locator('#buch-isbn');
	const knoepfe = [
		page.getByRole('button', { name: 'Daten aus dem Internet aktualisieren' }),
		page.getByRole('button', { name: 'Scan ISBN' })
	];
	const vorher = await Promise.all(knoepfe.map((k) => k.boundingBox()));

	await isbn.fill('12345');
	await isbn.press('Tab');
	await expect(isbn).toHaveAttribute('aria-invalid', 'true');
	const m = await miss(isbn);
	if (!m.symbol) throw new Error('ISBN: kein Fehlersymbol im Feld.');
	expect(await Promise.all(knoepfe.map((k) => k.boundingBox()))).toEqual(vorher);
	const ersterKnopf = /** @type {{ x: number }} */ (vorher[0]);
	expect(m.symbol.rechts, 'Das Symbol liegt nicht vor den Knöpfen.').toBeLessThanOrEqual(
		ersterKnopf.x
	);
	expect(m.symbol.links, 'Das Symbol liegt über dem Text.').toBeGreaterThanOrEqual(
		m.feld.rechts - m.innenRechts
	);
});

test('Leserakte: Der Knopf am Ausweisjahr liegt im Feld, das Fehlersymbol auch', async ({
	page
}) => {
	seedSQL(`
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ('FS-H-${s}', 'Hanna${s}', 'Feld', '7H1', 2031),
		       ('FS-V-${s}', 'Vera${s}', 'Feld', 'Vorkurs', 2031);
	`);
	try {
		await uiLogin(page);
		/** @param {string} vorname */
		const oeffne = async (vorname) => {
			await gehZu(page, '/schuelerdatei');
			await page.getByRole('searchbox', { name: 'Leser suchen' }).fill(vorname);
			await page.getByRole('button', { name: new RegExp(`Profil von ${vorname} `) }).click();
		};
		const jahr = page.getByLabel('Ablaufjahr des Ausweises');

		await oeffne(`Hanna${s}`);
		await jahr.fill('2035');
		const knopf = await page
			.getByRole('button', { name: /^Auf den vorgeschlagenen Wert/ })
			.boundingBox();
		const feld = await jahr.boundingBox();
		if (!knopf || !feld) throw new Error('Feld oder Knopf hat keine Fläche.');
		expect(knopf.x + knopf.width, 'Der Knopf steht rechts neben dem Feld.').toBeLessThanOrEqual(
			feld.x + feld.width
		);

		// Aus „Vorkurs" lässt sich kein Jahr rechnen: Das Feld steht leer und ungültig da.
		await oeffne(`Vera${s}`);
		await expect(jahr).toHaveAttribute('aria-invalid', 'true');
		symbolImFeld(await miss(jahr), 'Ausweisjahr');
	} finally {
		seedSQL(`DELETE FROM schueler WHERE barcode_id IN ('FS-H-${s}', 'FS-V-${s}');`);
	}
});

test('Datumsfeld: Das Symbol liegt neben dem Kalender des Browsers, nicht darauf', async ({
	page
}) => {
	await uiLogin(page);
	await gehZu(page, '/einstellungen');
	await einstellungsKategorie(page, 'LUSD & Versetzung').click();
	const bis = page.locator('#sommerferien-bis');
	await page.locator('#sommerferien-von').fill('2031-07-20');
	await bis.fill('2031-07-01');
	await expect(bis).toHaveAttribute('aria-invalid', 'true');
	symbolImFeld(await miss(bis), 'Ende der Sommerferien');
});
