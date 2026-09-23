import { test, expect } from '@playwright/test';
import {
	uiLogin,
	seedSQL,
	querySQL,
	gehZu,
	einstellungsKategorie,
	uniqueSuffix
} from './helpers.js';

// Die Pflegeseite der Schlagworte (docs/OFFEN.md 4.20, Migration 143) über den Klickpfad bis in
// die Datenbank: umbenennen, zusammenführen, Verweis anlegen, Filter setzen, löschen. Die
// Regeln selbst beweisen die PG-Tests am Server; hier geht es darum, dass jede Aktion der Seite
// die richtige Tür mit den richtigen Kennungen trifft — ein vertauschtes Ziel beim
// Zusammenführen sähe in der Liste nach Erfolg aus und hätte die Titel des falschen Worts
// umgehängt.
test('Schlagwort-Pflege: umbenennen, zusammenführen, Verweis, Filter, löschen', async ({
	page
}) => {
	const s = uniqueSuffix().slice(0, 6);
	const krimi = `Krimi ${s}`;
	const detektiv = `Detektiv ${s}`;
	const roman = `Kriminalroman ${s}`;
	const verweis = `Krimis ${s}`;
	const titel = `E2E-Pflege-Titel-${s}`;
	seedSQL(`
		WITH t AS (INSERT INTO buecher_titel (titel) VALUES ('${titel}') RETURNING id),
		     w AS (INSERT INTO schlagworte (wort) VALUES ('${krimi}'), ('${detektiv}') RETURNING id)
		INSERT INTO titel_schlagworte (titel_id, schlagwort_id) SELECT t.id, w.id FROM t, w;
	`);
	const amTitel = () =>
		querySQL(`
			SELECT coalesce(string_agg(sw.wort, '|' ORDER BY lower(sw.wort)), '')
			FROM titel_schlagworte ts
			JOIN schlagworte sw ON sw.id = ts.schlagwort_id
			JOIN buecher_titel t ON t.id = ts.titel_id
			WHERE t.titel = '${titel}'`);
	/** @param {string} wort @param {string} eintrag */
	async function menue(wort, eintrag) {
		await page.getByRole('button', { name: `Aktionen für „${wort}“` }).click();
		await page.getByRole('menuitem', { name: eintrag }).click();
	}
	/** @param {string} titelText @param {string} feld @param {string} wert @param {string} aktion */
	async function dialog(titelText, feld, wert, aktion) {
		const d = page.getByRole('dialog', { name: titelText });
		await expect(d).toBeVisible();
		await d.getByLabel(feld, { exact: true }).fill(wert);
		await d.getByRole('button', { name: aktion, exact: true }).click();
		await expect(d).toBeHidden({ timeout: 15000 });
	}

	try {
		await uiLogin(page);
		await gehZu(page, '/einstellungen');
		await einstellungsKategorie(page, 'Schlagworte').click();
		await page.getByRole('searchbox', { name: 'Schlagwort suchen' }).fill(s);
		await expect(page.getByRole('cell', { name: krimi, exact: true })).toBeVisible({
			timeout: 15000
		});

		await menue(krimi, 'Umbenennen');
		await dialog(`„${krimi}“ umbenennen`, 'Neue Schreibweise', roman, 'Umbenennen');
		await expect.poll(amTitel).toBe(`${detektiv}|${roman}`);

		await menue(detektiv, 'Zusammenführen mit …');
		await dialog(`„${detektiv}“ zusammenführen`, 'Mit Schlagwort', roman, 'Zusammenführen');
		await expect.poll(amTitel).toBe(roman);
		expect(
			querySQL(`SELECT z.wort FROM schlagworte v JOIN schlagworte z ON z.id = v.verweis_auf
			          WHERE v.wort = '${detektiv}'`)
		).toBe(roman);

		await menue(roman, 'Verweis anlegen …');
		await dialog(`Verweis auf „${roman}“`, 'Schreibweise', verweis, 'Verweis anlegen');
		await expect
			.poll(() =>
				querySQL(`SELECT count(*) FROM schlagworte v JOIN schlagworte z ON z.id = v.verweis_auf
				          WHERE v.wort = '${verweis}' AND z.wort = '${roman}'`)
			)
			.toBe('1');

		await page.getByRole('switch', { name: `„${roman}“ als Filter im Portal` }).click();
		await expect
			.poll(() => querySQL(`SELECT ist_filter FROM schlagworte WHERE wort = '${roman}'`))
			.toBe('t');

		await menue(roman, 'Löschen');
		const frage = page.getByRole('dialog', { name: `„${roman}“ löschen?` });
		await expect(frage).toContainText(
			'1 Titel verlieren das Schlagwort, 2 Verweise darauf fallen mit.'
		);
		await frage.getByRole('button', { name: 'Löschen' }).click();
		await expect
			.poll(() => querySQL(`SELECT count(*) FROM schlagworte WHERE wort LIKE '% ${s}'`))
			.toBe('0');
		expect(amTitel()).toBe('');
	} finally {
		seedSQL(`
			DELETE FROM schlagworte WHERE wort LIKE '% ${s}';
			DELETE FROM buecher_titel WHERE titel = '${titel}';
		`);
	}
});
