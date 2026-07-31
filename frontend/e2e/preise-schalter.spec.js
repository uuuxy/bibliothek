import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix } from './helpers.js';

// Der Schalter "Preise im Bestellwesen" (Einstellungen → Allgemein).
//
// Anlass: Preise wurden nie gepflegt — 2360 Exemplare im Bestand, kein einziges mit einem
// Einkaufspreis über 0. Bestellhistorie und alle drei Berichte summierten also Nullen und
// sahen dabei aus wie ein Ausgabennachweis. Ein Nachweis, der Null behauptet, ist
// schlimmer als keiner: Er wird geglaubt.
//
// Der Schalter steuert Erfassung UND Anzeige, nicht die Daten. Deshalb prüft dieser Test
// beide Richtungen — dass etwas verschwindet, und dass es zurückkommt.

/** @param {boolean} an */
function setzePreiseErfassen(an) {
	seedSQL(`
		INSERT INTO system_einstellungen (schluessel, wert) VALUES ('preise_erfassen', '${an}')
		ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert;
	`);
}

// Die Einstellung ist global und die Suite teilt sich eine Datenbank: Ohne dieses
// Zurücksetzen liefe jeder nachfolgende Test mit abgeschalteten Preisen.
test.afterEach(() => setzePreiseErfassen(true));

test('Preise aus: Warenkorb, Historie und Berichte zeigen Mengen statt Geld', async ({ page }) => {
	const s = uniqueSuffix();
	seedSQL(`
		WITH t AS (
			INSERT INTO buecher_titel (titel, autor, isbn) VALUES ('E2E-Preis-Titel ${s}', 'A', '978${s.slice(0, 10)}') RETURNING id
		), b AS (
			INSERT INTO bestellungen_verlauf (lieferant_name, lieferant_email, kundennummer, bestelldatum, gesamtbetrag, anzahl_exemplare)
			VALUES ('E2E-Preis-Lieferant ${s}', 'e2e@example.org', 'K-${s}', CURRENT_TIMESTAMP, 42.50, 2) RETURNING id
		)
		INSERT INTO bestellungen_positionen (bestellung_id, titel_id, titel_name, isbn, menge, einzelpreis)
		SELECT b.id, t.id, 'E2E-Preis-Titel ${s}', '', 2, 21.25 FROM b, t;
	`);

	setzePreiseErfassen(false);
	await uiLogin(page);
	await page.getByTitle('Bestellungen').click();
	await page.getByRole('button', { name: 'Bestellhistorie', exact: true }).click();

	// Kopfzeile: keine Ausgaben, sondern Exemplare.
	await expect(page.getByText('Gesamtausgaben')).toHaveCount(0);
	await expect(page.getByText('Bestellte Exemplare')).toBeVisible();

	// Aufgeklappte Position: keine Betragsspalten.
	await page.getByRole('button', { name: new RegExp(`E2E-Preis-Lieferant ${s}`) }).click();
	await expect(page.getByText(`E2E-Preis-Titel ${s}`)).toBeVisible();
	await expect(page.getByRole('columnheader', { name: 'Einzelpreis' })).toHaveCount(0);

	// Berichte: "Lieferantenabrechnung" waere ohne Preise schlicht falsch — abgerechnet
	// wird nichts.
	await page.getByRole('button', { name: 'Berichte', exact: true }).click();
	await expect(page.getByText('Lieferantenübersicht')).toBeVisible();
	await expect(page.getByText('Lieferantenabrechnung')).toHaveCount(0);

	// BEWEIS, dass nichts gelöscht wurde: Schalter zurück, Beträge wieder da.
	setzePreiseErfassen(true);
	await page.reload();
	await page.getByTitle('Bestellungen').click();
	await page.getByRole('button', { name: 'Bestellhistorie', exact: true }).click();
	await expect(page.getByText('Gesamtausgaben')).toBeVisible();
	await page.getByRole('button', { name: new RegExp(`E2E-Preis-Lieferant ${s}`) }).click();
	await expect(page.getByRole('columnheader', { name: 'Einzelpreis' })).toBeVisible();
});

// Das PDF muss zur Oberfläche passen: Wer keine Beträge sieht, darf im Bericht keine
// finden — sonst widersprechen sich zwei Ansichten derselben Daten.
test('Preise aus: der Bericht ist ein Mengenbericht', async ({ page }) => {
	setzePreiseErfassen(false);
	await uiLogin(page);

	const res = await page.request.get(
		'/api/bestellhistorie/bericht?von=2020-01-01&bis=2030-12-31&jahresansicht=true&titel=E2E-Mengenbericht'
	);
	expect(res.status()).toBe(200);
	expect(res.headers()['content-type']).toContain('application/pdf');

	// gofpdf komprimiert die Textströme, der Inhalt ist also nicht direkt lesbar. Geprüft
	// wird deshalb, dass überhaupt ein vollständiges PDF entsteht — die Spaltenlogik
	// selbst deckt der Go-Test ab (TestBerichtOhnePreise*).
	const body = await res.body();
	expect(body.length).toBeGreaterThan(1000);
	expect(body.subarray(0, 5).toString()).toBe('%PDF-');
});
