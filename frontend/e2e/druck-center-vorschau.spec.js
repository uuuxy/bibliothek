import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix, gehZu, navigationSteht } from './helpers.js';

// Das Vorschaublatt der Buch-Etiketten ist die erste Seite des Druck-PDFs, 140 mm breit, und
// steht in der schmaleren der zwei Spalten. Wo die Spalte schmaler ist als das Blatt, wird es
// als Ganzes verkleinert: Es bleibt in seiner Spalte, verdeckt kein Feld der Einstellungen
// daneben und macht den Bereich nicht seitlich rollbar. Wo Platz ist, bleibt es in voller Größe.

const BLATT_PX = (140 * 96) / 25.4;

test('Druck-Center: Das Vorschaublatt bleibt in seiner Spalte', async ({ page }) => {
	const s = uniqueSuffix();
	seedSQL(`
        WITH t AS (
            INSERT INTO buecher_titel (isbn, titel, autor, verlag)
            VALUES ('978v${s}', 'E2E Vorschau ${s}', 'Blatt Autor', 'Blattverlag') RETURNING id
        )
        INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
        SELECT id, 'B-VOR-${s}', true FROM t;
    `);
	try {
		await page.setViewportSize({ width: 1280, height: 900 });
		await uiLogin(page);
		await gehZu(page, '/druck-center');
		await page.getByText('Buch-Etiketten', { exact: true }).first().click();
		await page.getByRole('searchbox', { name: 'Titel im Katalog suchen' }).fill(`Vorschau ${s}`);
		await page.getByRole('button', { name: new RegExp(`E2E Vorschau ${s}`) }).click();
		const blatt = page.getByTestId('etiketten-blatt');
		await expect(blatt).toBeVisible();
		// Das Blatt ist ein Bild, das pdf.js aus dem PDF des Servers gezeichnet hat.
		expect(
			await blatt.evaluate((el) => /** @type {HTMLImageElement} */ (el).naturalWidth)
		).toBeGreaterThan(500);

		for (const breite of [1024, 1280, 1440, 1920]) {
			await test.step(`Fenster ${breite} px`, async () => {
				await page.setViewportSize({ width: breite, height: 900 });
				// Unter 1280 px beginnt die Navigation eingeklappt; gemessen wird erst, wenn sie steht.
				await navigationSteht(page, breite < 1280 ? 'eingeklappt' : 'ausgeklappt');

				const lage = () =>
					blatt.evaluate((el) => {
						const zeile = /** @type {HTMLElement} */ (el.parentElement?.parentElement);
						const vorschau = /** @type {HTMLElement} */ (zeile.parentElement);
						const einstellungen = /** @type {HTMLElement} */ (vorschau.previousElementSibling);
						/** @type {HTMLElement | null} */
						let rollt = vorschau;
						while (rollt && !/auto|scroll/.test(getComputedStyle(rollt).overflowY))
							rollt = rollt.parentElement;
						const b = el.getBoundingClientRect();
						const z = zeile.getBoundingClientRect();
						return {
							blattLinks: b.left,
							blattRechts: b.right,
							blattBreite: b.width,
							form: b.width / b.height,
							platzLinks: z.left,
							platzRechts: z.right,
							einstellungenRechts: einstellungen.getBoundingClientRect().right,
							querRollbar: rollt ? rollt.scrollWidth - rollt.clientWidth : -1
						};
					});
				// Die Größe folgt der Spalte einen Durchlauf später; warten, bis sie steht.
				await expect
					.poll(async () => Math.round((await lage()).blattBreite), {
						message: 'das Blatt ist so breit wie sein Platz, höchstens 140 mm'
					})
					.toBe(
						Math.round(Math.min(BLATT_PX, (await lage()).platzRechts - (await lage()).platzLinks))
					);
				const m = await lage();

				expect(m.blattLinks, 'das Blatt beginnt in seiner Spalte').toBeGreaterThanOrEqual(
					m.platzLinks - 0.5
				);
				expect(m.blattRechts, 'das Blatt endet in seiner Spalte').toBeLessThanOrEqual(
					m.platzRechts + 0.5
				);
				expect(
					m.einstellungenRechts,
					'das Blatt liegt nicht über den Einstellungen'
				).toBeLessThanOrEqual(m.blattLinks);
				expect(m.querRollbar, 'der Bereich rollt nicht seitlich').toBe(0);
				expect(m.form, 'das Blatt behält seine Form (140 zu 198 mm)').toBeCloseTo(140 / 198, 2);
				if (breite === 1920)
					expect(m.blattBreite, 'wo Platz ist, bleibt das Blatt in voller Größe').toBeCloseTo(
						BLATT_PX,
						0
					);

				// Probe aus der Sicht der Maus: Der Pfeil des Auswahlfelds am rechten Rand der
				// Einstellungen nimmt den Klick an. Liegt etwas darüber, läuft der Versuch ab.
				const feld = page.getByRole('combobox', { name: 'Etikettenformat' });
				const kasten = /** @type {{ width: number, height: number }} */ (await feld.boundingBox());
				await feld.click({
					trial: true,
					timeout: 3000,
					position: { x: kasten.width - 20, y: kasten.height / 2 }
				});
			});
		}
	} finally {
		seedSQL(`DELETE FROM buecher_titel WHERE isbn = '978v${s}';`);
	}
});

