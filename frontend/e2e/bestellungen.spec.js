import { test, expect } from '@playwright/test';
import { uiLogin, uniqueSuffix } from './helpers.js';

// Smoke-Flow Bestellwesen: Lieferant über die UI anlegen → erscheint in der
// Verwaltung → Berichte-Tab validiert Datumsbereiche (Zeitzonen-Fix-Umfeld).
test('Lieferant anlegen und Berichte-Validierung', async ({ page }) => {
	await uiLogin(page);

	// In den Bestellungen-Workspace
	await page.getByTitle('Bestellungen').click();

	// Sub-Tab „Lieferanten verwalten"
	await page.getByRole('button', { name: 'Lieferanten verwalten' }).click();

	const name = `E2E-Buchhandlung-${uniqueSuffix()}`;
	await page.locator('#n').fill(name);
	await page.locator('#e').fill('e2e@example.com');
	await page.locator('#c').fill('K-4711');
	await page.getByRole('button', { name: 'Lieferanten speichern' }).click();

	// Der neue Lieferant erscheint in der Verwaltung
	await expect(page.getByText(name)).toBeVisible();

	// Berichte-Tab: Lieferantenabrechnung wählen
	await page.getByRole('button', { name: 'Berichte' }).click();
	await page.getByRole('radio').nth(2).check();

	// Bewusst kein getByRole('link'): im disabled-Zustand entfällt das href,
	// und ein <a> ohne href hat keine Link-Rolle mehr.
	const download = page.locator('a').filter({ hasText: 'PDF herunterladen' });
	await expect(download).toBeVisible();
	await expect(download).not.toHaveAttribute('aria-disabled', 'true');

	// Von > Bis → Button disabled + Fehlermeldung (Client-Validierung)
	await page.locator('#von').fill('2026-07-10');
	await page.locator('#bis').fill('2026-07-01');
	await expect(page.getByText('Das Von-Datum liegt nach dem Bis-Datum.')).toBeVisible();
	await expect(download).toHaveAttribute('aria-disabled', 'true');

	// Zurück in den gültigen Bereich → wieder aktiv
	await page.locator('#bis').fill('2026-07-31');
	await expect(download).not.toHaveAttribute('aria-disabled', 'true');
});
