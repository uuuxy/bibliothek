// Gate: Jeder Symbolknopf ist mindestens 32 px groß und rund, und die 48 × 48 px um seine Mitte
// gehören ihm.
//
// Material 3, Icon buttons, Specs: Container „extra small" 32 dp, Shape „Round (default)" und
// „Extra small and small icon buttons must have a target size of 48x48dp or larger to be
// accessible." Größe, Rundung und Trefferfläche stehen in .icon-btn (styles/komponenten.css);
// was der Fläche ein Nachbar oder ein Container nimmt, misst e2e/trefferflaeche.js.
//
// Gemessen wird im Browser, weil die Fläche erst dort aus Symbol, Polsterung und Umgebung
// entsteht. Drei Zustände stellt der Test ausdrücklich her, weil man sie leicht übersieht: die
// geöffnete Bestellung (ihre Symbole stehen in der Detailansicht), die Exemplarkarten der
// Buchakte und die eingeklappte Navigation (ihr Umschalter ist ein anderer Knopf als der zum
// Einklappen).
import { test, expect } from '@playwright/test';
import {
	uiLogin,
	gehZu,
	seedSQL,
	querySQL,
	uniqueSuffix,
	seedBestellung,
	oeffneBestellungsDetail,
	oeffneSchuelerProfil
} from './helpers.js';
import { beobachteDatenanfragen, warteAufInhalt, pruefeNichtZuFruehGemessen } from './messhilfe.js';
import { SYMBOLKNOEPFE } from './trefferflaeche.js';

const MIN_FLAECHE = 32; // px — Material 3 Icon-Button „extra small"
const ZIEL = 48; // px — Trefferfläche nach Material 3

// Bestellung, Buch und Leser legt der Test selbst an: Er läuft damit auch auf einer leeren
// Datenbank, und die Exemplarkarte zeigt ihre Symbole für ein Etikett und einen Platzhalter.
let marke = '';
let s = '';
/** @type {(() => void) | undefined} */
let aufraeumen;
test.beforeEach(() => {
	({ marke, aufraeumen } = seedBestellung());
	s = uniqueSuffix();
	seedSQL(`
		WITH t AS (
			INSERT INTO buecher_titel (titel, autor) VALUES ('E2E-Treffer ${s}', 'E2E') RETURNING id
		)
		INSERT INTO buecher_exemplare (titel_id, barcode_id)
		SELECT t.id, b FROM t, unnest(ARRAY['B-TF${s.slice(0, 6)}', 'AUTO-TF${s.slice(0, 6)}']) b;
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ('E2E-TF-${s}', 'Treffer${s}', 'Fläche', '08a',
		        EXTRACT(YEAR FROM CURRENT_DATE)::int + 3);
	`);
});
test.afterEach(() => {
	aufraeumen?.();
	aufraeumen = undefined;
	seedSQL(`
		DELETE FROM schueler WHERE barcode_id = 'E2E-TF-${s}';
		DELETE FROM buecher_exemplare
		WHERE titel_id IN (SELECT id FROM buecher_titel WHERE titel = 'E2E-Treffer ${s}');
		DELETE FROM buecher_titel WHERE titel = 'E2E-Treffer ${s}';
	`);
});

// Jede Seite der Navigation (routenTabelle.js) außer den Bestandsbüchern: Dort steht kein
// Symbolknopf außer denen des Rahmens, und ihre Liste ist am Entwicklungsrechner 67.000 Zeilen
// lang; schon das erste Messen dauert dort länger als das Warten auf Ruhe.
const SEITEN = [
	['Ausleihe', '/kiosk'],
	['Medienkatalog', '/medienkatalog'],
	['Signaturen', '/signaturen'],
	['Druck-Center', '/druck-center'],
	['Klassensätze', '/schulklassen'],
	['Leserdatei', '/schuelerdatei'],
	['Mahnwesen', '/mahnwesen'],
	['Abgänger', '/abgaenger'],
	['Bestellungen', '/bestellungen'],
	['Inventur', '/inventur'],
	['Statistiken', '/statistiken'],
	['Bestellberichte', '/bestellberichte'],
	['System-Logs', '/system-logs'],
	['Schuljahreswechsel', '/schuljahr'],
	['Benutzer & Rechte', '/berechtigungen'],
	['Einstellungen', '/einstellungen'],
	['Mein Portal', '/kollegium-portal']
];

