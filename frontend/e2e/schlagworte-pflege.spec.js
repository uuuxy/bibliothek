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
// umgehängt. Seit dem 23.09.2026 mit dem Kästchen „als Verweis behalten" (Umbenennen mit der
// Vorbelegung, Zusammenführen abgewählt) und dem Wettlauf zweier Filter-Schalter.
test('Schlagwort-Pflege: umbenennen, zusammenführen, Verweis, Filter, löschen', async ({
	page
}) => {
	const s = uniqueSuffix().slice(0, 6);
	const krimi = `Krimi ${s}`;
	const detektiv = `Detektiv ${s}`;
	const roman = `Kriminalroman ${s}`;
	const verweis = `Krimis ${s}`;
	const abenteuer = `Abenteuer ${s}`;
	const titel = `E2E-Pflege-Titel-${s}`;
	seedSQL(`
		WITH t AS (INSERT INTO buecher_titel (titel) VALUES ('${titel}') RETURNING id),
		     w AS (INSERT INTO schlagworte (wort)
		           VALUES ('${krimi}'), ('${detektiv}'), ('${abenteuer}') RETURNING id)
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
	/**
	 * @param {string} titelText @param {string} feld @param {string} wert @param {string} aktion
	 * @param {string} [abwaehlen] - Beschriftung des Kästchens, das vor der Aktion abgewählt wird
	 */
	async function dialog(titelText, feld, wert, aktion, abwaehlen) {
		const d = page.getByRole('dialog', { name: titelText });
		await expect(d).toBeVisible();
		await d.getByLabel(feld, { exact: true }).fill(wert);
		if (abwaehlen) await d.getByRole('checkbox', { name: abwaehlen }).uncheck();
		await d.getByRole('button', { name: aktion, exact: true }).click();
		await expect(d).toBeHidden({ timeout: 15000 });
	}
	/** @param {string} wort */
	const verweisZiel = (wort) =>
		querySQL(`SELECT coalesce(max(z.wort), '') FROM schlagworte v JOIN schlagworte z ON z.id = v.verweis_auf
		          WHERE v.wort = '${wort}'`);
	/** @param {string} wort */
	const schalter = (wort) => page.getByRole('switch', { name: `„${wort}“ als Filter im Portal` });

	try {
		await uiLogin(page);
		await gehZu(page, '/einstellungen');
		await einstellungsKategorie(page, 'Schlagworte').click();
		await page.getByRole('searchbox', { name: 'Schlagwort suchen' }).fill(s);
		await expect(page.getByRole('cell', { name: krimi, exact: true })).toBeVisible({
			timeout: 15000
		});

		// Umbenennen: Das Kästchen steht vorbelegt, die alte Schreibweise bleibt als Verweis.
		await menue(krimi, 'Umbenennen');
		await expect(
			page.getByRole('dialog').getByRole('checkbox', { name: `„${krimi}“ als Verweis behalten` })
		).toBeChecked();
		await dialog(`„${krimi}“ umbenennen`, 'Neue Schreibweise', roman, 'Umbenennen');
		await expect.poll(amTitel).toBe(`${abenteuer}|${detektiv}|${roman}`);
		expect(verweisZiel(krimi)).toBe(roman);

		// Zusammenführen abgewählt: „Detektiv" fällt ganz weg.
		await menue(detektiv, 'Zusammenführen mit …');
		await dialog(
			`„${detektiv}“ zusammenführen`,
			'Mit Schlagwort',
			roman,
			'Zusammenführen',
			`„${detektiv}“ als Verweis behalten`
		);
		await expect.poll(amTitel).toBe(`${abenteuer}|${roman}`);
		expect(querySQL(`SELECT count(*) FROM schlagworte WHERE wort = '${detektiv}'`)).toBe('0');

		await menue(roman, 'Verweis anlegen …');
		await dialog(`Verweis auf „${roman}“`, 'Schreibweise', verweis, 'Verweis anlegen');
		await expect
			.poll(() =>
				querySQL(`SELECT count(*) FROM schlagworte v JOIN schlagworte z ON z.id = v.verweis_auf
				          WHERE v.wort = '${verweis}' AND z.wort = '${roman}'`)
			)
			.toBe('1');

		// Zwei Schalter kurz nacheinander. Die Liste, die nach dem ersten lädt, hält der Test
		// zurück, bis die nach dem zweiten da ist: Sie trägt den zweiten Schalter noch als aus
		// und darf ihn nicht zurückdrehen (OFFEN.md 4.20, Punkt a).
		/** @type {() => void} */
		let zugestellt = () => {};
		const alteListeDa = new Promise((fertig) => (zugestellt = fertig));
		let halten = true;
		await page.route('**/api/schlagworte/pflege', async (route) => {
			if (!halten) return route.continue();
			halten = false;
			const antwort = await route.fetch();
			await new Promise((weiter) => setTimeout(weiter, 1500));
			await route.fulfill({ response: antwort });
			zugestellt();
		});
		const ersteListe = page.waitForRequest('**/api/schlagworte/pflege');
		await schalter(roman).click();
		await ersteListe;
		await schalter(abenteuer).click();
		await expect
			.poll(() =>
				querySQL(`SELECT string_agg(ist_filter::text, ',' ORDER BY wort) FROM schlagworte
				          WHERE wort IN ('${abenteuer}', '${roman}')`)
			)
			.toBe('true,true');
		await alteListeDa;
		// Die zurückgehaltene Antwort ist zugestellt; einen Augenblick geben, bis die Seite sie
		// verarbeitet hätte — sonst prüfte die Erwartung den Stand davor.
		await page.waitForTimeout(500);
		await expect(schalter(abenteuer)).toHaveAttribute('aria-checked', 'true');
		await expect(schalter(roman)).toHaveAttribute('aria-checked', 'true');
		await page.unroute('**/api/schlagworte/pflege');

		await menue(roman, 'Löschen');
		const frage = page.getByRole('dialog', { name: `„${roman}“ löschen?` });
		await expect(frage).toContainText(
			'1 Titel verliert das Schlagwort, 2 Verweise darauf fallen mit.'
		);
		await frage.getByRole('button', { name: 'Löschen' }).click();
		await expect
			.poll(() =>
				querySQL(
					`SELECT count(*) FROM schlagworte WHERE wort IN ('${roman}', '${krimi}', '${verweis}')`
				)
			)
			.toBe('0');
		expect(amTitel()).toBe(abenteuer);
	} finally {
		seedSQL(`
			DELETE FROM schlagworte WHERE wort LIKE '% ${s}';
			DELETE FROM buecher_titel WHERE titel = '${titel}';
		`);
	}
});
