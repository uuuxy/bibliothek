import { test, expect } from '@playwright/test';
import { uiLogin, apiPost, querySQL, seedSQL, uniqueSuffix } from './helpers.js';

// Schlagworte am Titel (Migration 138) über den echten Klickpfad bis in die Datenbank.
//
// Das Buchformular schickt den ganzen Titel per PUT zurück; die Schlagworte gehen dabei
// als ganze Menge mit. Vier Dinge kann nur dieser Weg zeigen:
//   1. Eintragen per Enter und per Komma kommt als Wörter an — nicht als ein Wort mit Komma.
//   2. Öffnen und unverändert speichern erhält sie. Käme das Formular aus der schlanken
//      Katalogliste (dort sind sie null), würden sie nicht angezeigt — und ein Wort, das man
//      dann einträgt, ersetzte still alle anderen.
//   3. Das × entfernt genau eines.
//   4. Ein getipptes, nicht bestätigtes Wort geht beim Klick auf „Speichern" nicht verloren —
//      das Feld übernimmt es beim Verlassen, bevor das Formular abschickt.
test('Schlagworte: eintragen, erhalten beim Speichern ohne Änderung, einzeln entfernen', async ({
	page
}) => {
	await uiLogin(page);
	const s = uniqueSuffix().slice(0, 6);
	const isbn = `9783${String(Date.now()).slice(-9)}`;
	const titel = `E2E-Schlagwort-Buch-${s}`;
	const fantasy = `Fantasy ${s}`;
	const tiere = `Magische Tiere ${s}`;
	const freundschaft = `Freundschaft ${s}`;
	const woerter = () =>
		querySQL(`
			SELECT coalesce(string_agg(sw.wort, '|' ORDER BY lower(sw.wort)), '')
			FROM titel_schlagworte ts
			JOIN schlagworte sw ON sw.id = ts.schlagwort_id
			JOIN buecher_titel t ON t.id = ts.titel_id
			WHERE t.isbn = '${isbn}'`);

	/** Öffnet die Maske „Buch bearbeiten" über die Titel-Verwaltung. */
	async function oeffne() {
		await page.getByTitle('Medienkatalog').click();
		await page.getByRole('tab', { name: 'Titel-Verwaltung' }).click();
		const suche = page.getByRole('searchbox', { name: 'Bücher durchsuchen' });
		await expect(suche).toBeVisible({ timeout: 15000 });
		await suche.fill(titel);
		await page.getByText(titel).first().click();
		await expect(page.locator('#buch-schlagworte')).toBeVisible({ timeout: 15000 });
	}
	async function speichere() {
		await page.getByRole('button', { name: 'Speichern' }).click();
		await expect(page.getByText('Buch erfolgreich gespeichert!').first()).toBeVisible({
			timeout: 15000
		});
	}

	try {
		const angelegt = await apiPost(page, '/api/books', {
			isbn,
			title: titel,
			author: 'E2E Autor',
			signatur: 'E2E SIG',
			coverUrl: '/covers/e2e-dummy.jpg',
			stock: 1
		});
		expect(angelegt.ok(), `Buch anlegen: ${angelegt.status()}`).toBeTruthy();

		// 1. Eintragen: Enter und Komma
		await oeffne();
		const feld = page.locator('#buch-schlagworte');
		await feld.fill(fantasy);
		await feld.press('Enter');
		await feld.fill(tiere);
		await feld.press(',');
		await expect(feld).toHaveValue('');
		await speichere();
		expect(woerter()).toBe(`${fantasy}|${tiere}`);

		// 2. Öffnen, nichts ändern, speichern: Die Chips stehen da, und die Menge bleibt.
		await oeffne();
		await expect(page.getByRole('button', { name: `„${fantasy}“ entfernen` })).toBeVisible();
		await expect(page.getByRole('button', { name: `„${tiere}“ entfernen` })).toBeVisible();
		await speichere();
		expect(woerter()).toBe(`${fantasy}|${tiere}`);

		// 3. Ein Wort entfernen
		await oeffne();
		await page.getByRole('button', { name: `„${fantasy}“ entfernen` }).click();
		await speichere();
		expect(woerter()).toBe(tiere);

		// 4. Tippen und direkt speichern, ohne Enter
		await oeffne();
		await page.locator('#buch-schlagworte').fill(freundschaft);
		await speichere();
		expect(woerter()).toBe(`${freundschaft}|${tiere}`);
	} finally {
		seedSQL(`
			DELETE FROM buecher_exemplare WHERE titel_id IN (SELECT id FROM buecher_titel WHERE isbn = '${isbn}');
			DELETE FROM buecher_titel WHERE isbn = '${isbn}';
			DELETE FROM schlagworte WHERE wort IN ('${fantasy}', '${tiere}', '${freundschaft}');
		`);
	}
});
