import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix } from './helpers.js';

// Signaturen: Regal suchen (Präfix!) und das Vokabular pflegen, aus dem das
// Buchformular die Signatur vorschlägt.
//
// Beides gab es vorher nicht bedienbar: Die Signaturliste kam aus der Tabelle
// `signatures`, die kein Buch je referenzierte, und `systematik_kategorien` war
// nur lesbar — es gab keinen Weg, sie zu füllen, obwohl der Signatur-Vorschlag
// im Buchformular von ihr abhängt.
test('Signaturen: Regal per Präfix finden, Sachgruppe anlegen', async ({ page }) => {
	await uiLogin(page);
	const suffix = uniqueSuffix();
	const basis = `E2E BIB ${suffix}`;
	const kuerzel = `E2E${suffix}`.slice(0, 20);

	try {
		// Drei Titel: die Signatur selbst, ein Unterfach darunter — und ein Nachbar,
		// der dieselbe Zeichenfolge fortsetzt, aber ohne Grenze am Leerzeichen.
		seedSQL(`
            INSERT INTO buecher_titel (titel, autor, signatur) VALUES
                ('E2E-Sig-Basis-${suffix}',   'Autor A', '${basis}'),
                ('E2E-Sig-Unter-${suffix}',   'Autor B', '${basis} 5 KRUE'),
                ('E2E-Sig-Nachbar-${suffix}', 'Autor C', '${basis}X');
        `);

		await page.getByTitle('Signaturen').click();

		// Regal öffnen: über die Suche eingrenzen, dann die Signatur wählen.
		await page.getByLabel('Signatur suchen').fill(basis);
		await page.getByRole('button', { name: basis, exact: false }).first().click();

		// Das Präfix erfasst Basis UND Unterfach ...
		await expect(page.getByText(`E2E-Sig-Basis-${suffix}`)).toBeVisible();
		await expect(page.getByText(`E2E-Sig-Unter-${suffix}`)).toBeVisible();
		// ... aber NICHT den Nachbarn ohne Grenze am Leerzeichen. Ohne diese Grenze
		// würde eine Signatur-Inventur stillschweigend fremde Regale mitnehmen.
		await expect(
			page.getByText(`E2E-Sig-Nachbar-${suffix}`),
			'Nachbarsignatur ohne Leerzeichen-Grenze gehoert nicht ins Regal'
		).toHaveCount(0);

		// Vokabular pflegen: Sachgruppe anlegen — der Beleg liegt in der DB.
		await page.getByLabel('Kürzel').fill(kuerzel);
		await page.getByLabel('Bezeichnung', { exact: true }).fill(`E2E-Fach-${suffix}`);
		await page.getByRole('button', { name: 'Anlegen' }).click();

		await expect(page.getByText(`E2E-Fach-${suffix}`)).toBeVisible();
		expect(
			querySQL(`SELECT bezeichnung FROM systematik_kategorien WHERE kuerzel = '${kuerzel}'`)
		).toBe(`E2E-Fach-${suffix}`);
	} finally {
		seedSQL(`
            DELETE FROM buecher_titel WHERE titel LIKE 'E2E-Sig-%-${suffix}';
            DELETE FROM systematik_kategorien WHERE kuerzel = '${kuerzel}';
        `);
	}
});
