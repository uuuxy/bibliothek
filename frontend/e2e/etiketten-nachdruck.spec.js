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

// Die Verbindung, die vorher fehlte: Der Bedarf entsteht im Bestellwesen ("30 Bücher sind
// da, aber ohne Etikett"), das Werkzeug steht im Druck-Center. Dazwischen lag nichts —
// der Hinweis nach dem Wareneingang war flüchtig, und war der Moment vorbei, erfuhr
// niemand je wieder davon.
test('Bestellwesen weist auf offene Etiketten hin und führt in die Liste', async ({ page }) => {
	const s = uniqueSuffix();
	const titel = `E2E-Hinweis-Titel ${s}`;

	seedSQL(`
		WITH t AS (
			INSERT INTO buecher_titel (titel, autor) VALUES ('${titel}', 'Testautorin') RETURNING id
		)
		INSERT INTO buecher_exemplare (titel_id, barcode_id, etikett_gedruckt, erworben_am)
		SELECT t.id, 'E2E-HIN-${s}', false, CURRENT_DATE FROM t;
	`);

	await uiLogin(page);
	await page.getByTitle('Bestellungen').click();

	// Der Hinweis steht dauerhaft, nicht nur direkt nach einem Wareneingang.
	const hinweis = page.getByRole('button', { name: 'Im Druck-Center nachdrucken' });
	await expect(hinweis).toBeVisible();

	// BEWEIS: Der Verweis landet nicht irgendwo im Druck-Center, sondern in der Liste.
	await hinweis.click();
	await expect(page.getByLabel('Exemplare filtern')).toBeVisible();

	await page.getByLabel('Exemplare filtern').fill(`E2E-HIN-${s}`);
	await expect(page.getByLabel(`${titel} (E2E-HIN-${s}) auswählen`)).toBeVisible();
});

// Zwei Druckwege mit verschiedenem Gedächtnis: Der A4-Bogen buchte gegen, der Einzeldruck
// aus der Buchakte nicht. Ein dort gedrucktes Etikett blieb damit für immer auf der Liste
// "Fehlende Etiketten" — sie wurde also ausgerechnet durch Benutzung unbrauchbar.
test('Einzeldruck aus der Buchakte bucht das Etikett gegen', async ({ page }) => {
	const s = uniqueSuffix();
	const barcode = `B-E2E${s.slice(0, 8)}`;

	seedSQL(`
		WITH t AS (
			INSERT INTO buecher_titel (titel, autor) VALUES ('E2E-Einzel-Titel ${s}', 'Testautorin') RETURNING id
		)
		INSERT INTO buecher_exemplare (titel_id, barcode_id, etikett_gedruckt, erworben_am)
		SELECT t.id, '${barcode}', false, CURRENT_DATE FROM t;
	`);

	const exemplarID = querySQL(`SELECT id FROM buecher_exemplare WHERE barcode_id = '${barcode}'`);
	await uiLogin(page);

	// Der Weg, den die Buchakte nimmt: ein GET auf die Etikett-Route.
	const res = await page.request.get(`/api/print/etikett/${exemplarID}`);
	expect(res.status(), 'Ersatz-Etikett-PDF').toBe(200);
	expect(res.headers()['content-type']).toContain('application/pdf');

	// BEWEIS an der Datenbank: Vorher blieb der Wert auf 'f'.
	await expect
		.poll(() =>
			querySQL(`SELECT etikett_gedruckt FROM buecher_exemplare WHERE id = '${exemplarID}'`)
		)
		.toBe('t');
});

