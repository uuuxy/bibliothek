import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix } from './helpers.js';

/** Die Abgänger-Ansicht zeigt ihre Zeilen nur in der Saison (01.05.–31.07.), die der
 *  Server bestimmt. Außerhalb gibt es keine Zeile zum Anklicken — dann überspringen,
 *  sichtbar im Bericht, statt grün ohne Prüfung.
 *  @param {import('@playwright/test').Page} page */
async function saisonOderUeberspringen(page) {
	const { fenster } = await (await page.request.get('/api/abgaenger')).json();
	test.skip(!fenster.offen, `Abgängerliste außerhalb der Saison (${fenster.von}–${fenster.bis})`);
}

// Feature: aus Mahnwesen/Abgänger muss ein Klick auf den Schüler dessen Profil öffnen
// (vorher nur in der Schülerdatei möglich — inkonsistent). Hier über die Abgänger-Ansicht
// verifiziert; Mahnwesen nutzt exakt denselben Mechanismus (uiStore.requestedStudentId).
test('Abgänger-Zeile klickbar → Schülerprofil öffnet sich', async ({ page }) => {
	const s = uniqueSuffix();

	// Abschlussklasse MIT offenem Buch — nur solche erscheinen in der Abgänger-Ansicht,
	// und nur in der Saison (01.05.–31.07.); außerhalb wird der Test übersprungen, die
	// Verdrahtung deckt dann AbgaengerTabelle.test.js, die Liste graduates_pg_test.go.
	seedSQL(`
		WITH t AS (
			INSERT INTO buecher_titel (titel) VALUES ('E2E-Abg-Titel ${s}') RETURNING id
		),
		ex AS (
			INSERT INTO buecher_exemplare (titel_id, barcode_id)
			SELECT id, 'E2E-ABG-B-${s}' FROM t RETURNING id
		),
		sch AS (
			INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
			VALUES ('E2E-ABG-S-${s}', 'Abgklick${s}', 'Testschueler', '9H1',
			        EXTRACT(YEAR FROM CURRENT_DATE)::int)
			RETURNING id
		)
		INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		SELECT ex.id, sch.id, CURRENT_DATE - 5 FROM ex, sch;
	`);

	await uiLogin(page);
	await saisonOderUeberspringen(page);
	await page.getByTitle('Abgänger').click();

	// Die Abgänger-Zeile ist als Button zugänglich (a11y) — anklicken.
	await page
		.getByRole('button', { name: new RegExp(`Profil von Abgklick${s} Testschueler`) })
		.click();

	// BEWEIS: Das Profil ist offen, sobald die profil-spezifischen Tabs erscheinen
	// (die gibt es in der Abgänger-Liste nicht).
	await expect(page.getByText('Ausleihen & Historie')).toBeVisible();
	await expect(page.getByRole('heading', { name: new RegExp(`Abgklick${s}`) })).toBeVisible();
});

// Der Reiter folgt der ABSICHT, mit der das Profil geöffnet wurde — nicht der Route.
// Beide Wege landen in derselben Ansicht (Schülerdatei), meinen aber Verschiedenes:
// Wer aus den Abgängern kommt, fragt nach Büchern; wer selbst gesucht hat, will die
// Stammdaten (Elternkontakt, Adressabgleich). Vorher öffnete beides "Ausleihen".
test('Profil-Reiter folgt der Absicht: Abgänger → Ausleihen, eigene Suche → Stammdaten', async ({
	page
}) => {
	const s = uniqueSuffix();

	seedSQL(`
		WITH t AS (
			INSERT INTO buecher_titel (titel) VALUES ('E2E-Reiter-Titel ${s}') RETURNING id
		),
		ex AS (
			INSERT INTO buecher_exemplare (titel_id, barcode_id)
			SELECT id, 'E2E-REI-B-${s}' FROM t RETURNING id
		),
		sch AS (
			INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
			VALUES ('E2E-REI-S-${s}', 'Reiter${s}', 'Testschueler', '10R1',
			        EXTRACT(YEAR FROM CURRENT_DATE)::int)
			RETURNING id
		)
		INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		SELECT ex.id, sch.id, CURRENT_DATE - 5 FROM ex, sch;
	`);
	// Weg 2 sucht einen zweiten, eigenen Schüler — die Aussage dieses Tests (Reiter
	// folgt der Absicht) hängt nicht daran, wer es ist.
	seedSQL(`
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ('E2E-REI-AKTIV-${s}', 'Reiteraktiv${s}', 'Testschueler', '10b',
		        EXTRACT(YEAR FROM CURRENT_DATE)::int + 1);
	`);

	await uiLogin(page);
	await saisonOderUeberspringen(page);

	// Weg 1: aus den Abgängern heraus — die Frage ist "hat der noch Bücher?"
	await page.getByTitle('Abgänger').click();
	await page
		.getByRole('button', { name: new RegExp(`Profil von Reiter${s} Testschueler`) })
		.click();

	const ausleihReiter = page.getByRole('button', { name: 'Ausleihen & Historie' });
	await expect(ausleihReiter).toBeVisible();
	// Der aktive Reiter trägt die blaue Unterkante — daran hängt die Zusicherung,
	// nicht an der Sichtbarkeit der Schaltfläche (beide sind immer sichtbar).
	await expect(ausleihReiter).toHaveClass(/border-blue-600/);
	await expect(page.getByText(`E2E-Reiter-Titel ${s}`).first()).toBeVisible();

	// Weg 2: in der Schülerdatei selbst gesucht — die Frage ist "wie erreiche ich die Eltern?"
	// Erst das offene Profil schließen: Die Abgänger-Ansicht landet in DERSELBEN Ansicht,
	// solange dort ein Profil offen ist, gibt es keine Liste zum Suchen.
	await page.getByTitle('Schüler schließen (ESC)').click();
	await page.getByLabel('Schüler suchen').first().fill(`Reiteraktiv${s}`);
	await page.getByText(`Reiteraktiv${s} Testschueler`).first().click();

	await expect(page.getByRole('button', { name: 'Stammdaten & Adresse' })).toHaveClass(
		/border-blue-600/
	);
});
