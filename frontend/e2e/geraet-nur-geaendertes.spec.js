import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, querySQL, uniqueSuffix, menuepunkt } from './helpers.js';

// Die Maske „Gerät bearbeiten" und der Defekt-Knopf füllen sich aus der Zeile der Liste, und
// die lädt beim Öffnen der Seite. Ändert ein anderer Platz inzwischen Modell oder Zubehör,
// schrieb das Speichern den alten Stand zurück. Beide schicken nur, was sie selbst ändern.
test('Gerät: Maske und Defekt-Knopf überschreiben nicht, was inzwischen ein anderer Platz geändert hat', async ({
	page
}) => {
	const s = uniqueSuffix();
	const barcode = `G-E2E-NG-${s}`;
	seedSQL(`
		INSERT INTO geraete (modellname, barcode_id, zubehoer, zustand_notiz, ist_ausleihbar)
		VALUES ('E2E-Maske ${s}', '${barcode}', 'Ladekabel', 'Kratzer', true);
	`);
	const stand = () =>
		querySQL(
			`SELECT modellname || '|' || zubehoer || '|' || coalesce(zustand_notiz, '') || '|' || ist_ausleihbar FROM geraete WHERE barcode_id = '${barcode}'`
		);
	const naechsterAufruf = () =>
		page.waitForRequest((r) => r.method() === 'PUT' && /\/api\/geraete\//.test(r.url()));
	try {
		await uiLogin(page);
		await menuepunkt(page, 'Medienkatalog').click();
		await page.getByRole('tab', { name: 'Geräte' }).click();
		const zeile = page.locator('li').filter({ hasText: barcode });
		await zeile.getByRole('button', { name: 'Bearbeiten' }).click();
		const notiz = page.getByLabel('Zustandsnotiz');
		await expect(notiz).toHaveValue('Kratzer');

		// Ein anderer Platz berichtigt das Modell und meldet das Gerät defekt, während die Maske
		// offen ist.
		seedSQL(
			`UPDATE geraete SET modellname = 'E2E-Maske ${s} Pro', ist_ausleihbar = false WHERE barcode_id = '${barcode}';`
		);

		let gesendet = naechsterAufruf();
		await notiz.fill('Display gesprungen');
		await page.getByRole('button', { name: 'Speichern', exact: true }).click();
		expect(
			(await gesendet).postDataJSON(),
			'Die Maske schickt mehr als das geänderte Feld'
		).toEqual({ zustand_notiz: 'Display gesprungen' });
		await expect(page.getByText('Gerät gespeichert.')).toBeVisible();
		expect(
			stand(),
			'Das Speichern der Notiz hat Modell oder Defekt-Kennzeichen des anderen Platzes zurückgeschrieben'
		).toBe(`E2E-Maske ${s} Pro|Ladekabel|Display gesprungen|false`);

		// Die Liste ist neu geladen und zeigt das Gerät als defekt. Ein anderer Platz ergänzt das
		// Zubehör; „Wieder freigeben" nennt danach nur das Kennzeichen.
		const freigeben = zeile.getByRole('button', { name: 'Wieder freigeben' });
		await expect(freigeben).toBeVisible();
		seedSQL(`UPDATE geraete SET zubehoer = 'Ladekabel, Stift' WHERE barcode_id = '${barcode}';`);
		gesendet = naechsterAufruf();
		await freigeben.click();
		expect((await gesendet).postDataJSON(), 'Der Knopf schickt mehr als das Kennzeichen').toEqual({
			ist_ausleihbar: true
		});
		await expect(zeile.getByRole('button', { name: 'Defekt melden' })).toBeVisible();
		expect(stand(), 'Das Freigeben hat das Zubehör des anderen Platzes zurückgeschrieben').toBe(
			`E2E-Maske ${s} Pro|Ladekabel, Stift|Display gesprungen|true`
		);
	} finally {
		seedSQL(`DELETE FROM geraete WHERE barcode_id = '${barcode}';`);
	}
});
