import { test, expect } from '@playwright/test';
import { uiLogin, uniqueSuffix, einstellungsKategorie } from './helpers.js';

// Smoke-Flow Bestellwesen: Lieferant über die UI anlegen → erscheint in der
// Verwaltung → Berichte-Tab validiert Datumsbereiche (Zeitzonen-Fix-Umfeld).
test('Lieferant anlegen und Berichte-Validierung', async ({ page }) => {
	await uiLogin(page);

	// Lieferanten sind seit 25.08.2026 eine Einstellungs-Kategorie, kein Reiter mehr.
	await page.goto('/einstellungen');
	await einstellungsKategorie(page, 'Lieferanten').click();

	const name = `E2E-Buchhandlung-${uniqueSuffix()}`;
	await page.locator('#n').fill(name);
	await page.locator('#e').fill('e2e@example.com');
	await page.locator('#c').fill('K-4711');
	await page.getByRole('button', { name: 'Lieferanten speichern' }).click();

	// Der neue Lieferant erscheint in der Verwaltung
	await expect(page.getByText(name)).toBeVisible();

	// Bestellberichte: Lieferantenabrechnung wählen. Seit dem 17.09.2026 ein eigener
	// Bildschirm unter „Berichte"; die Lieferanten wohnen seit ihrem Umzug in den
	// Einstellungen.
	await page.goto('/bestellberichte');
	await page.getByRole('radio').nth(2).check();

	// Frei ein Link, gesperrt ein gesperrter Knopf (ui/Button mit href).
	const download = page.getByRole('link', { name: /Als PDF/ });
	await expect(download).toBeVisible();

	// Von > Bis → Button disabled + Fehlermeldung (Client-Validierung)
	await page.locator('#von').fill('2026-07-10');
	await page.locator('#bis').fill('2026-07-01');
	await expect(page.getByText('Das Von-Datum liegt nach dem Bis-Datum')).toBeVisible();
	await expect(page.getByRole('button', { name: /Als PDF/ })).toBeDisabled();
	await expect(download).toHaveCount(0);

	// Zurück in den gültigen Bereich → wieder aktiv
	await page.locator('#bis').fill('2026-07-31');
	await expect(download).toBeVisible();
});
