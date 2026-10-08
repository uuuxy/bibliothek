import { test, expect } from '@playwright/test';
import {
	uiLogin,
	apiPost,
	csrfToken,
	querySQL,
	seedSQL,
	uniqueSuffix,
	menuepunkt
} from './helpers.js';

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
		await menuepunkt(page, 'Medienkatalog').click();
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

// Gate zur schlanken Katalogliste: GET /api/books liefert erweiterteEigenschaften leer, und
// das Bearbeiten-Formular schickt per PUT das ganze Objekt zurück. Würde das Formular aus der
// Listenzeile befüllt, leerte „Bearbeiten → Speichern“ die Eigenschaften still. Das Formular
// muss deshalb vom Einzel-Read (/api/books/{id}) befüllt werden. Der Test geht den Klickpfad:
// öffnen, nichts ändern, speichern, und liest die Eigenschaften danach in der Tabelle.
test('Bücher: Bearbeiten ohne Änderung erhält die erweiterten Eigenschaften', async ({ page }) => {
	await uiLogin(page);
	const suffix = uniqueSuffix();
	const isbn = `9782${String(Date.now()).slice(-9)}`;
	const titel = `E2E-Blank-Buch-${suffix}`;

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
			erweiterteEigenschaften: { regal: `R-${suffix}`, notiz: `N-${suffix}` }
		});
		expect(created.ok(), `Buch anlegen: ${created.status()}`).toBeTruthy();

		await menuepunkt(page, 'Medienkatalog').click();
		await page.getByRole('tab', { name: 'Titel-Verwaltung' }).click();
		const suche = page.getByRole('searchbox', { name: 'Bücher durchsuchen' });
		await expect(suche).toBeVisible({ timeout: 15000 });
		await suche.fill(titel);
		await page.getByText(titel).first().click();

		// Die Maske ist geladen, sobald die Signatur dasteht.
		await expect(page.locator('#buch-signatur')).toHaveValue('E2E SIG', { timeout: 15000 });

		await page.getByRole('button', { name: 'Speichern' }).click();
		await expect(page.getByText('Buch erfolgreich gespeichert!')).toBeVisible({
			timeout: 15000
		});

		expect(
			querySQL(
				`SELECT erweiterte_eigenschaften->>'notiz' FROM buecher_titel WHERE isbn = '${isbn}'`
			)
		).toBe(`N-${suffix}`);
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
