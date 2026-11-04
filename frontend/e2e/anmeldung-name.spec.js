import { test, expect } from '@playwright/test';
import { seedSQL, querySQL, ADMIN_PASSWORD } from './helpers.js';

// Mit gesetzter Domain der Schule genügt der Name vor dem @: Die Seite nennt die Domain unter dem
// Feld, und der Server meldet mit der vollen Adresse an.
//
// Braucht SELBSTANMELDUNG_DOMAIN=test.local im lokalen Stack (docker-compose.local.yml).
test('Anmeldung nur mit dem Namen: Hinweis am Feld, Anmeldung mit der Adresse der Schule', async ({
	page
}) => {
	const name = `nora.name${Date.now().toString(36)}`;
	const adresse = `${name}@test.local`;
	const raeumeAuf = () =>
		seedSQL(`DELETE FROM audit_logs WHERE admin_id IN (SELECT id FROM benutzer WHERE email LIKE 'nora.name%@test.local');
		         DELETE FROM benutzer WHERE email LIKE 'nora.name%@test.local';
		         DELETE FROM leser WHERE nachname = 'Namenprobe' AND art = 'lehrkraft';`);
	raeumeAuf();
	seedSQL(`INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
	         VALUES ('Nora', 'Namenprobe', '${adresse}', 'admin', true);`);

	try {
		await page.goto('/');
		const feld = page.getByLabel('Name oder E-Mail');
		await expect(feld).toBeVisible();
		await expect(
			page.getByText('Der Name vor dem @ genügt – @test.local ergänzt das Programm.')
		).toBeVisible();

		// Wie uiLogin: erst fokussieren, dann füllen, dann prüfen.
		const passwort = page.locator('#login-password');
		await feld.click();
		await feld.fill(name);
		await passwort.click();
		await passwort.fill(ADMIN_PASSWORD);
		await expect(feld).toHaveValue(name);
		await page.getByRole('button', { name: 'Anmelden' }).click();
		await expect(page.getByRole('button', { name: 'Abmelden' })).toBeVisible();

		// Am Draht: Die Anmeldung traf das Konto mit der vollen Adresse.
		expect(
			querySQL(`SELECT count(*) FROM sitzungen s JOIN benutzer b ON b.id = s.benutzer_id
			          WHERE b.email = '${adresse}'`)
		).not.toBe('0');
	} finally {
		raeumeAuf();
	}
});
