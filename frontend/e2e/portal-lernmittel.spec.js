import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix } from './helpers.js';

// Lehrerportal → Reiter „Lernmittel" (Betreiber-Entscheidung 24.08.2026): Das
// Kollegium sieht Bestand und Mengen je Jahrgang sowie die Klassensatz-Zuordnung —
// hinter der Anmeldung, über die Portal-Türen /api/portal/* (Anmeldung genügt,
// KEIN view_books: die Rolle hat das Recht nicht, sonst käme sie nie an die Daten).
//
// Genau diese Rechte-Konstellation macht den e2e-Lauf zur Pflicht: Der alte Weg
// (/api/books) antwortet der Rolle mit 403 — ein Unit-Test am Handler sieht das nie.
const LEHRER_EMAIL = 'e2e-lehrer@test.local';

test.describe('Lehrerportal: Lernmittel', () => {
	const s = uniqueSuffix().slice(0, 6);
	const TITEL = `LM Mathe ${s}`;
	// '7B' ist die KANONISCHE Form (der Vokabular-Trigger aus Migration 079 schreibt
	// den Buchstaben groß — ein '7b' im Seed käme als '7B' wieder heraus, und die
	// Anzeige hieße anders als die Erwartung). Die class_books-FK verlangt den
	// Eintrag in `klassen`.
	const KLASSE = '7B';

	test.beforeAll(() => {
		seedSQL(`
			INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
			VALUES ('E2E', 'Lehrer', '${LEHRER_EMAIL}', 'kollegium', true)
			ON CONFLICT (email) DO UPDATE SET aktiv = true;

			INSERT INTO klassen (name) VALUES ('${KLASSE}') ON CONFLICT DO NOTHING;

			WITH t AS (
				-- kein subject: die Spalte trägt eine FK auf systematik_kategorien (Migration 078)
				INSERT INTO buecher_titel (isbn, titel, autor, jahrgang_von, jahrgang_bis, track)
				VALUES ('978lm${s}', '${TITEL}', 'Portal Autor', 7, 7, 'Gymnasium')
				RETURNING id
			), e AS (
				INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
				SELECT t.id, 'LM-${s}-' || g, true FROM t, generate_series(1, 3) AS g
			)
			INSERT INTO class_books (class_name, book_id) SELECT '${KLASSE}', t.id FROM t;
		`);
	});

	test.afterAll(() => {
		// Exemplare zuerst (FK), dann der Titel — class_books räumt die CASCADE ab.
		seedSQL(`
			DELETE FROM buecher_exemplare WHERE barcode_id LIKE 'LM-${s}-%';
			DELETE FROM buecher_titel WHERE isbn = '978lm${s}';
		`);
	});

	test('Lehrkraft sieht Klassensatz-Zuordnung und Bestand je Jahrgang mit Mengen', async ({
		page
	}) => {
		await uiLogin(page, LEHRER_EMAIL);
		await page.getByTitle('Mein Portal').click();
		await page.getByRole('tab', { name: 'Lernmittel' }).click();

		// Klassensätze: die Klasse aufklappen, der Titel steht mit Menge darin.
		await page.getByText(`Klasse ${KLASSE}`, { exact: true }).click();
		const zeile = page.locator('li').filter({ hasText: TITEL });
		await expect(zeile).toBeVisible();
		await expect(zeile).toContainText('3/3 verfügbar');

		// Bestand nach Jahrgang: die Spanne 7–7 wird zur Gruppe „Klasse 7 Gymnasium".
		const gruppe = page.getByRole('button', { name: /Klasse 7 Gymnasium/ });
		await expect(gruppe).toBeVisible();
		await gruppe.click();
		const kachel = page.locator('h3').filter({ hasText: TITEL });
		await expect(kachel).toBeVisible();

		// Die Mengenangabe der Kachel — nicht nur „irgendwo steht 3/3".
		await expect(
			page.locator('div').filter({ has: kachel }).getByText('3/3 Stück').first()
		).toBeVisible();
	});

	test('ohne Anmeldung bleiben die Portal-Türen zu', async ({ request }) => {
		for (const pfad of ['/api/portal/lernmittel', '/api/portal/klassensaetze']) {
			const res = await request.get(pfad);
			expect(res.status(), pfad).toBe(401);
		}
	});
});
