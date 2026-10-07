import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix, gehZu } from './helpers.js';

// Die Maske „Benutzer bearbeiten" füllt sich aus der Zeile der Liste, und die lädt beim Öffnen
// der Seite. Ändert ein anderer Platz inzwischen Rolle oder „aktiv", schrieb das Speichern den
// alten Stand zurück. Die Maske schickt nur, was sie seit dem Öffnen geändert hat.
test('Benutzer bearbeiten: das Speichern überschreibt nicht, was inzwischen ein anderer Platz geändert hat', async ({
	page
}) => {
	const s = uniqueSuffix();
	const email = `e2e-maske-${s}@example.invalid`;
	seedSQL(`
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Maske${s}', 'Vorher', '${email}', 'mitarbeiter', true);
	`);
	const stand = () =>
		querySQL(
			`SELECT nachname || '|' || rolle::text || '|' || aktiv FROM benutzer WHERE email = '${email}'`
		);
	try {
		await uiLogin(page);
		await gehZu(page, '/berechtigungen');
		await page.getByLabel('Benutzer suchen').fill(`Maske${s}`);
		const zeile = page.getByRole('row').filter({ hasText: email });
		await zeile.getByRole('button', { name: 'Bearbeiten' }).click();
		const nachname = page.getByLabel('Nachname');
		await expect(nachname).toHaveValue('Vorher');

		// Ein anderer Platz stuft das Konto herab und schaltet es ab, während die Maske offen ist.
		seedSQL(`UPDATE benutzer SET rolle = 'helfer', aktiv = false WHERE email = '${email}';`);

		const gesendet = page.waitForRequest(
			(r) => r.method() === 'PUT' && /\/api\/benutzer\//.test(r.url())
		);
		await nachname.fill('Nachher');
		await page.getByRole('button', { name: 'Speichern' }).click();
		expect(
			(await gesendet).postDataJSON(),
			'Die Maske schickt mehr als das geänderte Feld'
		).toEqual({ nachname: 'Nachher' });
		await expect(page.getByText('Benutzer erfolgreich aktualisiert.')).toBeVisible();

		expect(
			stand(),
			'Das Speichern des Nachnamens hat Rolle oder „aktiv" des anderen Platzes zurückgeschrieben'
		).toBe('Nachher|helfer|false');
	} finally {
		seedSQL(`DELETE FROM benutzer WHERE email = '${email}';`);
	}
});
