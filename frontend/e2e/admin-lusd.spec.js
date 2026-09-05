import { test, expect } from '@playwright/test';
import { uiLogin, uniqueSuffix, einstellungsKategorie } from './helpers.js';

test('LUSD-Import: Preview und Ausführung', async ({ page }) => {
	await uiLogin(page);

	// 1. Navigation zu Einstellungen -> Schuljahreswechsel
	await page.goto('/einstellungen');
	await einstellungsKategorie(page, 'Schuljahreswechsel').click();

	// 2. CSV generieren (1 neuer Schüler)
	const s = uniqueSuffix();
	const csvContent = `lusd_id,vorname,nachname,klasse,geburtsdatum\nLUSD_NEW_${s},Neu_${s},Schueler_${s},05a,01.01.2015`;

	// 3. Datei-Upload simulieren via FileChooser
	const fileChooserPromise = page.waitForEvent('filechooser');
	await page.getByText('LUSD-CSV oder -Excel auswählen').click();
	const fileChooser = await fileChooserPromise;
	await fileChooser.setFiles({
		name: 'lusd_test.csv',
		mimeType: 'text/csv',
		buffer: Buffer.from(csvContent)
	});

	// 4. Vorschau berechnen
	await page.getByRole('button', { name: 'Vorschau laden' }).click();

	// Warten auf das Ergebnis oder Fehlermeldung
	await expect(page.getByText('Neue Schüler')).toBeVisible({ timeout: 2000 });

	// Akkordeon öffnen um den Namen zu sehen
	await page.locator('summary').filter({ hasText: 'Neue Schüler' }).click();

	// Verifikation der Vorschau (Akkordeon)
	await expect(page.getByText('Import abgeschlossen')).not.toBeVisible();
	await expect(page.getByText(`Neu_${s}`)).toBeVisible();

	// 5. Finalisieren
	await page.getByRole('button', { name: 'Import finalisieren' }).click();

	// 6. Bestätigen (falls die Massenabgang-Bremse greift, weil die DB mehr als 10 Schüler
	// hat und diese nicht im CSV sind).
	//
	// ERST warten, DANN verzweigen. `if (await x.isVisible())` allein fragt den Zustand
	// in dem Moment ab, in dem die Antwort des Servers vielleicht noch unterwegs ist:
	// Die Bremse ist dann "nicht sichtbar", der Zweig wird übersprungen, und die Prüfung
	// darunter läuft in einen Timeout mit einer Meldung, die vom falschen Ort erzählt.
	// Schlimmer noch andersherum — griffe die Bremse eines Tages NICHT mehr, liefe der
	// Test genauso durch und behauptete, alles sei in Ordnung.
	const erfolgsmeldung = page.getByText('LUSD-Import erfolgreich übernommen.');
	const overrideButton = page.getByRole('button', {
		name: 'Ja, Import trotz hoher Abgängerquote erzwingen'
	});
	// Eines von beidem MUSS erscheinen — hier wird der Wächter ehrlich: Kommt keins,
	// ist der Test an dieser Zeile rot, nicht drei Zeilen später aus einem anderen Grund.
	await expect(erfolgsmeldung.or(overrideButton)).toBeVisible();
	if (await overrideButton.isVisible()) {
		await overrideButton.click();
	}

	// 7. Erfolg verifizieren
	await expect(erfolgsmeldung).toBeVisible();
});

// Schrottdatei-Pfad: falsche Header und Binärmüll dürfen keinen 500er
// produzieren, sondern müssen als verständliche Meldung im UI landen —
// das Sekretariat lädt hier echte Exporte hoch, Tippfehler inklusive.
test('LUSD-Import: Schrottdateien werden sauber abgewiesen', async ({ page }) => {
	await uiLogin(page);

	await page.goto('/einstellungen');
	await einstellungsKategorie(page, 'Schuljahreswechsel').click();

	const uploadAndPreview = async (name, buffer) => {
		// Direkt aufs versteckte File-Input — Label-Texte ändern sich nach dem
		// ersten Upload (zeigen den Dateinamen). Das LUSD-Input ist das zweite
		// (letzte) CSV-File-Input der Datenverwaltung.
		await page
			.locator('input[type="file"][accept=".csv,.xlsx"]')
			.last()
			.setInputFiles({ name, mimeType: 'text/csv', buffer });
		await page.getByRole('button', { name: 'Vorschau laden' }).click();
	};

	// Fall 1: komplett falsche Header → verständliche deutsche Meldung
	await uploadAndPreview('kaputt.csv', Buffer.from('foo;bar;baz\n1;2;3'));
	await expect(
		page.getByText(/Pflichtspalte '.*' fehlt in der CSV-Kopfzeile/).first()
	).toBeVisible();
	await expect(page.getByText('Neue Schüler')).toHaveCount(0);

	// Fall 2: Binärmüll (ungültige Kodierung) → Fehler statt Crash
	await uploadAndPreview('binaer.csv', Buffer.from([0xff, 0xfe, 0x00, 0x9c, 0x01, 0x02, 0x03]));
	// Am Zustand festmachen, nicht am Zeichen: Der Locator hing vorher am Emoji ⚠️
	// und fiel um, als die Oberfläche auf eine einheitliche Icon-Familie umstellte.
	await expect(page.getByRole('alert').last()).toBeVisible();
	await expect(page.getByText('Neue Schüler')).toHaveCount(0);

	// Die Seite lebt noch: Vorschau-Kontrolle weiterhin bedienbar
	await expect(page.getByRole('button', { name: 'Vorschau laden' })).toBeVisible();
});

// Der echte Export der Schule (LANIS-Klassenliste) hat weder Schüler-ID noch Geburtsdatum:
// `Nachname;Vorname;Klasse;…`, Semikolon, UTF-8-BOM. Er muss durchgehen — als Nur-Name-
// Stufe mit sichtbarer Warnung, nicht als Fehler „Pflichtspalte fehlt".
test('LUSD-Import: LANIS-Klassenliste ohne ID und Geburtsdatum (Nur-Name-Stufe)', async ({
	page
}) => {
	await uiLogin(page);
	await page.goto('/einstellungen');
	await einstellungsKategorie(page, 'Schuljahreswechsel').click();

	const s = uniqueSuffix();
	const csvContent = `\uFEFFNachname;Vorname;Klasse;BKU;Spanisch\nLanis_${s};Neu_${s};05G1;x;\n`;
	await page
		.locator('input[type="file"][accept=".csv,.xlsx"]')
		.last()
		.setInputFiles({
			name: 'LUSD_LANIS_Klassenliste.csv',
			mimeType: 'text/csv',
			buffer: Buffer.from(csvContent)
		});
	await page.getByRole('button', { name: 'Vorschau laden' }).click();

	// Stufe wird genannt und als Warnung gezeigt; der Neuzugang steht in der Vorschau.
	await expect(page.getByText('Zuordnung nur über Vor- und Nachname')).toBeVisible();
	await page.locator('summary').filter({ hasText: 'Neue Schüler' }).click();
	await expect(page.getByText(`Neu_${s}`)).toBeVisible();

	await page.getByRole('button', { name: 'Import finalisieren' }).click();
	// Erst auf eines von beidem warten, dann verzweigen — siehe oben.
	const erfolg = page.getByText('LUSD-Import erfolgreich übernommen.');
	const bremse = page.getByRole('button', { name: /Massenabgang bestätigen/ });
	await expect(erfolg.or(bremse)).toBeVisible();
	if (await bremse.isVisible()) {
		await bremse.click();
	}
	await expect(erfolg).toBeVisible();
});
