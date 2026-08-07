// Der Mahnlauf-Dialog über den echten Pfad — Klick für Klick, wie das Personal ihn bedient.
//
// WICHTIG: Der POST wird abgefangen und NIE an das Backend durchgelassen. Der lokale
// Stack trägt in der .env einen echten SMTP_HOST; ein durchgereichter Versand würde
// hier tatsächlich Mahnungen verschicken. Geprüft wird deshalb genau das, was der
// Dialog dem Server sagen WÜRDE — die Serverseite deckt api/mahnwesen_bulk_mail_test.go ab.
import { test, expect } from '@playwright/test';
import { uiLogin } from './helpers.js';

test('Mahnlauf: Auswahl und Override-Adresse landen im Request', async ({ page }) => {
	await uiLogin(page);

	/** @type {any} */
	let gesendeterBody = null;
	await page.route('**/api/mail/send-bulk-overdue', async (route) => {
		gesendeterBody = route.request().postDataJSON();
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({
				sent_count: 1,
				skipped_count: 0,
				message: 'an sekretariat@schule.de versendet'
			})
		});
	});

	await page.goto('/mahnwesen');
	await page
		.getByRole('button', {
			name: /Alle anmahnen – Mahnlauf konfigurieren und per E-Mail versenden/
		})
		.first()
		.click();

	// Auf den Dialog eingegrenzt: Die Mahnwesen-Tabelle dahinter bringt eigene
	// Checkboxen mit, die sonst mitgezählt (und mit abgewählt) würden.
	const dialog = page.getByRole('dialog');
	await expect(dialog.getByRole('heading', { name: 'Mahnlauf konfigurieren' })).toBeVisible();

	// Beim Öffnen ist alles vorgewählt: die Beschriftung nennt die Zahl der Klassen.
	const senden = dialog.getByRole('button', { name: /anmahnen$/ });
	const checkboxen = dialog.getByRole('checkbox');
	const anzahl = await checkboxen.count();
	expect(anzahl).toBeGreaterThan(0);
	await expect(senden).toContainText(`${anzahl}`);

	// Abwählen muss sich sofort in der Beschriftung niederschlagen — der Knopf ist
	// die einzige Stelle, an der ablesbar ist, wie viele Klassen der Lauf trifft.
	if (anzahl > 1) {
		await checkboxen.first().uncheck();
		await expect(senden).toContainText(`${anzahl - 1}`);
	}

	// Tippfehler in der Adresse sperren den Versand, statt ihn ins Leere laufen zu lassen.
	const feld = dialog.getByLabel(/Alternative Empfänger/);
	await feld.fill('sekretariat@');
	await expect(senden).toBeDisabled();

	await feld.fill('sekretariat@schule.de');
	await expect(senden).toBeEnabled();
	await senden.click();

	await expect(page.getByRole('heading', { name: 'Mahnlauf konfigurieren' })).toBeHidden();

	expect(gesendeterBody).not.toBeNull();
	expect(gesendeterBody.override_email).toBe('sekretariat@schule.de');
	expect(Array.isArray(gesendeterBody.klassen)).toBe(true);
	expect(gesendeterBody.klassen).toHaveLength(anzahl > 1 ? anzahl - 1 : anzahl);
});

test('Mahnlauf: Abbrechen schickt nichts und vergisst die Eingaben', async ({ page }) => {
	await uiLogin(page);

	let posted = false;
	await page.route('**/api/mail/send-bulk-overdue', async (route) => {
		posted = true;
		await route.abort();
	});

	await page.goto('/mahnwesen');
	await page
		.getByRole('button', {
			name: /Alle anmahnen – Mahnlauf konfigurieren und per E-Mail versenden/
		})
		.first()
		.click();

	const dialog = page.getByRole('dialog');
	const checkboxen = dialog.getByRole('checkbox');
	const anzahl = await checkboxen.count();
	await checkboxen.first().uncheck();
	await dialog.getByLabel(/Alternative Empfänger/).fill('vertretung@schule.de');
	await dialog.getByRole('button', { name: 'Abbrechen' }).click();

	await expect(page.getByRole('heading', { name: 'Mahnlauf konfigurieren' })).toBeHidden();
	expect(posted).toBe(false);

	// Zweiter Anlauf: Der Dialog bleibt gemountet — ohne Reset trüge er die Abwahl und
	// die fremde Adresse in den nächsten Lauf und verschickte still an jemand anderen.
	await page
		.getByRole('button', {
			name: /Alle anmahnen – Mahnlauf konfigurieren und per E-Mail versenden/
		})
		.first()
		.click();
	await expect(dialog.getByRole('button', { name: /anmahnen$/ })).toContainText(`${anzahl}`);
	await expect(dialog.getByLabel(/Alternative Empfänger/)).toHaveValue('');
});
