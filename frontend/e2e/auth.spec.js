import { test, expect } from '@playwright/test';
import { uiLogin } from './helpers.js';

// Smoke-Flow: Login über die echte UI → App sichtbar → Logout.
test('Login und Logout über die UI', async ({ page }) => {
    await uiLogin(page);

    // Eingeloggt: Sidebar mit Navigation und Abmelden-Knopf
    const logoutBtn = page.getByRole('button', { name: 'Abmelden' });
    await expect(logoutBtn).toBeVisible();
    await expect(page.getByRole('button', { name: 'Ausleihe' })).toBeVisible();

    await logoutBtn.click();

    // Wieder ausgeloggt: Login-Formular ist zurück
    await expect(page.locator('#login-email')).toBeVisible();
});
