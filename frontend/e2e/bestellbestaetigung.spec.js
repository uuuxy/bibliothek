import { test, expect } from '@playwright/test';
import { seedSQL, querySQL, uniqueSuffix, uiLogin } from './helpers.js';

// Der Bestätigungs-Link, den der Lieferant aus der Bestellmail öffnet.
//
// Warum als E2E und nicht nur als Go-Test: Die entscheidende Eigenschaft dieser Seite
// ist, dass sie OHNE ANMELDUNG funktioniert. Ein Go-Test ruft den Handler direkt auf und
// kann das gar nicht bemerken — käme versehentlich ein Auth-Wächter davor oder liefe der
// Pfad in die Login-Maske, bliebe er trotzdem grün, und der Lieferant stünde vor einem
// Anmeldebildschirm, für den er kein Konto hat.
//
// Der Test meldet sich deshalb bewusst NIE an.

/** Legt Lieferant + Bestellung samt Token an und liefert den Klartext-Token zurück. */
function seedBestellungMitLink(token, { gueltigTage = 30 } = {}) {
	const s = uniqueSuffix();
	seedSQL(`
		WITH l AS (
			INSERT INTO lieferanten (name, email, kundennummer, bietet_bestellbestaetigung)
			VALUES ('E2E-Naacher ${s}', 'e2e-${s}@example.invalid', 'K-${s}', true)
			RETURNING id
		),
		t AS (
			-- isbn bewusst NULL statt '': Die Spalte ist UNIQUE, und der Leerstring IST ein
			-- Wert — zwei Seeds ohne ISBN kollidierten. Der Produktivcode schreibt an dieser
			-- Stelle ebenfalls NULL (NULLIF in UpsertBookTitle).
			INSERT INTO buecher_titel (titel, autor, isbn, signatur)
			VALUES ('E2E-Bestaetigung ${s}', 'Autor', NULL, 'LMF-E2E ${s}')
			RETURNING id
		),
		b AS (
			INSERT INTO bestellungen_verlauf
				(lieferant_id, lieferant_name, lieferant_email, kundennummer, anzahl_exemplare,
				 bestaetigungs_token_hash, token_gueltig_bis)
			SELECT l.id, 'E2E-Naacher ${s}', 'e2e-${s}@example.invalid', 'K-${s}', 2,
			       encode(sha256('${token}'::bytea), 'hex'),
			       now() + make_interval(days => ${gueltigTage})
			FROM l RETURNING id
		),
		p AS (
			INSERT INTO bestellungen_positionen
				(bestellung_id, titel_id, titel_name, isbn, menge, einzelpreis, mit_vorab_barcode)
			SELECT b.id, t.id, 'E2E-Bestaetigung ${s}', '', 2, 0, true FROM b, t
			RETURNING bestellung_id
		)
		INSERT INTO buecher_exemplare (titel_id, barcode_id, bestellung_id, ist_ausleihbar)
		SELECT t.id, 'E2E-BC-${s}-' || g, b.id, false FROM t, b, generate_series(1,2) g;
	`);
	return s;
}

test('Lieferant öffnet den Link ohne Anmeldung, druckt Etiketten und bestätigt — genau einmal', async ({
	page
}) => {
	const token = `E2E-TOKEN-${uniqueSuffix()}`;
	const s = seedBestellungMitLink(token);

	await page.goto(`/bestellung/${token}`);

	// Kein Login-Bildschirm, sondern die Bestellung.
	await expect(page.getByRole('heading', { name: /Bestellung vom/ })).toBeVisible();
	await expect(page.getByText(`E2E-Bestaetigung ${s}`)).toBeVisible();
	await expect(page.getByRole('button', { name: /Anmelden|Login/ })).toHaveCount(0);

	// Die Etiketten sind abrufbar — geprüft am echten PDF, nicht am Knopf.
	const antwort = await page.request.get(`/api/public/bestellung/${token}/etiketten/gross`);
	expect(antwort.status()).toBe(200);
	expect(antwort.headers()['content-type']).toContain('application/pdf');

	await page.getByRole('button', { name: 'Bestellung jetzt bestätigen' }).click();
	await expect(page.getByRole('heading', { name: 'Bestellung bestätigt' })).toBeVisible();

	// BEWEIS in der Datenbank: Die Bibliothek sieht, dass der LIEFERANT bestätigt hat —
	// nicht jemand aus dem Haus.
	const durch = querySQL(
		`SELECT bestaetigt_durch FROM bestellungen_verlauf WHERE kundennummer = 'K-${s}';`
	);
	expect(durch.trim()).toBe('lieferant');

	// Nach dem Neuladen bleibt es bei der Quittung — kein zweiter Bestätigen-Knopf.
	await page.reload();
	await expect(page.getByRole('button', { name: 'Bestellung jetzt bestätigen' })).toHaveCount(0);
});

test('Ein ungültiger Link zeigt keine Daten und keinen Anmeldebildschirm', async ({ page }) => {
	await page.goto('/bestellung/DIESEN-TOKEN-GIBT-ES-NICHT');

	await expect(page.getByText('Dieser Link ist nicht mehr gültig')).toBeVisible();
	await expect(page.getByRole('heading', { name: /Bestellung vom/ })).toHaveCount(0);
});

test('Ein abgelaufener Link ist tot — auch wenn die Bestellung existiert', async ({ page }) => {
	const token = `E2E-ALT-${uniqueSuffix()}`;
	seedBestellungMitLink(token, { gueltigTage: -1 });

	await page.goto(`/bestellung/${token}`);
	await expect(page.getByText('Dieser Link ist nicht mehr gültig')).toBeVisible();
});

// Der Kreis schliesst sich erst hier: Was nützt die Bestätigung des Händlers, wenn sie im
// Haus niemand sieht? Vorher stand der Status IN der Lieferantenspalte, die truncate
// trägt — das Chip war auf wenige Pixel zerquetscht und praktisch unlesbar.
test('Die Bestellung erscheint im Haus als bestätigt — ohne die Zeile aufzuklappen', async ({
	page
}) => {
	const token = `E2E-SICHT-${uniqueSuffix()}`;
	const s = seedBestellungMitLink(token);

	// Der Händler bestätigt über seinen Link.
	await page.goto(`/bestellung/${token}`);
	await page.getByRole('button', { name: 'Bestellung jetzt bestätigen' }).click();
	await expect(page.getByRole('heading', { name: 'Bestellung bestätigt' })).toBeVisible();

	// Und die Bibliothek sieht es in der Bestellhistorie, in der eingeklappten Zeile.
	await uiLogin(page);
	await page.getByTitle('Bestellungen').click();
	await page.getByRole('button', { name: 'Bestellhistorie', exact: true }).click();

	const zeile = page.locator('tbody tr', { hasText: `K-${s}` }).first();
	await expect(zeile).toContainText('Bestätigt');
	await expect(zeile.getByText('Wartet auf Händler')).toHaveCount(0);
});