// Das große Lernmittel-Etikett ist ein weiteres Format der Buch-Etiketten, mit dem Erzeuger des
// Lieferanten-Wegs und für Bücher aus jedem Topf. Startfeld und Barcode-Art entfallen.
test('Druck-Center: Das große Lernmittel-Etikett, auch für die Schülerbücherei', async ({
	page
}) => {
	const s = uniqueSuffix();
	seedSQL(`
        WITH l AS (
            INSERT INTO buecher_titel (isbn, titel, autor, verlag, ist_lernmittel)
            VALUES ('978g${s}', 'E2E Gross Lernmittel ${s}', 'Autor', 'Verlag', true) RETURNING id
        ), b AS (
            INSERT INTO buecher_titel (isbn, titel, autor, verlag, ist_lernmittel)
            VALUES ('978h${s}', 'E2E Gross Buecherei ${s}', 'Autor', 'Verlag', false) RETURNING id
        )
        INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
        SELECT id, 'B-GL-${s}', true FROM l UNION ALL SELECT id, 'B-GB-${s}', true FROM b;
    `);
	try {
		await uiLogin(page);
		await gehZu(page, '/druck-center');
		await page.getByText('Buch-Etiketten', { exact: true }).first().click();
		const suche = page.getByRole('searchbox', { name: 'Titel im Katalog suchen' });
		await suche.fill(`Gross Lernmittel ${s}`);
		await page.getByRole('button', { name: new RegExp(`E2E Gross Lernmittel ${s}`) }).click();
		await expect(page.getByTestId('etiketten-blatt')).toBeVisible();

		const grossesBlatt = page.waitForResponse(
			(r) =>
				r.url().endsWith('/api/print/labels') &&
				r.request().postDataJSON()?.formatId === 'lernmittel_gross'
		);
		await page.getByRole('combobox', { name: 'Etikettenformat' }).click();
		await page.getByRole('option', { name: /Große Lernmittel-Etiketten/ }).click();
		expect((await grossesBlatt).status(), 'die Tür liefert das große Etikett').toBe(200);
		await expect(page.getByText('Zum Ausschneiden', { exact: true })).toBeVisible();
		await expect(
			page.getByRole('spinbutton', { name: 'Startposition auf dem A4-Bogen' })
		).toHaveCount(0);
		await expect(page.getByRole('combobox', { name: 'Barcode-Ausgabe' })).toHaveCount(0);
		await expect(page.getByTestId('etiketten-blatt')).toBeVisible();

		const buecherei = page.waitForResponse((r) => {
			const auftrag = r.url().endsWith('/api/print/labels') && r.request().postDataJSON();
			return (
				auftrag?.formatId === 'lernmittel_gross' &&
				auftrag.items?.some((/** @type {any} */ e) => e.BarcodeID === `B-GB-${s}`)
			);
		});
		await suche.fill(`Gross Buecherei ${s}`);
		await page.getByRole('button', { name: new RegExp(`E2E Gross Buecherei ${s}`) }).click();
		expect((await buecherei).status(), 'auch das Buch der Schülerbücherei').toBe(200);
		await expect(page.getByTestId('etiketten-blatt')).toBeVisible();
	} finally {
		seedSQL(`DELETE FROM buecher_titel WHERE isbn IN ('978g${s}', '978h${s}');`);
	}
});
