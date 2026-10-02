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
	await expect(page.getByText('Exemplare auswählen: 3 von 3')).toBeVisible();

	// Ein Etikett der Vorschau ist 42,3 × 25,4 mm; die Nummer steht darauf.
	const etiketten = page.locator('div[style*="42.3mm"][style*="25.4mm"]');
	await expect(etiketten).toHaveCount(3);

	await page
		.locator('label', { hasText: nummer(2) })
		.getByRole('checkbox')
		.uncheck();
	await expect(page.getByText('Exemplare auswählen: 2 von 3')).toBeVisible();
	await expect(etiketten).toHaveCount(2);
	await expect(etiketten.filter({ hasText: nummer(2) })).toHaveCount(0);

	await page.getByRole('button', { name: 'A4-Bogen drucken' }).click();
	await expect.poll(() => vermerk?.barcode_ids).toEqual([nummer(1), nummer(3)]);
	expect(druckauftrag.items.map((/** @type {any} */ i) => i.BarcodeID)).toEqual([
		nummer(1),
		nummer(3)
	]);
});

// Über der Liste steht ein Kästchen für alle und, sobald die Liste länger ist als ihr Kasten,
// ein Feld für die Nummer. Die Eingabetaste, wie ein Handscanner sie schickt, setzt das
// Exemplar mit genau dieser Nummer auf den Bogen. Ein ausgesondertes Exemplar steht nicht in
// der Liste: Für ein Buch, das nicht mehr im Regal steht, wäre ein Etikett immer falsch.
test('Etiketten: ein Kästchen für alle, die Nummer für ein einzelnes, kein ausgesondertes', async ({
	page
}) => {
	const s = uniqueSuffix();
	const titel = `E2E-Auswahl-Titel ${s}`;
	const nummer = (/** @type {number} */ n) => `B-AUSW${n}-${s}`;
	const ausgesondert = `B-AUSWX-${s}`;
	aufraeumen = () =>
		seedSQL(`
			DELETE FROM buecher_exemplare WHERE titel_id IN (SELECT id FROM buecher_titel WHERE titel = '${titel}');
			DELETE FROM buecher_titel WHERE titel = '${titel}';
		`);
	seedSQL(`
		INSERT INTO buecher_titel (titel, autor) VALUES ('${titel}', 'E2E');
		INSERT INTO buecher_exemplare (titel_id, barcode_id)
		SELECT t.id, 'B-AUSW' || n || '-${s}' FROM buecher_titel t, generate_series(1, 8) n
		WHERE t.titel = '${titel}';
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, ist_ausgesondert, aussonderung_grund)
		SELECT t.id, '${ausgesondert}', false, true, 'AUSSORTIERT' FROM buecher_titel t
		WHERE t.titel = '${titel}';
	`);

	/** @type {any} */
	let druckauftrag;
	await page.route('**/api/print/labels', async (route) => {
		druckauftrag = route.request().postDataJSON();
		await route.fulfill({ status: 200, contentType: 'application/pdf', body: '%PDF-1.4 Probe' });
	});
	await page.route('**/api/exemplare/etiketten-gedruckt', (route) =>
		route.fulfill({ status: 200, contentType: 'application/json', body: '{}' })
	);

	await uiLogin(page);
	await gehZu(page, '/druck-center');
	await page.getByRole('searchbox', { name: 'Buchtitel im Katalog suchen' }).fill(titel);
	await page.getByRole('button', { name: new RegExp(titel) }).click();

	const etiketten = page.locator('div[style*="42.3mm"][style*="25.4mm"]');
	const alle = page.getByRole('checkbox', { name: 'Alle 8 Exemplare' });
	const feld = page.getByRole('searchbox', { name: 'Exemplar nach Nummer suchen' });

	// Acht im Bestand, alle vorgehakt; das ausgesonderte fehlt in Liste und Vorschau.
	await expect(page.getByText('Exemplare auswählen: 8 von 8')).toBeVisible();
	await expect(alle).toBeChecked();
	await expect(etiketten).toHaveCount(8);
	await expect(page.getByText(ausgesondert)).toHaveCount(0);
	await expect(page.getByText('1 ausgesondertes Exemplar steht nicht in der Liste.')).toBeVisible();

	// Ein Klick nimmt alle vom Bogen; ohne Etikett gibt es nichts zu drucken.
	await alle.uncheck();
	await expect(page.getByText('Exemplare auswählen: 0 von 8')).toBeVisible();
	await expect(etiketten).toHaveCount(0);
	await expect(page.getByRole('button', { name: 'A4-Bogen drucken' })).toBeDisabled();

	// Die Nummer, klein getippt wie von Hand, und die Eingabetaste: genau dieses Exemplar.
	await feld.click();
	await page.keyboard.type(nummer(5).toLowerCase(), { delay: 5 });
	await expect(page.getByRole('checkbox', { name: 'Der eine Treffer' })).toBeVisible();
	await page.keyboard.press('Enter');
	await expect(feld).toHaveValue('');
	await expect(page.getByText('Exemplare auswählen: 1 von 8')).toBeVisible();
	await expect(etiketten).toHaveCount(1);
	await expect(etiketten.filter({ hasText: nummer(5) })).toHaveCount(1);
	await expect(alle).toHaveJSProperty('indeterminate', true);

	// Eine Nummer, die nicht zu diesem Titel gehört, wählt nichts und bleibt stehen.
	await page.keyboard.type(ausgesondert, { delay: 5 });
	await page.keyboard.press('Enter');
	await expect(
		page.getByText(`Kein Exemplar dieses Titels passt zu „${ausgesondert}“.`)
	).toBeVisible();
	await expect(etiketten).toHaveCount(1);
	await feld.fill('');

	// Das Kästchen mit dem Strich wählt wieder alle; gedruckt wird ohne das ausgesonderte.
	await alle.check();
	await expect(etiketten).toHaveCount(8);
	await page.getByRole('button', { name: 'A4-Bogen drucken' }).click();
	await expect
		.poll(() => druckauftrag?.items.map((/** @type {any} */ i) => i.BarcodeID))
		.toEqual([1, 2, 3, 4, 5, 6, 7, 8].map(nummer));
});
