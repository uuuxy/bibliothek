import { test, expect } from '@playwright/test';
import { uiLogin, gehZu, seedSQL, seedBenutzer, querySQL, uniqueSuffix } from './helpers.js';

// Den Designer öffnet, wer Ausweise druckt; das Design ändern darf nur, wer die Einstellungen
// verwaltet (api/routen_system.go). Das Öffnen schickte das unveränderte Design an den Server:
// Mitarbeiter und Leitung lasen danach „Speichern fehlgeschlagen: Keine Berechtigung".

/** @param {import('@playwright/test').Page} page */
function zaehleSchreiben(page) {
	/** @type {string[]} */
	const rumpfe = [];
	page.on('request', (anfrage) => {
		if (anfrage.method() === 'PUT' && anfrage.url().endsWith('/api/ausweis-layout')) {
			rumpfe.push(anfrage.postData() ?? '');
		}
	});
	return rumpfe;
}

/** @param {import('@playwright/test').Page} page */
async function oeffneDesigner(page) {
	await gehZu(page, '/druck-center');
	await page.getByRole('tab', { name: 'Schülerausweise' }).click();
	await expect(page.getByRole('button', { name: /Testdruck Vorderseite/ })).toBeVisible();
}

/** Länger als die Entprellung des Auto-Save (800 ms). @param {import('@playwright/test').Page} page */
const entprellungAbwarten = (page) => page.waitForTimeout(1200);

test('Ohne das Recht: gesperrt, kein Schreiben, keine Meldung; der Testdruck bleibt', async ({
	page
}) => {
	const email = `designer-mitarbeiter-${uniqueSuffix()}@schule.example`;
	seedBenutzer(email, 'mitarbeiter');
	const rumpfe = zaehleSchreiben(page);

	await uiLogin(page, email);
	await oeffneDesigner(page);
	await entprellungAbwarten(page);

	expect(rumpfe).toEqual([]);
	await expect(page.getByText(/Speichern fehlgeschlagen/)).toHaveCount(0);
	for (const name of ['+ Text', '+ Fläche', '+ Bild(er)', 'Standardwerte wiederherstellen']) {
		await expect(page.getByRole('button', { name, exact: true })).toBeDisabled();
	}
	await expect(page.getByRole('combobox', { name: 'Barcode-Typ' })).toBeDisabled();
	await expect(page.getByRole('button', { name: /Testdruck Vorderseite/ })).toBeEnabled();
	await expect(
		page.getByText('Das Design ändern kann, wer die Einstellungen verwalten darf.', {
			exact: false
		})
	).toBeVisible();
});

test('Mit dem Recht: Öffnen schreibt nichts, eine Änderung geht einmal hinaus', async ({
	page
}) => {
	const zeilenVorher = querySQL(
		`SELECT count(*) FROM system_einstellungen WHERE schluessel = 'ausweis_layout'`
	);
	const rumpfe = zaehleSchreiben(page);
	try {
		await uiLogin(page);
		await oeffneDesigner(page);
		await entprellungAbwarten(page);
		expect(rumpfe).toEqual([]);

		await page.getByRole('button', { name: 'Etikettenbogen', exact: true }).click();
		await expect(page.getByText(/Zentral gespeichert/)).toBeVisible();
		expect(rumpfe).toHaveLength(1);
		expect(JSON.parse(rumpfe[0]).printMode).toBe('etikett');
		// Der Kopf der Schule gehört zur Antwort, nicht zum Design.
		expect(JSON.parse(rumpfe[0]).schule).toBeUndefined();
	} finally {
		seedSQL(
			zeilenVorher === '0'
				? `DELETE FROM system_einstellungen WHERE schluessel = 'ausweis_layout';`
				: `UPDATE system_einstellungen SET wert = (wert::jsonb || '{"printMode":"card"}'::jsonb)::text
				   WHERE schluessel = 'ausweis_layout';`
		);
	}
});
