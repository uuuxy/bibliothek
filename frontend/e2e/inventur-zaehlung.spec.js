import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix, menuepunkt, apiPost } from './helpers.js';

// Im Regal stehen zwei erwartete Bücher, dazu ein gesperrtes und eines, das laut Programm
// verliehen ist. Das gesperrte erwartet der Abschluss nicht: Gescannt zählt es nicht als gefunden.
// Zählte die Seite jede Erfassung, zeigte sie „2 / 2 erfasst", die Rückfrage sagte „keines als
// verloren", und der Abschluss buchte doch einen Verlust. Das verliehene bucht der Scan zurück wie
// die Theke; es war beim Beginn verliehen und zählt nicht mit. Das zweite erwartete fehlt.
test('Inventur: Zählung wie beim Abschluss, ein verliehenes Buch wird zurückgebucht', async ({
	page
}) => {
	await uiLogin(page);
	const s = uniqueSuffix();
	const signatur = `E2E-INVZ-${s}`;
	const [da, fehlt, gesperrt, verliehen] = ['DA', 'FEHLT', 'GESP', 'VERL'].map(
		(b) => `B-INVZ${b}-${s}`
	);
	const barcodes = [da, fehlt, gesperrt, verliehen].map((b) => `'${b}'`).join(', ');

	try {
		seedSQL(`
			WITH t AS (
				INSERT INTO buecher_titel (titel, signatur)
				VALUES ('E2E-Zaehlbuch-${s}', '${signatur}') RETURNING id
			), ex AS (
				INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
				SELECT id, b, b <> '${gesperrt}' FROM t, unnest(ARRAY[${barcodes}]) AS b
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
		await scan.fill(gesperrt);
		await scan.press('Enter');
		await expect(page.getByText(`Barcode: ${gesperrt}`)).toBeVisible();
		await expect(fortschritt, 'das gesperrte Buch erwartet der Abschluss nicht').toHaveText(
			/1 \/ 2/
		);
		await scan.fill(verliehen);
		await scan.press('Enter');
		await expect(
			page.getByText('Zurückgebucht: Das Buch war laut Programm noch ausgeliehen.')
		).toBeVisible();
		await expect(fortschritt, 'beim Beginn verliehen, erwartet der Abschluss es nicht').toHaveText(
			/1 \/ 2/
		);
		expect(
			querySQL(`SELECT count(*) FROM ausleihen a JOIN buecher_exemplare e ON e.id = a.exemplar_id
				WHERE e.barcode_id = '${verliehen}' AND a.rueckgabe_am IS NULL`),
			'die Ausleihe ist beendet'
		).toBe('0');

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

// Ein Buch des Regals ist beim Beginn verliehen und kommt während der Inventur an der Theke zurück,
// nach dem letzten Scan. Es stand beim Beginn nicht im Regal: Der Abschluss bucht es nicht als
// verloren, und die Rückfrage nennt nur das Buch, das wirklich fehlt.
test('Inventur: Ein Buch, das während der Inventur zurückkommt, gilt nicht als verloren', async ({
	page
}) => {
	await uiLogin(page);
	const s = uniqueSuffix();
	const signatur = `E2E-INVT-${s}`;
	const [da, fehlt, zurueck] = ['DA', 'FEHLT', 'ZUR'].map((b) => `B-INVT${b}-${s}`);
	const barcodes = [da, fehlt, zurueck].map((b) => `'${b}'`).join(', ');

	try {
		seedSQL(`
			WITH t AS (
				INSERT INTO buecher_titel (titel, signatur)
				VALUES ('E2E-Rueckkehrbuch-${s}', '${signatur}') RETURNING id
			), ex AS (
				INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
				SELECT id, b, true FROM t, unnest(ARRAY[${barcodes}]) AS b
				RETURNING id, barcode_id
			), sch AS (
				INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
				VALUES ('E2E-INVT-S-${s}', 'Rueck${s}', 'Testschueler', '7a', 2030) RETURNING id
			)
			INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
			SELECT ex.id, sch.id, CURRENT_DATE + 14 FROM ex, sch WHERE ex.barcode_id = '${zurueck}';
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

		// An der Theke, nach dem letzten Scan: die Rückgabe, ohne geöffneten Leser.
		const rueckgabe = await apiPost(page, '/api/action', { query: zurueck });
		expect(rueckgabe.status(), 'die Theke bucht die Rückgabe').toBe(200);

		await page.getByRole('button', { name: 'Inventur abschließen' }).click();
		await expect(
			page.getByText(/^1 Buch aus dem Bereich dieser Inventur ist nicht gescannt\./)
		).toBeVisible();
		await page.getByRole('button', { name: 'Ja, unwiderruflich abschließen' }).click();
		await expect(page.getByRole('button', { name: 'Neue Bestandsprüfung starten' })).toBeVisible();
		expect(
			querySQL(
				`SELECT string_agg(barcode_id, ',') FROM buecher_exemplare
				 WHERE barcode_id IN (${barcodes}) AND ist_ausgesondert`
			),
			'gebucht ist nur das Buch, das beim Beginn im Regal stand und fehlt'
		).toBe(fehlt);
	} finally {
		seedSQL(`
			DELETE FROM ausleihen WHERE exemplar_id IN (
				SELECT id FROM buecher_exemplare WHERE barcode_id IN (${barcodes}));
			DELETE FROM inventur_sessions WHERE scope_signatur = '${signatur}';
			DELETE FROM buecher_exemplare WHERE barcode_id IN (${barcodes});
			DELETE FROM buecher_titel WHERE titel = 'E2E-Rueckkehrbuch-${s}';
			DELETE FROM schueler WHERE barcode_id = 'E2E-INVT-S-${s}';
		`);
	}
});

// Läuft im gewählten Bereich schon eine Inventur, nennt der Kasten über der Liste den Grund. Er
// erscheint nach dem Klick auf „Inventur Starten", ohne dass der Fokus zu ihm wandert; vorgelesen
// wird er deshalb nur als Meldung (role="alert"), wie dieselbe Meldung im Start-Dialog.
test('Inventur: Der Grund über der Liste wird als Meldung vorgelesen', async ({ page }) => {
	await uiLogin(page);
	const s = uniqueSuffix();
	const signatur = `E2E-INVM-${s}`;

	try {
		seedSQL(`
			INSERT INTO inventur_sessions (scope_type, scope_signatur, scope_label)
			VALUES ('signature', '${signatur}', 'Signatur ${signatur}');
		`);

		await menuepunkt(page, 'Inventur').click();
		await page.getByRole('button', { name: 'Neue Bestandsprüfung starten' }).click();
		await page.getByText('Nur bestimmte Signatur').click();
		await page.getByLabel('Signatur auswählen').fill(signatur);
		await page.getByRole('button', { name: 'Inventur Starten' }).click();

		// Der Dialog schließt nach der Ablehnung; dieselbe Meldung in ihm zählt nicht.
		await expect(page.getByRole('heading', { name: 'Bereich der Inventur wählen' })).toBeHidden();
		await expect(page.getByText('Für diesen Bereich läuft bereits eine Inventur')).toBeVisible();
		await expect(
			page.getByRole('alert').filter({ hasText: 'Für diesen Bereich läuft bereits eine Inventur' })
		).toBeVisible();
	} finally {
		seedSQL(`DELETE FROM inventur_sessions WHERE scope_signatur = '${signatur}';`);
	}
});
