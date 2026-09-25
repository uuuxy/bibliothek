import { test, expect } from '@playwright/test';
import { uiLogin, gehZu, seedSQL, querySQL, uniqueSuffix } from './helpers.js';
import { isbnFormen } from '../src/lib/utils/isbnFormen.js';

// „Neue Auflage bestellen" über die andere Länge der ISBN (docs/OFFEN.md 4.18, Stufe 4): Im
// Katalog steht die neue Auflage mit ihrer ISBN-10, eingegeben wird die EAN-13 vom Buchrücken.
// Die Tür legt nichts an und fragt; „Diesen Titel nehmen" ordnet den Titel aus dem Katalog zu,
// und unter der EAN-13 entsteht kein zweiter. Die ISBN-10 rechnet hier der Browser aus
// (isbnFormen), am Server ihr Zwilling (isbnutil.AndereForm) — die Frage kommt nur, wenn beide
// gleich rechnen. Kein Katalogdienst im Netz: Gefragt wird die DNB erst bei „Neu anlegen".
test('Bestellbedarf: neue Auflage über die andere Länge der ISBN — erst die Frage, dann der Titel aus dem Katalog', async ({
	page
}) => {
	const marke = `E2E-AndereForm-${uniqueSuffix()}`;
	const einstellung = (/** @type {string} */ schluessel) =>
		querySQL(`SELECT wert FROM system_einstellungen WHERE schluessel = '${schluessel}'`);
	const vorher = {
		bestellbedarf_warnung_aktiv: einstellung('bestellbedarf_warnung_aktiv'),
		bestellbedarf_schwelle: einstellung('bestellbedarf_schwelle')
	};
	const setze = (/** @type {string} */ schluessel, /** @type {string} */ wert) =>
		seedSQL(`INSERT INTO system_einstellungen (schluessel, wert) VALUES ('${schluessel}', '${wert}')
		         ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert;`);
	const ean = isbnFormen(`3${String(Date.now()).slice(-8)}0`)[1];
	const zehn = isbnFormen(ean)[1];
	const alt = querySQL(
		`INSERT INTO buecher_titel (titel, auflage, erscheinungsjahr, ist_lernmittel)
		 VALUES ('${marke}', '3. Aufl.', 2019, true) RETURNING id`
	).split('\n')[0];
	const neu = querySQL(
		`INSERT INTO buecher_titel (titel, auflage, erscheinungsjahr, ist_lernmittel, isbn)
		 VALUES ('${marke} Neubearbeitung', '4. Aufl.', 2024, false, '${zehn}') RETURNING id`
	).split('\n')[0];

	try {
		seedSQL(
			`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ('${alt}', '${marke}-1');`
		);
		setze('bestellbedarf_warnung_aktiv', 'true');
		setze('bestellbedarf_schwelle', '5');

		await uiLogin(page);
		await gehZu(page, '/bestellungen');
		await page.getByRole('searchbox', { name: 'Bestellvorschläge filtern' }).fill(marke);
		const zeilen = page.locator('div.group').filter({ hasText: marke });
		await expect(zeilen).toHaveCount(1);
		await zeilen
			.first()
			.getByRole('button', { name: `Weitere Aktionen zu ${marke}` })
			.click();
		await page.getByRole('menuitem', { name: /Neue Auflage bestellen/ }).click();
		const dialog = page.getByRole('dialog', { name: 'Neue Auflage bestellen' });
		await dialog.getByLabel('ISBN der neuen Auflage').fill(ean);
		await dialog.getByRole('button', { name: 'Suchen' }).click();

		// Die Frage: der Titel aus dem Katalog mit seiner ISBN-10, und bis zur Wahl nichts zu
		// bestätigen.
		await expect(
			dialog.getByText('Im Katalog steht diese ISBN in zehnstelliger Form.')
		).toBeVisible();
		const nehmen = dialog.getByRole('button', { name: /Diesen Titel nehmen/ });
		await expect(nehmen).toContainText(`${marke} Neubearbeitung`);
		await expect(nehmen).toContainText(zehn);
		await expect(dialog.getByRole('button', { name: /Neu anlegen/ })).toContainText(ean);
		await expect(dialog.getByRole('button', { name: 'Zuordnen und bestellen' })).toBeDisabled();
		expect(querySQL(`SELECT count(*) FROM buecher_titel WHERE isbn = '${ean}'`)).toBe('0');

		await nehmen.click();
		await expect(dialog.getByText(/in zehnstelliger Form/)).toBeHidden();
		await expect(dialog.getByText(`${marke} Neubearbeitung`)).toBeVisible();
		await dialog.getByRole('button', { name: 'Zuordnen und bestellen' }).click();
		await expect(dialog).toBeHidden();

		// Zugeordnet ist der Titel aus dem Katalog; unter der EAN-13 steht weiter keiner.
		await expect(zeilen.first()).toContainText('Bestand aus 2 Auflagen');
		expect(
			querySQL(
				`SELECT count(DISTINCT werk_id) || '/' || count(werk_id) || '/' || bool_and(ist_lernmittel)
				 FROM buecher_titel WHERE id IN ('${alt}', '${neu}')`
			)
		).toBe('1/2/true');
		expect(querySQL(`SELECT count(*) FROM buecher_titel WHERE isbn = '${ean}'`)).toBe('0');
	} finally {
		seedSQL(`
			DELETE FROM werke WHERE id IN (
				SELECT werk_id FROM buecher_titel WHERE id IN ('${alt}', '${neu}') AND werk_id IS NOT NULL);
			DELETE FROM buecher_titel WHERE id IN ('${alt}', '${neu}') OR isbn = '${ean}';
		`);
		for (const [schluessel, wert] of Object.entries(vorher)) {
			if (wert) setze(schluessel, wert);
			else seedSQL(`DELETE FROM system_einstellungen WHERE schluessel = '${schluessel}';`);
		}
	}
});
