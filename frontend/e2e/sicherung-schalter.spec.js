import { test, expect } from '@playwright/test';
import { uiLogin, einstellungsKategorie, menuepunkt, schalteSicherung } from './helpers.js';

// Der Schalter der eigenen Nachtsicherung über die Oberfläche: Einstellungen → Sicherung,
// einschalten, speichern; danach führt der Server „sicherung_an“. Der Stand davor wird
// zurückgestellt, denn die Datenbank teilen sich alle Specs.
test('Einstellungen → Sicherung schaltet die Nachtsicherung ein', async ({ page }) => {
	await uiLogin(page);
	const vorher = await schalteSicherung(page, false);
	try {
		await page.getByRole('button', { name: 'System', exact: true }).click();
		await menuepunkt(page, 'Einstellungen').click();
		await einstellungsKategorie(page, 'Sicherung').click();

		const schalter = page.getByRole('switch', { name: 'Nächtliche Sicherung umschalten' });
		await expect(schalter).toHaveAttribute('aria-checked', 'false');
		await schalter.click();
		await page.getByRole('button', { name: 'Sicherung speichern' }).click();

		await expect
			.poll(async () => (await (await page.request.get('/api/einstellungen')).json()).sicherung_an)
			.toBe(true);
	} finally {
		await schalteSicherung(page, vorher);
	}
});
