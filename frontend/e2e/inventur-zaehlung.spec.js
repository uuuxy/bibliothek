import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix, menuepunkt } from './helpers.js';

// Ein Buch, das laut Programm verliehen ist, steht im Regal und wird gescannt. Der Abschluss
// erwartet es nicht, gefunden ist damit keins der erwarteten. Zählte die Seite jede Erfassung,
// zeigte sie „2 / 2 erfasst", die Rückfrage sagte „keines als verloren", und der Abschluss buchte
// doch einen Verlust. Gescannt werden ein erwartetes Buch und das verliehene; das zweite
// erwartete fehlt.
test('Inventur: Ein verliehenes Buch im Regal zählt nicht als gefunden', async ({ page }) => {
	await uiLogin(page);
	const s = uniqueSuffix();
	const signatur = `E2E-INVZ-${s}`;
	const [da, fehlt, verliehen] = ['DA', 'FEHLT', 'VERL'].map((b) => `B-INVZ${b}-${s}`);
	const barcodes = [da, fehlt, verliehen].map((b) => `'${b}'`).join(', ');

	try {
		seedSQL(`
			WITH t AS (
				INSERT INTO buecher_titel (titel, signatur)
				VALUES ('E2E-Zaehlbuch-${s}', '${signatur}') RETURNING id
			), ex AS (
				INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
				SELECT id, b, true FROM t, unnest(ARRAY[${barcodes}]) AS b
				RETURNING id, barcode_id
			), sch AS (
				INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
				VALUES ('E2E-INVZ-S-${s}', 'Zaehl${s}', 'Testschueler', '7a', 2030) RETURNING id
			)
			INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
			SELECT ex.id, sch.id, CURRENT_DATE + 14 FROM ex, sch WHERE ex.barcode_id = '${verliehen}';
		`);

		await menuepunkt(page, 'Inventur').click();
		await page.getByRole('button', { name: 'Neue Bestandsprüfung starten' }).click();
		await page.getByText('Nur bestimmte Signatur').click();
		await page.getByLabel('Signatur auswählen').fill(signatur);
		await page.getByRole('button', { name: 'Inventur Starten' }).click();
		const fortschritt = page.getByText(/^\s*\d+ \/ \d+\s*erfasst\s*$/);
		await expect(fortschritt).toHaveText(/0 \/ 2/);

		const scan = page.getByPlaceholder('Barcode scannen...');
		await scan.fill(da);
		await scan.press('Enter');
		await expect(fortschritt).toHaveText(/1 \/ 2/);
		await scan.fill(verliehen);
		await scan.press('Enter');
		await expect(page.getByText('Buch ist laut System aktuell ausgeliehen.')).toBeVisible();
		await expect(fortschritt, 'das verliehene Buch erwartet der Abschluss nicht').toHaveText(
			/1 \/ 2/
		);

		await page.getByRole('button', { name: 'Inventur abschließen' }).click();
		await expect(
			page.getByText(/^1 Buch aus dem Bereich dieser Inventur ist nicht gescannt\./)
		).toBeVisible();
		await page.getByRole('button', { name: 'Ja, unwiderruflich abschließen' }).click();
		await expect(page.getByRole('button', { name: 'Neue Bestandsprüfung starten' })).toBeVisible();

		// An der Datenbank: gebucht ist genau das eine Buch, das die Rückfrage nannte.
		expect(
			querySQL(
				`SELECT string_agg(barcode_id, ',') FROM buecher_exemplare
				 WHERE barcode_id IN (${barcodes}) AND ist_ausgesondert`
			)
		).toBe(fehlt);
	} finally {
		seedSQL(`
			DELETE FROM ausleihen WHERE exemplar_id IN (
				SELECT id FROM buecher_exemplare WHERE barcode_id IN (${barcodes}));
			DELETE FROM inventur_sessions WHERE scope_signatur = '${signatur}';
			DELETE FROM buecher_exemplare WHERE barcode_id IN (${barcodes});
			DELETE FROM buecher_titel WHERE titel = 'E2E-Zaehlbuch-${s}';
			DELETE FROM schueler WHERE barcode_id = 'E2E-INVZ-S-${s}';
		`);
	}
});
