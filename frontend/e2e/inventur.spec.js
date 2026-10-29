import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix, menuepunkt } from './helpers.js';

// Inventur-Ablauf: starten (Signatur-Scope!) → scannen → abschließen.
// WICHTIG: Der Test nutzt bewusst NUR den Signatur-Scope — ein globaler
// Lauf würde auf einer geteilten DB alle nicht gescannten Exemplare als
// verloren aussondern. Der Signatur-Scope markiert nur Titel der eigenen
// Test-Signatur als 'ausstehend'; finish trifft nur diese.
test('Inventur: Signatur-Scope, gescannt bleibt, ungescannt wird Verlust', async ({ page }) => {
	await uiLogin(page);
	const suffix = uniqueSuffix();
	const sigName = `E2E-INV-${suffix}`;

	try {
		// Die Signatur steht als TEXT am Titel (buecher_titel.signatur) — sie ist das,
		// was physisch auf dem Buchrücken klebt. Der frühere Fremdschlüssel auf die
		// Tabelle `signatures` ist mit Migration 060 entfallen: Er wurde nie gepflegt,
		// weshalb die Signatur-Inventur in Wahrheit null Exemplare traf.
		seedSQL(`
            WITH t AS (
                INSERT INTO buecher_titel (titel, signatur)
                VALUES ('E2E-Inventurbuch-${suffix}', '${sigName}') RETURNING id
            )
            INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
            SELECT id, b, true FROM t, unnest(ARRAY['B-INVA-${suffix}', 'B-INVB-${suffix}']) AS b;
        `);

		await menuepunkt(page, 'Inventur').click();
		const neu = page.getByRole('button', { name: 'Neue Bestandsprüfung starten' });
		await neu.click();

		// Escape schließt den Dialog, und der Knopf öffnet ihn danach wieder.
		const startTitel = page.getByRole('heading', { name: 'Bereich der Inventur wählen' });
		await expect(startTitel).toBeVisible();
		await page.keyboard.press('Escape');
		await expect(startTitel).toBeHidden();
		await neu.click();
		await expect(startTitel).toBeVisible();

		// Scope: nur die Test-Signatur. Ohne Signatur lässt sich nicht starten.
		const starten = page.getByRole('button', { name: 'Inventur Starten' });
		await page.getByText('Nur bestimmte Signatur').click();
		await expect(starten).toBeDisabled();
		await page.getByLabel('Signatur auswählen').fill(sigName);
		await starten.click();

		// Exemplar A scannen → als erfasst bestätigt
		const scan = page.getByPlaceholder('Barcode scannen...');
		await expect(scan).toBeVisible();
		await scan.fill(`B-INVA-${suffix}`);
		await scan.press('Enter');
		await expect(page.getByText(`E2E-Inventurbuch-${suffix}`).first()).toBeVisible();

		// Abschließen → Exemplar B (nie gescannt) wird als Verlust ausgesondert. Die Rückfrage
		// nennt die Zahl; Escape ist „nein", die Inventur läuft weiter, und der Knopf fragt
		// danach wieder.
		const abschliessen = page.getByRole('button', { name: 'Inventur abschließen' });
		const frage = page.getByRole('heading', { name: 'Inventur abschließen?' });
		await abschliessen.click();
		await expect(frage).toBeVisible();
		await expect(page.getByText(/\b1\s+B(uch|ücher)\b/)).toBeVisible();
		await page.keyboard.press('Escape');
		await expect(frage).toBeHidden();
		// Der Fokus steht wieder im Scanfeld: Der nächste Scan wird gezählt.
		await expect(scan).toBeFocused();
		await abschliessen.click();
		await expect(frage).toBeVisible();
		await page.getByRole('button', { name: 'Ja, unwiderruflich abschließen' }).click();
		await expect(neu).toBeVisible();

		// Der Fehlbestandsbericht steht da — und nennt das FEHLENDE Buch, nicht nur eine Zahl.
		//
		// Vorher endete die Inventur mit „1 Bücher wurden als verloren markiert" im Toast,
		// der nach Sekunden verschwand. Damit kann niemand ins Regal gehen und nachsehen, ob
		// das Buch nur falsch einsortiert war. Rekonstruieren liess sich die Liste auch
		// nicht: Durch die Aussonderung fallen die Exemplare aus dem Scope, nach dem
		// gerechnet wird.
		const bericht = page.getByRole('heading', { name: /Fehlbestand/ });
		await expect(bericht, 'Nach dem Abschluss muss der Fehlbestand sichtbar sein').toBeVisible();
		await expect(
			page.getByText(`B-INVB-${suffix}`),
			'das nicht gescannte Exemplar gehoert in den Bericht'
		).toBeVisible();
		await expect(
			page.getByText(`B-INVA-${suffix}`),
			'das gescannte Exemplar darf NICHT im Bericht stehen'
		).toHaveCount(0);
		// Die Signatur traegt die Sortierung — ohne sie ist die Liste im Regal unbrauchbar.
		await expect(page.getByText(sigName).first()).toBeVisible();

		// Und er bleibt stehen, bis er ausdruecklich geschlossen wird.
		await page.getByRole('button', { name: 'Fehlbestandsbericht schließen' }).click();
		await expect(bericht).toHaveCount(0);

		// DB-Beweis: A unangetastet, B ausgesondert mit Inventur-Notiz
		expect(
			querySQL(
				`SELECT ist_ausgesondert FROM buecher_exemplare WHERE barcode_id = 'B-INVA-${suffix}'`
			)
		).toBe('f');
		expect(
			querySQL(
				`SELECT ist_ausgesondert || '|' || zustand_notiz FROM buecher_exemplare WHERE barcode_id = 'B-INVB-${suffix}'`
			)
		).toBe('true|Verlust bei Inventur');
	} finally {
		seedSQL(`
            DELETE FROM buecher_exemplare WHERE barcode_id IN ('B-INVA-${suffix}', 'B-INVB-${suffix}');
            DELETE FROM buecher_titel WHERE titel = 'E2E-Inventurbuch-${suffix}';
        `);
	}
});

