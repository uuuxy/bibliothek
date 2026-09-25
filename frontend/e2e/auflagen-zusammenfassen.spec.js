import { test, expect } from '@playwright/test';
import { uiLogin, gehZu, seedSQL, querySQL, uniqueSuffix } from './helpers.js';

// Auflagen eines Schulbuchs zusammenfassen (docs/OFFEN.md 4.18, Stufe 2): Der Weg, den eine
// Bibliothekskraft nimmt — Medienkatalog, Stift an der alten Auflage, Abschnitt „Auflagen",
// „Andere Auflage zuordnen", suchen, wählen, zuordnen — und wieder lösen. Geprüft wird am
// Bildschirm UND in der Datenbank: Beide Titel hängen danach am selben Werk, nach dem Lösen
// an keinem mehr, denn ein Buch mit einer Auflage ist keine Gruppe (repository/auflagen.go).
test('Auflagen: in der Titelmaske zuordnen und wieder lösen', async ({ page }) => {
	const marke = `E2E-Auflage-${uniqueSuffix()}`;
	const anlegen = (/** @type {string} */ auflage, /** @type {number} */ jahr) =>
		querySQL(
			`INSERT INTO buecher_titel (titel, auflage, erscheinungsjahr, ist_lernmittel)
			 VALUES ('${marke}', '${auflage}', ${jahr}, true) RETURNING id`
		).split('\n')[0];
	const alt = anlegen('3. Aufl.', 2019);
	const neu = anlegen('4. Aufl.', 2023);

	try {
		// Ein Exemplar je Auflage: Titel ohne Exemplar stehen in keinem Katalog.
		seedSQL(`INSERT INTO buecher_exemplare (titel_id, barcode_id)
		         VALUES ('${alt}', '${marke}-1'), ('${neu}', '${marke}-2');`);

		await uiLogin(page);
		await gehZu(page, '/medienkatalog');
		await page.getByRole('tab', { name: 'Suche & Filter' }).click();
		await page
			.getByRole('searchbox', { name: 'Suchen nach Titel, Fach, Klasse oder Autor' })
			.fill(marke);
		const karteAlt = page
			.locator('div.m3-state')
			.filter({ hasText: marke })
			.filter({ hasText: '3. Aufl.' });
		await karteAlt.getByRole('button', { name: 'Buch schnell bearbeiten' }).click();
		await expect(page.getByRole('heading', { name: 'Buch bearbeiten' })).toBeVisible();

		await expect(page.getByRole('heading', { name: 'Auflagen', exact: true })).toBeVisible();
		await expect(page.getByText('Keine andere Auflage zugeordnet.')).toBeVisible();
		await page.getByRole('button', { name: 'Andere Auflage zuordnen' }).click();

		const dialog = page.getByRole('dialog', { name: 'Andere Auflage zuordnen' });
		await expect(dialog.getByRole('button', { name: 'Zuordnen' })).toBeDisabled();
		await dialog.getByLabel('Andere Auflage suchen').fill(marke);
		// Die alte Auflage selbst ist kein Kandidat — sie gehört schon zu diesem Buch.
		await expect(dialog.getByRole('button', { name: /3\. Aufl\. · 2019/ })).toHaveCount(0);
		await dialog.getByRole('button', { name: /4\. Aufl\. · 2023/ }).click();
		await dialog.getByRole('button', { name: 'Zuordnen' }).click();
		await expect(dialog).toBeHidden();

		const liste = page.getByRole('list', { name: 'Auflagen dieses Buchs' });
		await expect(liste.getByRole('listitem')).toHaveCount(2);
		await expect(liste.getByRole('listitem').first()).toContainText('4. Aufl. · 2023');
		await expect(liste.getByRole('listitem').last()).toContainText(
			/3\. Aufl\. · 2019 — diese Auflage/
		);
		await expect(page.getByText('Zusammen: 2 von 2 verfügbar')).toBeVisible();
		expect(
			querySQL(
				`SELECT count(DISTINCT werk_id) || '/' || count(werk_id) FROM buecher_titel WHERE id IN ('${alt}', '${neu}')`
			)
		).toBe('1/2');

		await liste.getByRole('button', { name: '4. Aufl. · 2023 aus den Auflagen lösen' }).click();
		await expect(page.getByText('Keine andere Auflage zugeordnet.')).toBeVisible();
		expect(
			querySQL(`SELECT count(werk_id) FROM buecher_titel WHERE id IN ('${alt}', '${neu}')`)
		).toBe('0');
	} finally {
		seedSQL(`
			DELETE FROM werke WHERE id IN (
				SELECT werk_id FROM buecher_titel WHERE id IN ('${alt}', '${neu}') AND werk_id IS NOT NULL);
			DELETE FROM buecher_titel WHERE id IN ('${alt}', '${neu}');
		`);
	}
});