// Die Verweise in der Bestellhistorie. Der Betreiber suchte den Nachdruck dort, weil dort
// der Anlass entsteht ("ich habe diese Titel bestellt, wo sind ihre Etiketten?").
//
// Bedingung: Der Verweis erscheint NUR, wenn es für den Titel offene Etiketten gibt. Ein
// Verweis, der eine leere Liste öffnet, entwertet alle anderen gleich mit.
test('Bestellhistorie verweist auf Nachdruck und Titelsatz — und nur, wenn es etwas zu drucken gibt', async ({
	page
}) => {
	const s = uniqueSuffix();
	const offenerTitel = `E2E-Histo-Offen ${s}`;
	const fertigerTitel = `E2E-Histo-Fertig ${s}`;

	// Zwei Positionen in EINER Bestellung: eine mit offenem Etikett, eine ohne.
	seedSQL(`
		WITH t1 AS (
			INSERT INTO buecher_titel (titel, autor) VALUES ('${offenerTitel}', 'A') RETURNING id
		), t2 AS (
			INSERT INTO buecher_titel (titel, autor) VALUES ('${fertigerTitel}', 'B') RETURNING id
		), e1 AS (
			INSERT INTO buecher_exemplare (titel_id, barcode_id, etikett_gedruckt, erworben_am)
			SELECT id, 'E2E-HO-${s}', false, CURRENT_DATE FROM t1
		), e2 AS (
			INSERT INTO buecher_exemplare (titel_id, barcode_id, etikett_gedruckt, erworben_am)
			SELECT id, 'E2E-HF-${s}', true, CURRENT_DATE FROM t2
		), b AS (
			INSERT INTO bestellungen_verlauf (lieferant_name, lieferant_email, kundennummer, bestelldatum, gesamtbetrag, anzahl_exemplare)
			VALUES ('E2E-Lieferant ${s}', 'e2e@example.org', 'K-${s}', CURRENT_TIMESTAMP, 0, 2) RETURNING id
		)
		INSERT INTO bestellungen_positionen (bestellung_id, titel_id, titel_name, isbn, menge, einzelpreis)
		SELECT b.id, t1.id, '${offenerTitel}', '', 1, 0 FROM b, t1
		UNION ALL
		SELECT b.id, t2.id, '${fertigerTitel}', '', 1, 0 FROM b, t2;
	`);

	await uiLogin(page);
	await page.getByTitle('Bestellungen').click();
	await page.getByRole('button', { name: 'Bestellhistorie', exact: true }).click();

	// Bestellung aufklappen
	await page.getByRole('button', { name: new RegExp(`E2E-Lieferant ${s}`) }).click();
	await expect(page.getByText(offenerTitel)).toBeVisible();

	// Der Titelsatz-Verweis steht bei BEIDEN Positionen.
	await expect(
		page.getByRole('button', { name: `Titelsatz von ${offenerTitel} öffnen` })
	).toBeVisible();
	await expect(
		page.getByRole('button', { name: `Titelsatz von ${fertigerTitel} öffnen` })
	).toBeVisible();

	// BEWEIS: Der Nachdruck-Verweis steht NUR bei der Position mit offenem Etikett.
	// Angesprochen wird der Knopf selbst, nicht seine Zeile: Die aufgeklappte Bestellzeile
	// enthaelt die gesamte Positionstabelle, ein row-Treffer waere also mehrdeutig.
	const nachdruckOffen = page.getByRole('button', {
		name: `Etiketten für ${offenerTitel} nachdrucken`
	});
	await expect(nachdruckOffen).toBeVisible();
	await expect(
		page.getByRole('button', { name: `Etiketten für ${fertigerTitel} nachdrucken` })
	).toHaveCount(0);

	// Und der Verweis landet in der Liste, bereits auf diesen Titel gefiltert.
	await nachdruckOffen.click();
	await expect(page.getByLabel('Exemplare filtern')).toHaveValue(offenerTitel);
	await expect(page.getByLabel(`${offenerTitel} (E2E-HO-${s}) auswählen`)).toBeVisible();
});

