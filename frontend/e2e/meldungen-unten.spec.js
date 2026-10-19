import { test, expect } from '@playwright/test';
import {
	uiLogin,
	seedSQL,
	querySQL,
	gehZu,
	einstellungsKategorie,
	uniqueSuffix
} from './helpers.js';

// Die Meldungen stehen unten und mittig über der Fläche rechts der Navigation und rücken über
// eine Leiste am unteren Rand (OFFEN 5.78). Material 3, Snackbar, „Placement": „Snackbars should
// be placed at the bottom of a UI, in front of the main content. In some cases, snackbars can
// be nudged upwards to avoid overlapping with other UI elements near the bottom"; „Avoid
// placing snackbars in front of navigation components". Die Rechnung steht in
// src/lib/meldungsPlatz.js; hier wird gemessen.

test.use({ viewport: { width: 1280, height: 720 } });

/** @param {import('@playwright/test').Locator} l */
async function kasten(l) {
	const k = await l.boundingBox();
	if (!k) throw new Error('Das Element hat keine Fläche.');
	return k;
}

test('Eine Meldung steht unten, mittig rechts der Navigation, eingeklappt wie ausgeklappt', async ({
	page
}) => {
	await uiLogin(page);
	await gehZu(page, '/einstellungen');
	await einstellungsKategorie(page, 'Ausleihe & Fristen').click();
	await page.getByLabel('Tage / Buch').waitFor();
	const navigation = page
		.locator('aside')
		.filter({ has: page.getByRole('button', { name: /^Navigation / }) });
	// Speichern ohne Änderung schickt nichts und meldet „Gespeichert." (einstellungen-kategorien).
	const speichern = page.getByRole('button', { name: 'Ausleihe & Fristen speichern' });

	for (const breite of [256, 64]) {
		if (breite === 64) {
			await page.getByRole('button', { name: 'Navigation einklappen' }).click();
			await expect.poll(async () => (await kasten(navigation)).width).toBe(64);
		}
		await speichern.click();
		const meldung = page.getByRole('status').filter({ hasText: 'Gespeichert.' }).last();
		await expect(meldung).toBeVisible();
		const n = await kasten(navigation);
		const rechtsDerNavigation = n.x + n.width;
		await expect
			.poll(async () => {
				const k = await kasten(meldung);
				return Math.round(k.x + k.width / 2);
			}, `Die Meldung steht nicht mittig rechts der Navigation (${breite} px).`)
			.toBe(Math.round((rechtsDerNavigation + 1280) / 2));
		const m = await kasten(meldung);
		expect(720 - (m.y + m.height), 'Abstand zum unteren Rand').toBeCloseTo(24, 0);
		expect(m.x, 'Die Meldung liegt über der Navigation.').toBeGreaterThanOrEqual(
			rechtsDerNavigation
		);
	}
});

test('Eine Meldung rückt über die Auswahlleiste, die unten am Rand klebt', async ({ page }) => {
	const s = uniqueSuffix();
	// So viele Exemplare, dass die Liste über das Fenster reicht und die Leiste unten klebt.
	seedSQL(`
		WITH t AS (
			INSERT INTO buecher_titel (titel, autor, isbn) VALUES ('E2E-Meldung ${s}', 'Probe', '978m${s}') RETURNING id
		)
		INSERT INTO buecher_exemplare (titel_id, barcode_id)
		SELECT t.id, 'B-mu-${s}-' || n FROM t, generate_series(1, 25) n;
	`);
	try {
		const id = querySQL(`SELECT id FROM buecher_titel WHERE isbn = '978m${s}'`);
		await uiLogin(page);
		await gehZu(page, `/medienkatalog/buch/${id}`);
		await page.getByRole('tab', { name: /^Exemplare/ }).click();
		await page.getByRole('checkbox', { name: `Exemplar B-mu-${s}-1 auswählen` }).check();
		const leiste = page.getByRole('region', { name: 'Aktionen für die markierten Exemplare' });
		await expect(leiste).toBeVisible();
		const l = await kasten(leiste);
		// Sie klebt 16 px über dem Rand der Arbeitsfläche, die unten 24 px Innenabstand hat: dort,
		// wo die Meldung ohne sie stünde.
		expect(720 - (l.y + l.height), 'Die Leiste klebt nicht unten.').toBeCloseTo(40, 0);

		// Der Server lehnt ab; die Markierung und mit ihr die Leiste bleiben stehen.
		await page.route('**/api/exemplare/standort', (route) =>
			route.fulfill({
				status: 409,
				contentType: 'application/json',
				body: JSON.stringify({ error: 'Probe: Standort abgelehnt.' })
			})
		);
		await leiste.getByRole('button', { name: 'Standort ändern' }).click();
		const dialog = page.getByRole('dialog', { name: 'Standort ändern' });
		await dialog.getByLabel('Standort', { exact: true }).fill('Probe');
		await dialog.getByRole('button', { name: 'Standort ändern' }).click();

		const meldung = page.getByRole('alert').filter({ hasText: 'Probe: Standort abgelehnt.' });
		await expect(meldung).toBeVisible();
		await expect
			.poll(async () => {
				const m = await kasten(meldung);
				return Math.round(l.y - (m.y + m.height));
			}, 'Abstand zwischen Meldung und Leiste')
			.toBe(16);
	} finally {
		seedSQL(`
			DELETE FROM buecher_exemplare WHERE titel_id IN (SELECT id FROM buecher_titel WHERE isbn = '978m${s}');
			DELETE FROM buecher_titel WHERE isbn = '978m${s}';
		`);
	}
});

test('Eine Meldung rückt über die Leiste der Leserdatei', async ({ page }) => {
	const s = uniqueSuffix();
	const vorname = `Mu${s}`;
	seedSQL(`
		INSERT INTO schueler (vorname, nachname, klasse, barcode_id, geburtsdatum, abgaenger_jahr)
		VALUES ('${vorname}', 'Probe', '7A', 'MU-${s}', '2012-01-01', 2030);
	`);
	const id = querySQL(`SELECT id FROM schueler WHERE barcode_id = 'MU-${s}'`);
	try {
		// Das Ausweis-Design lässt sich nicht laden: „Ausweise drucken" meldet das, und die
		// Markierung und mit ihr die Leiste bleiben stehen.
		await page.route('**/api/ausweis-layout', (route) => route.fulfill({ status: 503, body: '' }));
		await uiLogin(page);
		await gehZu(page, '/schuelerdatei');
		await page.getByRole('searchbox', { name: 'Leser suchen' }).fill(vorname);
		await page
			.getByRole('checkbox', { name: `${vorname} Probe für den Ausweisdruck markieren` })
			.check();
		const leiste = page.getByRole('region', { name: 'Aktionen für die markierten Schüler' });
		await expect(leiste).toBeVisible();
		const l = await kasten(leiste);

		await leiste.getByRole('button', { name: /drucken$/ }).click();
		const meldung = page.getByRole('alert').filter({ hasText: 'Nicht gedruckt' });
		await expect(meldung).toBeVisible();
		await expect
			.poll(async () => {
				const m = await kasten(meldung);
				return Math.round(l.y - (m.y + m.height));
			}, 'Abstand zwischen Meldung und Leiste')
			.toBe(16);
	} finally {
		seedSQL(`DELETE FROM schueler WHERE id = '${id}';`);
	}
});
