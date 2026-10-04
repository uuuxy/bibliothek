import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, navigationSteht } from './helpers.js';

// Die Ausleihliste der Akte: eine Zeile je Buch, und der Titel ist auch im Fenster von
// 1280 px zu lesen. Ein Schüler hat acht bis achtzehn Lernmittel; trägt jede Zeile Marke,
// Nummer, Datum, Status und drei Knöpfe nebeneinander, bleibt dem Titel keine Breite.
// Nur Ziffern: Die Nummern der Exemplare sehen aus wie die dreizehnstelligen der Schule.
const s = String(Date.now()).slice(-7);
const AUSWEIS = `S-AZ-${s}`;
const AUTORIN = `Autorin ${s}`;

// Längen wie in der Schulbuchliste: die Hälfte über 30 Zeichen.
const TITEL = [
	'Fundamente der Mathematik 8 Gymnasium Hessen',
	'Green Line 4 Schülerbuch Klasse 8',
	'Natur und Technik - Biologie 7 - 10',
	'Forum Geschichte 4 (Schulbuch Klasse 9)',
	'Deutschbuch 8 Sprach- und Lesebuch',
	'Diercke Weltatlas',
	'Découvertes 3 Série jaune',
	'Chemie heute SI Gesamtband',
	'Impulse Physik Mittelstufe',
	'Politik & Co. 2 Hessen',
	'Kursbuch Religion Elementar 7/8',
	'Musik um uns 2/3'
].map((titel) => `${titel} ${s}`);
const nummer = (/** @type {number} */ i) => `5896${String(i).padStart(2, '0')}${s}`;

/** Tippt blind wie ein Handscanner. @param {import('@playwright/test').Page} page @param {string} code */
async function scanne(page, code) {
	await page.keyboard.type(code, { delay: 5 });
	await page.keyboard.press('Enter');
}

/**
 * Jeder Titel hat Breite zum Lesen, und jedes Buch steht in einer Zeile.
 * @param {import('@playwright/test').Page} page
 * @param {import('@playwright/test').Locator} tabelle
 */
async function titelSindZuLesen(page, tabelle) {
	for (const titel of TITEL) {
		const text = tabelle.getByText(titel, { exact: true });
		const zeile = tabelle.getByRole('row').filter({ has: page.getByText(titel, { exact: true }) });
		const titelKasten = await text.boundingBox();
		const zeilenKasten = await zeile.boundingBox();
		expect(titelKasten?.width ?? 0, `„${titel}“ hat keine Breite zum Lesen`).toBeGreaterThanOrEqual(
			140
		);
		expect(zeilenKasten?.height ?? 999, `„${titel}“ steht in mehr als einer Zeile`).toBeLessThan(
			56
		);
	}
}

