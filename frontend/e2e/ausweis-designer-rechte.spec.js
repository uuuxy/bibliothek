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
		// includes, nicht endsWith: Das Speichern trägt den Änderungsstand als ?stand=….
		if (anfrage.method() === 'PUT' && anfrage.url().includes('/api/ausweis-layout')) {
			rumpfe.push(anfrage.postData() ?? '');
		}
	});
	return rumpfe;
}

/** @param {import('@playwright/test').Page} page */
async function oeffneDesigner(page) {
	await gehZu(page, '/druck-center');
	await page.getByRole('tab', { name: 'Ausweis-Design' }).click();
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
		// Der Kopf der Schule und der Änderungsstand gehören zur Antwort, nicht zum Design.
		expect(JSON.parse(rumpfe[0]).schule).toBeUndefined();
		expect(JSON.parse(rumpfe[0]).stand).toBeUndefined();
	} finally {
		seedSQL(
			zeilenVorher === '0'
				? `DELETE FROM system_einstellungen WHERE schluessel = 'ausweis_layout';`
				: `UPDATE system_einstellungen SET wert = (wert::jsonb || '{"printMode":"card"}'::jsonb)::text
				   WHERE schluessel = 'ausweis_layout';`
		);
	}
});

// Zwei Plätze, ein Design. Platz A hat den Designer offen, Platz B speichert danach. Die nächste
// Änderung an A ersetzte früher das ganze Design von B, ohne Hinweis an einem der beiden Plätze.
// Jetzt lehnt der Server den überholten Stand ab, und A lädt auf Knopfdruck den von B.
test('Zwei Plätze: Der ältere Stand überschreibt den neueren nicht', async ({ page, browser }) => {
	// Das Design der Datenbank wird beiseitegelegt und am Ende genau so zurückgelegt.
	seedSQL(`
		DELETE FROM system_einstellungen WHERE schluessel = 'e2e_ausweis_layout_sicherung';
		INSERT INTO system_einstellungen (schluessel, wert)
		SELECT 'e2e_ausweis_layout_sicherung', wert FROM system_einstellungen WHERE schluessel = 'ausweis_layout';
	`);
	const kontextB = await browser.newContext();
	const platzB = await kontextB.newPage();
	try {
		await uiLogin(page);
		await oeffneDesigner(page);
		await uiLogin(platzB);
		await oeffneDesigner(platzB);

		// B stellt auf den Etikettenbogen; A zeigt noch den Kartendrucker.
		await platzB.getByRole('button', { name: 'Etikettenbogen', exact: true }).click();
		await expect(platzB.getByText(/Zentral gespeichert/)).toBeVisible();

		// A ändert etwas anderes, auf seinem alten Stand.
		await page.getByRole('combobox', { name: 'Barcode-Typ' }).click();
		await page.getByRole('option').nth(1).click();
		await expect(
			page.getByText(
				'Speichern fehlgeschlagen: Das Ausweis-Design wurde inzwischen an einem anderen Platz geändert. Nichts gespeichert.'
			)
		).toBeVisible();
		expect(
			querySQL(
				`SELECT wert::jsonb ->> 'printMode' FROM system_einstellungen WHERE schluessel = 'ausweis_layout'`
			),
			'gespeichert bleibt das Design von Platz B'
		).toBe('etikett');

		await page.getByRole('button', { name: 'Neu laden und verwerfen' }).click();
		await expect(page.getByText(/Speichern fehlgeschlagen/)).toHaveCount(0);
		await expect(page.getByRole('button', { name: 'Etikettenbogen', exact: true })).toHaveAttribute(
			'aria-pressed',
			'true'
		);
	} finally {
		await kontextB.close();
		seedSQL(`
			DELETE FROM system_einstellungen WHERE schluessel = 'ausweis_layout';
			UPDATE system_einstellungen SET schluessel = 'ausweis_layout'
			WHERE schluessel = 'e2e_ausweis_layout_sicherung';
		`);
	}
});
