// Der Klassensatz-Dialog: bisher ohne jede e2e-Abdeckung.
//
// Aufgefallen ist die Lücke beim Vereinheitlichen der Suchfelder (10.08.2026). Der Dialog
// trägt das letzte handgebaute Suchfeld des Projekts, und eine Änderung daran wäre von
// keinem Test bemerkt worden: Der Rundgang über alle Routen erreicht nur Bildschirme, keine
// Dialoge, und klassen-ein-ort.spec.js legt Klassensätze über die API an, nicht über die
// Oberfläche.
//
// Der Test geht deshalb den Weg, den eine Bibliothekskraft geht: Schulklassen öffnen,
// „Klasse hinzufügen", Zielklasse eintippen, im Dialog nach einem Buch SUCHEN, es
// auswählen, speichern — und danach nachsehen, ob der Satz wirklich in der Datenbank steht.
//
// Der Kern ist die Suche: Ohne sie findet man in einem Bestand von mehreren tausend Titeln
// kein einziges Buch, der Dialog ist dann unbenutzbar. Genau deshalb prüft der Test nicht
// „das Feld ist da", sondern dass es die Kachelmenge tatsächlich eingrenzt.
import { test, expect } from '@playwright/test';
import { uiLogin, csrfToken, querySQL, uniqueSuffix } from './helpers.js';

const s = uniqueSuffix();
const KLASSE = `zz${s}`.slice(0, 12);

test.afterAll(async ({ request }) => {
	// Nur die eine angelegte Zuordnung zurücknehmen. Kein Rundumschlag: Ein Teardown hat
	// in diesem Projekt schon einmal echte Konfiguration mitgenommen.
	await request
		.delete(`/api/admin/class-books?className=${KLASSE.toUpperCase()}`, {
			failOnStatusCode: false
		})
		.catch(() => {});
});

test('Klassensatz über den Dialog anlegen — Suche grenzt ein, Auswahl landet in der Datenbank', async ({
	page
}) => {
	await uiLogin(page);

	// Einen Titel aus dem Bestand nehmen, statt einen zu erfinden: Der Dialog zeigt nur,
	// was es wirklich gibt.
	const buecher = await (await page.request.get('/api/books')).json();
	const titel = (buecher.data ?? [])[0]?.title;
	expect(titel, 'Testdaten: mindestens ein Buch nötig').toBeTruthy();

	await page.goto('/schulklassen');
	await page.getByRole('button', { name: 'Klasse hinzufügen' }).click();

	// Über die id und nicht über den Platzhalter: Dieser Test entstand unmittelbar VOR dem
	// Umbau des Feldes auf das gemeinsame Bauteil. Nur ein Selektor, der beide Fassungen
	// trifft, kann belegen, dass der Umbau am Verhalten nichts ändert.
	const suchfeld = page.locator('#book-search-field');
	await expect(suchfeld, 'der Dialog muss ein Suchfeld haben').toBeVisible();

	// Auch im Dialog dieselbe Suchpille wie auf den Bildschirmen. Der Rundgang und
	// suchpille-einheitlich.spec.js erreichen nur Routen, keine Dialoge — ohne diese zwei
	// Zeilen könnte genau hier wieder eine eigene Fassung entstehen.
	const pille = await suchfeld.evaluate((el) => {
		const p = /** @type {HTMLElement} */ (el.parentElement);
		const s = getComputedStyle(p);
		return { hoehe: Math.round(p.getBoundingClientRect().height), radius: s.borderRadius };
	});
	expect(pille.hoehe, 'die Suchpille im Dialog misst 48 px wie überall').toBe(48);
	expect(Math.round(parseFloat(pille.radius)), 'rund, nicht eckig').toBeGreaterThanOrEqual(24);

	// Vor der Suche: der ganze Bestand. Die Zahl wird gleich als Vergleich gebraucht —
	// ohne sie wäre „nach der Suche sind es weniger" keine Aussage.
	//
	// Gewartet werden MUSS: Der Dialog holt die Bücher erst beim Mounten über /api/books.
	// Ein sofortiges count() liefert 0 und liesse den Test aus dem falschen Grund scheitern.
	const kacheln = page.locator('[aria-pressed]');
	await expect.poll(() => kacheln.count(), { timeout: 20000 }).toBeGreaterThan(1);
	const vorher = await kacheln.count();

	await suchfeld.fill(titel);
	await expect.poll(() => kacheln.count(), { timeout: 5000 }).toBeLessThan(vorher);
	const treffer = await kacheln.count();
	expect(treffer, 'die Suche darf den gesuchten Titel nicht wegfiltern').toBeGreaterThan(0);

	// Zielklasse eintragen und auswählen.
	await page.locator('#class-input').fill(KLASSE);
	await page.locator('#class-input').press('Enter');
	await kacheln.first().click();

	await page.getByRole('button', { name: 'Speichern' }).click();

	// Am Ergebnis prüfen, nicht an der Oberfläche: Ein Dialog, der sich schliesst, sagt
	// nichts darüber, ob etwas gespeichert wurde.
	//
	// upper() auf beiden Seiten, weil die Anwendung Klassennamen auf Grossbuchstaben
	// normalisiert („zz…" wird zu „ZZ…"). Ohne das verglich der Test an der Wirklichkeit
	// vorbei und meldete 0, obwohl die Zeile längst dastand.
	await expect
		.poll(
			() =>
				querySQL(`SELECT count(*) FROM class_books WHERE upper(class_name) = upper('${KLASSE}');`),
			{ timeout: 10000 }
		)
		.toBe('1');
});
