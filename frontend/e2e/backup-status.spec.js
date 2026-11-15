import { test, expect } from '@playwright/test';
import { uiLogin, schalteSicherung } from './helpers.js';

// Smoke-Flow Backup-Wächter: Ist der Alert an den echten Endpunkt verdrahtet?
//
// Diese Datei prüfte bis zum 06.08.2026 eine Umgebungsannahme statt eines Verhaltens:
// "Der lokale Stack setzt bewusst KEINEN BACKUP_ENCRYPTION_KEY". Genau das hat
// docker-compose.local.yml seit dem Durchreichen des Schlüssels (Zeile 62) aufgehoben —
// der Test wurde rot, obwohl das Feature stimmte. Ein Test, dessen Vorbedingung eine
// andere Datei im selben Repo still umstellt, misst nicht das Programm, sondern die
// Konfiguration.
//
// Die Textlogik je Zustand ist vollständig unit-getestet (src/lib/components/system/backupStatusText.test.js).
// Hier bleibt die Frage, die nur am laufenden Stack zu beantworten ist: Kommt der echte
// Status vom Server bis in den Alert? Deshalb wird der Zustand gelesen und die dazu
// passende Meldung erwartet — der Test gilt in jeder Stack-Konfiguration und bleibt
// trotzdem eine echte Behauptung.
//
// Die eigene Nachtsicherung ist von Haus aus aus (Einstellungen → Sicherung); dann warnt der
// Wächter nicht. Dieser Fall schaltet sie für die Dauer des Tests ein.
test('Backup-Wächter zeigt den echten Serverstatus an', async ({ page }) => {
	await uiLogin(page);
	const vorher = await schalteSicherung(page, true);
	try {
		await page.reload();
		await pruefeWaechter(page);
	} finally {
		await schalteSicherung(page, vorher);
	}
});

// Ohne eingeschaltete Sicherung meldet der Server "aus", und der Hinweis bleibt weg: Er stünde
// sonst über jeder Seite, obwohl die Schule die Sicherung mit Absicht dem Server überlässt.
test('Backup-Wächter schweigt, solange die eigene Sicherung aus ist', async ({ page }) => {
	await uiLogin(page);
	const vorher = await schalteSicherung(page, false);
	try {
		const res = await page.request.get('/api/admin/system/backup-status');
		expect((await res.json()).status).toBe('aus');
		await page.reload();
		await expect(page.getByText('Eigene Sicherung aus')).toBeVisible();
		await expect(page.getByRole('alert')).toBeHidden();
	} finally {
		await schalteSicherung(page, vorher);
	}
});

/**
 * Leitet die erwartete Meldung aus dem Stand des Servers ab und prüft den Hinweis.
 * @param {import('@playwright/test').Page} page
 */
async function pruefeWaechter(page) {
	const res = await page.request.get('/api/admin/system/backup-status');
	expect(res.ok(), 'backup-status muss erreichbar sein').toBeTruthy();
	const status = await res.json();

	// Erwartete Meldung aus dem Serverzustand ableiten — dieselbe Rangfolge wie
	// backupMessage(), aber bewusst hier ausgeschrieben: Würde der Test die Funktion
	// importieren, prüfte er sie gegen sich selbst.
	if (!status.encryption_key_set) {
		await erwarteAlert(page, 'Backup-Verschlüsselungs-Key fehlt');
	} else if (!status.last_backup_at) {
		await erwarteAlert(page, 'Noch kein Backup vorhanden');
	} else if (status.encryption_key_weak && status.status === 'warning') {
		await erwarteAlert(page, 'Backup-Schlüssel zu kurz');
	} else if (status.status === 'ok') {
		// Kein Handlungsbedarf: Der Alert darf dann NICHT stehen. Auch das ist eine
		// Behauptung — ein Wächter, der immer warnt, ist so nutzlos wie einer, der nie warnt.
		await expect(page.getByRole('alert')).toBeHidden();
	} else {
		await erwarteAlert(page, 'kein Backup');
	}
}

/**
 * Der Alert muss sichtbar sein, die Meldung tragen und einen Weg zur Behebung anbieten.
 * @param {import('@playwright/test').Page} page
 * @param {string} meldung
 */
async function erwarteAlert(page, meldung) {
	const alert = page.getByRole('alert');
	await expect(alert).toBeVisible();
	await expect(alert).toContainText(meldung);
	// Ein Alert ohne Weg zur Behebung ist eine Sackgasse (siehe BackupHinweis.svelte).
	await expect(alert.getByRole('button', { name: 'Betriebsbereitschaft öffnen' })).toBeVisible();
}
