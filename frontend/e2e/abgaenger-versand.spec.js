// Kontoauszug-Versand der Abgänger über den echten Klickpfad.
//
// WICHTIG (wie beim Mahnlauf): Der POST wird abgefangen und NIE durchgelassen. Der
// lokale Stack trägt einen echten SMTP_HOST — ein durchgereichter Lauf würde hier
// tatsächlich Kontoauszüge an die hinterlegten Klassenleitungen schicken. Geprüft
// wird der Payload, den das Frontend schicken WÜRDE; die Serverseite deckt
// api/graduates_mail_test.go ab.
import { test, expect } from '@playwright/test';
import { uiLogin } from './helpers.js';

test('Abgänger: Klassenauswahl und Override landen im Versand-Request', async ({ page }) => {
	await uiLogin(page);

	/** @type {any} */
	let gesendeterBody = null;
	await page.route('**/api/abgaenger/mail', async (route) => {
		gesendeterBody = route.request().postDataJSON();
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({ sent_count: 1, skipped_count: 0, message: 'Kontoauszüge versendet.' })
		});
	});

	await page.goto('/abgaenger');
	await page.getByRole('button', { name: /An Klassenleitungen mailen/ }).click();

	const dialog = page.getByRole('dialog');
	await expect(dialog.getByRole('heading', { name: 'Kontoauszüge versenden' })).toBeVisible();

	const senden = dialog.getByRole('button', { name: /senden$/ });
	const checkboxen = dialog.getByRole('checkbox');
	const anzahl = await checkboxen.count();
	expect(anzahl).toBeGreaterThan(0);
	await expect(senden).toContainText(`${anzahl}`);

	if (anzahl > 1) {
		await checkboxen.first().uncheck();
		await expect(senden).toContainText(`${anzahl - 1}`);
	}

	const feld = dialog.getByLabel(/Alternative Empfänger/);
	await feld.fill('sekretariat@');
	await expect(senden).toBeDisabled();

	await feld.fill('sekretariat@schule.de');
	await expect(senden).toBeEnabled();
	await senden.click();

	await expect(page.getByRole('heading', { name: 'Kontoauszüge versenden' })).toBeHidden();
	expect(gesendeterBody.override_email).toBe('sekretariat@schule.de');
	expect(gesendeterBody.klassen.length).toBe(anzahl > 1 ? anzahl - 1 : anzahl);
});
