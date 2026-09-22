import { test, expect } from '@playwright/test';
import { uiLogin, apiPost, csrfToken, querySQL, seedSQL, uniqueSuffix } from './helpers.js';

// Bücher-CRUD über die /api/books-Schnittstelle (die auch das Admin-Formular
// nutzt) + Katalog-Suche im UI + der Signatur-Schutz aus Migration 038:
// Ein Littera-Import ohne Signaturspalte darf eine bestehende Buchrücken-
// Signatur NIE überschreiben (COALESCE(NULLIF(…)) in UpsertBookTitle).
test('Bücher: anlegen, Exemplare, Katalog-Suche, Signatur übersteht Littera-Import', async ({
	page
}) => {
	await uiLogin(page);
	const suffix = uniqueSuffix();
	// 13-stellige, garantiert eindeutige Ziffern-ISBN (Format-Validierung!)
	const isbn = `9781${String(Date.now()).slice(-9)}`;
	const titel = `E2E-CRUD-Buch-${suffix}`;

	try {
		// 1. Anlegen mit Signatur und 2 Exemplaren (stock erzeugt Barcodes)
		// coverUrl gesetzt: sonst startet der Handler einen externen
		// ISBN-Metadaten-Lookup (langsam/offline-abhängig)
		const created = await apiPost(page, '/api/books', {
			isbn,
			title: titel,
			author: 'E2E Autor',
			signatur: 'E2E SIG',
			coverUrl: '/covers/e2e-dummy.jpg',
			subject: '',
			gradeLevel: 7,
			track: '',
			stock: 2
		});
		expect(created.ok(), `Buch anlegen: ${created.status()}`).toBeTruthy();
		expect(querySQL(`SELECT signatur FROM buecher_titel WHERE isbn = '${isbn}'`)).toBe('E2E SIG');
		expect(
			querySQL(
				`SELECT count(*) FROM buecher_exemplare e JOIN buecher_titel t ON t.id = e.titel_id WHERE t.isbn = '${isbn}'`
			)
		).toBe('2');

		// 2. Katalog-Suche im UI: Titel finden, Karte zeigt die ISBN
		await page.getByTitle('Medienkatalog').click();
		await page.getByRole('tab', { name: 'Suche & Filter' }).click();
		// Ueber den zugaenglichen Namen, nicht ueber den Platzhalter: Der Text im Feld ist
		// Beschriftung und darf sich aendern (am 10.08.2026 tat er das, als alle Suchfelder
		// auf ein Bauteil kamen). Der aria-label sagt, WAS das Feld ist.
		const suche = page.getByRole('searchbox', {
			name: 'Suchen nach Titel, Fach, Klasse oder Autor'
		});
		await expect(suche).toBeVisible({ timeout: 15000 });
		await suche.fill(titel);
		await expect(page.getByText(titel).first()).toBeVisible();
		await expect(page.getByText(`ISBN: ${isbn}`).first()).toBeVisible();

		// 3. Signatur-Schutz über den ECHTEN Import-Pfad: Littera-CSV hat
		//    keine Buchrücken-Signaturspalte → Upsert kommt mit leerer
		//    Signatur an und darf 'E2E SIG' nicht überschreiben
		const csv = `Titel,ISBN,Barcode\n${titel},${isbn},LIT-${suffix}`;
		const token = await csrfToken(page);
		const imported = await page.request.post('/api/import/littera', {
			headers: { 'X-CSRF-Token': token },
			multipart: {
				file: { name: 'littera.csv', mimeType: 'text/csv', buffer: Buffer.from(csv) }
			}
		});
		expect(imported.ok(), `Littera-Import: ${imported.status()}`).toBeTruthy();

		expect(querySQL(`SELECT signatur FROM buecher_titel WHERE isbn = '${isbn}'`)).toBe('E2E SIG');
		// Das importierte Exemplar ist zusätzlich angekommen
		expect(
			querySQL(`SELECT count(*) FROM buecher_exemplare WHERE barcode_id = 'LIT-${suffix}'`)
		).toBe('1');
	} finally {
		seedSQL(`
            DELETE FROM buecher_exemplare WHERE titel_id IN (SELECT id FROM buecher_titel WHERE isbn = '${isbn}');
            DELETE FROM buecher_titel WHERE isbn = '${isbn}';
        `);
	}
});

// Regressions-Gate zur schlanken Katalogliste: GET /api/books liefert beschreibung/
// erweiterteEigenschaften bewusst LEER (Payload), und das Bearbeiten-Formular schickt
// per PUT das GANZE Objekt zurück. Würde das Formular aus der Listenzeile befüllt,
// leerte "Bearbeiten → Speichern" beide Felder still (Upsert-Blanking-Bugklasse —
// exakt so am 20.08.2026 als Regression gebaut und in der Nachprüfung gefunden).
// Das Formular muss deshalb vom Einzel-Read (/api/books/{id}) befüllt werden. Dieser
// Test beweist es über den echten Klickpfad: öffnen, NICHTS ändern, speichern —
// beide Felder unversehrt. Die toHaveValue-Prüfung schlägt zusätzlich schon beim
// Öffnen fehl, wenn das Formular aus der schlanken Liste käme (leere Textarea).
test('Bücher: Bearbeiten ohne Änderung erhält Beschreibung und erweiterte Eigenschaften', async ({
	page
}) => {
	await uiLogin(page);
	const suffix = uniqueSuffix();
	const isbn = `9782${String(Date.now()).slice(-9)}`;
	const titel = `E2E-Blank-Buch-${suffix}`;
	const beschreibung = `Wertvolle Beschreibung ${suffix}`;

	try {
		const created = await apiPost(page, '/api/books', {
			isbn,
			title: titel,
			author: 'E2E Autor',
			signatur: 'E2E SIG',
			coverUrl: '/covers/e2e-dummy.jpg',
			subject: '',
			gradeLevel: 7,
			track: '',
			stock: 1,
			beschreibung,
			erweiterteEigenschaften: { regal: `R-${suffix}` }
		});
		expect(created.ok(), `Buch anlegen: ${created.status()}`).toBeTruthy();

		await page.getByTitle('Medienkatalog').click();
		await page.getByRole('tab', { name: 'Titel-Verwaltung' }).click();
		const suche = page.getByRole('searchbox', { name: 'Bücher durchsuchen' });
		await expect(suche).toBeVisible({ timeout: 15000 });
		await suche.fill(titel);
		await page.getByText(titel).first().click();

		// Beweis der Quelle: Das Formular zeigt die Beschreibung — die schlanke
		// Listenzeile hätte hier eine leere Textarea.
		const feld = page.locator('#buch-beschreibung');
		await expect(feld).toBeVisible({ timeout: 15000 });
		await expect(feld).toHaveValue(beschreibung);

		await page.getByRole('button', { name: 'Speichern' }).click();
		await expect(page.getByText('Buch erfolgreich gespeichert!')).toBeVisible({
			timeout: 15000
		});

		expect(querySQL(`SELECT beschreibung FROM buecher_titel WHERE isbn = '${isbn}'`)).toBe(
			beschreibung
		);
		expect(
			querySQL(
				`SELECT erweiterte_eigenschaften->>'regal' FROM buecher_titel WHERE isbn = '${isbn}'`
			)
		).toBe(`R-${suffix}`);
	} finally {
		seedSQL(`
            DELETE FROM buecher_exemplare WHERE titel_id IN (SELECT id FROM buecher_titel WHERE isbn = '${isbn}');
            DELETE FROM buecher_titel WHERE isbn = '${isbn}';
        `);
	}
});