// Eine verworfene Inventur steht unter „Frühere Inventuren" als verworfen, nicht als
// „vollständig", und bietet keinen Fehlbestand an (docs/OFFEN.md 5.32, Migration 149).
// „Verwerfen" schrieb dieselben Spalten wie ein Abschluss ohne Verlust; wer die Liste las,
// hielt den Bereich für geprüft.
test('Inventur: verworfen steht nicht als vollständig in der Liste', async ({ page }) => {
	await uiLogin(page);
	const suffix = uniqueSuffix();
	const sigName = `E2E-VERW-${suffix}`;
	const label = `Signatur ${sigName}`;

	try {
		seedSQL(`
            WITH t AS (
                INSERT INTO buecher_titel (titel, signatur)
                VALUES ('E2E-Verwerfbuch-${suffix}', '${sigName}') RETURNING id
            )
            INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
            SELECT id, b, true FROM t, unnest(ARRAY['B-VERWA-${suffix}', 'B-VERWB-${suffix}']) AS b;
        `);

		await menuepunkt(page, 'Inventur').click();
		await page.getByRole('button', { name: 'Neue Bestandsprüfung starten' }).click();
		await page.getByText('Nur bestimmte Signatur').click();
		await page.getByLabel('Signatur auswählen').fill(sigName);
		await page.getByRole('button', { name: 'Inventur Starten' }).click();

		const scan = page.getByPlaceholder('Barcode scannen...');
		await expect(scan).toBeVisible();
		await scan.fill(`B-VERWA-${suffix}`);
		await scan.press('Enter');
		await expect(page.getByText(`E2E-Verwerfbuch-${suffix}`).first()).toBeVisible();

		// Nach dem Neuladen steht die Inventur unter „Laufende Inventuren" — dort wird verworfen.
		// In beiden Listen steht die Beschriftung im Textblock (Name und Zeile darunter), der
		// Textblock in der Zeile mit den Knöpfen.
		await page.reload();
		const beschriftung = page.getByText(label, { exact: true });
		const text = beschriftung.locator('..');
		const zeile = beschriftung.locator('../..');

		// Erst die Rückfrage: „Abbrechen" lässt die Inventur laufen, „Verwerfen" beendet sie.
		const rueckfrage = page.getByRole('dialog', { name: `Inventur „${label}“ verwerfen?` });
		await zeile.getByRole('button', { name: 'Verwerfen' }).click();
		await rueckfrage.getByRole('button', { name: 'Abbrechen' }).click();
		await expect(rueckfrage).toHaveCount(0);
		await expect(zeile.getByRole('button', { name: 'Fortsetzen' })).toBeVisible();
		expect(
			querySQL(
				`SELECT abgeschlossen_am IS NULL FROM inventur_sessions WHERE scope_label = '${label}'`
			)
		).toBe('t');
		await zeile.getByRole('button', { name: 'Verwerfen' }).click();
		await rueckfrage.getByRole('button', { name: 'Verwerfen' }).click();

		// Ohne Neuladen in der Liste früherer Inventuren, als verworfen und ohne Bericht.
		await expect(text).toContainText(/1 erfasst\s*·\s*verworfen/);
		await expect(text).not.toContainText('vollständig');
		await expect(zeile.getByRole('button', { name: 'Fehlbestand' })).toHaveCount(0);

		expect(querySQL(`SELECT verworfen FROM inventur_sessions WHERE scope_label = '${label}'`)).toBe(
			't'
		);
		// Verworfen bucht keinen Verlust: Das ungescannte Buch bleibt im Umlauf.
		expect(
			querySQL(
				`SELECT ist_ausgesondert FROM buecher_exemplare WHERE barcode_id = 'B-VERWB-${suffix}'`
			)
		).toBe('f');
	} finally {
		seedSQL(`
            DELETE FROM inventur_sessions WHERE scope_label = '${label}';
            DELETE FROM buecher_exemplare WHERE barcode_id IN ('B-VERWA-${suffix}', 'B-VERWB-${suffix}');
            DELETE FROM buecher_titel WHERE titel = 'E2E-Verwerfbuch-${suffix}';
        `);
	}
});

