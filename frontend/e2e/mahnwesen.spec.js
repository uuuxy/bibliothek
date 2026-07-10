import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix } from './helpers.js';

// Mahnwesen: überfällige Ausleihe erscheint in der Übersicht, und die
// Mahnliste kommt als echtes PDF (Smoke-Assert statt visueller Prüfung).
test('Mahnwesen: überfälliger Schüler erscheint, Mahnliste-PDF antwortet', async ({ page }) => {
    await uiLogin(page);
    const suffix = uniqueSuffix();

    seedSQL(`
        WITH s AS (
            INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr)
            VALUES ('E2E', 'Saeumig-${suffix}', '8M', 'S-${suffix}', 2030) RETURNING id
        ), t AS (
            INSERT INTO buecher_titel (titel) VALUES ('E2E-Mahnbuch-${suffix}') RETURNING id
        ), e AS (
            INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
            SELECT id, 'B-${suffix}', true FROM t RETURNING id
        )
        INSERT INTO ausleihen (exemplar_id, schueler_id, bearbeiter_id, ausgeliehen_am, rueckgabe_frist)
        SELECT e.id, s.id, (SELECT id FROM benutzer ORDER BY erstellt_am LIMIT 1), NOW() - INTERVAL '30 days', NOW() - INTERVAL '10 days' FROM e, s;
    `);

    await page.getByTitle('Mahnwesen').click();
    // Die Tabelle listet Schüler (Medien nur als Zähler, keine Titel)
    const zeile = page.getByRole('row', { name: new RegExp(`Saeumig-${suffix}`) });
    await expect(zeile).toBeVisible();
    await expect(zeile).toContainText('8M');

    // PDF-Smoke: Status, Content-Type und nicht-leerer Body genügen —
    // visuelle PDF-Prüfung lohnt den Wartungsaufwand nicht.
    const pdf = await page.request.get('/api/mahnwesen/pdf');
    expect(pdf.status(), 'Mahnliste-PDF Status').toBe(200);
    expect(pdf.headers()['content-type']).toContain('application/pdf');
    expect((await pdf.body()).length).toBeGreaterThan(1000);
});
