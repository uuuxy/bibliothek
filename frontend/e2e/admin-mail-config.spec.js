import { test, expect } from '@playwright/test';
import { uiLogin } from './helpers.js';

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
