import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix } from './helpers.js';

// „Was macht die Uhr da? Da passiert nichts." — aus dem Betrieb gemeldet. Die
// Verlängerung funktionierte, aber ihre einzige Wirkung war eine still geänderte Zahl
// in einer anderen Spalte. Dieser Test hält beides fest: dass sie wirkt UND dass sie
// es sagt.
test('Einzel-Verlängerung: verschiebt die Frist und meldet es zurück', async ({ page }) => {
	const s = uniqueSuffix();
	seedSQL(`
		WITH t AS (INSERT INTO buecher_titel (titel) VALUES ('E2E-Verl-Titel ${s}') RETURNING id),
		ex AS (INSERT INTO buecher_exemplare (titel_id, barcode_id) SELECT id, 'E2E-VERL-B-${s}' FROM t RETURNING id),
		sch AS (INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_abgaenger)
			VALUES ('E2E-VERL-S-${s}', 'Verl${s}', 'Testschueler', '10a', EXTRACT(YEAR FROM CURRENT_DATE)::int, true) RETURNING id)
		INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
		SELECT ex.id, sch.id, CURRENT_DATE - 5 FROM ex, sch;
	`);

	await uiLogin(page);
	page.on('dialog', async (d) => {
		console.log('ALERT:', d.message());
		await d.dismiss();
	});
	const antworten = [];
	page.on('response', (r) => {
		if (r.url().includes('verlaengern')) antworten.push(`${r.status()} ${r.url()}`);
	});

	await page.getByTitle('Abgänger').click();
	await page.getByRole('button', { name: new RegExp(`Profil von Verl${s} Testschueler`) }).click();
	await expect(page.getByText('Ausleihen & Historie')).toBeVisible();

	const zelle = page.locator('td', { hasText: /^\s*\d{1,2}\.\d{1,2}\.\d{4}/ }).first();
	const vorher = (await zelle.innerText()).trim();

	await page.getByRole('button', { name: 'Ausleihe verlängern' }).first().click();

	// 1. Die Frist wandert sichtbar.
	await expect(zelle).not.toHaveText(vorher, { timeout: 5000 });
	// 2. Und die Aktion sagt, was sie getan hat — mit dem neuen Datum.
	await expect(page.getByText(/Verlängert bis \d{1,2}\.\d{1,2}\.\d{4}/)).toBeVisible();

	expect(antworten.some((a) => a.startsWith('200'))).toBe(true);
});

// Aus dem Betrieb gemeldet: „Ich drücke auf Verlängern, dann kommt ein Datum in der
// Vergangenheit, aber sonst passiert nichts."
//
// Ursache war, dass die neue Frist auf die ALTE Frist addiert wurde. Bei einer Ausleihe,
// die länger überfällig ist als die Leihfrist (21 Tage), landet sie damit WIEDER in der
// Vergangenheit: Das Buch bleibt „Überfällig", die Zeile sieht unverändert aus, und die
// Erfolgsmeldung nennt ein Datum, das schon vorbei ist.
//
// Warum der ältere Test das nicht fand: Er setzt die Frist auf CURRENT_DATE - 5. Fünf Tage
// überfällig plus 21 Tage Verlängerung ergibt ein Datum in der Zukunft — der Fehler
// beginnt erst jenseits der Leihfrist. Deshalb hier bewusst 60 Tage.
test('Verlängerung einer lange überfälligen Ausleihe endet in der Zukunft', async ({ page }) => {
	const s = uniqueSuffix();
	seedSQL(`
		WITH t AS (INSERT INTO buecher_titel (titel) VALUES ('E2E-Spaet-Titel ${s}') RETURNING id),
		ex AS (INSERT INTO buecher_exemplare (titel_id, barcode_id) SELECT id, 'E2E-SPAET-B-${s}' FROM t RETURNING id),
		sch AS (INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_abgaenger)
			VALUES ('E2E-SPAET-S-${s}', 'Spaet${s}', 'Testschueler', '10a', EXTRACT(YEAR FROM CURRENT_DATE)::int, true) RETURNING id)
		INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist, mahnstufe)
		SELECT ex.id, sch.id, CURRENT_DATE - 60, 2 FROM ex, sch;
	`);

	await uiLogin(page);
	await page.getByTitle('Abgänger').click();
	await page.getByRole('button', { name: new RegExp(`Profil von Spaet${s} Testschueler`) }).click();
	await expect(page.getByText('Ausleihen & Historie')).toBeVisible();

	await page.getByRole('button', { name: 'Ausleihe verlängern' }).first().click();
	await expect(page.getByText(/Verlängert bis \d{1,2}\.\d{1,2}\.\d{4}/)).toBeVisible();

	// Am Zustand belegt, nicht am Text: Eine Verlängerung, die in der Vergangenheit endet,
	// ist keine. Der Nutzer soll das Buch danach weiter ausleihen dürfen.
	const frist = querySQL(`
		SELECT rueckgabe_frist::date > CURRENT_DATE
		FROM ausleihen a
		JOIN buecher_exemplare e ON e.id = a.exemplar_id
		WHERE e.barcode_id = 'E2E-SPAET-B-${s}'
	`).trim();
	expect(frist, 'neue Frist muss in der Zukunft liegen').toBe('t');

	// Und die Mahn-Eskalation gehört zurückgesetzt, sobald die Frist wieder trägt —
	// sonst springt dasselbe Buch beim nächsten Überziehen sofort in die Rechnungsstufe.
	const mahnstufe = querySQL(`
		SELECT a.mahnstufe
		FROM ausleihen a
		JOIN buecher_exemplare e ON e.id = a.exemplar_id
		WHERE e.barcode_id = 'E2E-SPAET-B-${s}'
	`).trim();
	expect(mahnstufe, 'Mahnstufe muss zurückgesetzt sein').toBe('0');
});
