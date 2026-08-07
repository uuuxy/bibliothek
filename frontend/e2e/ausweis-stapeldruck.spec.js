import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix } from './helpers.js';

// Stapeldruck der Schülerausweise aus der Schülerdatei.
//
// Geprüft wird am DOM des Druckbereichs, nicht am Klick: „Ausweise drucken" öffnet den
// Systemdialog des Browsers, den Playwright nicht bedienen kann. Aussagekräftig ist
// ohnehin die Frage davor — steht für jeden markierten Schüler eine Karte bereit, und
// trägt sie das richtige Ablaufjahr?
//
// Das Ablaufjahr kommt aus dem Bildungsgang (internal/ausweis): 7H1 endet mit Jahrgang 9,
// 8G2 mit 10. Eine Bezeichnung ohne erkennbaren Jahrgang ("Vorkurs") liefert keins —
// dort muss die Oberfläche vor dem Druck warnen, statt „31.07.–" zu drucken.
//
// Bewusst NICHT "5a" als Warnfall: Klassen ohne Zweigbuchstaben bekommen seit dem
// 07.08.2026 den längsten Mittelstufenweg und damit sehr wohl ein Datum.
test.describe('Ausweis-Stapeldruck', () => {
	const marke = uniqueSuffix().slice(0, 6);
	const schuljahrEnde = new Date().getMonth() >= 7 ? new Date().getFullYear() + 1 : new Date().getFullYear();

	test.beforeEach(() => {
		seedSQL(`
			INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
			VALUES ('STAPEL-H-${marke}', 'Hein', 'Stapel${marke}', '7H1', 2031),
			       ('STAPEL-G-${marke}', 'Gerd', 'Stapel${marke}', '8G2', 2031),
			       ('STAPEL-X-${marke}', 'Xena', 'Stapel${marke}', 'Vorkurs', 2031);
		`);
	});

	test.afterEach(() => {
		seedSQL(`DELETE FROM schueler WHERE barcode_id LIKE 'STAPEL-%-${marke}';`);
	});

	test('markierte Schüler landen als Karten im Druckbereich, mit korrektem Ablaufjahr', async ({
		page
	}) => {
		await uiLogin(page);
		await page.getByTitle('Schülerdatei').click();

		// Auf die drei Testschüler eingrenzen — die Suche läuft auf dem Server.
		await page.getByLabel('Schüler suchen').fill(`Stapel${marke}`);
		await expect(page.locator('tbody tr').filter({ hasText: `Stapel${marke}` })).toHaveCount(3);

		// Ohne Markierung darf kein Aktionsbalken stehen.
		await expect(page.getByRole('region', { name: /Aktionen für die markierten/ })).toBeHidden();

		// Kopf-Checkbox markiert genau die angezeigten Treffer.
		await page.getByRole('checkbox', { name: /Alle angezeigten Schüler/ }).check();

		const balken = page.getByRole('region', { name: /Aktionen für die markierten/ });
		await expect(balken).toBeVisible();
		await expect(balken).toContainText('3');
		// Xena ("Vorkurs") hat keinen erkennbaren Jahrgang — davor muss gewarnt werden.
		await expect(balken).toContainText('1 davon ohne Ablaufjahr');

		// Der Druckbereich trägt je markiertem Schüler eine Karte. Er ist am Bildschirm
		// unsichtbar (nur @media print), deshalb wird der DOM-Inhalt geprüft.
		const druckbereich = page.locator('.print-section-card');
		await expect(druckbereich).toHaveCount(1);
		await expect(druckbereich.locator('.print-card-box')).toHaveCount(3);

		// Die Ablaufjahre stehen auf den Karten und folgen dem Bildungsgang.
		const inhalt = await druckbereich.innerText();
		expect(inhalt).toContain(`31.07.${schuljahrEnde + 2}`); // 7H1: Jahrgang 7 → Ende 9
		expect(inhalt).toContain(`31.07.${schuljahrEnde + 2}`); // 8G2: Jahrgang 8 → Ende 10
		expect(inhalt).toContain('31.07.–'); // Vorkurs: kein Jahrgang, nicht ableitbar

		// Markierung aufheben räumt Balken UND Druckbereich ab — ein unsichtbar im DOM
		// hängender Kartensatz wäre sonst der nächste Fehldruck.
		await balken.getByRole('button', { name: 'Markierung aufheben' }).click();
		await expect(balken).toBeHidden();
		await expect(page.locator('.print-section-card')).toHaveCount(0);
	});

	test('Einzelauswahl öffnet nicht das Profil', async ({ page }) => {
		await uiLogin(page);
		await page.getByTitle('Schülerdatei').click();
		await page.getByLabel('Schüler suchen').fill(`Stapel${marke}`);
		await expect(page.locator('tbody tr').filter({ hasText: `Stapel${marke}` })).toHaveCount(3);

		// Die ganze Zeile öffnet das Profil. Das Häkchen darf das NICHT auslösen, sonst
		// ist die Markierung nach dem ersten Klick weg (stopPropagation in der Zelle).
		await page.getByRole('checkbox', { name: /Hein Stapel/ }).check();

		await expect(page.getByRole('region', { name: /Aktionen für die markierten/ })).toContainText(
			'1'
		);
		// Immer noch die Liste, nicht das Profil.
		await expect(page.getByLabel('Schüler suchen')).toBeVisible();
		await expect(page.locator('.print-section-card .print-card-box')).toHaveCount(1);
	});
});
