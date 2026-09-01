import { test, expect } from '@playwright/test';
import { uiLogin } from './helpers.js';

// Smoke-Flow: Login über die echte UI → App sichtbar → Session übersteht
// einen Reload (Boot-Restore) → Logout invalidiert auch serverseitig.
test('Login, Session-Restore nach Reload, Logout', async ({ page }) => {
	await uiLogin(page);

	// Eingeloggt: Sidebar mit Navigation und Abmelden-Knopf
	const logoutBtn = page.getByRole('button', { name: 'Abmelden' });
	await expect(logoutBtn).toBeVisible();
	await expect(page.getByRole('button', { name: 'Ausleihe' })).toBeVisible();

	// F5 darf nicht mehr ausloggen: der Boot-Restore stellt die Session
	// aus dem HttpOnly-Cookie wieder her (GET /api/auth/me).
	await page.reload();
	await expect(logoutBtn).toBeVisible();
	await expect(page.locator('#login-email')).not.toBeVisible();

	// Logout: UI zurück am Login …
	await logoutBtn.click();
	await expect(page.locator('#login-email')).toBeVisible();

	// … und die Session ist auch serverseitig tot — ein Reload darf sie
	// NICHT wiederbeleben (Token-Blacklist via POST /api/auth/logout).
	await page.reload();
	await expect(page.locator('#login-email')).toBeVisible();
	await expect(page.getByRole('button', { name: 'Abmelden' })).not.toBeVisible();
});
