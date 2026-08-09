import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix } from './helpers.js';

// Escape bringt von überall zurück an die Theke. Der Kurzbefehl galt bedingungslos —
// und machte damit die Berichte unbenutzbar: Die Ansicht besteht aus Monats-, Jahres- und
// Datumsfeldern, und Escape ist dort die normale Art, ein aufgeklapptes Auswahlfenster
// zuzumachen. Statt des Auswahlfensters schloss sich die ganze Ansicht; für den Benutzer
// sprang das Programm grundlos in die Ausleihe.
//
// Betroffen war nicht nur diese eine Ansicht, sondern jedes Feld der Anwendung. Die
// Berichte fielen nur zuerst auf, weil sie fast nur aus solchen Feldern bestehen.
test('Escape in einem Eingabefeld verlässt die Ansicht nicht', async ({ page }) => {
	await uiLogin(page);
	await page.getByTitle('Bestellungen').click();
	await page.getByRole('tab', { name: 'Berichte', exact: true }).click();
	await expect(page.getByText('Bericht erstellen')).toBeVisible();

	// So bedient man ein Monatsfeld: anklicken, Auswahlfenster mit Escape schließen.
	await page.locator('#monat').click();
	await page.keyboard.press('Escape');

	await expect(page.getByText('Bericht erstellen')).toBeVisible();

	// Auch im Auswahlfeld des Jahresberichts.
	await page.getByRole('radio', { name: /Jahresbericht/ }).check();
	await page.locator('#jahr').click();
	await page.keyboard.press('Escape');
	await expect(page.getByText('Bericht erstellen')).toBeVisible();
});

// Die Gegenrichtung: Ohne Fokus in einem Feld muss der Kurzbefehl weiter funktionieren —
// sonst wäre der Fehler nur gegen den Verlust der Funktion eingetauscht.
test('Escape außerhalb von Eingabefeldern führt weiter zur Ausleihe', async ({ page }) => {
	await uiLogin(page);
	await page.getByTitle('Bestellungen').click();
	await expect(page.getByRole('tab', { name: 'Berichte', exact: true })).toBeVisible();

	await page.locator('body').click({ position: { x: 5, y: 5 } });
	await page.keyboard.press('Escape');

	await expect(page.locator('#omnibox-input')).toBeVisible();
});

// Der zweite Teil derselben Fehlerklasse: Ein aufgeklapptes Menü ist kein Eingabefeld,
// sein Auslöser ist ein Knopf. Die Feld-Ausnahme greift hier also nicht — das Menü muss
// den Tastendruck selbst als verarbeitet melden. Tat es keins, schloss Escape das Menü
// UND warf den Benutzer aus dem Schülerprofil in die Ausleihe.
test('Escape in einem offenen Menü schliesst nur das Menü', async ({ page }) => {
	const s = uniqueSuffix();
	seedSQL(`
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_abgaenger)
		VALUES ('E2E-ESC-${s}', 'Escmenue${s}', 'Testschueler', '9a',
		        EXTRACT(YEAR FROM CURRENT_DATE)::int + 3, false);
	`);

	await uiLogin(page);
	await page.getByTitle('Schülerdatei').click();
	await page
		.getByPlaceholder(/filtern|suchen/i)
		.first()
		.fill(`Escmenue${s}`);
	await page.getByText(`Escmenue${s} Testschueler`).first().click();
	await expect(page.getByRole('heading', { name: new RegExp(`Escmenue${s}`) })).toBeVisible();

	await page.getByRole('button', { name: 'Ausweisseiten wählen' }).click();
	await page.keyboard.press('Escape');

	// Das Profil steht noch — nur das Menü ist zu.
	await expect(page.getByRole('heading', { name: new RegExp(`Escmenue${s}`) })).toBeVisible();
});