test('Symbolknöpfe: 32 px, rund, 48 px Trefferfläche', async ({ page }) => {
	test.setTimeout(300_000);
	const anfragen = beobachteDatenanfragen(page);
	const zaehle = async () =>
		(await page.evaluate(SYMBOLKNOEPFE, { ziel: ZIEL, messen: false })).length;
	/** @type {{ zuKlein: string[], eckig: string[], flaeche: string[] }} */
	const befunde = { zuKlein: [], eckig: [], flaeche: [] };
	let untersucht = 0;
	let runde = 0;

	/**
	 * Misst die Symbolknöpfe der Seite und trägt jeden Verstoß unter ihrem Namen ein.
	 * @param {string} seite
	 */
	const pruefe = async (seite) => {
		// Eine Sprechblase über den Messpunkten wäre ein Fund, den es ohne Maus nicht gibt.
		await page.mouse.move(0, 0);
		for (const k of await page.evaluate(SYMBOLKNOEPFE, { ziel: ZIEL, messen: true })) {
			untersucht++;
			const wer = `${seite}: "${k.label}"`;
			if (k.breite < MIN_FLAECHE || k.hoehe < MIN_FLAECHE) {
				befunde.zuKlein.push(`${wer} ist ${k.breite}×${k.hoehe} px [${k.klassen}]`);
			}
			if (/(^|\s)icon-btn(\s|$)/.test(k.klassen)) {
				runde++;
				if (k.radius < Math.min(k.breite, k.hoehe) / 2) {
					befunde.eckig.push(`${wer} hat ${k.radius} px Rundung bei ${k.breite}×${k.hoehe} px`);
				}
			}
			if (k.fremd.length > 0) befunde.flaeche.push(`${wer}: ${k.fremd.join('; ')}`);
			if (k.ueberdeckt.length > 0) {
				befunde.flaeche.push(`${wer} überdeckt ${k.ueberdeckt.join('; ')}`);
			}
		}
	};

	await uiLogin(page);
	for (const [seite, pfad] of SEITEN) {
		await gehZu(page, pfad);
		let gemessen = await warteAufInhalt(anfragen, zaehle).catch((err) => {
			throw new Error(`${seite}: ${err.message}`, { cause: err });
		});
		// Ein übersprungener Messpunkt sähe aus wie ein bestandener: Der Helfer wartet hart auf
		// beide Symbole der Bestellung.
		if (pfad === '/bestellungen') {
			await oeffneBestellungsDetail(page, marke);
			gemessen = await warteAufInhalt(anfragen, zaehle);
		}
		await pruefeNichtZuFruehGemessen(page, anfragen, zaehle, gemessen, seite);
		await pruefe(seite);
	}

	const titelID = querySQL(`SELECT id FROM buecher_titel WHERE titel = 'E2E-Treffer ${s}'`);
	await gehZu(page, `/medienkatalog/buch/${titelID}`);
	await page.getByRole('tab', { name: /^Exemplare/ }).click();
	for (const nummer of [`B-TF${s.slice(0, 6)}`, `AUTO-TF${s.slice(0, 6)}`]) {
		await page.getByRole('checkbox', { name: `Exemplar ${nummer} auswählen` }).waitFor();
	}
	await page.getByRole('button', { name: 'Exemplar löschen' }).first().waitFor();
	await pruefe('Buchakte, Exemplare');

	await oeffneSchuelerProfil(page, `Treffer${s}`);
	await pruefe('Leserakte');

	// Eingeklappte Navigation: Ihr Umschalter existiert nur in diesem Zustand.
	await gehZu(page, '/bestellungen');
	await warteAufInhalt(anfragen, zaehle);
	await page.getByRole('button', { name: 'Navigation einklappen' }).click();
	await page.getByRole('button', { name: 'Navigation ausklappen' }).waitFor();
	await pruefe('Navigation eingeklappt');

	// Selbstschutz: Bricht eine Navigation oder ein Selektor, misst der Test still nichts mehr.
	// Die Untergrenze ist grob; sie fängt den Totalausfall und schreibt keine Zahl fest.
	expect(
		untersucht,
		'Der Test hat fast keine Symbolknöpfe gefunden: Vermutlich bricht eine Navigation, und er ' +
			'misst nichts mehr.'
	).toBeGreaterThan(40);
	expect(runde, 'Der Test hat keinen Knopf mit .icon-btn gefunden.').toBeGreaterThan(0);

	// Weich, damit ein Lauf alle drei Arten von Funden nennt.
	expect
		.soft(
			befunde.zuKlein,
			`Symbolknöpfe unter ${MIN_FLAECHE}×${MIN_FLAECHE} px.\n` +
				`Behebung: die Klasse .icon-btn (styles/komponenten.css) statt eigener Polsterung.\n\n` +
				befunde.zuKlein.join('\n')
		)
		.toEqual([]);
	expect
		.soft(
			befunde.eckig,
			`Symbolknöpfe mit .icon-btn, die nicht rund sind (${runde} gemessen).\n` +
				`Behebung: Die Rundung steht in .icon-btn (styles/komponenten.css).\n\n` +
				befunde.eckig.join('\n')
		)
		.toEqual([]);
	expect
		.soft(
			befunde.flaeche,
			`Symbolknöpfe, denen die ${ZIEL}×${ZIEL} px um ihre Mitte nicht gehören.\n` +
				`Behebung: .icon-btn setzen; zu Nachbarn 16 px Abstand, zum Rand eines Containers, der\n` +
				`abschneidet, 8 px.\n\n` +
				befunde.flaeche.join('\n')
		)
		.toEqual([]);
});