// Der Altbestand. etikett_gedruckt wurde bis vor Kurzem NIRGENDS gesetzt — für den
// gesamten Bestand steht deshalb "kein Etikett", nicht weil keins da wäre, sondern weil es
// nie jemand vermerkt hat. Ohne Aufräummöglichkeit zeigte diese Liste dauerhaft den
// ganzen Bestand, und der Hinweis im Bestellwesen nennte eine Zahl ohne Bedeutung.
//
// Bewusst KEINE Migration: Die hätte beim Update stillschweigend zugeschlagen und dabei
// genau die Exemplare mitversteckt, wegen denen die Liste überhaupt entstand. Der Stichtag
// gehört dem Betreiber — deshalb prüft der Test auch, dass NEUERES stehen bleibt.
test('Altbestand aufräumen vermerkt nur Exemplare bis zum Stichtag', async ({ page }) => {
	const s = uniqueSuffix();
	seedSQL(`
		WITH t AS (
			INSERT INTO buecher_titel (titel, autor) VALUES ('E2E-Alt-Titel ${s}', 'A') RETURNING id
		)
		INSERT INTO buecher_exemplare (titel_id, barcode_id, etikett_gedruckt, erworben_am)
		SELECT t.id, 'E2E-ALT-${s}', false, CURRENT_DATE - 400 FROM t
		UNION ALL
		SELECT t.id, 'E2E-NEU-${s}', false, CURRENT_DATE FROM t;
	`);

	await uiLogin(page);
	await page.getByTitle('Druck-Center').click();
	await page.getByRole('button', { name: 'Fehlende Etiketten' }).click();

	await page.getByRole('group').filter({ hasText: 'Altbestand aufräumen' }).click();
	// Stichtag: gestern — trifft das alte Exemplar, nicht das heutige.
	const gestern = new Date(Date.now() - 86400000).toISOString().slice(0, 10);
	await page.locator('#stichtag').fill(gestern);
	await page.locator('#stichtag').dispatchEvent('change');

	// Die Zahl steht VOR der Bestätigung — die Aktion ist nicht umkehrbar.
	const knopf = page.getByRole('button', { name: /Exemplare als erledigt vermerken/ });
	await expect(knopf).toBeEnabled();
	await knopf.click();

	// BEWEIS an der Datenbank: das alte vermerkt, das neue unberührt.
	await expect
		.poll(() =>
			querySQL(`SELECT etikett_gedruckt FROM buecher_exemplare WHERE barcode_id = 'E2E-ALT-${s}'`)
		)
		.toBe('t');
	expect(
		querySQL(`SELECT etikett_gedruckt FROM buecher_exemplare WHERE barcode_id = 'E2E-NEU-${s}'`)
	).toBe('f');
});

// Der Weg zurück — beide Richtungen von Hand, ohne zu drucken.
//
// Vorher setzten alle drei Wege das Kennzeichen nur in EINE Richtung: Stapeldruck,
// Einzeldruck aus der Buchakte und das Altbestand-Aufräumen. Blieb der Bogen im Drucker
// stecken oder war der Stichtag zu weit gefasst, galten Exemplare als erledigt, ohne dass
// ein Etikett am Buch klebte — und sie waren dauerhaft aus der Liste verschwunden.
//
// Belegt wird am Datenbankzustand, nicht an der Meldung: Ob ein Exemplar wieder auf der
// Liste erscheint, entscheidet allein etikett_gedruckt.
test('Etiketten von Hand vermerken und wieder öffnen', async ({ page }) => {
	const s = uniqueSuffix();
	const titel = `E2E-Hand ${s}`;
	seedSQL(`
		WITH t AS (INSERT INTO buecher_titel (titel, autor) VALUES ('${titel}', 'Hand') RETURNING id)
		INSERT INTO buecher_exemplare (titel_id, barcode_id, etikett_gedruckt, erworben_am)
		SELECT id, 'E2E-HAND-${s}', false, CURRENT_DATE FROM t;
	`);

	const flag = () =>
		querySQL(
			`SELECT etikett_gedruckt FROM buecher_exemplare WHERE barcode_id = 'E2E-HAND-${s}'`
		).trim();
	expect(flag(), 'Ausgangslage: Etikett steht aus').toBe('f');

	await uiLogin(page);
	await page.getByTitle('Druck-Center').click();
	await page.getByRole('button', { name: /Fehlende Etiketten/ }).click();

	// 1. Von Hand als erledigt vermerken — ohne Druck.
	await page.getByPlaceholder(/Nach Titel oder Barcode filtern/).fill(titel);
	const zeile = page.locator('tr', { hasText: titel });
	await zeile.first().waitFor();
	await zeile.getByRole('checkbox').check();
	await page.getByRole('button', { name: /als erledigt vermerken/ }).click();
	await expect.poll(flag, { timeout: 5000, message: 'Vermerk muss ankommen' }).toBe('t');

	// Und damit ist es aus der Liste „Offen" verschwunden.
	await expect(page.locator('tr', { hasText: titel })).toHaveCount(0);

	// 2. Der Notfallweg: wieder öffnen. Ohne die Ansicht „Erledigt" wäre das Exemplar
	//    unerreichbar — man kann nur zurückholen, was man sehen kann.
	await page.getByRole('button', { name: 'Erledigt', exact: true }).click();
	const zeile2 = page.locator('tr', { hasText: titel });
	await zeile2.first().waitFor();
	await zeile2.getByRole('checkbox').check();
	await page.getByRole('button', { name: /wieder als offen vermerken/ }).click();
	await expect.poll(flag, { timeout: 5000, message: 'Zurücksetzen muss ankommen' }).toBe('f');
});
