import { test, expect } from '@playwright/test';
import { uiLogin, apiPost, seedSQL, uniqueSuffix } from './helpers.js';

/**
 * Seedet einen Schüler mit aktiver Ausleihe direkt in der DB —
 * deterministischer Ausgangszustand ohne UI-Umwege.
 */
function seedStudentWithLoan(suffix) {
    seedSQL(`
        WITH s AS (
            INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr)
            VALUES ('E2E', 'Leiher-${suffix}', '7B', 'S-${suffix}', 2030) RETURNING id
        ), t AS (
            INSERT INTO buecher_titel (titel) VALUES ('E2E-Rueckgabebuch-${suffix}') RETURNING id
        ), e AS (
            INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
            SELECT id, 'B-${suffix}', true FROM t RETURNING id
        )
        INSERT INTO ausleihen (exemplar_id, schueler_id, bearbeiter_id, ausgeliehen_am, rueckgabe_frist)
        SELECT e.id, s.id, (SELECT id FROM benutzer ORDER BY erstellt_am LIMIT 1), NOW(), NOW() + INTERVAL '14 days' FROM e, s;
    `);
    return { bookBarcode: `B-${suffix}`, bookTitle: `E2E-Rueckgabebuch-${suffix}` };
}

// Der häufigste Alltagsfluss: entliehenes Buch ohne aktive Sitzung scannen
// → Rückgabe wird verbucht.
test('Rückgabe: entliehenes Buch scannen bucht es zurück', async ({ page }) => {
    await uiLogin(page);
    const { bookBarcode, bookTitle } = seedStudentWithLoan(uniqueSuffix());

    await page.getByTitle('Ausleihe').click();
    const scanInput = page.getByPlaceholder(/scannen/i).first();
    await scanInput.fill(bookBarcode);
    await scanInput.press('Enter');

    await expect(page.getByText(`„${bookTitle}" erfolgreich zurückgegeben.`)).toBeVisible();
});

// Fremdrückgabe: Buch von Schüler A wird in der Sitzung von Schüler B gescannt
// → automatische Ausbuchung bei A, Neuausleihe an B, unübersehbare Warnung.
test('Fremdrückgabe: Scan in fremder Sitzung bucht um und warnt', async ({ page }) => {
    await uiLogin(page);
    const suffix = uniqueSuffix();
    const { bookBarcode, bookTitle } = seedStudentWithLoan(suffix);

    // Schüler B anlegen und dessen Konto öffnen
    const created = await apiPost(page, '/api/schueler', {
        vorname: 'E2E',
        nachname: `Zweitleiher-${suffix}`,
        klasse: '7C',
        barcode_id: `S-Z${suffix}`,
    });
    expect(created.ok(), `Schüler-Seeding: ${created.status()}`).toBeTruthy();

    await page.getByTitle('Ausleihe').click();
    const scanInput = page.getByPlaceholder(/scannen/i).first();
    await scanInput.fill(`S-Z${suffix}`);
    await scanInput.press('Enter');
    await expect(page.getByText(`Zweitleiher-${suffix}`).first()).toBeVisible();

    // Buch von Schüler A in Bs Sitzung scannen
    await scanInput.fill(bookBarcode);
    await scanInput.press('Enter');

    await expect(page.getByText(/Fremdrückgabe erfolgt \(Vorbesitzer: E2E Leiher-/)).toBeVisible();
    // Das Buch hängt jetzt an B
    await expect(page.getByText(bookTitle).first()).toBeVisible();
});
