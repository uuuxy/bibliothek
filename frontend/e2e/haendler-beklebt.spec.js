// „Händler beklebt die Bücher" — der Rundlauf durch die Oberfläche.
//
// Die Wirkung der Einstellung (Exemplare erscheinen nicht auf der Nachdruck-Liste) ist in
// api/haendler_beklebt_pg_test.go am Verhalten belegt; der Weg über POST /api/bestellungen
// verschickt Mail und gehört deshalb nicht in einen E2E-Lauf.
//
// Hier geht es um die andere Fehlerklasse: ein Formularfeld, das die Oberfläche anbietet,
// das Backend aber still verwirft — Antwort 200, Einstellung weg, niemand merkt es. Der
// Test belegt deshalb an der DATENBANK, dass der Haken ankommt, und danach, dass er beim
// Bearbeiten nicht wieder verloren geht.
import { test, expect } from '@playwright/test';
import { uiLogin, querySQL, uniqueSuffix } from './helpers.js';

/** @param {import('@playwright/test').Page} page */
async function zuLieferanten(page) {
	await page.goto('/bestellungen');
	await page.getByRole('button', { name: 'Lieferanten verwalten' }).click();
	await page.getByRole('heading', { name: 'Neuer Lieferant' }).waitFor();
}

/** @param {string} name */
function flagInDB(name) {
	return querySQL(`SELECT liefert_mit_barcode FROM lieferanten WHERE name = '${name}'`).trim();
}

test('Haken „Händler beklebt" kommt in der Datenbank an und überlebt das Bearbeiten', async ({
	page
}) => {
	const s = uniqueSuffix();
	const name = `E2E-Bekleb-${s}`;
	await uiLogin(page);
	await zuLieferanten(page);

	await page.getByLabel('Name').fill(name);
	await page.getByLabel('E-Mail').fill(`b${s}@example.invalid`);
	await page.getByLabel('Kundennummer').fill(`K-${s}`);
	// Ein Zustand, kein Auswahlpunkt: Die Oberfläche zeigt hier einen M3-Schalter
	// (role="switch"), kein Häkchen.
	await page.getByRole('switch', { name: 'Händler beklebt die Bücher' }).click();
	await page.getByRole('button', { name: 'Lieferanten speichern' }).click();

	// 1. Der Haken erreicht die Datenbank — nicht nur die Oberfläche.
	await expect
		.poll(() => flagInDB(name), { timeout: 5000, message: 'Haken muss gespeichert sein' })
		.toBe('t');

	// 2. Und die Liste zeigt ihn auch an — in der Spalte „Etikettendruck" steht, WER druckt.
	//    Bewusst zwei gleichwertige Wörter statt Chip gegen graue Kleinschrift: eine
	//    Ja/Nein-Angabe verdient eine Darstellung, nicht zwei.
	const zeile = page.locator('tr', { hasText: name });
	await expect(zeile.getByText('Händler', { exact: true })).toBeVisible();

	// 3. Wer nur die E-Mail korrigiert, darf die Einstellung nicht verlieren.
	//    Genau hier stand der Haken vorher immer auf „aus", weil die Bearbeiten-Maske ihn
	//    nicht aus dem Datensatz übernahm.
	await zeile.getByRole('button', { name: 'Bearbeiten' }).click();
	const bearbeiten = page.locator('tr', { hasText: /Speichern/ });
	await bearbeiten.locator('input[type="email"]').fill(`neu${s}@example.invalid`);
	await bearbeiten.getByRole('button', { name: 'Speichern' }).click();

	await expect
		.poll(() => flagInDB(name), {
			timeout: 5000,
			message: 'Haken darf beim Bearbeiten nicht verloren gehen'
		})
		.toBe('t');
});

test('Ohne Haken bleibt es beim bisherigen Verhalten', async ({ page }) => {
	const s = uniqueSuffix();
	const name = `E2E-Selbst-${s}`;
	await uiLogin(page);
	await zuLieferanten(page);

	await page.getByLabel('Name').fill(name);
	await page.getByLabel('E-Mail').fill(`s${s}@example.invalid`);
	await page.getByLabel('Kundennummer').fill(`K-${s}`);
	await page.getByRole('button', { name: 'Lieferanten speichern' }).click();

	await expect.poll(() => flagInDB(name), { timeout: 5000 }).toBe('f');
	await expect(
		page.locator('tr', { hasText: name }).getByText('Bibliothek', { exact: true })
	).toBeVisible();
});

// Standardlieferant: Wer immer beim selben Händler bestellt, soll ihn nicht jedes Mal
// neu auswählen müssen — einmal vergessen heisst, die Bestellung geht an den falschen raus.
//
// Vorher gewann schlicht der alphabetisch erste (die Liste kommt mit ORDER BY name).
// Deshalb heisst der Testlieferant hier absichtlich mit "Z" — wäre die Vorauswahl
// wirkungslos, stünde ein anderer im Feld, und der Test wäre rot.
test('Der Standardlieferant ist beim Bestellen vorausgewählt', async ({ page }) => {
	const s = uniqueSuffix();
	const name = `ZZZ-Standard-${s}`;
	await uiLogin(page);
	await zuLieferanten(page);

	await page.getByLabel('Name').fill(name);
	await page.getByLabel('E-Mail').fill(`std${s}@example.invalid`);
	await page.getByLabel('Kundennummer').fill(`K-STD-${s}`);
	await page.getByRole('switch', { name: 'Voreingestellt beim Bestellen' }).click();
	await page.getByRole('button', { name: 'Lieferanten speichern' }).click();

	// 1. In der Datenbank steht genau einer — die Invariante, auf der alles beruht.
	await expect
		.poll(() => querySQL(`SELECT count(*) FROM lieferanten WHERE ist_standard`).trim(), {
			timeout: 5000,
			message: 'Es darf immer nur einen Standardlieferanten geben'
		})
		.toBe('1');
	expect(
		querySQL(`SELECT name FROM lieferanten WHERE ist_standard`).trim(),
		'und zwar der gerade gesetzte'
	).toBe(name);

	// 2. Und das Bestellformular übernimmt ihn — sonst wäre die Einstellung Zierrat.
	//
	// Bewusst nach einem Neuladen: Die Vorauswahl greift beim Aufbau des Bestellwesens,
	// nicht mitten in der Arbeit. Eine schon getroffene Auswahl bleibt stehen — sonst
	// wechselte jemandem, der gerade einen Warenkorb füllt, unter der Hand der Lieferant.
	await page.goto('/bestellungen');
	const auswahl = page.locator('select#supplier');
	await auswahl.waitFor();
	await expect(
		auswahl,
		'Das Bestellformular muss den hinterlegten Standardlieferanten vorauswählen'
	).toHaveValue(querySQL(`SELECT id FROM lieferanten WHERE ist_standard`).trim());
	await expect(auswahl).toContainText(name);
});
