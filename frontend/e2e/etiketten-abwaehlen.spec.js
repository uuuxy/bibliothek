import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix, gehZu } from './helpers.js';

// Druck-Center, Schritt 2: Die vorhandenen Exemplare eines Titels sind vorgehakt. Was abgewählt
// wird, steht weder in der Vorschau noch im Druckauftrag noch im Vermerk „Etikett gedruckt".
//
// Der Druck selbst wird abgefangen: Geprüft wird, was der Browser schicken würde, und kein
// Exemplar bekommt den Vermerk.
/** @type {(() => void) | undefined} */
let aufraeumen;
test.afterEach(() => {
	aufraeumen?.();
	aufraeumen = undefined;
});

test('Etiketten: ein abgewähltes Exemplar steht weder in der Vorschau noch im Druck', async ({
	page
}) => {
	const s = uniqueSuffix();
	const titel = `E2E-Abwahl-Titel ${s}`;
	const nummer = (/** @type {number} */ n) => `B-ABW${n}-${s}`;
	aufraeumen = () =>
		seedSQL(`
			DELETE FROM buecher_exemplare WHERE titel_id IN (SELECT id FROM buecher_titel WHERE titel = '${titel}');
			DELETE FROM buecher_titel WHERE titel = '${titel}';
		`);
	seedSQL(`
		INSERT INTO buecher_titel (titel, autor) VALUES ('${titel}', 'E2E');
		INSERT INTO buecher_exemplare (titel_id, barcode_id)
		SELECT t.id, 'B-ABW' || n || '-${s}' FROM buecher_titel t, generate_series(1, 3) n
		WHERE t.titel = '${titel}';
	`);

	/** @type {any} */
	let druckauftrag;
	/** @type {any} */
	let vermerk;
	await page.route('**/api/print/labels', async (route) => {
		druckauftrag = route.request().postDataJSON();
		await route.fulfill({ status: 200, contentType: 'application/pdf', body: '%PDF-1.4 Probe' });
	});
	await page.route('**/api/exemplare/etiketten-gedruckt', async (route) => {
		vermerk = route.request().postDataJSON();
		await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' });
	});

	await uiLogin(page);
	await gehZu(page, '/druck-center');
	await page.getByRole('searchbox', { name: 'Buchtitel im Katalog suchen' }).fill(titel);
	await page.getByRole('button', { name: new RegExp(titel) }).click();
	await expect(page.getByText('Exemplare auswählen (3 gefunden)')).toBeVisible();

	// Ein Etikett der Vorschau ist 42,3 × 25,4 mm; die Nummer steht darauf.
	const etiketten = page.locator('div[style*="42.3mm"][style*="25.4mm"]');
	await expect(etiketten).toHaveCount(3);

	await page
		.locator('label', { hasText: nummer(2) })
		.getByRole('checkbox')
		.uncheck();
	await expect(etiketten).toHaveCount(2);
	await expect(etiketten.filter({ hasText: nummer(2) })).toHaveCount(0);

	await page.getByRole('button', { name: 'A4-Bogen drucken' }).click();
	await expect.poll(() => vermerk?.barcode_ids).toEqual([nummer(1), nummer(3)]);
	expect(druckauftrag.items.map((/** @type {any} */ i) => i.BarcodeID)).toEqual([
		nummer(1),
		nummer(3)
	]);
});
