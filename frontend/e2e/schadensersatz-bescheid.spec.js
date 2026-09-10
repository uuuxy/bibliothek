import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix, gehZu } from './helpers.js';

// Der Weg zum Schadensersatz-Bescheid, wie ihn das Sekretariat geht: im Mahnwesen einen
// Schüler markieren, den Knopf drücken, im Dialog prüfen, erstellen.
//
// Warum als E2E und nicht nur als PG-Test: Die Entscheidung, WO der Bescheid entsteht,
// war der Kern der Absprache (Auswahlleiste bei genau einem markierten Schüler, vierter
// Reiter für die Briefe). Ob der Knopf dort wirklich erscheint und der Dialog seine
// Vorschläge zeigt, sieht kein Go-Test — nur der Browser.

// Kurzer Suffix: barcode_id und isbn sind varchar(20), ein voller uniqueSuffix() sprengt
// sie. Die letzten sechs Zeichen genügen für einen parallelen Lauf.
const suffix = uniqueSuffix().slice(-6);
const NACHNAME = `Bescheidkind${suffix}`;

test.describe('Schadensersatz-Bescheid', () => {
	test.beforeAll(() => {
		// Pflichtangaben, ohne die kein Bescheid entsteht — einschließlich der
		// Schul-Anschrift, die im Brief das Absender- und das Widerspruchsfeld füllt.
		// Ohne sie zeigt der Dialog seine Fehlliste (siehe zweiter Test).
		seedSQL(`
			INSERT INTO system_einstellungen (schluessel, wert) VALUES
				('bescheid_bereich_nr', '5830'),
				('bescheid_schulnummer', '1234'),
				('bescheid_aufsicht', 'Staatliches Schulamt, Musterstraße 1, 12345 Musterstadt'),
				('bescheid_schulleitung', 'Dr. Beispiel, Schulleitung'),
				('schule_name', 'Testschule'),
				('schule_strasse', 'Schulweg 1'),
				('schule_plz', '61381'),
				('schule_ort', 'Musterstadt')
			ON CONFLICT (schluessel) DO UPDATE SET wert = EXCLUDED.wert;`);

		// Ein Kind mit einer überfälligen Ausleihe UND einer offenen Forderung darauf.
		seedSQL(`
			WITH s AS (
				INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, strasse, hausnummer, plz, ort)
				VALUES ('S-BESCH-${suffix}', 'Test', '${NACHNAME}', '08G2', 0, 'Musterweg', '12', '61381', 'Musterstadt')
				RETURNING id
			), t AS (
				INSERT INTO buecher_titel (titel, isbn, ist_lernmittel, meldebestand)
				VALUES ('LMF-Bescheidbuch ${suffix}', '978-3-1${suffix}', true, 1)
				RETURNING id
			), e AS (
				INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, einkaufspreis, erworben_am)
				SELECT t.id, 'BESCH-EX-${suffix}', true, 24.90, CURRENT_DATE - INTERVAL '2 years' FROM t
				RETURNING id
			), a AS (
				INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist)
				SELECT e.id, s.id, CURRENT_DATE - 60, CURRENT_DATE - 30 FROM e, s
				RETURNING id
			)
			INSERT INTO schadensfaelle (exemplar_id, schueler_id, ausleihe_id, beschreibung, betrag, art)
			SELECT e.id, s.id, a.id, 'Bescheidbuch nicht zurückgegeben', 0, 'nicht_zurueckgegeben'
			FROM e, s, a;`);
	});

	test.afterAll(() => {
		// Reihenfolge von innen nach außen: Forderungen und Ausleihen hängen mit RESTRICT
		// am Schüler, der Bescheid mit SET NULL. Ohne diese Reihenfolge bricht das
		// Aufräumen ab und lässt Testdaten in der Anlage zurück.
		seedSQL(`
			DELETE FROM schadensersatz_bescheide WHERE schueler_id IN
				(SELECT id FROM schueler WHERE nachname = '${NACHNAME}');
			DELETE FROM schadensfaelle WHERE schueler_id IN
				(SELECT id FROM schueler WHERE nachname = '${NACHNAME}');
			DELETE FROM ausleihen WHERE schueler_id IN
				(SELECT id FROM schueler WHERE nachname = '${NACHNAME}');
			DELETE FROM schueler WHERE nachname = '${NACHNAME}';
			DELETE FROM buecher_exemplare WHERE barcode_id = 'BESCH-EX-${suffix}';
			DELETE FROM buecher_titel WHERE titel LIKE 'LMF-Bescheidbuch ${suffix}%';`);
	});

	test('entsteht aus der Auswahlleiste und landet im vierten Reiter', async ({ page }) => {
		await uiLogin(page);
		await gehZu(page, '/mahnwesen');

		// Der Reiter „Bescheide" steht in der Reihe nach Dringlichkeit.
		await expect(page.getByRole('tab', { name: /Bescheide/ })).toBeVisible();

		// Ohne Markierung gibt es den Knopf nicht — der Bescheid ist ein Einzelfall.
		await expect(page.getByRole('button', { name: /Schadensersatz-Bescheid/ })).toHaveCount(0);

		// Genau EINEN Schüler markieren.
		await page.getByRole('checkbox', { name: new RegExp(`${NACHNAME}.*auswählen`) }).click();
		const knopf = page.getByRole('button', { name: /Schadensersatz-Bescheid/ });
		await expect(knopf).toBeVisible();
		await knopf.click();

		// Der Dialog zeigt die offene Forderung mit Vorschlag und Herleitung.
		const dialog = page.getByRole('dialog');
		await expect(dialog.getByRole('heading', { name: /Schadensersatz-Bescheid/ })).toBeVisible();
		await expect(dialog.getByText(/Verleihjahr/)).toBeVisible();
		await expect(dialog.getByText(/Nicht ordnungsgemäß zurückgegeben/)).toBeVisible();

		await dialog.getByRole('button', { name: 'Bescheid erstellen' }).click();

		// Die Bestätigung nennt die Referenznummer.
		await expect(page.getByText(/Bescheid 5830 /)).toBeVisible({ timeout: 10000 });

		// Und der Brief steht im vierten Reiter. Gesucht wird in der ZEILE des Kindes:
		// Die Referenznummer steht auch in der Bestätigungsmeldung, ein Textabgleich über
		// die ganze Seite träfe beide.
		await page.getByRole('tab', { name: /Bescheide/ }).click();
		const zeile = page.getByRole('row', { name: new RegExp(NACHNAME) });
		await expect(zeile).toBeVisible();
		await expect(zeile).toContainText(/5830 \d{4} 1234 \d{4}/);
		await expect(zeile.getByRole('button', { name: 'Nachdruck' })).toBeVisible();
		// Der Status ist „offen": Die Frist läuft, die Übergabe ist noch nicht möglich.
		await expect(zeile).toContainText('offen');
		await expect(zeile.getByRole('button', { name: 'Übergeben' })).toHaveCount(0);

		// Die Datenbank trägt genau einen Brief mit einer Position. querySQL liefert die
		// Ausgabe von psql als Text (Felder mit „|" getrennt), nicht als Zeilenliste.
		const zeilen = querySQL(`
			SELECT b.referenznummer,
			       (SELECT count(*) FROM schadensfaelle f WHERE f.bescheid_id = b.id)
			FROM schadensersatz_bescheide b
			JOIN schueler s ON s.id = b.schueler_id
			WHERE s.nachname = '${NACHNAME}';`)
			.split('\n')
			.filter((z) => z.trim() !== '');
		expect(zeilen.length, 'genau ein Bescheid').toBe(1);
		const [nummer, positionen] = zeilen[0].split('|');
		expect(nummer, 'Referenznummer im Format des Musterschreibens').toMatch(
			/^5830 \d{4} 1234 \d{4}$/
		);
		expect(positionen, 'eine Position auf dem Brief').toBe('1');
	});
});
