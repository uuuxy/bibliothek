import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix } from './helpers.js';

// Mehrplatz-Betrieb (bis zu 10 PCs): Das System verspricht per SSE, dass
// alle Plätze denselben Stand sehen. Zwei echte Browser-Kontexte:
// PC B hat das Schülerkonto offen, PC A bucht die Rückgabe — B muss die
// Änderung OHNE Reload sehen (Omnibox lauscht auf action-Events).
test('Livesync: Rückgabe an PC A aktualisiert das offene Konto an PC B', async ({ browser }) => {
    const ctxA = await browser.newContext();
    const ctxB = await browser.newContext();
    const pageA = await ctxA.newPage();
    const pageB = await ctxB.newPage();

    try {
        const suffix = uniqueSuffix();
        seedSQL(`
            WITH s AS (
                INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr)
                VALUES ('E2E', 'Sync-${suffix}', '7B', 'S-${suffix}', 2030) RETURNING id
            ), t AS (
                INSERT INTO buecher_titel (titel) VALUES ('E2E-Syncbuch-${suffix}') RETURNING id
            ), e AS (
                INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
                SELECT id, 'B-${suffix}', true FROM t RETURNING id
            )
            INSERT INTO ausleihen (exemplar_id, schueler_id, bearbeiter_id, ausgeliehen_am, rueckgabe_frist)
            SELECT e.id, s.id, (SELECT id FROM benutzer ORDER BY erstellt_am LIMIT 1),
                   NOW(), NOW() + INTERVAL '14 days' FROM e, s;
        `);

        await uiLogin(pageA);
        await uiLogin(pageB);

        // PC B: Konto öffnen, Buch ist sichtbar
        await pageB.getByTitle('Ausleihe').click();
        const scanB = pageB.getByPlaceholder(/scannen/i).first();
        await scanB.fill(`S-${suffix}`);
        await scanB.press('Enter');
        await expect(pageB.getByText(`E2E-Syncbuch-${suffix}`).first()).toBeVisible();

        // PC A: Rückgabe buchen
        await pageA.getByTitle('Ausleihe').click();
        const scanA = pageA.getByPlaceholder(/scannen/i).first();
        await scanA.fill(`B-${suffix}`);
        await scanA.press('Enter');
        await expect(pageA.getByText(`„E2E-Syncbuch-${suffix}" erfolgreich zurückgegeben.`)).toBeVisible();

        // PC B: Buch verschwindet OHNE Reload (SSE → reloadProfile)
        await expect(pageB.getByText(`E2E-Syncbuch-${suffix}`)).not.toBeVisible({ timeout: 10000 });
    } finally {
        await ctxA.close();
        await ctxB.close();
    }
});
