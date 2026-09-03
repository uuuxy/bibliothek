import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix } from './helpers.js';

// Lehrerportal → Reiter „Schulbücher" (Peter, 03.09.2026): Die Fachsprecher wollen wissen,
// wie viele Mathebücher die Schule hat. Grundlage ist allein der Lernmittel-Schalter am
// Titel; das Fach gruppiert. Türen /api/portal/lernmittel[/export] hinter der Anmeldung,
// ohne view_books — deshalb als e2e mit der Rolle kollegium, nicht nur am Handler.
const LEHRER_EMAIL = 'e2e-lehrer-schulbuecher@test.local';

test.describe('Lehrerportal: Schulbücher je Fach', () => {
	const s = uniqueSuffix().slice(0, 6);
	const FACH = `E2E-Fach ${s}`;
	const TITEL = `Schulbuch Mathe ${s}`;

	test.beforeAll(() => {
		seedSQL(`
			INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
			VALUES ('E2E', 'Fachsprecher', '${LEHRER_EMAIL}', 'kollegium', true)
			ON CONFLICT (email) DO UPDATE SET aktiv = true;
			INSERT INTO systematik_kategorien (kuerzel, bezeichnung) VALUES ('E2EF${s}', '${FACH}');
			WITH t AS (
				INSERT INTO buecher_titel (isbn, titel, autor, subject, ist_lernmittel)
				VALUES ('978sb${s}', '${TITEL}', 'Portal Autor', '${FACH}', true)
				RETURNING id
			), f AS (
				-- Kein Lernmittel: darf in der Fach-Kachel NICHT mitzählen.
				INSERT INTO buecher_titel (isbn, titel, autor, subject, ist_lernmittel)
				VALUES ('978fh${s}', 'Freihand ${s}', 'Portal Autor', '${FACH}', false)
				RETURNING id
			)
			INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
			SELECT t.id, 'SB-${s}-' || g, true FROM t, generate_series(1, 4) AS g
			UNION ALL
			SELECT f.id, 'FH-${s}-1', true FROM f;
		`);
	});

	test.afterAll(() => {
		seedSQL(`
			DELETE FROM buecher_exemplare WHERE barcode_id LIKE 'SB-${s}-%' OR barcode_id LIKE 'FH-${s}-%';
			DELETE FROM buecher_titel WHERE isbn IN ('978sb${s}', '978fh${s}');
			DELETE FROM systematik_kategorien WHERE bezeichnung = '${FACH}';
		`);
	});

	test('Fach-Kachel zählt nur Lernmittel, Titel und Excel-Export folgen', async ({ page }) => {
		await uiLogin(page, LEHRER_EMAIL);
		await page.getByTitle('Mein Portal').click();
		await page.getByRole('tab', { name: 'Schulbücher' }).click();

		// Der Filter-Chip des Fachs: 4 Exemplare (der Freihand-Titel zählt nicht).
		const chip = page.getByRole('button', { name: new RegExp(`^${FACH}\\s+4$`) });
		await expect(chip).toBeVisible();
		await chip.click();
		await expect(page.getByTestId('schulbuecher-antwort')).toContainText(
			`${FACH} · 1 Titel · 4 Exemplare`
		);

		const titel = page.getByTestId('schulbuecher-titel').locator('h3').filter({ hasText: TITEL });
		await expect(titel).toBeVisible();
		await expect(page.getByTestId('schulbuecher-titel').locator('h3')).toHaveCount(1);

		// Export: dieselbe Sitzung, Excel-Datei mit dem Titel drin (XLSX ist ein Zip —
		// der Inhalt liegt in xl/sharedStrings.xml; hier genügt Typ, Größe und Dateiname).
		const res = await page.request.get(
			`/api/portal/lernmittel/export?fach=${encodeURIComponent(FACH)}`
		);
		expect(res.status()).toBe(200);
		expect(res.headers()['content-type']).toContain('spreadsheetml');
		expect(res.headers()['content-disposition']).toContain('schulbuecher_e2e-fach');
		expect((await res.body()).length).toBeGreaterThan(2000);
	});
});
