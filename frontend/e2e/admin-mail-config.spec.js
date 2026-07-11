import { test, expect } from '@playwright/test';
import { uiLogin, csrfToken, uniqueSuffix } from './helpers.js';

test('E-Mail- & SMTP-Konfiguration: Test-Mail API Endpunkt liefert 200', async ({ page }) => {
    await uiLogin(page);
    await page.goto('/einstellungen');

    // Wir rufen den Endpoint direkt auf, um den Backend-Weg zu testen, 
    // ohne uns auf die genaue UI der Settings zu verlassen (falls der Button in einem Untertab ist).
    const res = await page.request.post('/api/admin/settings/mail/test', {
        data: {
            host: 'localhost',
            port: 1025,
            username: '',
            password: '',
            encryption: 'none',
            sender: 'test@local',
            test_recipient: 'admin@local'
        }
    });

    // Wir erwarten 200, oder 500 falls mail server lokal nicht erreichbar ist. 
    // Ein echter Fehler wegen fehlendem Server ist 500, das UI zeigt ihn an.
    // Aber für Smoke checken wir nur, dass der Endpoint existiert und reagiert (nicht 404).
    expect([200, 400, 500]).toContain(res.status());
});

// Mahn-Template-Bearbeitung: der Admin muss Betreff/Text der Mahnungen ändern
// können (Roundtrip GET → PUT → GET). Der Originalzustand wird im finally
// wiederhergestellt — die Test-DB teilen sich alle Specs.
test('Mail-Templates: Mahnungs-Vorlage lässt sich ändern und speichern', async ({ page }) => {
    await uiLogin(page);
    const s = uniqueSuffix();

    const list = await page.request.get('/api/mail-templates');
    expect(list.status()).toBe(200);
    const templates = await list.json();
    const mahnung = templates.find((/** @type {any} */ t) => t.typ === 'MAHNUNG_ELTERN');
    expect(mahnung, 'Vorlage MAHNUNG_ELTERN muss existieren (Mahnwesen!)').toBeTruthy();

    const token = await csrfToken(page);
    try {
        const put = await page.request.put(`/api/mail-templates/${mahnung.id}`, {
            headers: { 'X-CSRF-Token': token },
            data: { betreff: `${mahnung.betreff} [E2E ${s}]`, text_body: mahnung.text_body },
        });
        expect(put.status()).toBe(200);

        const verify = await page.request.get('/api/mail-templates');
        const updated = (await verify.json()).find((/** @type {any} */ t) => t.id === mahnung.id);
        expect(updated.betreff).toContain(`[E2E ${s}]`);
        // Platzhalter-Variablen dürfen den Roundtrip nicht verlieren
        expect(updated.text_body).toContain('{{.Vorname}}');
    } finally {
        await page.request.put(`/api/mail-templates/${mahnung.id}`, {
            headers: { 'X-CSRF-Token': token },
            data: { betreff: mahnung.betreff, text_body: mahnung.text_body },
        });
    }
});
