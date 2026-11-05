import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix } from './helpers.js';

// Titel löschen mit Grund: „Alter Bestand" ist vorgewählt. Ein verliehenes Exemplar steht mit
// seinem Leser im Dialog und bleibt beim Leser, bis sein eigenes Kästchen angekreuzt ist; der
// Knopf nennt, was geschieht. Jedes Exemplar steht danach mit seinem Grund im Abgangsbuch.
const s = uniqueSuffix().slice(0, 6);
const TITEL = `Loeschdialog ${s}`;
const LESER = `Loesch${s}`;
const BARCODES = [1, 2, 3].map((n) => `B-LD-${n}-${s}`);

/** Exemplare des Titels als „im Bestand/ausgesondert mit Grund/offene Ausleihen". */
const stand = () =>
	querySQL(`
		SELECT count(*) FILTER (WHERE NOT e.ist_ausgesondert) || '/' ||
		       count(*) FILTER (WHERE e.ist_ausgesondert AND e.aussonderung_grund = 'BESCHAEDIGUNG') || '/' ||
		       (SELECT count(*) FROM ausleihen a JOIN buecher_exemplare x ON x.id = a.exemplar_id
		        WHERE x.barcode_id LIKE 'B-LD-%-${s}' AND a.rueckgabe_am IS NULL)
		FROM buecher_exemplare e WHERE e.barcode_id LIKE 'B-LD-%-${s}'`);

/** @param {import('@playwright/test').Page} page */
async function oeffneMaske(page) {
	await uiLogin(page);
	await page.goto('/medienkatalog');
	await page.getByRole('tab', { name: 'Titel-Verwaltung' }).click();
	await page.getByRole('searchbox', { name: 'Titel durchsuchen' }).fill(TITEL);
	await page.getByText(TITEL).first().click();
	await expect(page.getByRole('heading', { name: 'Titel bearbeiten' })).toBeVisible();
	await page.getByRole('button', { name: 'Titel löschen' }).click();
	return page.getByRole('dialog', { name: `„${TITEL}“ löschen?` });
}

test.describe.serial('Titel löschen mit Grund', () => {
	test.beforeAll(() => {
		seedSQL(`
			WITH t AS (
				INSERT INTO buecher_titel (titel, autor, signatur, medientyp)
				VALUES ('${TITEL}', 'E2E', 'Loe 1', 'Buch') RETURNING id
			), e AS (
				INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar)
				SELECT t.id, b, true FROM t, unnest(ARRAY['${BARCODES.join("','")}']) b
				RETURNING id, barcode_id
			), sch AS (
				INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
				VALUES ('S-LD-${s}', '${LESER}', 'Testschueler', '10R1', 2030) RETURNING id
			)
			INSERT INTO ausleihen (exemplar_id, schueler_id, rueckgabe_frist)
			SELECT e.id, sch.id, now() + interval '14 days' FROM e, sch WHERE e.barcode_id = '${BARCODES[0]}';`);
	});

	test.afterAll(() => {
		seedSQL(`
			DELETE FROM ausleihen WHERE schueler_id IN (SELECT id FROM schueler WHERE barcode_id = 'S-LD-${s}');
			DELETE FROM buecher_exemplare WHERE barcode_id LIKE 'B-LD-%-${s}';
			DELETE FROM buecher_titel WHERE titel = '${TITEL}';
			DELETE FROM geloeschte_exemplare WHERE barcode LIKE 'B-LD-%-${s}';
			DELETE FROM schueler WHERE barcode_id = 'S-LD-${s}';`);
	});

	test('ohne Kästchen bleibt das verliehene beim Leser, die im Haus gehen mit dem Grund', async ({
		page
	}) => {
		const dialog = await oeffneMaske(page);
		await expect(
			dialog.getByText('Die 2 Exemplare im Haus kommen mit diesem Grund ins Abgangsbuch.')
		).toBeVisible();
		await expect(dialog.getByRole('radio', { name: 'Alter Bestand' })).toBeChecked();
		await expect(dialog.getByText('Ein Exemplar ist verliehen:')).toBeVisible();
		await expect(dialog.getByText(`${LESER} Testschueler (10R1)`)).toBeVisible();
		await expect(dialog.getByRole('checkbox')).not.toBeChecked();

		await dialog.getByRole('radio', { name: 'Beschädigt' }).check();
		await dialog.getByRole('button', { name: '2 Exemplare löschen' }).click();

		await expect(
			page.getByText('Der Titel bleibt, bis die verliehenen Exemplare zurück sind')
		).toBeVisible();
		expect(stand(), 'eines im Bestand, zwei beschädigt ausgesondert, die Ausleihe offen').toBe(
			'1/2/1'
		);
	});

	test('mit Kästchen geht alles, und jedes Exemplar steht mit seinem Grund im Abgangsbuch', async ({
		page
	}) => {
		const dialog = await oeffneMaske(page);
		await expect(dialog.getByText('Im Haus ist kein Exemplar.')).toBeVisible();
		await expect(dialog.getByRole('button', { name: 'Löschen' })).toBeDisabled();

		await dialog
			.getByRole('checkbox', {
				name: 'Auch dieses Exemplar löschen und aus dem Leserkonto austragen'
			})
			.check();
		await dialog.getByRole('button', { name: 'Das Exemplar löschen' }).click();

		await expect(page.getByText('Titel gelöscht')).toBeVisible();
		expect(querySQL(`SELECT count(*) FROM buecher_titel WHERE titel = '${TITEL}'`)).toBe('0');
		expect(
			querySQL(`
				SELECT string_agg(barcode || ':' || grund, ',' ORDER BY barcode)
				FROM geloeschte_exemplare WHERE barcode LIKE 'B-LD-%-${s}'`)
		).toBe(`${BARCODES[0]}:AUSSORTIERT,${BARCODES[1]}:BESCHAEDIGUNG,${BARCODES[2]}:BESCHAEDIGUNG`);
	});
});
