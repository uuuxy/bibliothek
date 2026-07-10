import { test, expect } from '@playwright/test';
import { uiLogin, apiPost, seedSQL, uniqueSuffix } from './helpers.js';

// RBAC-Negativpfad: Wir testen sonst nur den voll autorisierten Admin.
// Hier der Beweis, dass die Kette Server-403 → UI-Ausblendung für
// Nicht-Admins hält und nichts leakt. Die Benutzer werden idempotent
// geseedet (Mock-IMAP akzeptiert beim Login jedes Passwort).

const MITARBEITER_EMAIL = 'e2e-mitarbeiter@test.local';
const LEHRER_EMAIL = 'e2e-lehrer@test.local';

function seedUsers() {
    seedSQL(`
        INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
        VALUES ('E2E', 'Mitarbeiter', '${MITARBEITER_EMAIL}', 'mitarbeiter', true),
               ('E2E', 'Lehrer', '${LEHRER_EMAIL}', 'lehrer', true)
        ON CONFLICT DO NOTHING;
    `);
}

test('Mitarbeiter: manage_users-Endpoints liefern 403, Admin-UI bleibt verborgen', async ({ page }) => {
    seedUsers();
    await uiLogin(page, MITARBEITER_EMAIL);

    // Mitarbeiter dürfen Schüler anlegen (create_students) — als Testobjekt
    const suffix = uniqueSuffix();
    const created = await apiPost(page, '/api/schueler', {
        vorname: 'E2E',
        nachname: `Rbac-${suffix}`,
        klasse: '6A',
        barcode_id: `S-${suffix}`,
    });
    expect(created.ok(), `Schüler-Seeding als Mitarbeiter: ${created.status()}`).toBeTruthy();
    const { id: studentId } = await created.json();

    // DSGVO-Auskunft bündelt ALLE Daten eines Kindes → nur manage_users
    const auskunft = await page.request.get(`/api/schueler/${studentId}/dsgvo-auskunft`);
    expect(auskunft.status(), 'DSGVO-Auskunft für Mitarbeiter').toBe(403);
    expect(await auskunft.text()).not.toContain(`Rbac-${suffix}`);

    // Backup-Status ist Admin-Territorium
    const backup = await page.request.get('/api/admin/system/backup-status');
    expect(backup.status(), 'Backup-Status für Mitarbeiter').toBe(403);

    // …und deshalb darf das Backup-Alert-Badge im UI nicht auftauchen
    await expect(page.getByText('Backup-Verschlüsselungs-Key fehlt')).toHaveCount(0);
});

test('Lehrer: /abgaenger direkt aufgerufen leakt keine Schülerdaten', async ({ page }) => {
    seedUsers();
    await uiLogin(page, LEHRER_EMAIL);

    // Server blockt hart (view_graduates: false)
    const api = await page.request.get('/api/abgaenger');
    expect(api.status(), 'Abgänger-API für Lehrer').toBe(403);

    // Direkter URL-Aufruf: kein Crash, keine Datenzeilen
    await page.goto('/abgaenger');
    await expect(page.locator('table')).toHaveCount(0);
    await expect(page.getByText('Barcode-ID')).toHaveCount(0);

    // Schreibende Admin-API ebenfalls dicht
    const createUser = await apiPost(page, '/api/benutzer', {
        vorname: 'Boese', nachname: 'Absicht', email: 'x@x.local', rolle: 'admin',
    });
    expect(createUser.status(), 'Benutzer anlegen als Lehrer').toBe(403);
});
