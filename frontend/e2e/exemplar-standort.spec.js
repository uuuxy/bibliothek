import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, gehZu, uniqueSuffix, scanneWieScanner } from './helpers.js';

// Der Standort am Exemplar (docs/OFFEN.md 5.53): Die Karte nennt ihn, markierte Exemplare
// bekommen ihn über „Standort ändern", und Kopf der Akte und Titel-Verwaltung nennen die
// Standorte der Exemplare mit ihrer Zahl. Regeln und Grenzen prüfen die Go-Tests
// (api/exemplar_standort_pg_test.go); hier steht der Weg durch den Browser bis in die Tabelle.

// Was ein Test anlegt, räumt er über die Kennung des Titels wieder ab: Ein Rest stünde sonst
// als Vorschlag im Dialog des nächsten Laufs.
/** @type {string[]} */
const angelegt = [];
test.afterEach(() => {
	for (const id of angelegt.splice(0)) seedSQL(`DELETE FROM buecher_titel WHERE id = '${id}';`);
});

/** @param {string} titel @param {string[]} barcodes Das erste Exemplar steht im Lehrerschrank. */
function seedTitel(titel, barcodes) {
	const zeilen = barcodes
		.map(
			(b, i) => `SELECT t.id, '${b}', CURRENT_DATE, ${i === 0 ? "'Lehrerschrank'" : 'NULL'} FROM t`
		)
		.join(' UNION ALL ');
	seedSQL(`
		WITH t AS (
			INSERT INTO buecher_titel (titel, autor, signatur) VALUES ('${titel}', 'Beyer', 'LMF Bio 7') RETURNING id
		)
		INSERT INTO buecher_exemplare (titel_id, barcode_id, erworben_am, standort) ${zeilen};
	`);
	const id = querySQL(`SELECT titel_id FROM buecher_exemplare WHERE barcode_id = '${barcodes[0]}'`);
	angelegt.push(id);
	return id;
}

/** @param {string[]} barcodes */
const standorteInDerTabelle = (barcodes) =>
	querySQL(`SELECT string_agg(coalesce(standort, '-'), '|' ORDER BY barcode_id)
	          FROM buecher_exemplare WHERE barcode_id IN (${barcodes.map((b) => `'${b}'`).join(', ')})`);

