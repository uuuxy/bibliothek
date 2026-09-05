import { test, expect } from '@playwright/test';
import { uiLogin, apiPost, seedBenutzer, seedSQL, uniqueSuffix } from './helpers.js';

// LMF-Plan (Register, Entscheidung 3, 05.09.2026): Die Bibliothek trägt Rückgabe- und
// Ausgabetermine je Klasse ein — statt der Excel-Liste, die per Mail ans Kollegium ging.
// Das Kollegium liest denselben Plan im Portal (für alle gleich), als PDF in der
// gewohnten Form. Geprüft über den echten Klickpfad: anlegen → in der Tabelle → im
// Portal einer Lehrkraft → PDF; dazu die Regel „ohne Klasse braucht es einen Vermerk".
const LEHRER_EMAIL = 'e2e-lehrer-lmfplan@test.local';

test('LMF-Plan: Termin anlegen, im Kollegiums-Portal sehen, PDF laden', async ({
	page,
	browser
}) => {
	const s = uniqueSuffix();
	const vermerk = `E2E-Plan-${s}`;
	seedBenutzer(LEHRER_EMAIL, 'kollegium');

	await uiLogin(page);
	await page.goto('/lmf-plan');

	// Ohne Klasse und ohne Vermerk lehnt der Server ab — bevor eine leere Zeile im Plan steht.
	const leer = await apiPost(page, '/api/lmf-termine', {
		datum: '2027-06-28',
		stunde: 3,
		art: 'rueckgabe',
		klassen: [],
		vermerk: ''
	});
	expect(leer.status(), 'leerer Termin').toBe(400);

	// Über den Dialog: Datum, Klasse als „weitere Klasse" (unabhängig vom Seed), Vermerk.
	await page.getByRole('button', { name: 'Termin hinzufügen' }).click();
	const dialog = page.getByRole('dialog');
	await dialog.getByLabel('Datum').fill('2027-06-28');
	await dialog.getByLabel('Weitere Klasse').fill('09H1');
	await dialog.getByRole('button', { name: 'Hinzufügen' }).click();
	await expect(dialog.getByRole('checkbox', { name: '09H1' })).toBeChecked();
	await dialog.getByLabel('Besonderheiten').fill(vermerk);
	await dialog.getByRole('button', { name: 'Speichern' }).click();

	// In der Tabelle: unter Bücherrückgabe, mit Wochentag und Klasse.
	const zeile = page.getByRole('row').filter({ hasText: vermerk });
	await expect(zeile).toBeVisible();
	await expect(zeile).toContainText('Montag');
	await expect(zeile).toContainText('09H1');
	await expect(page.getByRole('region', { name: 'Bücherrückgabe' })).toContainText(vermerk);

	// Das Kollegium sieht denselben Plan im Portal — ohne edit_books, nur mit Sitzung.
	const lehrerKontext = await browser.newContext();
	const lehrer = await lehrerKontext.newPage();
	try {
		await uiLogin(lehrer, LEHRER_EMAIL);
		await lehrer.getByTitle('Mein Portal').click();
		await lehrer.getByRole('tab', { name: 'LMF-Plan' }).click();
		await expect(lehrer.getByRole('row').filter({ hasText: vermerk })).toBeVisible();
		// Lesend: keine Aktionen im Portal.
		await expect(lehrer.getByRole('button', { name: /Bearbeiten/ })).toHaveCount(0);

		const download = lehrer.waitForEvent('download');
		await lehrer.getByRole('button', { name: /Als PDF/ }).click();
		expect((await download).suggestedFilename()).toBe('LMF-Plan.pdf');

		// Schreiben ist der Rolle verwehrt.
		const verboten = await apiPost(lehrer, '/api/lmf-termine', {
			datum: '2027-06-29',
			stunde: 1,
			art: 'ausgabe',
			klassen: ['09H1'],
			vermerk: 'verboten'
		});
		expect(verboten.status(), 'Kollegium schreibt den Plan').toBe(403);
	} finally {
		await lehrerKontext.close();
		seedSQL(`DELETE FROM lmf_termine WHERE vermerk = '${vermerk}';`);
	}
});
