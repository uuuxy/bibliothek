import { test, expect } from '@playwright/test';
import { uiLogin, apiPost, uniqueSuffix } from './helpers.js';

// Die Geräteausleihe am ganzen Weg (16.08.2026 — vorher war sie ein Backend-Torso:
// keine Verwaltung, und die Omnibox konnte die Zubehör-Checkliste nie bestätigen).
//
// Der Weg: Gerät im Medienkatalog anlegen → am Kiosk Schüler scannen → Gerät scannen
// → Checklisten-Dialog nennt die Zubehörteile → bestätigen → ausgeliehen. Danach
// derselbe Scan als Rückgabe — wieder über die Checkliste (fehlt ein Teil bei der
// Rückgabe, bricht das Personal hier ab).
test('Geräteausleihe: anlegen, Checkliste bestätigen, ausleihen, zurückgeben', async ({ page }) => {
	const s = uniqueSuffix();
	const BARCODE = `G-E2E-${s}`;

	await uiLogin(page);

	// 1. Gerät im Medienkatalog anlegen (Bereich „Geräte").
	await page.getByTitle('Medienkatalog').click();
	await page.getByRole('tab', { name: 'Geräte' }).click();
	await page.getByRole('button', { name: 'Gerät anlegen' }).click();
	await page.getByLabel('Modellname *').fill(`E2E-Tablet ${s}`);
	await page.getByLabel('Barcode (G-…) *').fill(BARCODE);
	await page.getByLabel(/Zubehör/).fill('Ladekabel, Eingabestift');
	await page.getByRole('button', { name: 'Anlegen', exact: true }).click();
	const zeile = page.locator('li').filter({ hasText: `E2E-Tablet ${s}` });
	await expect(zeile).toBeVisible();
	await expect(zeile.getByText('im Schrank')).toBeVisible();

	// 2. Schüler für die Ausleihe anlegen (API, wie die übrigen Kiosk-Specs).
	const created = await apiPost(page, '/api/schueler', {
		vorname: 'E2E',
		nachname: `Geraet-${s}`,
		klasse: '7b',
		barcode_id: `S-${s}`
	});
	expect(created.ok(), `Schüler-Seeding: ${created.status()}`).toBeTruthy();

	// 3. Kiosk: Schüler scannen, Gerät scannen → Checkliste erscheint, nichts ist gebucht.
	await page.getByTitle('Ausleihe').click();
	const scan = page.getByPlaceholder(/scannen/i).first();
	await scan.fill(`S-${s}`);
	await scan.press('Enter');
	await expect(page.getByText(`Geraet-${s}`).first()).toBeVisible();

	await scan.fill(BARCODE);
	await scan.press('Enter');
	await expect(page.getByRole('heading', { name: 'Zubehör prüfen' })).toBeVisible();
	await expect(page.getByText('Ladekabel', { exact: true })).toBeVisible();
	await expect(page.getByText('Eingabestift', { exact: true })).toBeVisible();

	// 4. Bestätigen → Ausleihe geht durch.
	await page.getByRole('button', { name: 'Alles vollständig — weiter' }).click();
	await expect(page.getByText(`„E2E-Tablet ${s}" ausgeliehen an E2E.`)).toBeVisible();

	// 5. Rückgabe: derselbe Scan, wieder über die Checkliste.
	await scan.fill(BARCODE);
	await scan.press('Enter');
	await expect(page.getByRole('heading', { name: 'Zubehör prüfen' })).toBeVisible();
	await page.getByRole('button', { name: 'Alles vollständig — weiter' }).click();
	await expect(page.getByText(`„E2E-Tablet ${s}" erfolgreich zurückgegeben.`)).toBeVisible();
});