test.describe.serial('Leserakte: Ausleihliste', () => {
	test.beforeAll(() => {
		const werte = TITEL.map(
			(titel, i) => `('${titel.replace(/'/g, "''")}', '4${String(i).padStart(2, '0')}${s}', ${i})`
		).join(',');
		seedSQL(`
			WITH sch AS (
				INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
				VALUES ('${AUSWEIS}', 'Zeile${s}', 'Testschueler', '08a', EXTRACT(YEAR FROM CURRENT_DATE)::int + 3)
				RETURNING id
			),
			t AS (
				INSERT INTO buecher_titel (titel, isbn, ist_lernmittel, autor)
				SELECT v.titel, v.isbn, true, '${AUTORIN}' FROM (VALUES ${werte}) AS v(titel, isbn, n)
				RETURNING id, titel
			),
			-- Die Nummer kommt aus der Liste, nicht aus der gespeicherten ISBN: Eine gültige
			-- zehnstellige ISBN speichert das Programm dreizehnstellig.
			ex AS (
				INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
				SELECT t.id, '5896' || lpad(v.n::text, 2, '0') || '${s}', true
				FROM t JOIN (VALUES ${werte}) AS v(titel, isbn, n) ON v.titel = t.titel
				RETURNING id
			)
			INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
			SELECT ex.id, sch.id, CURRENT_DATE + 20 FROM ex, sch;
		`);
	});
	test.afterAll(() => {
		seedSQL(`
			DELETE FROM ausleihen WHERE schueler_id IN (SELECT id FROM schueler WHERE barcode_id = '${AUSWEIS}');
			DELETE FROM buecher_exemplare WHERE titel_id IN (SELECT id FROM buecher_titel WHERE autor = '${AUTORIN}');
			DELETE FROM buecher_titel WHERE autor = '${AUTORIN}';
			DELETE FROM schueler WHERE barcode_id = '${AUSWEIS}';
		`);
	});

	test('1280 px: jeder Titel ist zu lesen, jedes Buch steht in einer Zeile, und die Sprechblase nennt den ganzen Titel', async ({
		page
	}) => {
		await page.setViewportSize({ width: 1280, height: 900 });
		await uiLogin(page);
		await expect
			.poll(() => page.evaluate(() => document.activeElement?.id ?? ''), { timeout: 5000 })
			.toBe('omnibox-input');
		await scanne(page, AUSWEIS);
		await expect(page.getByText('Entliehene Bücher (12)')).toBeVisible();

		const tabelle = page.getByRole('table', { name: 'Ausgeliehene Bücher' });
		await titelSindZuLesen(page, tabelle);

		// Die Liste läuft nicht über ihren Platz hinaus, und die Symbole halten ihr Mindestmaß.
		// Das Miniaturbild ist ein Bild, kein Symbol; es öffnet die Großansicht des Covers.
		const mass = await tabelle.evaluate((t) => {
			const knoepfe = [...t.querySelectorAll('tbody button:has(svg)')].map((b) =>
				b.getBoundingClientRect()
			);
			return {
				laeuftUeber: t.scrollWidth > (t.parentElement?.clientWidth ?? 0) + 1,
				knoepfe: knoepfe.length,
				zuKlein: knoepfe.filter((r) => r.width < 32 || r.height < 32).length
			};
		});
		expect(mass.laeuftUeber, 'die Tabelle ist breiter als ihr Platz').toBe(false);
		expect(mass.knoepfe, 'je Zeile Datum ändern, verlängern, Schaden, zurückgeben').toBe(48);
		expect(mass.zuKlein, 'Symbol-Knöpfe unter 32 px').toBe(0);

		// Jede Zeile führt zur Großansicht ihres Covers (e2e/cover-grossansicht.spec.js).
		await expect(tabelle.getByRole('button', { name: /^Cover von .+ anzeigen$/ })).toHaveCount(12);
		await expect(
			tabelle.getByRole('button', { name: `Cover von ${TITEL[0]} anzeigen` })
		).toBeVisible();

		// Was die Zeile kürzt oder weglässt, steht beim Zeigen auf dem Titel.
		await expect(tabelle.getByRole('columnheader', { name: 'Barcode' })).toBeHidden();
		await tabelle.getByText(TITEL[0], { exact: true }).hover();
		const blase = page.locator('[data-tooltip-blase]');
		await expect(blase).toBeVisible();
		await expect(blase).toHaveText(`${TITEL[0]} · ${AUTORIN} · ${nummer(0)}`);
	});

	// Unter 1280 px Fensterbreite beginnt die Navigation eingeklappt (Sidebar.svelte). Neben
	// ihr in voller Breite und der Leserkarte blieb dem Titel bei 1100 px kein Platz und bei
	// 1200 px 82 px.
	test('unter 1280 px beginnt die Navigation eingeklappt; bei 1100 px ist jeder Titel zu lesen', async ({
		page
	}) => {
		await page.setViewportSize({ width: 1279, height: 900 });
		await uiLogin(page);
		await expect(page.getByRole('button', { name: 'Navigation ausklappen' })).toBeVisible();
		await page.setViewportSize({ width: 1280, height: 900 });
		await expect(page.getByRole('button', { name: 'Navigation einklappen' })).toBeVisible();
		await page.setViewportSize({ width: 1100, height: 900 });
		await expect(page.getByRole('button', { name: 'Navigation ausklappen' })).toBeVisible();
		await navigationSteht(page, 'eingeklappt');
		await expect
			.poll(() => page.evaluate(() => document.activeElement?.id ?? ''), { timeout: 5000 })
			.toBe('omnibox-input');
		await scanne(page, AUSWEIS);
		await expect(page.getByText('Entliehene Bücher (12)')).toBeVisible();

		const tabelle = page.getByRole('table', { name: 'Ausgeliehene Bücher' });
		await titelSindZuLesen(page, tabelle);
		const laeuftUeber = await tabelle.evaluate(
			(t) => t.scrollWidth > (t.parentElement?.clientWidth ?? 0) + 1
		);
		expect(laeuftUeber, 'die Tabelle ist breiter als ihr Platz').toBe(false);

		// Der Doppelpfeil klappt aus, und dabei bleibt es: Die Wahl gilt vor der Fensterbreite.
		await page.getByRole('button', { name: 'Navigation ausklappen' }).click();
		await expect(page.getByRole('button', { name: 'Navigation einklappen' })).toBeVisible();
		await page.setViewportSize({ width: 1000, height: 900 });
		await expect(page.getByRole('button', { name: 'Navigation einklappen' })).toBeVisible();
	});

	// Die Sprechblase nennt Titel und Nummer eines Buchs dieses Lesers. Bleibt der Zeiger auf
	// der Zeile liegen, bis die Theke sich leert oder sperrt, darf sie nicht stehen bleiben.
	test('die Sprechblase geht mit ihrer Zeile: nach „Theke leeren“ und hinter der Sperre steht sie nicht mehr', async ({
		page
	}) => {
		await page.clock.install();
		const fristen = page.waitForResponse(
			(r) => new URL(r.url()).pathname === '/api/einstellungen/sitzung'
		);
		await uiLogin(page);
		await fristen;
		await expect
			.poll(() => page.evaluate(() => document.activeElement?.id ?? ''), { timeout: 5000 })
			.toBe('omnibox-input');
		await scanne(page, AUSWEIS);
		await expect(page.getByText('Entliehene Bücher (12)')).toBeVisible();

		const blase = page.locator('[data-tooltip-blase]');
		await page
			.getByRole('table', { name: 'Ausgeliehene Bücher' })
			.getByText(TITEL[0], { exact: true })
			.hover();
		await page.clock.runFor(400);
		await expect(blase).toBeVisible();

		// Fünf Minuten ohne Bedienung: Die Theke leert sich, der Zeiger liegt noch dort.
		await page.clock.fastForward('05:10');
		await expect(page.getByText('Entliehene Bücher (12)')).toHaveCount(0);
		await expect(blase, 'die Sprechblase steht ohne ihre Zeile da').toBeHidden();

		// Dasselbe an einem Knopf, der bleibt: Hinter dem Sperrbildschirm ist er ausgeblendet.
		const knopf = page.getByRole('button', { name: /Navigation einklappen/ });
		await knopf.hover();
		await page.clock.runFor(400);
		await expect(blase).toBeVisible();
		await page.clock.fastForward('15:10');
		await expect(page.getByTestId('sperrbildschirm')).toBeVisible();
		await expect(blase, 'die Sprechblase steht über dem Sperrbildschirm').toBeHidden();
	});

	test('1920 px: die Titel stehen ungekürzt, und die Nummer des Exemplars hat ihre Spalte', async ({
		page
	}) => {
		await page.setViewportSize({ width: 1920, height: 1080 });
		await uiLogin(page);
		await expect
			.poll(() => page.evaluate(() => document.activeElement?.id ?? ''), { timeout: 5000 })
			.toBe('omnibox-input');
		await scanne(page, AUSWEIS);
		await expect(page.getByText('Entliehene Bücher (12)')).toBeVisible();

		const tabelle = page.getByRole('table', { name: 'Ausgeliehene Bücher' });
		await expect(tabelle.getByRole('columnheader', { name: 'Barcode' })).toBeVisible();
		await expect(tabelle.getByRole('cell', { name: nummer(0), exact: true })).toBeVisible();
		for (const titel of TITEL) {
			const gekuerzt = await tabelle
				.getByText(titel, { exact: true })
				.evaluate((e) => e.scrollWidth > e.clientWidth);
			expect(gekuerzt, `„${titel}“ ist bei 1920 px gekürzt`).toBe(false);
		}
	});
});
