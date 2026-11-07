import { test, expect } from '@playwright/test';
import { uiLogin, querySQL, seedSQL, uniqueSuffix } from './helpers.js';

// Mein Portal, Reiter „Schulbücher": Der Fachsprecher bestellt nach und braucht die genaue ISBN.
// Ein Klick auf die Kachel öffnet rechts das Seitenblatt mit Bestand und ISBN; der Knopf legt sie
// in die Zwischenablage. Das Blatt steht neben dem Raster, nicht darüber. Gemessen im Browser,
// weil die Zwischenablage und die Lage erst dort entstehen.
const s = uniqueSuffix();
const TITEL = `E2E-Blatt ${s}`;
const ISBN = `SBL1${s.slice(-12)}`;
// Ein eigenes Fach: Die Datenbank der CI kennt keine Fächer, die lokale schon.
const FACH = `E2E-Fach ${s}`;

test.beforeEach(() => {
	seedSQL(
		`INSERT INTO systematik_kategorien (kuerzel, bezeichnung) VALUES ('E2EB${s}', '${FACH}');`
	);
	querySQL(`
		WITH t AS (
			INSERT INTO buecher_titel (titel, autor, isbn, subject, jahrgang_von, jahrgang_bis, ist_lernmittel)
			VALUES ('${TITEL}', 'Blattprobe ${s}', '${ISBN}', '${FACH}', 7, 7, true)
			RETURNING id
		),
		ex AS (
			INSERT INTO buecher_exemplare (titel_id, barcode_id)
			SELECT id, 'B-SBL-' || g || '-${s}' FROM t, generate_series(1, 3) g
			RETURNING titel_id
		),
		-- Ein viertes ist bestellt und noch nicht da: Es steht im Blatt als „bestellt“, nicht im Bestand.
		zulauf AS (
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, bestellstatus)
			SELECT id, 'B-SBL-Z-${s}', false, 'im_zulauf' FROM t
			RETURNING titel_id
		)
		SELECT DISTINCT titel_id FROM ex UNION SELECT titel_id FROM zulauf;
	`);
});

test.afterEach(() => {
	seedSQL(`
		DELETE FROM buecher_exemplare WHERE titel_id IN (SELECT id FROM buecher_titel WHERE isbn = '${ISBN}');
		DELETE FROM buecher_titel WHERE isbn = '${ISBN}';
		DELETE FROM systematik_kategorien WHERE bezeichnung = '${FACH}';
	`);
});

test('Mein Portal: Ein Klick auf die Kachel zeigt die ISBN im Seitenblatt, der Knopf kopiert sie', async ({
	page,
	context
}) => {
	await context.grantPermissions(['clipboard-read', 'clipboard-write']);
	await page.setViewportSize({ width: 1280, height: 900 });
	await uiLogin(page);
	await page.goto('/kollegium-portal');
	await page.getByRole('tab', { name: 'Schulbücher' }).click();
	// Erst die gefilterte Liste abwarten: Vorher steht die ungefilterte da, mit anderen Fächern.
	const gefiltert = page.waitForResponse((r) => r.url().includes('/api/portal/lernmittel?q='));
	await page.getByRole('searchbox', { name: 'Schulbücher durchsuchen' }).fill(TITEL);
	await gefiltert;
	await page.getByRole('button', { name: new RegExp(FACH) }).click();

	const kachel = page.getByRole('button', { name: new RegExp(`^${TITEL} .*Angaben zeigen$`) });
	await kachel.click();
	const blatt = page.getByRole('complementary', { name: TITEL });
	await expect(blatt).toBeVisible();
	await expect(blatt).toContainText(`${FACH} · Jahrgang 7`);
	await expect(blatt).toContainText('3 von 3');
	await expect(
		blatt.locator('dt', { hasText: /^bestellt$/ }).locator('xpath=preceding-sibling::dd[1]')
	).toHaveText('1');
	await expect(kachel).toHaveAttribute('aria-pressed', 'true');

	// Das Blatt steht neben dem Inhalt, nicht darüber: „Als PDF" am rechten Ende der Fachzeile
	// endet links vom Blatt.
	const pdf = await page
		.getByTitle(`${FACH}: Bestandsliste als PDF, mit Buchbildern`)
		.boundingBox();
	const b = await blatt.boundingBox();
	expect(
		pdf && b && pdf.x + pdf.width <= b.x,
		`Als PDF ${JSON.stringify(pdf)}, Blatt ${JSON.stringify(b)}`
	).toBe(true);

	await blatt.getByRole('button', { name: `ISBN: ${ISBN}` }).click();
	await expect(blatt.getByText('Kopiert!')).toBeVisible();
	expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(ISBN);

	await blatt.getByRole('button', { name: 'Schließen' }).click();
	await expect(blatt).toHaveCount(0);
	await expect(kachel).toHaveAttribute('aria-pressed', 'false');
});
