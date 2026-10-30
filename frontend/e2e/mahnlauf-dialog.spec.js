// Der Mahnlauf-Dialog über den echten Pfad — Klick für Klick, wie das Personal ihn bedient.
//
// WICHTIG: Der POST wird abgefangen und NIE an das Backend durchgelassen. Der lokale
// Stack trägt in der .env einen echten SMTP_HOST; ein durchgereichter Versand würde
// hier tatsächlich Mahnungen verschicken. Geprüft wird deshalb genau das, was der
// Dialog dem Server sagen WÜRDE — die Serverseite deckt api/mahnwesen/mahnlisten_versand_test.go ab.
import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix } from './helpers.js';

// Der Knopf „Alle anmahnen" steht nur da, wenn etwas überfällig ist. Die Fälle legt die Datei
// selbst an: Sonst hing sie an den Resten früherer Tests und war auf einer frischen Datenbank rot.
// Zwei Klassen, damit das Abwählen einer Klasse jedes Mal geprüft wird.
const s = uniqueSuffix();
test.beforeEach(() => {
	seedSQL(`
		WITH t AS (INSERT INTO buecher_titel (titel) VALUES ('E2E-Mahnlauf ${s}') RETURNING id),
		ex AS (INSERT INTO buecher_exemplare (titel_id, barcode_id)
		       SELECT t.id, 'B-MAHN' || g || '${s}' FROM t, generate_series(1, 2) g RETURNING id, barcode_id),
		l AS (INSERT INTO schueler (vorname, nachname, klasse, barcode_id, abgaenger_jahr)
		      SELECT 'Kind' || g, 'Mahnlauf-${s}', '0' || (6 + g) || 'M', 'S-MAHN' || g || '${s}', 2031
		      FROM generate_series(1, 2) g RETURNING id, barcode_id)
		INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		SELECT ex.id, l.id, CURRENT_DATE - 30
		FROM ex JOIN l ON substring(ex.barcode_id FROM 7) = substring(l.barcode_id FROM 7);
	`);
});
test.afterEach(() => {
	seedSQL(`
		DELETE FROM ausleihen WHERE exemplar_id IN (
			SELECT id FROM buecher_exemplare WHERE barcode_id LIKE 'B-MAHN%${s}');
		DELETE FROM buecher_exemplare WHERE barcode_id LIKE 'B-MAHN%${s}';
		DELETE FROM buecher_titel WHERE titel = 'E2E-Mahnlauf ${s}';
		DELETE FROM leser WHERE barcode_id LIKE 'S-MAHN%${s}';
	`);
});

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
	expect(anzahl).toBeGreaterThan(1);
	await expect(senden).toContainText(`${anzahl}`);

	// Abwählen muss sich sofort in der Beschriftung niederschlagen — der Knopf ist
	// die einzige Stelle, an der ablesbar ist, wie viele Klassen der Lauf trifft.
	await checkboxen.first().uncheck();
	await expect(senden).toContainText(`${anzahl - 1}`);

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
	expect(gesendeterBody.klassen).toHaveLength(anzahl - 1);
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

// Ein Mahnlauf dauert je Klasse eine Mail lang. Solange er läuft, ist „Alle anmahnen" gesperrt
// und zeigt den Lauf; ein zweiter Start verschickte jede Mahnliste noch einmal. Die Antwort
// hält der Test zurück, bis er den gesperrten Knopf gesehen hat.
test('Mahnlauf: Solange er läuft, ist „Alle anmahnen" gesperrt', async ({ page }) => {
	await uiLogin(page);

	let anfragen = 0;
	/** @type {(wert?: unknown) => void} */
	let freigeben = () => {};
	const freigegeben = new Promise((fertig) => (freigeben = fertig));
	await page.route('**/api/mail/send-bulk-overdue', async (route) => {
		anfragen++;
		await freigegeben;
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({ sent_count: 1, message: 'Mahnlisten von 1 Klasse versendet' })
		});
	});

	await page.goto('/mahnwesen');
	const knopf = page
		.getByRole('button', {
			name: /Alle anmahnen – Mahnlauf konfigurieren und per E-Mail versenden/
		})
		.first();
	await knopf.click();
	await page
		.getByRole('dialog')
		.getByRole('button', { name: /anmahnen$/ })
		.click();

	await expect(knopf).toBeDisabled();
	await expect(knopf.getByRole('progressbar')).toBeVisible();

	freigeben();
	await expect(page.getByText('Mahnlisten von 1 Klasse versendet')).toBeVisible();
	await expect(knopf).toBeEnabled();
	await expect(knopf.getByRole('progressbar')).toHaveCount(0);
	expect(anfragen).toBe(1);
});
