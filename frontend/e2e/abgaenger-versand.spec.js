// Kontoauszug-Versand der Abgänger über den echten Klickpfad.
//
// WICHTIG (wie beim Mahnlauf): Der POST wird abgefangen und NIE durchgelassen. Der
// lokale Stack trägt einen echten SMTP_HOST — ein durchgereichter Lauf würde hier
// tatsächlich Kontoauszüge an die hinterlegten Klassenleitungen schicken. Geprüft
// wird der Payload, den das Frontend schicken WÜRDE; die Serverseite deckt
// api/graduates_mail_test.go ab.
import { test, expect } from '@playwright/test';
import { uiLogin, apiPost, csrfToken, seedSQL, uniqueSuffix } from './helpers.js';

/** Ein Schüler einer Abschlussklasse mit offenem Buch — damit der Dialog überhaupt eine
 *  Klasse anzubieten hat, unabhängig vom Seed. Zeilen gibt es nur in der Saison
 *  (01.05.–31.07.); außerhalb wird übersprungen, sichtbar im Bericht.
 *  @param {import('@playwright/test').Page} page
 *  @returns {Promise<any[]>} die Abgängerliste */
async function abgaengerBereitstellen(page) {
	const s = uniqueSuffix();
	seedSQL(`
		WITH t AS (INSERT INTO buecher_titel (titel) VALUES ('E2E-Versand-Titel ${s}') RETURNING id),
		ex AS (INSERT INTO buecher_exemplare (titel_id, barcode_id) SELECT id, 'E2E-VER-B-${s}' FROM t RETURNING id),
		sch AS (INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		        VALUES ('E2E-VER-S-${s}', 'Versand${s}', 'Testschueler', '10R1', 2030) RETURNING id)
		INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		SELECT ex.id, sch.id, CURRENT_DATE - 5 FROM ex, sch;
	`);
	const { fenster, abgaenger } = await (await page.request.get('/api/abgaenger')).json();
	test.skip(!fenster.offen, `Abgängerliste außerhalb der Saison (${fenster.von}–${fenster.bis})`);
	return abgaenger;
}

test('Abgänger: Klassenauswahl und Override landen im Versand-Request', async ({ page }) => {
	await uiLogin(page);
	await abgaengerBereitstellen(page);

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

	await page.goto('/schuljahr/abgaenger');
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
	expect(gesendeterBody.klassen).toHaveLength(anzahl > 1 ? anzahl - 1 : anzahl);
});

// Regression: Eine hinterlegte Klassenleitung MUSS im Dialog ankommen. Vorher stand
// dort bei jeder Klasse „keine E-Mail" — /api/abgaenger lieferte die Adressen gar
// nicht mit, und die Schreibweise im Mapping („5A" gegen „5a") wurde exakt verglichen.
// Beides zusammen sah aus, als sei der Versand kaputt.
test('Abgänger: hinterlegte Klassenleitung erscheint im Dialog', async ({ page }) => {
	await uiLogin(page);

	// Erst die Klassen der Abgänger holen, dann für die erste eine Adresse hinterlegen —
	// bewusst in ABWEICHENDER Schreibweise, denn genau daran scheiterte es.
	const abgaenger = await abgaengerBereitstellen(page);
	const klasse = abgaenger[0]?.klasse;
	expect(klasse, 'Testdaten ohne Abgänger').toBeTruthy();

	await apiPost(page, '/api/klassen-mapping', {
		klasse: ` ${String(klasse).toUpperCase()} `,
		lehrer_email: 'pflasch@philipp-reis-schule.de'
	});

	await page.goto('/schuljahr/abgaenger');
	await page.getByRole('button', { name: /An Klassenleitungen mailen/ }).click();

	const dialog = page.getByRole('dialog');
	const zeile = dialog.locator('label').filter({ hasText: klasse }).first();
	await expect(zeile).toBeVisible();
	await expect(zeile).not.toContainText('keine E-Mail');

	// Gegenprobe im selben Lauf: Eine Klasse OHNE Mapping muss den Hinweis weiter
	// tragen. Ohne sie würde der Test auch dann grün, wenn das Abzeichen gar nicht
	// mehr gerendert wird — geprüft wäre dann nichts.
	const ohneMapping = abgaenger.find((/** @type {any} */ s) => s.klasse !== klasse);
	if (ohneMapping) {
		// Voraussetzung selbst herstellen statt sie vorauszusetzen: Ein Mapping aus einem
		// früheren Lauf würde die Gegenprobe still entwerten.
		await page.request.delete(`/api/klassen-mapping/${encodeURIComponent(ohneMapping.klasse)}`, {
			headers: { 'X-CSRF-Token': await csrfToken(page) }
		});
		await page.reload();
		await page.getByRole('button', { name: /An Klassenleitungen mailen/ }).click();

		const andere = dialog.locator('label').filter({ hasText: ohneMapping.klasse }).first();
		await expect(andere).toContainText('keine E-Mail');
	}
});
