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
// Zweites Kind für Stufe 2: ein überfälliges Buch, KEINE Forderung — der Brief bucht den
// Verlust selbst.
const VERLUSTKIND = `Verlustkind${suffix}`;

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

		seedSQL(`
			WITH s AS (
				INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, strasse, hausnummer, plz, ort)
				VALUES ('S-VERL-${suffix}', 'Test', '${VERLUSTKIND}', '08G2', 0, 'Musterweg', '14', '61381', 'Musterstadt')
				RETURNING id
			), t AS (
				INSERT INTO buecher_titel (titel, isbn, ist_lernmittel, meldebestand)
				VALUES ('LMF-Verlustbuch ${suffix}', '978-3-2${suffix}', true, 1)
				RETURNING id
			), e AS (
				INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, einkaufspreis, erworben_am)
				SELECT t.id, 'VERL-EX-${suffix}', true, 24.90, CURRENT_DATE - INTERVAL '2 years' FROM t
				RETURNING id
			)
			INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist)
			SELECT e.id, s.id, CURRENT_DATE - 60, CURRENT_DATE - 30 FROM e, s;`);
	});

	test.afterAll(() => {
		// Reihenfolge von innen nach außen: Forderungen und Ausleihen hängen mit RESTRICT
		// am Schüler, der Bescheid mit SET NULL. Ohne diese Reihenfolge bricht das
		// Aufräumen ab und lässt Testdaten in der Anlage zurück.
		seedSQL(`
			DELETE FROM schadensersatz_bescheide WHERE schueler_id IN
				(SELECT id FROM schueler WHERE nachname IN ('${NACHNAME}', '${VERLUSTKIND}'));
			DELETE FROM schadensfaelle WHERE schueler_id IN
				(SELECT id FROM schueler WHERE nachname IN ('${NACHNAME}', '${VERLUSTKIND}'));
			DELETE FROM ausleihen WHERE schueler_id IN
				(SELECT id FROM schueler WHERE nachname IN ('${NACHNAME}', '${VERLUSTKIND}'));
			DELETE FROM schueler WHERE nachname IN ('${NACHNAME}', '${VERLUSTKIND}');
			DELETE FROM buecher_exemplare WHERE barcode_id IN ('BESCH-EX-${suffix}', 'VERL-EX-${suffix}');
			DELETE FROM buecher_titel WHERE titel LIKE 'LMF-Bescheidbuch ${suffix}%'
			   OR titel LIKE 'LMF-Verlustbuch ${suffix}%';`);
	});

	test('steht als Forderung im vierten Reiter, entsteht aus der Auswahlleiste und bleibt dort', async ({
		page
	}) => {
		await uiLogin(page);
		await gehZu(page, '/mahnwesen');

		// Der Reiter „Schadensersatz" steht in der Reihe nach Dringlichkeit — und zeigt das
		// Kind SCHON VOR dem Brief: Seit dem 15.09.2026 hat die Stufe „Forderung offen, noch
		// kein Bescheid" eine eigene Zeile. Vorher war ein Kind mit gemeldetem Verlust
		// nirgends zu sehen, sobald die Mahnliste neu geladen war (ReportDamage beendet die
		// Ausleihe).
		await expect(page.getByRole('tab', { name: /Schadensersatz/ })).toBeVisible();
		await page.getByRole('tab', { name: /Schadensersatz/ }).click();
		const wartend = page.getByRole('row', { name: new RegExp(NACHNAME) });
		await expect(wartend).toContainText('Bescheid noch nicht erstellt');
		await expect(wartend.getByRole('button', { name: 'Bescheid erstellen' })).toBeVisible();

		// Zweite Tür: der Name führt in die Akte, und dort steht der Knopf an der
		// Gebühren-Karte — bei der Forderung, aus der der Brief entsteht.
		await wartend.getByRole('button', { name: /Akte von .* öffnen/ }).click();
		await expect(page.getByRole('button', { name: /Bescheid erstellen/ })).toBeVisible();
		await gehZu(page, '/mahnwesen');

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
		await page.getByRole('tab', { name: /Schadensersatz/ }).click();
		const zeile = page.getByRole('row', { name: new RegExp(NACHNAME) });
		await expect(zeile).toBeVisible();
		await expect(zeile).toContainText(/5830 \d{4} 1234 \d{4}/);
		await expect(zeile.getByRole('button', { name: 'Nachdruck' })).toBeVisible();
		// Der Stand nennt den nächsten Schritt: Die Frist läuft, die Übergabe ist noch nicht
		// möglich — und die Zeile „noch nicht erstellt" ist weg, es gibt genau eine.
		await expect(zeile).toContainText('Frist läuft');
		await expect(zeile.getByRole('button', { name: 'Übergeben' })).toHaveCount(0);
		await expect(page.getByRole('row', { name: new RegExp(NACHNAME) })).toHaveCount(1);

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

	test('entsteht aus dem überfälligen Buch, das der Brief als Verlust bucht', async ({ page }) => {
		await uiLogin(page);
		await gehZu(page, '/mahnwesen');

		// Stufe 2 (15.09.2026): Das Kind steht mit seinem überfälligen Buch in der
		// Mahnliste und hat KEINE Forderung. Der Knopf öffnet den Dialog trotzdem mit
		// Inhalt — vorher war er hier eine leere Tür.
		await page.getByRole('checkbox', { name: new RegExp(`${VERLUSTKIND}.*auswählen`) }).click();
		await page.getByRole('button', { name: /Schadensersatz-Bescheid/ }).click();

		const dialog = page.getByRole('dialog');
		await expect(dialog.getByText(`LMF-Verlustbuch ${suffix}`)).toBeVisible();
		await expect(
			dialog.getByText(/Fällig seit .* wird mit dem Brief als Verlust gebucht/)
		).toBeVisible();
		await expect(
			dialog.getByRole('checkbox', { name: /LMF-Verlustbuch .* in den Bescheid aufnehmen/ })
		).toBeChecked();
		await dialog.getByRole('button', { name: 'Bescheid erstellen' }).click();
		await expect(page.getByText(/Bescheid 5830 /)).toBeVisible({ timeout: 10000 });

		// Die Ausleihe ist beendet: Das Kind ist aus der Mahnliste verschwunden und steht
		// im Reiter „Schadensersatz" mit laufender Frist.
		await expect(
			page.getByRole('checkbox', { name: new RegExp(`${VERLUSTKIND}.*auswählen`) })
		).toHaveCount(0);
		await page.getByRole('tab', { name: /Schadensersatz/ }).click();
		const zeile = page.getByRole('row', { name: new RegExp(VERLUSTKIND) });
		await expect(zeile).toContainText('Frist läuft');

		// Papier == Datenbank: Ausleihe beendet, Exemplar VERLUST, Forderung „nicht
		// zurückgegeben" am Brief mit der Referenznummer in der Beschreibung.
		const [beendet, grund, art, amBrief, beschreibung] = querySQL(`
			SELECT a.rueckgabe_am IS NOT NULL, e.aussonderung_grund, f.art, f.bescheid_id IS NOT NULL,
			       f.beschreibung
			FROM ausleihen a
			JOIN buecher_exemplare e ON e.id = a.exemplar_id
			JOIN schadensfaelle f ON f.ausleihe_id = a.id
			WHERE e.barcode_id = 'VERL-EX-${suffix}';`)
			.trim()
			.split('|');
		expect(beendet, 'Ausleihe beendet').toBe('t');
		expect(grund, 'Exemplar als Verlust ausgesondert').toBe('VERLUST');
		expect(art, 'Fallgruppe des Briefs').toBe('nicht_zurueckgegeben');
		expect(amBrief, 'Forderung hängt am Brief').toBe('t');
		expect(beschreibung, 'Referenznummer in der Forderung').toMatch(
			/Bescheid 5830 \d{4} 1234 \d{4}/
		);
	});
});
