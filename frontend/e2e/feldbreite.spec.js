// Gate: Kein Textfeld ist breiter als eine Maske (44rem, --container-maske in
// styles/theme-mass.css) und keine Suchleiste breiter als 720 px (45rem, --container-suche);
// Seiten und Listen bleiben breit.
//
// Material 3, Text fields, Guidelines: „For medium and expanded breakpoints, text fields should
// be bound by flexible margins or other containers." und „Text fields shouldn't span the full
// width of a large screen." Material 3, Search, Specs: „Container | Width | Min: 360dp, max:
// 720dp". Gemessen wird im Browser, weil die Breite erst beim Layout entsteht: auf jeder Seite
// der Navigation, in jeder Kategorie der Einstellungen, in „Stammdaten bearbeiten" und in der
// Buchmaske.
import { test, expect } from '@playwright/test';
import { uiLogin, gehZu, einstellungsKategorie, seedSQL, uniqueSuffix } from './helpers.js';
import { SEITEN } from './seiten.js';
import { beobachteDatenanfragen, warteAufInhalt } from './messhilfe.js';

const MASKE = 704; // px — 44rem
const SUCHE = 720; // px — 45rem
// Selbstschutz: so viele Suchleisten findet der Rundgang mindestens.
const SUCHLEISTEN_MIN = 10;

/**
 * Die sichtbaren Textfelder und Suchleisten der Seite und die breiteren unter ihnen. Von einer
 * Suchleiste zählt das Element um das Eingabefeld: die Pille, das Formular der Theke, beim
 * kleinen Suchfeld die Hülle. Läuft im Browser.
 * @param {{ maske: number, suche: number }} grenzen
 */
const MESSE = (grenzen) => {
	const OHNE = ['checkbox', 'radio', 'hidden', 'file', 'range', 'color', 'button', 'submit'];
	const felder = [
		...document.querySelectorAll('input, textarea, select, [role="combobox"]')
	].filter(
		(e) =>
			!OHNE.includes(e.getAttribute('type') || '') &&
			e.getBoundingClientRect().height > 0 &&
			e.checkVisibility()
	);
	const istSuche = (/** @type {Element} */ e) =>
		e.getAttribute('type') === 'search' || e.id === 'omnibox-input';
	const breiter = felder.flatMap((e) => {
		const suche = istSuche(e);
		const breite = Math.round(((suche && e.parentElement) || e).getBoundingClientRect().width);
		if (breite <= (suche ? grenzen.suche : grenzen.maske)) return [];
		const name = e.id || e.getAttribute('aria-label') || e.tagName.toLowerCase();
		return [`${suche ? 'Suchleiste ' : ''}${name} ist ${breite} px breit`];
	});
	return { gemessen: felder.length, suchleisten: felder.filter(istSuche).length, breiter };
};

test('Kein Textfeld ist breiter als eine Maske, keine Suchleiste breiter als 720 px', async ({
	page
}) => {
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
	let suchleisten = 0;
	const grenzen = { maske: MASKE, suche: SUCHE };
	const anfragen = beobachteDatenanfragen(page);
	const zaehle = async () => (await page.evaluate(MESSE, grenzen)).gemessen;
	/** @param {string} wo */
	const pruefe = async (wo) => {
		await warteAufInhalt(anfragen, zaehle).catch((err) => {
			throw new Error(`${wo}: ${err.message}`, { cause: err });
		});
		const m = await page.evaluate(MESSE, grenzen);
		gemessen += m.gemessen;
		suchleisten += m.suchleisten;
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
	expect(suchleisten, 'Der Test hat fast keine Suchleisten gefunden.').toBeGreaterThan(
		SUCHLEISTEN_MIN
	);
	expect(
		funde,
		`Textfelder breiter als die Maske (${MASKE} px) oder Suchleisten breiter als ${SUCHE} px.\n` +
			'Behebung: Die Felder der Maske in eine Hülle mit max-w-maske legen, die Suchleiste ' +
			'über ui/Suchpille bauen (max-w-suche).\n\n' +
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

// An der Theke steht „Schnellrückgabe" direkt neben dem Scanfeld und auf seiner Linie, nicht am
// rechten Rand des Inhalts.
test('Theke: Scanfeld höchstens 720 px, „Schnellrückgabe" direkt daneben', async ({ page }) => {
	await page.setViewportSize({ width: 1920, height: 1080 });
	await uiLogin(page);
	await gehZu(page, '/kiosk');
	const feld = await page.locator('#omnibox-input').locator('xpath=..').boundingBox();
	const knopf = await page.getByRole('button', { name: 'Schnellrückgabe' }).boundingBox();
	if (!feld || !knopf) throw new Error('Scanfeld oder Knopf hat keine Fläche');
	expect(feld.width, 'Breite des Scanfelds').toBe(SUCHE);
	expect(knopf.x - (feld.x + feld.width), 'Abstand zwischen Feld und Knopf').toBe(12);
	expect(Math.round(knopf.y + knopf.height / 2), 'Knopf und Feld auf einer Linie').toBe(
		Math.round(feld.y + feld.height / 2)
	);
});