// Unter „Frühere Inventuren" standen nur die zehn jüngsten; ältere waren samt
// Fehlbestandsbericht nicht mehr erreichbar (docs/OFFEN.md 5.112). „Mehr laden" holt sie, und eine
// frühere Inventur lässt sich nach der Rückfrage löschen.
test('Inventur: ältere frühere Inventuren über „Mehr laden“, eine davon löschen', async ({
	page
}) => {
	await uiLogin(page);
	const praefix = `E2E-ALT-${uniqueSuffix()}`;
	const aeltester = `${praefix}-1`;

	try {
		// Elf abgeschlossene Läufe im Jahr 2001: älter als alles, was andere Tests hinterlassen;
		// sie stehen also hinter der ersten Seite.
		seedSQL(`
            INSERT INTO inventur_sessions (scope_type, scope_signatur, scope_label, gestartet_am,
                                           abgeschlossen_am, erfasst_gemeldet, verloren_gemeldet)
            SELECT 'signature', '${praefix}-' || i, '${praefix}-' || i,
                   timestamptz '2001-01-01' + i * interval '1 day',
                   timestamptz '2001-01-01' + i * interval '1 day' + interval '1 hour', 0, 0
            FROM generate_series(1, 11) AS i;
        `);

		await menuepunkt(page, 'Inventur').click();
		const zeile = page.getByText(aeltester, { exact: true });
		const zeilen = page.getByRole('button', { name: /^Aktionen für „/ });
		const mehr = page.getByRole('button', { name: /^Mehr laden \(\d+ weitere\)$/ });
		await expect(zeilen.first()).toBeVisible();
		// Jeder Klick holt zehn; wie viele Seiten davor liegen, hängt an den Läufen anderer Tests.
		for (let i = 0; i < 50 && !(await zeile.isVisible()); i++) {
			const vorher = await zeilen.count();
			await mehr.click();
			await expect(zeilen).not.toHaveCount(vorher);
		}
		await expect(zeile).toBeVisible();

		// Erst die Rückfrage: „Abbrechen" lässt die Inventur stehen, „Löschen" nimmt sie heraus.
		const menue = page.getByRole('button', { name: `Aktionen für „${aeltester}“` });
		const rueckfrage = page.getByRole('dialog', { name: `Inventur „${aeltester}“ löschen?` });
		await menue.click();
		await page.getByRole('menuitem', { name: 'Löschen' }).click();
		await expect(rueckfrage).toContainText('Sie war vollständig und hat keinen Fehlbestand.');
		await rueckfrage.getByRole('button', { name: 'Abbrechen' }).click();
		await expect(rueckfrage).toHaveCount(0);
		await expect(zeile).toBeVisible();

		await menue.click();
		await page.getByRole('menuitem', { name: 'Löschen' }).click();
		await rueckfrage.getByRole('button', { name: 'Löschen' }).click();
		await expect(zeile).toHaveCount(0);
		expect(
			querySQL(`SELECT count(*) FROM inventur_sessions WHERE scope_label LIKE '${praefix}-%'`)
		).toBe('10');
	} finally {
		seedSQL(`DELETE FROM inventur_sessions WHERE scope_label LIKE '${praefix}-%';`);
	}
});