test('Standort: Karte und Kopf nennen ihn, markierte Exemplare bekommen und verlieren ihn', async ({
	page
}) => {
	const s = uniqueSuffix();
	const titel = `E2E-Standort ${s}`;
	const barcodes = ['A', 'B', 'C'].map((x) => `B-STO${s.slice(0, 6)}${x}`);
	const [a, b] = barcodes;
	const titelID = seedTitel(titel, barcodes);

	await uiLogin(page);
	await gehZu(page, `/medienkatalog/buch/${titelID}`);
	await page.getByRole('tab', { name: /^Exemplare/ }).click();

	// Anzeige: an der Karte des einen Exemplars und im Kopf der Akte hinter der Signatur.
	const karten = page.getByText('Standort:', { exact: true });
	await expect(page.getByText('Standort: Lehrerschrank', { exact: true })).toBeVisible();
	await expect(karten).toHaveCount(1);
	await expect(page.getByText('· Lehrerschrank (1)')).toBeVisible();

	// Ändern: erst mit einem eingetragenen Standort frei.
	await page.getByRole('checkbox', { name: `Exemplar ${b} auswählen` }).check();
	const leiste = page.getByRole('region', { name: 'Aktionen für die markierten Exemplare' });
	await leiste.getByRole('button', { name: 'Standort ändern' }).click();
	const dialog = page.getByRole('dialog', { name: 'Standort ändern' });
	await expect(dialog).toContainText('Gilt für 1 Exemplar.');
	const bestaetigen = dialog.getByRole('button', { name: 'Standort ändern' });
	await expect(bestaetigen).toBeDisabled();
	// Der vorhandene Standort steht in der Vorschlagsliste des Felds.
	await expect(dialog.locator('datalist option[value="Lehrerschrank"]')).toHaveCount(1);
	await dialog.getByLabel('Standort', { exact: true }).fill('Bibliothek, Regal 3B');
	await bestaetigen.click();

	await expect(dialog).toBeHidden();
	await expect(page.getByText('Standort: Bibliothek, Regal 3B', { exact: true })).toBeVisible();
	await expect(karten).toHaveCount(2);
	await expect(page.getByText('· Bibliothek, Regal 3B (1) · Lehrerschrank (1)')).toBeVisible();
	await expect
		.poll(() => standorteInDerTabelle(barcodes))
		.toBe('Lehrerschrank|Bibliothek, Regal 3B|-');
	expect(
		querySQL(`SELECT details->>'standort_alt' || '>' || (details->>'standort_neu') FROM audit_log
		          WHERE kontext = 'Standort geändert'
		            AND datensatz_id = (SELECT id FROM buecher_exemplare WHERE barcode_id = '${b}')`)
	).toBe('>Bibliothek, Regal 3B');

	// Die Titel-Verwaltung nennt dieselben Standorte mit derselben Zahl.
	await gehZu(page, '/medienkatalog');
	await page.getByRole('tab', { name: 'Titel-Verwaltung' }).click();
	await page.getByRole('searchbox', { name: 'Bücher durchsuchen' }).fill(titel);
	await expect(page.getByRole('row').filter({ hasText: titel })).toContainText(
		'Bibliothek, Regal 3B (1) · Lehrerschrank (1)'
	);

	// Entfernen: für alle Exemplare auf einmal.
	await gehZu(page, `/medienkatalog/buch/${titelID}`);
	await page.getByRole('tab', { name: /^Exemplare/ }).click();
	await page.getByRole('checkbox', { name: `Exemplar ${a} auswählen` }).check();
	await leiste.getByRole('button', { name: 'Alle auswählen' }).click();
	await expect(leiste).toContainText('3 markiert');
	await leiste.getByRole('button', { name: 'Standort ändern' }).click();
	await expect(dialog).toContainText('Gilt für 3 Exemplare.');
	await dialog.getByLabel('Standort entfernen').check();
	await expect(dialog.getByLabel('Standort', { exact: true })).toBeDisabled();
	await bestaetigen.click();

	await expect(dialog).toBeHidden();
	await expect.poll(() => standorteInDerTabelle(barcodes)).toBe('-|-|-');
	await expect(page.getByRole('checkbox', { name: `Exemplar ${a} auswählen` })).toBeVisible();
	await expect(karten).toHaveCount(0);
});

// Ein Handscanner tippt in das Feld mit dem Fokus und endet mit Enter. Im offenen Dialog
// steht der Fokus im Feld „Standort", und Enter bestätigt: Ohne Schutz trüge jedes markierte
// Exemplar die Nummer des gescannten Buchs als Standort.
test('Standort: ein Scan in den offenen Dialog ändert nichts', async ({ page }) => {
	const s = uniqueSuffix();
	const barcodes = ['A', 'B'].map((x) => `B-STS${s.slice(0, 6)}${x}`);
	const titelID = seedTitel(`E2E-Standort-Scan ${s}`, barcodes);

	await uiLogin(page);
	await gehZu(page, `/medienkatalog/buch/${titelID}`);
	await page.getByRole('tab', { name: /^Exemplare/ }).click();
	await page.getByRole('checkbox', { name: `Exemplar ${barcodes[0]} auswählen` }).check();
	await page
		.getByRole('region', { name: 'Aktionen für die markierten Exemplare' })
		.getByRole('button', { name: 'Standort ändern' })
		.click();
	const dialog = page.getByRole('dialog', { name: 'Standort ändern' });
	const feld = dialog.getByLabel('Standort', { exact: true });
	await expect(feld).toBeFocused();

	await scanneWieScanner(page, 'B-90231');

	await expect(feld).toHaveValue('');
	await expect(dialog).toBeVisible();
	await expect(dialog.getByRole('button', { name: 'Standort ändern' })).toBeDisabled();
	expect(standorteInDerTabelle(barcodes)).toBe('Lehrerschrank|-');
});
