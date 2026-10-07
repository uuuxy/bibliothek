import { test, expect } from '@playwright/test';
import { uiLogin, seedSQL, uniqueSuffix, oeffneSchuelerProfil } from './helpers.js';

// Ein Name aus einem langen Wort bleibt in der Leserkarte. Ohne Umbruch im Wort ragte er aus
// der Karte und lag daneben auf der ersten Zeile des Reiters.
test('Leserakte: ein langer Name bricht um und bleibt in der Karte', async ({ page }) => {
	const s = uniqueSuffix();
	const ausweis = `E2E-LN-${s}`;
	const nachname = 'Wolfeschlegelsteinhausen';
	seedSQL(`
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ('${ausweis}', 'Lang${s}', '${nachname}', '08a', EXTRACT(YEAR FROM CURRENT_DATE)::int + 3);
	`);
	try {
		await page.setViewportSize({ width: 1280, height: 900 });
		await uiLogin(page);
		await oeffneSchuelerProfil(page, `Lang${s}`);

		const name = page.getByRole('heading', { level: 3, name: new RegExp(nachname) });
		await expect(name).toBeVisible();
		const mass = await name.evaluate((el) => ({ sichtbar: el.clientWidth, ganz: el.scrollWidth }));
		expect(
			mass.ganz,
			`Der Name ist ${mass.ganz} px breit, seine Überschrift ${mass.sichtbar} px: Er ragt ` +
				`${mass.ganz - mass.sichtbar} px aus der Leserkarte.`
		).toBeLessThanOrEqual(mass.sichtbar);
	} finally {
		seedSQL(`DELETE FROM schueler WHERE barcode_id = '${ausweis}';`);
	}
});
