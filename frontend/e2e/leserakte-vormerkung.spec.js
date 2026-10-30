import { test, expect } from '@playwright/test';
import { uiLogin, oeffneSchuelerProfil, seedSQL, uniqueSuffix } from './helpers.js';

// Die Karte „Vorgemerkte Bücher" der Akte las ein Feld, das der Server nie schickt, und zeigte
// bei jeder Vormerkung „Unbekannter Titel". Ihre Unit-Tests lieferten das falsche Feld selbst mit;
// deshalb hier die echte Antwort.
test('Akte: eine Vormerkung nennt ihren Titel', async ({ page }) => {
	const s = uniqueSuffix();
	const titel = `E2E-Vorgemerkt ${s}`;
	seedSQL(`
		WITH t AS (INSERT INTO buecher_titel (titel) VALUES ('${titel}') RETURNING id),
		l AS (INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		      VALUES ('E2E-VOR-${s}', 'Vorm${s}', 'Merkung', '7a', 2031) RETURNING id)
		INSERT INTO vormerkungen (titel_id, schueler_id) SELECT t.id, l.id FROM t, l;
	`);
	try {
		await uiLogin(page);
		await oeffneSchuelerProfil(page, `Vorm${s}`);

		await expect(page.getByRole('heading', { name: 'Vorgemerkte Bücher (1)' })).toBeVisible();
		await expect(page.getByRole('heading', { name: titel })).toBeVisible();
	} finally {
		seedSQL(`
			DELETE FROM vormerkungen WHERE schueler_id IN (SELECT id FROM leser WHERE barcode_id = 'E2E-VOR-${s}');
			DELETE FROM buecher_titel WHERE titel = '${titel}';
			DELETE FROM leser WHERE barcode_id = 'E2E-VOR-${s}';
		`);
	}
});
