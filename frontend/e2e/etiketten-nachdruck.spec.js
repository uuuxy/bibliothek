import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix } from './helpers.js';

// Der Anlass (Betreiber): Eine Lieferung ist im System freigegeben, aber die Etiketten
// kamen nie aus dem Drucker. Danach gab es keinen Weg mehr zu genau diesen Exemplaren
// zurück — man hätte jeden Titel einzeln suchen müssen, ohne zu wissen, welche es sind.
test('Fehlende Etiketten: Exemplare finden, auswählen und an den Druck übergeben', async ({
	page
}) => {
	const s = uniqueSuffix();
	const titel = `E2E-Etikett-Titel ${s}`;

	// Drei Exemplare ohne gedrucktes Etikett, mit heutigem Zugang — damit sie in der
	// nach Zugang absteigend sortierten Liste oben stehen.
	seedSQL(`
		WITH t AS (
			INSERT INTO buecher_titel (titel, autor) VALUES ('${titel}', 'Testautorin') RETURNING id
		)
		INSERT INTO buecher_exemplare (titel_id, barcode_id, etikett_gedruckt, erworben_am)
		SELECT t.id, 'E2E-ETI-${s}-' || g, false, CURRENT_DATE
		FROM t, generate_series(1, 3) AS g;
	`);

	await uiLogin(page);
	await page.getByTitle('Druck-Center').click();
	await page.getByRole('button', { name: 'Fehlende Etiketten' }).click();

	// Auf den eigenen Bestand eingrenzen — die Liste zeigt alles, was noch kein Etikett hat.
	const filter = page.getByLabel('Exemplare filtern');
	await filter.click();
	await filter.fill(`E2E-ETI-${s}`);
	await expect(page.getByRole('row', { name: new RegExp(`E2E-ETI-${s}-1`) })).toBeVisible();

	// Zwei der drei auswählen — die Auswahl muss einzeln möglich sein, nicht nur "alle".
	await page.getByLabel(`${titel} (E2E-ETI-${s}-1) auswählen`).check();
	await page.getByLabel(`${titel} (E2E-ETI-${s}-2) auswählen`).check();

	await page.getByRole('button', { name: /2 an den Druck übergeben/ }).click();

	// BEWEIS 1: Die Übergabe landet im Etikettendruck — dort steht die Vorschau bereit.
	// Es gibt keinen zweiten Druckweg; die Auswahl geht durch dieselbe printQueue wie
	// der Wareneingang.
	await expect(page.getByRole('button', { name: 'A4-Bogen drucken' })).toBeEnabled();
});

// Die Gegenbuchung ist der Teil, ohne den die Liste wertlos wäre: etikett_gedruckt wurde
// vorher NIRGENDS auf true gesetzt — der Wert stand seit dem Anlegen der Tabelle auf false.
// Die Liste hätte also dauerhaft den gesamten Bestand gezeigt statt der Nachzügler.
test('Nach dem Druck sind die Exemplare als gedruckt vermerkt', async ({ page, context }) => {
	const s = uniqueSuffix();
	const titel = `E2E-Vermerk-Titel ${s}`;

	seedSQL(`
		WITH t AS (
			INSERT INTO buecher_titel (titel, autor) VALUES ('${titel}', 'Testautorin') RETURNING id
		)
		INSERT INTO buecher_exemplare (titel_id, barcode_id, etikett_gedruckt, erworben_am)
		SELECT t.id, 'E2E-VER-${s}', false, CURRENT_DATE FROM t;
	`);

	await uiLogin(page);
	await page.getByTitle('Druck-Center').click();
	await page.getByRole('button', { name: 'Fehlende Etiketten' }).click();

	const filter = page.getByLabel('Exemplare filtern');
	await filter.click();
	await filter.fill(`E2E-VER-${s}`);
	await page.getByLabel(`${titel} (E2E-VER-${s}) auswählen`).check();
	await page.getByRole('button', { name: /1 an den Druck übergeben/ }).click();

	// Der Druck öffnet das PDF in einem neuen Tab — den fangen wir ab, sonst bleibt er offen.
	const neuerTab = context.waitForEvent('page');
	await page.getByRole('button', { name: 'A4-Bogen drucken' }).click();
	await (await neuerTab).close();

	// BEWEIS: an der Datenbank, nicht an der Oberfläche.
	await expect
		.poll(() =>
			querySQL(`SELECT etikett_gedruckt FROM buecher_exemplare WHERE barcode_id = 'E2E-VER-${s}'`)
		)
		.toBe('t');

	// Und damit ist das Exemplar aus der Liste verschwunden.
	await page.getByRole('button', { name: 'Fehlende Etiketten' }).click();
	await page.getByLabel('Exemplare filtern').fill(`E2E-VER-${s}`);
	await expect(page.getByText(`Kein Exemplar ohne Etikett passt zu „E2E-VER-${s}"`)).toBeVisible();
});
