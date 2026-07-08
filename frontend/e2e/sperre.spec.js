import { test, expect } from '@playwright/test';
import { uiLogin, apiPost, apiPatch, seedSQL, uniqueSuffix } from './helpers.js';

// Smoke-Flow Sperre: gesperrter Schüler löst beim Buch-Checkout den
// Block-Alert aus; „Sperre dauerhaft aufheben" (der frühere Geister-Aufruf!)
// entsperrt und die Ausleihe läuft durch.
test('Gesperrter Schüler: Block-Alert und Sperre aufheben', async ({ page }) => {
    await uiLogin(page);

    const suffix = uniqueSuffix();
    const studentBarcode = `S-${suffix}`;
    const bookBarcode = `B-${suffix}`;
    const bookTitle = `E2E-Sperrbuch-${suffix}`;

    // Schüler anlegen und manuell sperren (gleicher Endpoint wie StudentLockModal)
    const created = await apiPost(page, '/api/schueler', {
        vorname: 'E2E',
        nachname: `Gesperrt-${suffix}`,
        klasse: '7B',
        barcode_id: studentBarcode,
    });
    expect(created.ok(), `Schüler-Seeding: ${created.status()}`).toBeTruthy();
    const { id: studentId } = await created.json();

    const locked = await apiPatch(page, `/api/admin/students/${studentId}/lock`, { is_locked: true });
    expect(locked.ok(), `Sperren: ${locked.status()}`).toBeTruthy();

    // Ausleihbares Buch-Exemplar seeden (kein einfacher API-Weg vorhanden)
    seedSQL(`
        WITH t AS (INSERT INTO buecher_titel (titel) VALUES ('${bookTitle}') RETURNING id)
        INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
        SELECT id, '${bookBarcode}', true FROM t;
    `);

    // Kiosk: Schüler scannen → Konto öffnet sich
    await page.getByRole('button', { name: 'Ausleihe' }).click();
    const scanInput = page.getByPlaceholder(/scannen/i).first();
    await scanInput.fill(studentBarcode);
    await scanInput.press('Enter');
    await expect(page.getByText(`Gesperrt-${suffix}`).first()).toBeVisible();

    // Buch scannen → 403 „Manuelle Sperre" → Block-Alert
    await scanInput.fill(bookBarcode);
    await scanInput.press('Enter');
    await expect(page.getByRole('heading', { name: 'Ausleihe blockiert' })).toBeVisible();

    // Sperre dauerhaft aufheben → Alert verschwindet, Ausleihe wird nachgeholt
    await page.getByRole('button', { name: 'Sperre dauerhaft aufheben' }).click();
    await expect(page.getByRole('heading', { name: 'Ausleihe blockiert' })).not.toBeVisible();
    await expect(page.getByText(bookTitle).first()).toBeVisible();
});
