// Gate: Kein Textfeld ist breiter als eine Maske (44rem, --container-maske in
// styles/theme-mass.css); Seiten und Listen bleiben breit.
//
// Material 3, Text fields, Guidelines: „For medium and expanded breakpoints, text fields should
// be bound by flexible margins or other containers." und „Text fields shouldn't span the full
// width of a large screen." Gemessen wird im Browser, weil die Breite erst beim Layout entsteht:
// auf jeder Seite der Navigation, in jeder Kategorie der Einstellungen, in „Stammdaten
// bearbeiten" und in der Buchmaske.
//
// Ausgenommen sind die Suchleisten (Suchfelder und das Scanfeld der Theke): Für sie gilt
// Material 3, Search, mit eigener Breite (OFFEN 5.86).
import { test, expect } from '@playwright/test';
import { uiLogin, gehZu, einstellungsKategorie, seedSQL, uniqueSuffix } from './helpers.js';
import { SEITEN } from './seiten.js';
import { beobachteDatenanfragen, warteAufInhalt } from './messhilfe.js';

const MASKE = 704; // px — 44rem

/**
 * Die sichtbaren Textfelder der Seite und die breiteren unter ihnen. Läuft im Browser.
 * @param {number} grenze
 */
const MESSE = (grenze) => {
	const OHNE = ['checkbox', 'radio', 'hidden', 'file', 'range', 'color', 'button', 'submit'];
	const felder = [
		...document.querySelectorAll('input, textarea, select, [role="combobox"]')
	].filter(
		(e) =>
			!OHNE.includes(e.getAttribute('type') || '') &&
			e.getAttribute('type') !== 'search' &&
			e.id !== 'omnibox-input' &&
			e.getBoundingClientRect().height > 0 &&
			e.checkVisibility()
	);
	return {
		gemessen: felder.length,
		breiter: felder
			.filter((e) => e.getBoundingClientRect().width > grenze)
			.map(
				(e) =>
					`${e.id || e.getAttribute('aria-label') || e.tagName.toLowerCase()} ist ` +
					`${Math.round(e.getBoundingClientRect().width)} px breit`
			)
	};
};

test('Kein Textfeld ist breiter als eine Maske', async ({ page }) => {
	test.setTimeout(300_000);
	await page.setViewportSize({ width: 1710, height: 952 });
	const s = uniqueSuffix();
	const vorname = `Breit${s}`;
	seedSQL(`
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ('E2E-FB-${s}', '${vorname}', 'Feld', '08a', EXTRACT(YEAR FROM CURRENT_DATE)::int + 3);
	`);
	/** @type {string[]} */
	const funde = [];
	let gemessen = 0;
	const anfragen = beobachteDatenanfragen(page);
	const zaehle = async () => (await page.evaluate(MESSE, MASKE)).gemessen;
	/** @param {string} wo */
	const pruefe = async (wo) => {
		await warteAufInhalt(anfragen, zaehle).catch((err) => {
			throw new Error(`${wo}: ${err.message}`, { cause: err });
		});
		const m = await page.evaluate(MESSE, MASKE);
		gemessen += m.gemessen;
		funde.push(...m.breiter.map((b) => `${wo}: ${b}`));
	};

	try {
		await uiLogin(page);
		for (const [seite, pfad] of SEITEN) {
			await gehZu(page, pfad);
			await pruefe(seite);
		}

		await gehZu(page, '/einstellungen');
		const kategorien = page.getByRole('navigation', { name: 'Einstellungs-Kategorien' });
		const titel = (await kategorien.getByRole('button').allInnerTexts()).map((t) =>
			t.split('\n')[0].trim()
		);
		expect(titel.length, 'keine Kategorien gefunden').toBeGreaterThan(10);
		for (const t of titel) {
			await einstellungsKategorie(page, t).click();
			await page.getByRole('heading', { level: 2, name: t, exact: true }).first().waitFor();
			await pruefe(`Einstellungen → ${t}`);
		}

		await gehZu(page, '/schuelerdatei');
		await page.getByRole('searchbox', { name: 'Leser suchen' }).fill(vorname);
		await page.getByRole('button', { name: new RegExp(`Profil von ${vorname} `) }).click();
		await page.getByRole('tab', { name: 'Stammdaten & Adresse' }).click();
		await page.getByRole('button', { name: 'Bearbeiten' }).click();
		await page.locator('#ort').waitFor();
		await pruefe('Stammdaten bearbeiten');
	} finally {
		seedSQL(`DELETE FROM schueler WHERE barcode_id = 'E2E-FB-${s}';`);
	}

	// Selbstschutz: Bricht eine Navigation, misst der Test still nichts mehr.
	expect(gemessen, 'Der Test hat fast keine Textfelder gefunden.').toBeGreaterThan(40);
	expect(
		funde,
		`Textfelder breiter als die Maske (${MASKE} px).\n` +
			'Behebung: Die Felder der Maske in eine Hülle mit max-w-maske legen.\n\n' +
			funde.join('\n')
	).toEqual([]);
});

for (const fenster of [
	{ width: 1710, height: 952 },
	{ width: 1366, height: 768 }
]) {
	test(`Buchmaske im Fenster von ${fenster.width} px: Felder höchstens 704 px breit`, async ({
		page
	}) => {
		await page.setViewportSize(fenster);
		await uiLogin(page);
		await gehZu(page, '/medienkatalog');
		await page.getByRole('tab', { name: 'Titel-Verwaltung' }).click();
		await page.getByRole('button', { name: 'Neues Buch' }).first().click();

		for (const id of ['buch-titel', 'buch-autor']) {
			const feld = await page.locator(`#${id}`).boundingBox();
			expect(feld, `#${id} hat keine Fläche`).not.toBeNull();
			expect(feld?.width, `#${id} ist breiter als 44rem`).toBeLessThanOrEqual(MASKE);
			// Gegenprobe: Das Feld ist nicht zusammengefallen.
			expect(feld?.width, `#${id} ist schmaler als erwartet`).toBeGreaterThan(600);
		}

		// Begrenzt sind die Felder, nicht die Seite: Der Kopf der Maske reicht weiter über die Fläche.
		const kopf = await page
			.getByRole('heading', { name: 'Neues Buch' })
			.locator('xpath=..')
			.boundingBox();
		expect(kopf?.width, 'der Kopf der Maske').toBeGreaterThan(fenster.width - 400);
	});
}
