import { test, expect } from '@playwright/test';
import { uiLogin, gehZu, seedSQL, querySQL, uniqueSuffix } from './helpers.js';
import { isbnFormen } from '../src/lib/utils/isbnFormen.js';

// „Neue Auflage bestellen" über die andere Länge der ISBN (Migration 157): Die neue
// Auflage steht im Katalog, zehnstellig angelegt wie vom Titelblatt; der Katalog führt die ISBN
// dreizehnstellig. Eingegeben wird die zehnstellige: Die Tür findet den Titel, fragt nicht und
// legt nichts an — die zehn- und die dreizehnstellige Form sind dieselbe Nummer. Kein
// Katalogdienst im Netz: Die DNB fragt die Tür nur zu einer ISBN, die der Katalog nicht kennt.
test('Bestellbedarf: neue Auflage über die andere Länge der ISBN — der Titel aus dem Katalog, ohne Frage', async ({
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
		await dialog.getByLabel('ISBN der neuen Auflage').fill(zehn);
		await dialog.getByRole('button', { name: 'Suchen' }).click();

		// Der Titel aus dem Katalog mit der ISBN, wie der Katalog sie trägt; keine Frage.
		await expect(dialog.getByText(`${marke} Neubearbeitung`)).toBeVisible();
		await expect(dialog.getByText(ean)).toBeVisible();
		await expect(dialog.getByText(/in zehnstelliger Form|in dreizehnstelliger Form/)).toHaveCount(
			0
		);
		await expect(dialog.getByRole('button', { name: /Neu anlegen/ })).toHaveCount(0);

		await dialog.getByRole('button', { name: 'Zuordnen und bestellen' }).click();
		await expect(dialog).toBeHidden();

		// Zugeordnet ist der Titel aus dem Katalog; ein zweiter ist nicht entstanden.
		await expect(zeilen.first()).toContainText('Bestand aus 2 Auflagen');
		expect(
			querySQL(
				`SELECT count(DISTINCT werk_id) || '/' || count(werk_id) || '/' || bool_and(ist_lernmittel)
				 FROM buecher_titel WHERE id IN ('${alt}', '${neu}')`
			)
		).toBe('1/2/true');
		expect(querySQL(`SELECT count(*) FROM buecher_titel WHERE isbn IN ('${ean}', '${zehn}')`)).toBe(
			'1'
		);
		expect(querySQL(`SELECT isbn FROM buecher_titel WHERE id = '${neu}'`)).toBe(ean);
	} finally {
		seedSQL(`
			DELETE FROM werke WHERE id IN (
				SELECT werk_id FROM buecher_titel WHERE id IN ('${alt}', '${neu}') AND werk_id IS NOT NULL);
			DELETE FROM buecher_titel WHERE id IN ('${alt}', '${neu}') OR isbn IN ('${ean}', '${zehn}');
		`);
		for (const [schluessel, wert] of Object.entries(vorher)) {
			if (wert) setze(schluessel, wert);
			else seedSQL(`DELETE FROM system_einstellungen WHERE schluessel = '${schluessel}';`);
		}
	}
});
