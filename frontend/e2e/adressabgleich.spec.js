import { test, expect } from '@playwright/test';
import { uiLogin, einstellungsKategorie, menuepunkt, apiPut } from './helpers.js';

// Der Zugang zum Webmail für den Adressabgleich über die Oberfläche: Einstellungen → Mail,
// Zugang speichern, Probelauf. Ein echtes Roundcube gibt es im Stack nicht; die Adresse liegt
// unter .invalid (RFC 2606), der Probelauf endet also mit dem Satz für ein unerreichbares
// Webmail. Damit ist der Weg durch Router, Verschlüsselung und Anzeige belegt, ohne dass eine
// Anfrage die Schule erreicht. Den Abgleich selbst prüfen api/system/adressabgleich_pg_test.go
// und internal/adressabgleich gegen einen Nachbau und ein echtes Roundcube.
test('Einstellungen → Mail: Zugang zum Webmail speichern und Probelauf', async ({ page }) => {
	await uiLogin(page);
	const vorher = await (await page.request.get('/api/admin/settings/adressabgleich')).json();
	try {
		await page.getByRole('button', { name: 'System', exact: true }).click();
		await menuepunkt(page, 'Einstellungen').click();
		await einstellungsKategorie(page, 'Mail').click();

		const abschnitt = page.getByRole('group', { name: 'Adressen aus dem Webmail' });
		await abschnitt.getByLabel('Webmail-Adresse').fill('http://webmail.invalid');
		await abschnitt.getByLabel('Benutzername').fill('e2e-abgleich@schule.example');
		await abschnitt.getByLabel('Passwort').fill('e2e-geheim');

		// Unverschlüsselt ginge das Passwort im Klartext durchs Netz.
		await page.getByRole('button', { name: 'Zugang speichern' }).click();
		await expect(
			page.getByText('Die Adresse des Webmails muss mit https:// beginnen')
		).toBeVisible();

		await abschnitt.getByLabel('Webmail-Adresse').fill('https://webmail.invalid/?_task=mail');
		await page.getByRole('button', { name: 'Zugang speichern' }).click();
		await expect
			.poll(
				async () =>
					(await (await page.request.get('/api/admin/settings/adressabgleich')).json()).adresse
			)
			.toBe('https://webmail.invalid');

		// Die Meldung steht dauerhaft im Abschnitt; der Toast der API-Hülle zeigt sie zusätzlich.
		await page.getByRole('button', { name: 'Probelauf' }).click();
		await expect(abschnitt.getByRole('alert')).toContainText('Webmail ist nicht erreichbar', {
			timeout: 40_000
		});
	} finally {
		await apiPut(page, '/api/admin/settings/adressabgleich', {
			adresse: vorher.adresse || '',
			benutzer: vorher.benutzer || '',
			passwort: ''
		});
	}
});
