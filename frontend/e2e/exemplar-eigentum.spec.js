import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, gehZu, uniqueSuffix } from './helpers.js';

// Das Eigentum in der Buchakte (docs/OFFEN.md 4.24, Stufe 3, freigegeben am 29.09.2026): Die
// Exemplarkarte nennt es mit Herkunft, und markierte Exemplare bekommen es über die Leiste
// „Eigentum ändern" — wie in Littera „Exemplardaten anpassen". Was das Eigentum bewirkt
// (Etikett, Bestandsbücher, Schadensersatz), prüfen die Go-Tests; hier steht der Weg durch
// den Browser bis in die Tabelle.
test('Eigentum: Karte zeigt die Herkunft, markierte Exemplare werden mit Grund geändert', async ({
	page
}) => {
	const s = uniqueSuffix();
	const titel = `E2E-Eigentum ${s}`;
	const a = `B-EIG${s.slice(0, 6)}A`;
	const b = `B-EIG${s.slice(0, 6)}B`;
	seedSQL(`
		WITH t AS (
			INSERT INTO buecher_titel (titel, autor) VALUES ('${titel}', 'Lessing') RETURNING id
		)
		INSERT INTO buecher_exemplare (titel_id, barcode_id, erworben_am, eigentum, eigentum_quelle,
		                               erweiterte_eigenschaften)
		SELECT t.id, '${a}', CURRENT_DATE, 'land', 'littera', '{"littera_eigentumsvermerk": "Land Hessen"}'::jsonb FROM t
		UNION ALL
		SELECT t.id, '${b}', CURRENT_DATE, NULL, NULL, '{}'::jsonb FROM t;
	`);
	const titelID = querySQL(`SELECT titel_id FROM buecher_exemplare WHERE barcode_id = '${a}'`);

	await uiLogin(page);
	await gehZu(page, `/medienkatalog/buch/${titelID}`);
	await page.getByRole('tab', { name: /^Exemplare/ }).click();

	// Anzeige: Wert und Herkunft je Karte.
	await expect(page.getByText('laut Littera („Land Hessen“)')).toBeVisible();
	await expect(page.getByText('Vorgabe (kein Lernmittel)')).toBeVisible();

	// Markieren: Die gemeinsame Leiste erscheint mit „Alle auswählen".
	await page.getByRole('checkbox', { name: `Exemplar ${b} auswählen` }).check();
	const leiste = page.getByRole('region', { name: 'Aktionen für die markierten Exemplare' });
	await expect(leiste).toContainText('1 markiert');
	await leiste.getByRole('button', { name: 'Alle auswählen' }).click();
	await expect(leiste).toContainText('2 markiert');

	// Ändern: erst mit Auswahl und Grund frei.
	await leiste.getByRole('button', { name: 'Eigentum ändern' }).click();
	const dialog = page.getByRole('dialog', { name: 'Eigentum ändern' });
	await expect(dialog).toContainText('Gilt für 2 Exemplare.');
	const bestaetigen = dialog.getByRole('button', { name: 'Eigentum ändern' });
	await dialog.getByLabel('Land', { exact: true }).check();
	await expect(bestaetigen).toBeDisabled();
	await dialog.getByLabel('Grund der Änderung').fill('Klassensatz aus LMF-Mitteln');
	await bestaetigen.click();

	// Beweis an der Karte und an der Tabelle.
	await expect(page.getByText('von Hand gesetzt · Littera: „Land Hessen“')).toBeVisible();
	await expect(dialog).toBeHidden();
	await expect
		.poll(() =>
			querySQL(`SELECT string_agg(eigentum || '/' || eigentum_quelle, ',' ORDER BY barcode_id)
			          FROM buecher_exemplare WHERE barcode_id IN ('${a}', '${b}')`)
		)
		.toBe('land/hand,land/hand');
	expect(
		querySQL(`SELECT count(*) FROM audit_log WHERE kontext = 'Eigentum von Hand geändert'
		          AND details->>'grund' = 'Klassensatz aus LMF-Mitteln'
		          AND datensatz_id IN (SELECT id FROM buecher_exemplare WHERE barcode_id IN ('${a}', '${b}'))`)
	).toBe('2');
});
