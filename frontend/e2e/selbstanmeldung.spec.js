import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, ADMIN_PASSWORD } from './helpers.js';

// Der ganze Weg einer Lehrkraft OHNE Konto (26.08.2026). Bis dahin war jedes Glied
// einzeln getestet (PG-Test für die Anlage, Unit-Test für die Meldung), aber nie die
// Kette: Erster Login → „Zugang beantragt" → die Bibliothek SIEHT den Antrag →
// Freischalten → die Lehrkraft kommt herein und sieht NUR „Mein Portal".
//
// Braucht SELBSTANMELDUNG_DOMAIN=test.local im lokalen Stack (docker-compose.local.yml).
test('Selbstanmeldung: Antrag → sichtbar → freigeschaltet → nur Mein Portal', async ({ page }) => {
	const s = Date.now().toString(36);
	const EMAIL = `erika.selbst${s}@test.local`;
	// Auch Leichen früherer Läufe: Die Zeile über der Tabelle zählt ALLE offenen Anträge.
	seedSQL(`DELETE FROM audit_logs WHERE admin_id IN (SELECT id FROM benutzer WHERE email LIKE 'erika.selbst%@test.local');
	         DELETE FROM benutzer WHERE email LIKE 'erika.selbst%@test.local';`);

	// 1. Erster Login ohne Konto: kein Zugang, aber eine Meldung, die STEHEN bleibt.
	await page.goto('/');
	// Wie uiLogin: erst fokussieren, dann füllen, dann prüfen — die Bindings rendern
	// kurz nach dem Mount, ein ungeduldiges fill landet sonst im falschen Feld.
	const email = page.locator('#login-email');
	const passwort = page.locator('#login-password');
	await email.click();
	await email.fill(EMAIL);
	await passwort.click();
	await passwort.fill(ADMIN_PASSWORD);
	await expect(email).toHaveValue(EMAIL);
	await expect(passwort).toHaveValue(ADMIN_PASSWORD);
	await page.getByRole('button', { name: 'Anmelden' }).click();
	const meldung = page.getByText('Zugang beantragt — die Bibliothek muss ihn noch freischalten');
	await expect(meldung).toBeVisible();
	await page.waitForTimeout(4500); // die 401-Meldung wäre nach 4 s weg — diese nicht
	await expect(meldung).toBeVisible();
	await expect(page.getByRole('button', { name: 'Abmelden' })).toHaveCount(0);

	// Am Draht: inaktiv, Rolle kollegium, Antrag markiert, Name aus der Adresse, Audit-Zeile.
	expect(
		querySQL(
			`SELECT aktiv || '|' || rolle || '|' || (zugang_beantragt_am IS NOT NULL) || '|' || vorname
			 FROM benutzer WHERE email = '${EMAIL}'`
		)
	).toBe('false|kollegium|true|Erika');
	expect(
		querySQL(
			`SELECT count(*) FROM audit_logs a JOIN benutzer b ON b.id = a.admin_id
			 WHERE a.aktion = 'SELBSTANMELDUNG' AND b.email = '${EMAIL}'`
		)
	).toBe('1');

	// 2. Die Bibliothek sieht den Antrag — als Zeile ÜBER der Tabelle, nicht als grauer
	//    Punkt irgendwo darin — und schaltet frei.
	await uiLogin(page);
	await page.getByRole('button', { name: 'System', exact: true }).click(); // Gruppe aufklappen
	await page.getByTitle('Benutzer & Rechte').click();
	await expect(page.getByRole('status')).toContainText(/Zugangsanfrage[\s\S]*Erika Selbst/);
	await page.getByRole('searchbox', { name: 'Benutzer suchen' }).fill(EMAIL);
	const zeile = page.locator('tr').filter({ hasText: EMAIL });
	await expect(zeile).toContainText('Zugang beantragt');
	await zeile.getByRole('button', { name: 'Bearbeiten' }).click();
	await page.getByLabel('Benutzerkonto ist aktiv').check();
	await page.getByRole('button', { name: 'Speichern' }).click();
	await expect(zeile).toContainText('Aktiv');
	await expect(page.getByRole('status')).toHaveCount(0);
	await page.getByRole('button', { name: 'Abmelden' }).click();

	// 3. Die Lehrkraft kommt herein — und sieht genau einen Menüpunkt.
	await uiLogin(page, EMAIL);
	const punkte = page.locator('nav [title]');
	await expect(punkte).toHaveCount(1);
	await expect(punkte.first()).toHaveAttribute('title', 'Mein Portal');
	await expect(
		page.getByRole('textbox', { name: 'Bücher für einen Klassensatz suchen' })
	).toBeVisible();
});
