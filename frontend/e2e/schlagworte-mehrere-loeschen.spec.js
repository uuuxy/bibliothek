import { test, expect } from '@playwright/test';
import {
	uiLogin,
	seedSQL,
	querySQL,
	gehZu,
	einstellungsKategorie,
	uniqueSuffix
} from './helpers.js';

// Mehrere Schlagworte auf einmal löschen (docs/OFFEN.md 4.20, wie Littera „Datenbearbeitung")
// über den Klickpfad bis in die Datenbank: markieren, die Leiste unten löscht, die Rückfrage
// nennt Wörter, Titelzahl und den Verweis, der mitfällt. Die Regeln (alle oder keins, jeder
// Titel einmal gezählt) beweist repository/schlagworte_loeschen_pg_test.go; hier geht es darum,
// dass genau die markierten Kennungen die Tür erreichen — und die nicht markierte nicht.
test('Schlagwort-Pflege: mehrere Wörter auf einmal löschen', async ({ page }) => {
	const s = uniqueSuffix().slice(0, 6);
	const magie = `Magie ${s}`;
	const muehle = `Mühle ${s}`;
	const freundschaft = `Freundschaft ${s}`;
	const zauberei = `Zauberei ${s}`;
	const titel = `E2E-Mehrfach-Titel-${s}`;
	seedSQL(`
		WITH t AS (INSERT INTO buecher_titel (titel) VALUES ('${titel}') RETURNING id),
		     w AS (INSERT INTO schlagworte (wort)
		           VALUES ('${magie}'), ('${muehle}'), ('${freundschaft}') RETURNING id)
		INSERT INTO titel_schlagworte (titel_id, schlagwort_id) SELECT t.id, w.id FROM t, w;
		INSERT INTO schlagworte (wort, verweis_auf) SELECT '${zauberei}', id FROM schlagworte WHERE wort = '${magie}';
	`);
	const markieren = (/** @type {string} */ wort) =>
		page.getByRole('checkbox', { name: `„${wort}“ markieren` }).check();

	try {
		await uiLogin(page);
		await gehZu(page, '/einstellungen');
		await einstellungsKategorie(page, 'Schlagworte').click();
		await page.getByRole('searchbox', { name: 'Schlagwort suchen' }).fill(s);
		await expect(page.getByRole('cell', { name: muehle, exact: true })).toBeVisible({
			timeout: 15000
		});

		// Ohne Treffer gibt es nichts zu markieren: Das Kästchen im Kopf ist gesperrt. Anklickbar
		// hakte es sich an, ohne dass etwas markiert war (Kaestchen hält einen eigenen Stand,
		// solange die Eigenschaft sich nicht ändert).
		const suche = page.getByRole('searchbox', { name: 'Schlagwort suchen' });
		const alle = page.getByRole('checkbox', { name: 'Alle angezeigten Schlagworte markieren' });
		await suche.fill(`${s}-kein-treffer`);
		await expect(page.getByText('Kein Schlagwort passt zur Suche.')).toBeVisible();
		await expect(alle).toBeDisabled();
		await suche.fill(s);
		await expect(alle).toBeEnabled();

		const leiste = page.getByRole('region', { name: 'Aktionen für die markierten Schlagworte' });
		await expect(leiste).toBeHidden();
		await markieren(magie);
		await markieren(muehle);
		await expect(leiste).toContainText('2 markiert');

		// Die Symbole erklären sich mit der Blase der Anwendung (data-tip), nicht mit dem
		// Tooltip des Browsers. Das Gate icon-tooltips sieht beide nicht: Es markiert nichts,
		// und es öffnet die Kategorie Schlagworte nicht.
		const blase = page.locator('[data-tooltip-blase]');
		await leiste.getByRole('button', { name: 'Markierung aufheben' }).hover();
		await expect(blase).toHaveText('Markierung aufheben');
		await page.getByRole('button', { name: `Aktionen für „${muehle}“` }).hover();
		await expect(blase).toHaveText(`Aktionen für „${muehle}“`);
		await page.mouse.move(0, 0);

		await leiste.getByRole('button', { name: 'Löschen' }).click();
		const frage = page.getByRole('dialog', { name: '2 Schlagworte löschen?' });
		await expect(frage).toContainText(
			`„${magie}“ und „${muehle}“. Sie stehen zusammen 2-mal an Titeln. 1 Verweis darauf fällt mit.`
		);
		await frage.getByRole('button', { name: 'Löschen' }).click();

		await expect
			.poll(() =>
				querySQL(
					`SELECT count(*) FROM schlagworte WHERE wort IN ('${magie}', '${muehle}', '${zauberei}')`
				)
			)
			.toBe('0');
		expect(
			querySQL(`
				SELECT string_agg(sw.wort, '|') FROM titel_schlagworte ts
				JOIN schlagworte sw ON sw.id = ts.schlagwort_id
				JOIN buecher_titel t ON t.id = ts.titel_id WHERE t.titel = '${titel}'`)
		).toBe(freundschaft);
		await expect(
			page.getByText('2 Schlagworte gelöscht. 1 Titel hat Schlagworte verloren.')
		).toBeVisible();
		await expect(leiste).toBeHidden();
		await expect(page.getByRole('cell', { name: freundschaft, exact: true })).toBeVisible();
	} finally {
		seedSQL(`
			DELETE FROM schlagworte WHERE wort LIKE '% ${s}';
			DELETE FROM buecher_titel WHERE titel = '${titel}';
		`);
	}
});
