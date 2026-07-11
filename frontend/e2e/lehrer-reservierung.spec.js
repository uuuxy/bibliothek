import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix } from './helpers.js';

// Positiv-Pfad der Klassensatz-Reservierung: bisher war nur das Abschließen
// durch den Admin getestet (klassensatz-reservierung.spec.js) — hier legt
// eine LEHRKRAFT die Anfrage im Lehrerportal an.
const LEHRER_EMAIL = 'e2e-lehrer@test.local';

test('Lehrerportal: Lehrkraft reserviert einen Klassensatz', async ({ page }) => {
    const s = uniqueSuffix();

    seedSQL(`
        INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
        VALUES ('E2E', 'Lehrer', '${LEHRER_EMAIL}', 'lehrer', true)
        ON CONFLICT (email) DO UPDATE SET aktiv = true;

        INSERT INTO buecher_titel (isbn, titel, autor)
        VALUES ('978-${s}', 'E2E Lehrerwunsch ${s}', 'Portal Autor');
    `);

    // Lehrer-Login, dann über den Menüpunkt "Mein Portal" ins Lehrerportal
    // (es ist NICHT die Startseite — auch Lehrkräfte landen zuerst im Standard-Tab).
    await uiLogin(page, LEHRER_EMAIL);
    await page.getByTitle('Mein Portal').click();
    await expect(page.getByRole('heading', { name: 'Mein Lehrerportal' })).toBeVisible();

    // Buch suchen (debounced Suchfeld)
    await page.getByPlaceholder('Titel, Autor oder ISBN suchen …').fill(`E2E Lehrerwunsch ${s}`);
    await expect(page.getByText(`E2E Lehrerwunsch ${s}`).first()).toBeVisible();

    // Reservierungs-Formular öffnen und ausfüllen
    await page.getByRole('button', { name: 'Klassensatz reservieren' }).first().click();
    await page.getByLabel('Klasse *').fill('08b');
    await page.getByLabel('Anzahl').fill('25');
    await page.getByPlaceholder(/Benötigt ab/i).fill('E2E Notiz — bitte ignorieren');
    await page.getByRole('button', { name: 'Anfrage senden' }).click();

    // Erfolgs-Feedback im Portal (die Karte zeigt das Badge "✓ Gesendet")
    await expect(page.getByText('✓ Gesendet')).toBeVisible();

    // DB-Zustand: Anfrage liegt mit Klasse und Anzahl in klassensatz_reservierungen
    const row = querySQL(`
        SELECT r.klasse || '|' || r.anzahl
        FROM klassensatz_reservierungen r
        JOIN buecher_titel t ON t.id = r.titel_id
        WHERE t.isbn = '978-${s}'
    `);
    expect(row).toBe('08b|25');
});
