// Der Hauptlieferant — der Rundlauf durch die Oberfläche.
//
// Eine Schule bestellt in der Regel bei EINEM Händler. Der ist beim Bestellen
// vorausgewählt, bekommt statt der reinen Bestellmail den Bestelllink (Etikettengröße +
// Bestätigung) und beklebt die Bücher selbst — seine Exemplare stehen deshalb nicht auf
// der Nachdruck-Liste. Alle anderen bekommen einfach nur die Bestellmail.
//
// Vorher waren das DREI einzelne Schalter mit drei einzelnen Tests. Sie beschrieben
// denselben Händler, mussten aber einzeln gesetzt werden — und „Bestelllink ohne beklebt"
// hiess: Der Händler klebt, die Bibliothek druckt trotzdem noch einmal. Migration 066 hat
// sie zusammengelegt, dieser Test entsprechend die drei alten aus haendler-beklebt.spec.js.
//
// Die WIRKUNG auf die Nachdruck-Liste ist in api/haendler_beklebt_pg_test.go am Verhalten
// belegt; der Weg über POST /api/bestellungen verschickt Mail und gehört deshalb nicht in
// einen E2E-Lauf. Hier geht es um die andere Fehlerklasse: ein Formularfeld, das die
// Oberfläche anbietet, das Backend aber still verwirft — Antwort 200, Einstellung weg,
// niemand merkt es. Belegt wird deshalb an der DATENBANK.
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
	return querySQL(`SELECT ist_hauptlieferant FROM lieferanten WHERE name = '${name}'`).trim();
}

/**
 * @param {import('@playwright/test').Page} page
 * @param {string} name
 * @param {string} s
 * @param {boolean} alsHaupt
 */
async function lieferantAnlegen(page, name, s, alsHaupt) {
	await page.getByLabel('Name').fill(name);
	await page.getByLabel('E-Mail').fill(`h${s}@example.invalid`);
	await page.getByLabel('Kundennummer').fill(`K-${s}`);
	if (alsHaupt) {
		// Ein Zustand, kein Auswahlpunkt: Die Oberfläche zeigt hier einen M3-Schalter
		// (role="switch"), kein Häkchen.
		await page.getByRole('switch', { name: 'Hauptlieferant der Schule' }).click();
	}
	await page.getByRole('button', { name: 'Lieferanten speichern' }).click();
}

// Der Testlieferant heisst absichtlich mit "Z": Die Liste kommt sonst mit ORDER BY name,
// und eine wirkungslose Vorauswahl fiele nicht auf, weil zufällig der richtige oben stünde.
test('Hauptlieferant: gespeichert, angezeigt, beim Bestellen vorausgewählt', async ({ page }) => {
	const s = uniqueSuffix();
	const name = `ZZZ-Hauptlieferant-${s}`;
	await uiLogin(page);
	await zuLieferanten(page);
	await lieferantAnlegen(page, name, s, true);

	// 1. Der Schalter erreicht die Datenbank — nicht nur die Oberfläche.
	await expect
		.poll(() => flagInDB(name), { timeout: 5000, message: 'Schalter muss gespeichert sein' })
		.toBe('t');

	// 2. Es ist GENAU einer. Die Invariante, auf der alles Weitere beruht.
	expect(
		querySQL(`SELECT count(*) FROM lieferanten WHERE ist_hauptlieferant`).trim(),
		'Es darf immer nur einen Hauptlieferanten geben'
	).toBe('1');

	// 3. Und die Liste benennt die Rolle.
	const zeile = page.locator('tr', { hasText: name });
	await expect(zeile.getByText('Hauptlieferant', { exact: true })).toBeVisible();

	// 4. Wer nur die E-Mail korrigiert, darf die Rolle nicht verlieren. Genau hier stand
	//    der Schalter früher immer auf „aus", weil die Bearbeiten-Maske ihn nicht aus dem
	//    Datensatz übernahm.
	await zeile.getByRole('button', { name: 'Bearbeiten' }).click();
	const bearbeiten = page.locator('tr', { hasText: /Speichern/ });
	await bearbeiten.locator('input[type="email"]').fill(`neu${s}@example.invalid`);
	await bearbeiten.getByRole('button', { name: 'Speichern' }).click();
	await expect
		.poll(() => flagInDB(name), {
			timeout: 5000,
			message: 'Rolle darf beim Bearbeiten nicht verloren gehen'
		})
		.toBe('t');

	// 5. Das Bestellformular übernimmt ihn — sonst wäre die Einstellung Zierrat.
	//
	// Bewusst nach einem Neuladen: Die Vorauswahl greift beim Aufbau des Bestellwesens,
	// nicht mitten in der Arbeit. Eine schon getroffene Auswahl bleibt stehen — sonst
	// wechselte jemandem, der gerade einen Warenkorb füllt, unter der Hand der Lieferant.
	//
	// Über die ROLLE gesucht, nicht über das Element: Die Lieferantenauswahl ist seit dem
	// 04.08.2026 kein natives <select> mehr, sondern die M3-Komponente
	// (button[role=combobox] + listbox). Der alte Selektor `select#supplier` prüfte die
	// Bauart statt das Verhalten und wurde beim Austausch rot, obwohl die Vorauswahl
	// weiterhin stimmte.
	await page.goto('/bestellungen');
	const auswahl = page.locator('#supplier[role="combobox"]');
	await auswahl.waitFor();
	await expect(
		auswahl,
		'Das Bestellformular muss den hinterlegten Hauptlieferanten vorauswählen'
	).toContainText(name);
});

test('Ohne den Schalter bleibt es bei der reinen Bestellmail', async ({ page }) => {
	const s = uniqueSuffix();
	const name = `E2E-NurMail-${s}`;
	await uiLogin(page);
	await zuLieferanten(page);
	await lieferantAnlegen(page, name, s, false);

	await expect.poll(() => flagInDB(name), { timeout: 5000 }).toBe('f');
	await expect(
		page.locator('tr', { hasText: name }).getByText('nur Bestellmail', { exact: true })
	).toBeVisible();
});
