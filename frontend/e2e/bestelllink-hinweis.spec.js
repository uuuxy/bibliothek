import { test, expect } from '@playwright/test';
import { seedSQL, querySQL, uniqueSuffix, uiLogin } from './helpers.js';

// Der Hinweis auf die fehlende öffentliche Adresse.
//
// Der Anlass ist ein Ausfall, den niemand sehen konnte: Am 06.08.2026 ging eine Bestellung
// an den Hauptlieferanten raus, die Oberfläche meldete "erfolgreich gesendet" — aber in der
// Mail stand kein Bestätigungs-Link, weil in den Einstellungen keine öffentliche Adresse
// hinterlegt war. Der Händler bekam vier PDFs und keine Möglichkeit zu bestätigen. Am Code
// war nichts kaputt; es fehlte eine Einstellung, deren Fehlen sich nirgends zeigte.
//
// Der Test prüft deshalb BEIDE Richtungen. Nur "Hinweis erscheint" wäre kein Gate: Ein
// versehentlich immer sichtbarer Hinweis bestünde ihn genauso, und ein Hinweis, der immer
// steht, wird nach drei Tagen nicht mehr gelesen.

/** @returns {() => void} Aufräumen: alter Zustand der Einstellung zurück */
function adresseEntfernen() {
	const vorher = querySQL(
		`SELECT wert FROM system_einstellungen WHERE schluessel = 'oeffentliche_adresse'`
	);
	seedSQL(`DELETE FROM system_einstellungen WHERE schluessel = 'oeffentliche_adresse';`);

	// Der Test setzt die Adresse unterwegs selbst — ohne dieses Zurücksetzen bliebe eine
	// erfundene Adresse im lokalen Stack stehen, und der nächste Lauf prüfte sie mit.
	return () => {
		seedSQL(
			vorher === ''
				? `DELETE FROM system_einstellungen WHERE schluessel = 'oeffentliche_adresse';`
				: `INSERT INTO system_einstellungen (schluessel, wert)
				   VALUES ('oeffentliche_adresse', '${vorher}')
				   ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert;`
		);
	};
}

/** Legt einen Hauptlieferanten an (E2E-…, der globalTeardown räumt ihn ab). */
function hauptlieferantAnlegen() {
	const s = uniqueSuffix();
	seedSQL(`
		-- Erst räumen, dann setzen — wie setzeHauptlieferant im Handler. Der Teil-Index aus
		-- Migration 066 lässt nur EINEN zu.
		UPDATE lieferanten SET ist_hauptlieferant = false WHERE ist_hauptlieferant;
		INSERT INTO lieferanten (name, email, kundennummer, ist_hauptlieferant)
		VALUES ('E2E-Linkhinweis ${s}', 'e2e-${s}@example.invalid', 'K-${s}', true);
	`);
}

test('Fehlende öffentliche Adresse: das Bestellwesen sagt es vorher — und schweigt, sobald sie da ist', async ({
	page
}) => {
	const adresseZurueck = adresseEntfernen();
	hauptlieferantAnlegen();

	try {
		await uiLogin(page);
		await page.goto('/bestellungen');

		const hinweis = page.getByRole('alert').filter({ hasText: 'ohne Bestätigungs-Link' });
		await expect(hinweis).toBeVisible();
		// Ein Hinweis ohne Weg zur Behebung ist eine Sackgasse (der Test läuft als Admin).
		await expect(hinweis.getByRole('button', { name: 'Einstellungen öffnen' })).toBeVisible();

		// Gegenrichtung: Adresse hinterlegt → der Hinweis muss weg sein. Ohne diesen Teil
		// wäre ein festverdrahtetes "immer sichtbar" grün.
		seedSQL(`
			INSERT INTO system_einstellungen (schluessel, wert)
			VALUES ('oeffentliche_adresse', 'https://e2e.example.invalid')
			ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert;
		`);
		await page.goto('/bestellungen');
		await page.getByRole('tab', { name: 'Lieferanten verwalten' }).waitFor();
		await expect(hinweis).toHaveCount(0);
	} finally {
		adresseZurueck();
	}
});

// Der Reparaturweg, den der Hinweis anbietet — an der Datenbank belegt.
//
// Das Feld gibt es seit dem 04.08.2026, benutzt hat es nie jemand. Ein Eingabefeld, das
// die Oberfläche anbietet und dessen Wert das Backend still verwirft (Antwort 200,
// Einstellung weg), sähe von aussen exakt so aus wie der Zustand, der diesen Fix ausgelöst
// hat: Man trägt die Adresse ein, klickt speichern — und die Mails gehen weiter ohne Link.
test('Der Knopf im Hinweis führt zum Feld, und die eingetragene Adresse kommt in der Datenbank an', async ({
	page
}) => {
	const adresseZurueck = adresseEntfernen();
	hauptlieferantAnlegen();
	const adresse = `https://e2e-${uniqueSuffix()}.example.invalid`;

	try {
		await uiLogin(page);
		await page.goto('/bestellungen');

		await page
			.getByRole('alert')
			.filter({ hasText: 'ohne Bestätigungs-Link' })
			.getByRole('button', { name: 'Einstellungen öffnen' })
			.click();

		const feld = page.getByLabel('Öffentliche Adresse');
		await feld.waitFor();
		await feld.fill(adresse);
		await page.getByRole('button', { name: 'Globale Einstellungen speichern' }).click();
		await expect(page.getByText('Einstellungen gespeichert.')).toBeVisible();

		const gespeichert = querySQL(
			`SELECT wert FROM system_einstellungen WHERE schluessel = 'oeffentliche_adresse'`
		);
		expect(gespeichert).toBe(adresse);
	} finally {
		adresseZurueck();
	}
});
