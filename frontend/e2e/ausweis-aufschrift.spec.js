import { test, expect } from '@playwright/test';
import { uiLogin, apiPost, seedSQL, querySQL, uniqueSuffix, menuepunkt, gehZu } from './helpers.js';

// Die Aufschrift des gedruckten Ausweises, gemessen am LIVE-Pfad.
//
// Gemeldet am 16.09.2026 : „wenn ein Lehrer einen Ausweis drucken möchte, sollte da
// natürlich Lehrerausweis stehen anstatt schülerausweis".
//
// Warum das ein e2e-Fall ist und nicht nur ein Unit-Test: Der Titel war ein gewöhnliches
// Textelement mit fest eingetipptem Inhalt, und das Ausweis-Design liegt ZENTRAL in der
// Datenbank. Es überschreibt beim Laden jede Vorgabe im Code. Ein Unit-Test über die
// Vorgabe wäre also grün gewesen, während die Anlage der Schule weiter „Schülerausweis"
// gedruckt hätte. Diese Spec setzt deshalb absichtlich den ALTEN Stand ins gespeicherte
// Design und prüft, was danach in der Druckkarte steht.
//
// Die Karte liegt im DOM (`.single-card-print-section`, display:none) und wird erst per
// @media print sichtbar — genau dieses Stück Markup geht in den Drucker.

/**
 * Setzt das gespeicherte Design auf den Stand vor dem 16.09.2026 zurück. Auf einer frischen
 * Datenbank ist noch keines gespeichert; das Öffnen des Designers schreibt keines, erst eine
 * Änderung.
 * @param {import('@playwright/test').Page} page
 */
async function alterTitelImGespeichertenDesign(page) {
	const gespeichert = querySQL(
		`SELECT count(*) FROM system_einstellungen WHERE schluessel = 'ausweis_layout' AND wert::jsonb ? 'front'`
	);
	if (gespeichert === '0') {
		await gehZu(page, '/druck-center');
		await page.getByRole('tab', { name: 'Schülerausweise' }).click();
		const schreiben = page.waitForResponse(
			(a) => a.url().endsWith('/api/ausweis-layout') && a.request().method() === 'PUT'
		);
		await page.getByRole('combobox', { name: 'Karten-Hintergrund' }).click();
		await page.getByRole('option').nth(1).click();
		expect((await schreiben).ok()).toBe(true);
	}
	seedSQL(`
		UPDATE system_einstellungen
		SET wert = jsonb_set(
			wert::jsonb, '{front,elements}',
			(SELECT jsonb_agg(CASE WHEN e->>'id' = 'title'
				THEN (e - 'type' - 'content' - 'style')
				     || jsonb_build_object('type','text','content','SCHÜLERAUSWEIS',
				                           'style', (e->'style') - 'textTransform')
				ELSE e END)
			 FROM jsonb_array_elements((wert::jsonb)->'front'->'elements') e)
		)::text
		WHERE schluessel = 'ausweis_layout';
	`);
}

/** @param {import('@playwright/test').Page} page */
async function oeffneAkte(page, vorname, nachname) {
	await menuepunkt(page, 'Leserdatei').click();
	const suche = page.getByLabel('Leser suchen');
	await suche.click();
	await suche.fill(nachname);
	const zeile = page.getByRole('row', { name: new RegExp(nachname) });
	await expect(zeile).toBeVisible();
	await zeile
		.getByRole('button', { name: new RegExp(`Profil von ${vorname} ${nachname}`) })
		.click();
	await expect(page.getByRole('heading', { name: `${vorname} ${nachname}` })).toBeVisible();
	return page.locator('.single-card-print-section');
}

test('Die Karte einer Lehrkraft heisst Lehrerausweis — auch bei einem Design von früher', async ({
	page
}) => {
	const s = uniqueSuffix();
	const nachname = `Ausweis${s}`;

	await uiLogin(page);
	await alterTitelImGespeichertenDesign(page);
	const angelegt = await apiPost(page, '/api/schueler', {
		art: 'lehrkraft',
		vorname: 'Lena',
		nachname,
		email: `lena.${nachname.toLowerCase()}@test.local`
	});
	expect(angelegt.ok(), `Anlegen fehlgeschlagen: ${angelegt.status()}`).toBeTruthy();

	const karte = await oeffneAkte(page, 'Lena', nachname);
	await expect(karte).toContainText('Lehrerausweis');
	await expect(karte).not.toContainText('Schülerausweis');
	// Ein Kollege hat keine Klasse, aus der ein Ablaufjahr zu rechnen wäre, und sein Ausweis
	// läuft mit keinem Schuljahr ab. Ohne die Weiche stünde hier „Gültig bis: 31.07.–".
	await expect(karte).not.toContainText('Gültig bis');
});

test('Die Karte einer Schülerin bleibt der Schülerausweis, mit Gültigkeit', async ({ page }) => {
	const s = uniqueSuffix();
	const nachname = `Ausweis${s}`;

	await uiLogin(page);
	await alterTitelImGespeichertenDesign(page);
	const angelegt = await apiPost(page, '/api/schueler', {
		art: 'schueler',
		geburtsdatum: '2012-03-04',
		vorname: 'Mia',
		nachname,
		klasse: '10A'
	});
	expect(angelegt.ok(), `Anlegen fehlgeschlagen: ${angelegt.status()}`).toBeTruthy();

	const karte = await oeffneAkte(page, 'Mia', nachname);
	await expect(karte).toContainText('Schülerausweis');
	await expect(karte).not.toContainText('Lehrerausweis');
	await expect(karte).toContainText('Gültig bis');
});

// Der Platzhalter „Musterstadt" wurde nur im Designer ersetzt. Wer aus der Akte druckte, bevor dort
// jemand gewesen war, bekam eine Karte mit dem Namen einer Schule, die es nicht gibt. Die Schlüssel
// liegen während des Tests unter einem anderen Namen, damit er auf jedem Stand gleich läuft.
test('Die Karte trägt den Namen der Schule, auch ohne dass jemand den Designer geöffnet hat', async ({
	page
}) => {
	const s = uniqueSuffix();
	const nachname = `Ausweis${s}`;
	const SCHLUESSEL = `('ausweis_layout', 'schule_name', 'schule_strasse', 'schule_plz', 'schule_ort')`;
	seedSQL(`
		UPDATE system_einstellungen SET schluessel = 'e2e-sicherung-' || schluessel WHERE schluessel IN ${SCHLUESSEL};
		INSERT INTO system_einstellungen (schluessel, wert) VALUES
			('schule_name', 'Schule ${s}'), ('schule_strasse', 'Schulweg 1'), ('schule_plz', '12345'), ('schule_ort', 'Neustadt');
	`);
	try {
		await uiLogin(page);
		const angelegt = await apiPost(page, '/api/schueler', {
			art: 'schueler',
			geburtsdatum: '2012-03-04',
			vorname: 'Mia',
			nachname,
			klasse: '10A'
		});
		expect(angelegt.ok(), `Anlegen fehlgeschlagen: ${angelegt.status()}`).toBeTruthy();

		const karte = await oeffneAkte(page, 'Mia', nachname);
		await expect(karte).toContainText(`Schule ${s}`);
		await expect(karte).toContainText('Schulweg 1, 12345 Neustadt');
		await expect(karte).not.toContainText(/Musterst/i);
	} finally {
		seedSQL(`
			DELETE FROM system_einstellungen WHERE schluessel IN ${SCHLUESSEL};
			UPDATE system_einstellungen SET schluessel = substr(schluessel, 15) WHERE schluessel LIKE 'e2e-sicherung-%';
		`);
	}
});
