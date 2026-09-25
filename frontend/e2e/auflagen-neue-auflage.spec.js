import { test, expect } from '@playwright/test';
import { uiLogin, gehZu, seedSQL, querySQL, uniqueSuffix } from './helpers.js';

// „Neue Auflage bestellen" (docs/OFFEN.md 4.18, Stufe 4): Die alte Auflage steht im
// Bestellbedarf, der Besteller gibt im Menü der Zeile die ISBN der neuen ein, bestätigt den
// Vorschlag — und danach steht das Buch als eine Zeile mit beiden Auflagen da, die neue liegt
// im Warenkorb (Topf Land) und ist ein Lernmittel. Die neue Auflage liegt schon im Katalog,
// damit die ISBN-Tür keinen Katalogdienst im Netz fragt.
test('Bestellbedarf: neue Auflage über das Menü der Zeile zuordnen und bestellen', async ({
	page
}) => {
	const marke = `E2E-NeuAufl-${uniqueSuffix()}`;
	const einstellung = (/** @type {string} */ schluessel) =>
		querySQL(`SELECT wert FROM system_einstellungen WHERE schluessel = '${schluessel}'`);
	const vorher = {
		bestellbedarf_warnung_aktiv: einstellung('bestellbedarf_warnung_aktiv'),
		bestellbedarf_schwelle: einstellung('bestellbedarf_schwelle')
	};
	const setze = (/** @type {string} */ schluessel, /** @type {string} */ wert) =>
		seedSQL(`INSERT INTO system_einstellungen (schluessel, wert) VALUES ('${schluessel}', '${wert}')
		         ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert;`);
	const ziffern = String(Date.now()).slice(-8);
	const isbnNeu = `97832${ziffern}`;
	const alt = querySQL(
		`INSERT INTO buecher_titel (titel, auflage, erscheinungsjahr, ist_lernmittel)
		 VALUES ('${marke}', '3. Aufl.', 2019, true) RETURNING id`
	).split('\n')[0];
	const neu = querySQL(
		`INSERT INTO buecher_titel (titel, auflage, erscheinungsjahr, ist_lernmittel, isbn)
		 VALUES ('${marke} Neubearbeitung', '4. Aufl.', 2024, false, '${isbnNeu}') RETURNING id`
	).split('\n')[0];

	try {
		seedSQL(
			`INSERT INTO buecher_exemplare (titel_id, barcode_id) VALUES ('${alt}', '${marke}-1');`
		);
		setze('bestellbedarf_warnung_aktiv', 'true');
		setze('bestellbedarf_schwelle', '5');

		await uiLogin(page);
		await gehZu(page, '/bestellungen');
		const filter = page.getByRole('searchbox', { name: 'Bestellvorschläge filtern' });
		await filter.fill(marke);
		const zeilen = page.locator('div.group').filter({ hasText: marke });
		await expect(zeilen).toHaveCount(1);

		await zeilen
			.first()
			.getByRole('button', { name: `Weitere Aktionen zu ${marke}` })
			.click();
		await page.getByRole('menuitem', { name: /Neue Auflage bestellen/ }).click();
		const dialog = page.getByRole('dialog', { name: 'Neue Auflage bestellen' });
		await dialog.getByLabel('ISBN der neuen Auflage').fill(isbnNeu);
		await dialog.getByRole('button', { name: 'Suchen' }).click();
		await expect(dialog.getByText(`${marke} Neubearbeitung`)).toBeVisible();
		await expect(dialog.getByText(/als Lernmittel geführt/)).toBeVisible();
		await dialog.getByRole('button', { name: 'Zuordnen und bestellen' }).click();
		await expect(dialog).toBeHidden();

		// Die Zeile ist jetzt das Buch: die neue Auflage vorn, beide in der Aufschlüsselung.
		await expect(zeilen).toHaveCount(1);
		await expect(zeilen.first()).toContainText(`${marke} Neubearbeitung`);
		await expect(zeilen.first()).toContainText(
			'Bestand aus 2 Auflagen: 4. Aufl. · 2024 (0), 3. Aufl. · 2019 (1)'
		);
		await expect(page.getByRole('region', { name: 'Bestellung Lernmittelfreiheit' })).toContainText(
			`${marke} Neubearbeitung`
		);
		expect(
			querySQL(
				`SELECT count(DISTINCT werk_id) || '/' || count(werk_id) || '/' || bool_and(ist_lernmittel)
				 FROM buecher_titel WHERE id IN ('${alt}', '${neu}')`
			)
		).toBe('1/2/true');
	} finally {
		seedSQL(`
			DELETE FROM werke WHERE id IN (
				SELECT werk_id FROM buecher_titel WHERE id IN ('${alt}', '${neu}') AND werk_id IS NOT NULL);
			DELETE FROM buecher_titel WHERE id IN ('${alt}', '${neu}');
		`);
		for (const [schluessel, wert] of Object.entries(vorher)) {
			if (wert) setze(schluessel, wert);
			else seedSQL(`DELETE FROM system_einstellungen WHERE schluessel = '${schluessel}';`);
		}
	}
});
