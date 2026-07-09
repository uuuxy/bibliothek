import { test, expect } from '@playwright/test';
import { uiLogin, uniqueSuffix } from './helpers.js';

test('LUSD-Import: Preview und Ausführung', async ({ page }) => {
    await uiLogin(page);

    // 1. Navigation zu Einstellungen -> Datenverwaltung
    await page.getByRole('button', { name: 'System', exact: true }).click();
    await page.getByRole('button', { name: 'Einstellungen' }).click();
    await page.getByRole('button', { name: 'Datenverwaltung' }).click();

    // 2. CSV generieren (1 neuer Schüler)
    const s = uniqueSuffix();
    const csvContent = `lusd_id,vorname,nachname,klasse,geburtsdatum\nLUSD_NEW_${s},Neu_${s},Schueler_${s},05a,01.01.2015`;

    // 3. Datei-Upload simulieren via FileChooser
    const fileChooserPromise = page.waitForEvent('filechooser');
    await page.getByText('LUSD-CSV auswählen').click();
    const fileChooser = await fileChooserPromise;
    await fileChooser.setFiles({
        name: 'lusd_test.csv',
        mimeType: 'text/csv',
        buffer: Buffer.from(csvContent)
    });

    // 4. Vorschau berechnen
    await page.getByRole('button', { name: 'Vorschau laden' }).click();

    // Warten auf das Ergebnis oder Fehlermeldung
    try {
        await expect(page.getByText('Neue Schüler')).toBeVisible({ timeout: 2000 });
    } catch (e) {
        const bodyText = await page.locator('body').innerText();
        console.log("Body text on failure:", bodyText);
        throw e;
    }

    // Akkordeon öffnen um den Namen zu sehen
    await page.locator('summary').filter({ hasText: 'Neue Schüler' }).click();

    // Verifikation der Vorschau (Akkordeon)
    await expect(page.getByText('Import abgeschlossen')).not.toBeVisible();
    await expect(page.getByText(`Neu_${s}`)).toBeVisible();

    // 5. Finalisieren
    await page.getByRole('button', { name: 'Import finalisieren' }).click();

    // 6. Bestätigen (falls die Massenabgang-Bremse greift, weil die DB mehr als 10 Schüler hat und diese nicht im CSV sind)
    // Wir fangen das ab, indem wir prüfen, ob der Override-Button erscheint.
    const overrideButton = page.getByRole('button', { name: 'Ja, Import trotz hoher Abgängerquote erzwingen' });
    if (await overrideButton.isVisible()) {
        await overrideButton.click();
    }

    // 7. Erfolg verifizieren
    await expect(page.getByText('LUSD-Import erfolgreich übernommen.')).toBeVisible();
});
