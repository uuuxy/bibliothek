import { test, expect } from '@playwright/test';
import { uiLogin, csrfToken, gehZu, seedSQL, uniqueSuffix } from './helpers.js';

// Eine Änderung an den Einstellungen steht im Protokoll der Verwaltung, samt dem, was geändert
// wurde. Die erste Fassung dieses Tests rief zwei Adressen, die es nicht gibt, und prüfte nur, ob
// die Seite irgendeine Tabelle zeigt: Am Protokoll hätte niemand etwas gemerkt.
test('System-Logs: Administrative Aktionen werden protokolliert', async ({ page }) => {
	const marke = `E2E-Vermerk ${uniqueSuffix()}`;
	await uiLogin(page);
	const antwort = await page.request.put('/api/einstellungen', {
		headers: { 'X-CSRF-Token': await csrfToken(page) },
		data: { etikett_eigentumsvermerk: marke }
	});
	expect(antwort.ok(), `Speichern: ${antwort.status()}`).toBe(true);

	try {
		await gehZu(page, '/system-logs');
		await page.getByRole('tab', { name: 'Admin-Audit-Log' }).click();
		const zeile = page.getByRole('row').filter({ hasText: marke });
		await expect(zeile).toHaveCount(1);
		await expect(zeile).toContainText('UPDATE_SETTINGS');
	} finally {
		seedSQL(`
			DELETE FROM system_einstellungen WHERE schluessel = 'etikett_eigentumsvermerk' AND wert = '${marke}';
			DELETE FROM audit_log WHERE details::text LIKE '%${marke}%';
		`);
	}
});
