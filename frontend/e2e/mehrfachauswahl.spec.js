import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix } from './helpers.js';

// M3, Desktop: „Select a range: Shift + click“, „Select all: Control + A, or Cmd + A (Mac)“.
// Die Regel steht in actions/mehrfachauswahl.js; hier an der Leserdatei im Browser: Die Aktion
// klickt die Kästchen selbst, und erst der Browser zeigt, dass Svelte die Klicks wie echte
// verarbeitet.
test('Leserdatei: Umschalt+Klick wählt einen Bereich, Strg+A alle, im Suchfeld bleibt Strg+A beim Text', async ({
	page
}) => {
	const s = uniqueSuffix();
	const vorname = `Mf${s}`;
	seedSQL(`
		INSERT INTO schueler (vorname, nachname, klasse, barcode_id, geburtsdatum, abgaenger_jahr)
		SELECT '${vorname}', 'Probe' || n, '7A', 'MF-${s}-' || n, '2012-01-01', 2030
		FROM generate_series(1, 6) n;
	`);
	const ids = querySQL(
		`SELECT string_agg(quote_literal(id::text), ',') FROM schueler WHERE barcode_id LIKE 'MF-${s}-%'`
	);
	try {
		await uiLogin(page);
		await page.goto('/schuelerdatei');
		const suche = page.getByRole('searchbox', { name: 'Leser suchen' });
		await suche.fill(vorname);
		const kaestchen = (/** @type {number} */ n) =>
			page.getByRole('checkbox', { name: `${vorname} Probe${n} für den Ausweisdruck markieren` });
		await expect(kaestchen(6)).toBeVisible();
		const leiste = page.getByRole('region', { name: 'Aktionen für die markierten Schüler' });

		await kaestchen(2).click();
		await kaestchen(5).click({ modifiers: ['Shift'] });
		await expect(leiste).toContainText('4 Schüler markiert');
		for (const n of [2, 3, 4, 5]) await expect(kaestchen(n)).toBeChecked();
		for (const n of [1, 6]) await expect(kaestchen(n)).not.toBeChecked();
		expect(
			await page.evaluate(() => window.getSelection()?.toString() ?? ''),
			'Umschalt+Klick zieht keine Textmarkierung über die Zeilen'
		).toBe('');

		// Im Suchfeld markiert Strg+A den Text, die Auswahl bleibt.
		await suche.focus();
		await page.keyboard.press('ControlOrMeta+a');
		expect(
			await suche.evaluate((/** @type {HTMLInputElement} */ f) => [
				f.selectionStart,
				f.selectionEnd
			])
		).toEqual([0, vorname.length]);
		await expect(leiste).toContainText('4 Schüler markiert');

		await kaestchen(5).focus();
		await page.keyboard.press('ControlOrMeta+a');
		await expect(leiste).toContainText('6 Schüler markiert');
		for (const n of [1, 2, 3, 4, 5, 6]) await expect(kaestchen(n)).toBeChecked();
	} finally {
		seedSQL(`DELETE FROM schueler WHERE id IN (${ids});`);
	}
});
