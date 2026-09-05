import { test, expect } from '@playwright/test';
import { uiLogin, uniqueSuffix, einstellungsKategorie, seedSQL, querySQL } from './helpers.js';

// Umbenennung ohne Schüler-ID, am echten Stack: Ein bestätigter Bestandsschüler kommt im
// Export unter neuem Nachnamen (gleiches Geburtsdatum, gleicher Schuleintritt). Die
// Vorschau muss ihn als sicheres Paar vorschlagen; nach dem Import trägt DERSELBE
// Datensatz (Barcode unverändert) den neuen Namen, und es gibt keine zweite Zeile.
//
// Der Bestand wird VOR dem Lauf gesät und die Datei enthält alle anderen bestätigten
// Schüler nicht — deshalb kann die Massenabgang-Bremse greifen; sie wird wie in
// admin-lusd.spec.js ehrlich abgewartet, nicht mit if-visible weggeraten.
test('LUSD-Import: umbenannter Schüler wird als Paar erkannt und behält seinen Datensatz', async ({
	page
}) => {
	const s = uniqueSuffix();
	const barcode = `UMB-${s}`;
	seedSQL(`
		INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr, geburtsdatum,
		                      schul_eintritt_am, lusd_bestaetigt_am)
		VALUES ('Anna${s}', 'Alt${s}', '05F1', '${barcode}', 2031, '2013-05-04', '2024-08-19', NOW());
	`);

	await uiLogin(page);
	await page.goto('/einstellungen');
	await einstellungsKategorie(page, 'Schuljahreswechsel').click();

	const csv =
		'Schueler_Vorname;Schueler_Nachname;Klassen_Klassenbezeichnung;Schueler_Geburtsdatum;Schueler_Eintritt_AktuelleSchule\n' +
		`Anna${s};Neu${s};06F1;04.05.2013;19.08.2024\n`;
	await page
		.locator('input[type="file"][accept=".csv,.xlsx"]')
		.last()
		.setInputFiles({ name: 'umbenennung.csv', mimeType: 'text/csv', buffer: Buffer.from(csv) });
	await page.getByRole('button', { name: 'Vorschau laden' }).click();

	// Das Paar steht in der Vorschau, als „sicher" und vorangekreuzt.
	await expect(page.getByText('Vermutlich dieselbe Person')).toBeVisible();
	const paar = page.getByRole('checkbox', {
		name: `Paar bestätigen: Anna${s} Alt${s} ist Anna${s} Neu${s}`
	});
	await expect(paar).toBeVisible();
	await expect(paar).toBeChecked();

	await page.getByRole('button', { name: 'Import finalisieren' }).click();
	const erfolg = page.getByText('LUSD-Import erfolgreich übernommen.');
	const bremse = page.getByRole('button', { name: /Massenabgang bestätigen/ });
	await expect(erfolg.or(bremse)).toBeVisible();
	if (await bremse.isVisible()) {
		await bremse.click();
	}
	await expect(erfolg).toBeVisible();
	await expect(page.getByText('1 von 1 Paaren bestätigt')).toBeVisible();

	// An der Datenbank: eine Zeile, derselbe Barcode, neuer Name, kein Abgänger.
	const zeilen = querySQL(
		`SELECT count(*) FROM schueler WHERE deleted_at IS NULL AND vorname = 'Anna${s}'`
	);
	expect(zeilen).toBe('1');
	const stand = querySQL(
		`SELECT nachname || '|' || barcode_id || '|' || ist_abgaenger FROM schueler WHERE vorname = 'Anna${s}'`
	);
	expect(stand).toBe(`Neu${s}|${barcode}|false`);
});

// Zusammenführen von Hand: zwei Datensätze derselben Testperson, die Akte des einen
// öffnen, den anderen suchen (auch als Abgänger auffindbar), zusammenführen. Danach ein
// Datensatz mit dem Barcode des behaltenen und dem Namen des LUSD-frischeren.
test('Schülerakte: zwei Datensätze zusammenführen', async ({ page }) => {
	const s = uniqueSuffix();
	seedSQL(`
		INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr, geburtsdatum,
		                      ist_abgaenger, ist_gesperrt, block_reason, abgaenger_seit, lusd_bestaetigt_am)
		VALUES ('Zf${s}', 'Alt${s}', '07A', 'ZFA-${s}', 2031, '2012-03-03',
		        true, true, 'Automatisierte Abgänger-Sperre (Karenzzeit vor Anonymisierung)', NOW(), NOW() - interval '1 year');
		INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr, geburtsdatum, lusd_bestaetigt_am)
		VALUES ('Zf${s}', 'Neu${s}', '08A', 'ZFN-${s}', 2031, '2012-03-03', NOW());
	`);
	const neuID = querySQL(`SELECT id FROM schueler WHERE barcode_id = 'ZFN-${s}'`);

	await uiLogin(page);
	// In die Akte des NEUEN Datensatzes über die Schülerdatei; der alte ist Abgänger und
	// steht nicht in der Aktivliste — genau das soll die Kandidatensuche überbrücken.
	await page.goto('/schuelerdatei');
	await page.getByLabel('Schüler suchen').first().fill(`Neu${s}`);
	await page.getByText(`Zf${s} Neu${s}`).first().click();
	await page.getByRole('button', { name: 'Stammdaten & Adresse' }).click();
	await page.getByRole('button', { name: 'Mit anderem Datensatz zusammenführen' }).click();

	const dialog = page.getByRole('dialog', { name: 'Datensatz zusammenführen' });
	await expect(dialog).toBeVisible();
	await dialog.getByLabel('Anderen Datensatz suchen').fill(`Alt${s}`);
	await dialog.getByRole('button', { name: new RegExp(`Zf${s}\\s+Alt${s}`) }).click();
	await expect(dialog.getByText('Welcher Datensatz bleibt?')).toBeVisible();
	// Es bleibt der ALTE (Ausweis in der Hand des Kindes).
	await dialog.getByLabel(new RegExp(`Alt${s}`)).check();
	await dialog.getByRole('button', { name: 'Zusammenführen' }).click();

	await expect(page.getByText(/Zusammengeführt:/)).toBeVisible();
	const zeilen = querySQL(`SELECT count(*) FROM schueler WHERE vorname = 'Zf${s}'`);
	expect(zeilen).toBe('1');
	// Barcode des behaltenen, Name/Klasse des LUSD-frischeren, kein Abgänger mehr, entsperrt.
	const stand = querySQL(
		`SELECT barcode_id || '|' || nachname || '|' || klasse || '|' || ist_abgaenger || '|' || ist_gesperrt FROM schueler WHERE vorname = 'Zf${s}'`
	);
	expect(stand).toBe(`ZFA-${s}|Neu${s}|08A|false|false`);
	const audit = querySQL(
		`SELECT count(*) FROM audit_logs WHERE aktion = 'SCHUELER_ZUSAMMENGEFUEHRT' AND details->>'aufgeloest_id' = '${neuID}'`
	);
	expect(audit).toBe('1');
});
