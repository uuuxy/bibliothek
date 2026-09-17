import { test, expect } from '@playwright/test';
import { uiLogin, seedBenutzer } from './helpers.js';

// Rolle Leitung (Migration 121/122): alles außer den zwei Türen der Systempflege.
//
// Warum als e2e und nicht nur am Handler: Die Rolle ist erst dann gebaut, wenn sie einen
// Menschen an den richtigen Bildschirm bringt. Dazwischen liegen vier Stellen, von denen
// jede einzeln grün sein kann, während das Ganze nicht funktioniert — der ENUM-Wert, die
// Rechte in role_permissions, die Weiche nach der Anmeldung und die Sichtbarkeitsregel
// des Menüs. Genau so war die Rolle Helfer einmal „fertig": im Server vollständig, in der
// Oberfläche unerreichbar.
//
// Der Nachweis läuft über die ECHTE Anmeldung (Mock-IMAP im lokalen Stack nimmt jedes
// Passwort), nicht über einen gesetzten Cookie: Die Weiche, die hier geprüft wird, läuft
// nur im Login-Pfad.
const LEITUNG_EMAIL = 'e2e-leitung@test.local';

test.describe('Rolle Leitung', () => {
	// Idempotent und ohne Aufräumen — wie bei den anderen Rollen-Specs: Die Anmeldung
	// schreibt Zeilen ins Logbuch, die auf das Konto zeigen. Ein Löschen in afterAll
	// wäre entweder blockiert oder müsste die Spur der Anmeldung mitnehmen.
	test.beforeAll(() => seedBenutzer(LEITUNG_EMAIL, 'leitung'));

	test('führt die Bibliothek, pflegt aber nicht das System', async ({ page }) => {
		await uiLogin(page, LEITUNG_EMAIL);

		// Verwaltung statt Portal. Das ist der Teil, den die Login-Weiche entscheidet:
		// Solange sie „admin oder mitarbeiter" aufzählte, wäre eine Leitung hier im
		// Kollegiums-Portal gelandet — ohne Fehlermeldung, einfach mit dem falschen
		// Bildschirm.
		await expect(page.getByTitle('Leserdatei')).toBeVisible();
		await expect(page.getByTitle('Mahnwesen')).toBeVisible();
		await expect(page.getByTitle('Medienkatalog')).toBeVisible();

		// „Statistiken" steht seit dem 17.09.2026 in der Sektion „Berichte" und ist damit
		// OHNE Aufklappen sichtbar — deshalb steht die Prüfung vor dem Klick auf „System".
		await expect(page.getByTitle('Statistiken')).toBeVisible();

		// Die Gruppe „System" ist zugeklappt (so gebaut) — dort stehen die übrigen Punkte,
		// um die es hier geht. Also aufklappen und DANN hinsehen: Die Prüfung soll den
		// Unterschied zwischen „darf nicht" und „ist gerade zugeklappt" nicht verwischen.
		await page.getByRole('button', { name: 'System', exact: true }).click();

		// Das Logbuch gehört zur Führung der Bibliothek (audit_logs) und unterscheidet
		// die Leitung vom Mitarbeiter, der es ab Werk nicht sieht.
		await expect(page.getByTitle('System-Logs')).toBeVisible();

		// „Benutzer & Rechte" bleibt weg — der Punkt hängt an genau einem Recht.
		await expect(page.getByTitle('Benutzer & Rechte')).toHaveCount(0);

		// „Einstellungen" bleibt SICHTBAR, und das ist richtig: Der Punkt ist ein
		// Sammelpunkt über sechs Kategorien mit verschiedenen Rechten (LUSD & Versetzung,
		// Datenverwaltung, LMF-Aktionen, Lieferanten …). Eine Leitung, die die Bibliothek
		// führt, braucht diese Kategorien; verschlossen ist nur, was an manage_settings
		// hängt — Schule, Fristen, Mailversand. Gemessen am 16.09.2026: Erst dieser Lauf
		// hat gezeigt, dass „außer Einstellungen" nicht „außer dem Menüpunkt" heißt.
		await expect(page.getByTitle('Einstellungen')).toBeVisible();

		// Die Grenze zieht der Server, nicht das Menü: Ein fehlender Menüpunkt ist eine
		// Anzeige, kein Schloss — und ein VORHANDENER Menüpunkt sagt nichts darüber, was
		// dahinter aufgeht.
		const rechte = await page.request.get('/api/admin/permissions');
		expect(rechte.status()).toBe(403);
		const einstellungen = await page.request.get('/api/einstellungen');
		expect(einstellungen.status()).toBe(403);
		const mail = await page.request.get('/api/admin/settings/mail');
		expect(mail.status()).toBe(403);
	});
});
