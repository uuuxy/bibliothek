import { test, expect } from '@playwright/test';
import { uiLogin } from './helpers.js';

// Smoke-Flow Backup-Wächter: Der lokale Stack setzt bewusst KEINEN
// BACKUP_ENCRYPTION_KEY — genau der stille Ausfall, den das Badge
// unübersehbar machen muss.
test('Backup-Wächter warnt bei fehlendem Verschlüsselungs-Key', async ({ page }) => {
	await uiLogin(page);

	const alert = page.getByRole('alert');
	await expect(alert).toBeVisible();
	await expect(alert).toContainText('Backup-Verschlüsselungs-Key fehlt');
});
