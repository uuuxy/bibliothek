import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix } from './helpers.js';

// Feature: aus Mahnwesen/Abgänger muss ein Klick auf den Schüler dessen Profil öffnen
// (vorher nur in der Schülerdatei möglich — inkonsistent). Hier über die Abgänger-Ansicht
// verifiziert; Mahnwesen nutzt exakt denselben Mechanismus (uiStore.requestedStudentId).
test('Abgänger-Zeile klickbar → Schülerprofil öffnet sich', async ({ page }) => {
	const s = uniqueSuffix();

	// Abgänger MIT offenem Buch — nur solche erscheinen in der Abgänger-Ansicht.
	seedSQL(`
		WITH t AS (
			INSERT INTO buecher_titel (titel) VALUES ('E2E-Abg-Titel ${s}') RETURNING id
		),
		ex AS (
			INSERT INTO buecher_exemplare (titel_id, barcode_id)
			SELECT id, 'E2E-ABG-B-${s}' FROM t RETURNING id
		),
		sch AS (
			INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_abgaenger)
			VALUES ('E2E-ABG-S-${s}', 'Abgklick${s}', 'Testschueler', '10a',
			        EXTRACT(YEAR FROM CURRENT_DATE)::int, true)
			RETURNING id
		)
		INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		SELECT ex.id, sch.id, CURRENT_DATE - 5 FROM ex, sch;
	`);

	await uiLogin(page);
	await page.getByTitle('Abgänger').click();

	// Die Abgänger-Zeile ist als Button zugänglich (a11y) — anklicken.
	await page
		.getByRole('button', { name: new RegExp(`Profil von Abgklick${s} Testschueler`) })
		.click();

	// Verifiziert, dass das Profil korrekt geöffnet wurde.
	// Das Profil ist offen, sobald die profil-spezifischen Tabs erscheinen.
	await expect(page.getByText('Ausleihen & Historie')).toBeVisible();
	await expect(page.getByRole('heading', { name: new RegExp(`Abgklick${s}`) })).toBeVisible();
});
