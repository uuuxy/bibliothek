import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, gehZu, uniqueSuffix } from './helpers.js';

// Zwei Plätze an demselben Exemplar. Der Status-Dialog der Buchakte schickt den Status mit, den
// er beim Öffnen gezeigt hat; hat ein anderer Platz das Exemplar inzwischen ausgesondert, etwa
// mit einer Schadensmeldung, lehnt der Server ab, statt den alten Status zurückzuschreiben. Den
// anderen Platz spielt die Datenbank. Der erste Fall prüft, dass der Dialog ohne Zwischenfall
// speichert: Ein falscher Stand beim Öffnen würde jedes Speichern ablehnen.

function legeExemplarAn() {
	const s = uniqueSuffix();
	const barcode = `B-STA${s}`;
	seedSQL(`
		WITH t AS (
			INSERT INTO buecher_titel (titel, autor) VALUES ('E2E-Status ${s}', 'Fontane') RETURNING id
		)
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
		SELECT t.id, '${barcode}', true FROM t;
	`);
	const titelID = querySQL(
		`SELECT titel_id FROM buecher_exemplare WHERE barcode_id = '${barcode}'`
	);
	return { barcode, titelID };
}

/** @param {import('@playwright/test').Page} page @param {string} titelID */
async function oeffneStatusDialog(page, titelID) {
	await uiLogin(page);
	await gehZu(page, `/medienkatalog/buch/${titelID}`);
	await page.getByRole('tab', { name: /^Exemplare/ }).click();
	await page.getByRole('button', { name: 'Status ändern' }).click();
}

test('Status-Dialog speichert, wenn niemand dazwischen geändert hat', async ({ page }) => {
	const { barcode, titelID } = legeExemplarAn();
	await oeffneStatusDialog(page, titelID);

	await page.getByRole('combobox', { name: 'Status des Exemplars' }).click();
	await page.getByRole('option', { name: 'Gesperrt (Defekt/Reserviert)' }).click();
	await page.getByRole('button', { name: 'Speichern', exact: true }).click();

	await expect(page.getByText('Status erfolgreich gespeichert')).toBeVisible();
	await expect
		.poll(() =>
			querySQL(`SELECT ist_ausleihbar FROM buecher_exemplare WHERE barcode_id = '${barcode}'`)
		)
		.toBe('f');
});

test('Status-Dialog schreibt keine Aussonderung zurück, die ein anderer Platz inzwischen gebucht hat', async ({
	page
}) => {
	const { barcode, titelID } = legeExemplarAn();
	await oeffneStatusDialog(page, titelID);

	seedSQL(`
		UPDATE buecher_exemplare
		SET ist_ausleihbar = false, ist_ausgesondert = true, aussonderung_grund = 'BESCHAEDIGUNG',
		    zustand_notiz = 'Wasserschaden'
		WHERE barcode_id = '${barcode}';
	`);
	await page.getByRole('button', { name: 'Speichern', exact: true }).click();

	await expect(
		page
			.getByRole('alert')
			.filter({ hasText: 'Ein anderer Platz hat den Status dieses Exemplars inzwischen geändert' })
	).toBeVisible();
	expect(
		querySQL(`SELECT ist_ausgesondert || '/' || aussonderung_grund || '/' || zustand_notiz
		          FROM buecher_exemplare WHERE barcode_id = '${barcode}'`)
	).toBe('true/BESCHAEDIGUNG/Wasserschaden');
});
