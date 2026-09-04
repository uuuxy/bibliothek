// Gate für die M3-BAUFORM: keine Fläche trägt Rahmen UND Erhebung zugleich.
//
// Die Regel ist nicht ausgelegt, sondern in Googles Token-Spezifikation nachgezählt
// (material-web v0.192, „Design system display name: Google Material 3", alle 84
// Bauteile ausgewertet):
//
//   6 Bauteile tragen einen Rahmen   — data-table, outlined-button, outlined-card,
//                                      outlined-menu-button, outlined-segmented-button,
//                                      outlined-text-field
//   36 Bauteile tragen eine Erhebung — dialog level3, menu level2, snackbar level3,
//                                      FAB level3, elevated-card/-button level1, …
//   Die SCHNITTMENGE IST LEER.
//
// Rahmen und Erhebung sind in M3 zwei Wege, eine Fläche abzugrenzen — nie eine Summe.
// Welcher der beiden Teile weicht, entscheidet die Bauteilrolle: Beim Dialog geht der
// Rahmen (er hat kein outline-Token, aber level3), beim outlined-Button und bei der
// data-table geht der Schatten (sie haben 1px Rahmen und kein elevation-Token).
//
// GEMESSEN, NICHT GEGREPT. Drei Fallen stecken darin, und jede hat bei der Erhebung
// dieses Befundes schon einmal ein falsches Ergebnis geliefert:
//
//   1. Tailwind rendert eine ungesetzte Schatten-Variable als
//      „rgba(0, 0, 0, 0) 0px 0px 0px 0px" — ein Schatten mit Deckkraft 0, der nichts
//      malt. Wer nur auf `boxShadow !== 'none'` prüft, findet die halbe Anwendung.
//   2. `ring-*` ist bei Tailwind ebenfalls ein box-shadow, nur ohne Weichzeichnung.
//      Ein Fokusring ist keine Erhebung — deshalb zählt nur blur > 0.
//   3. Eine outlined card DARF bei :hover auf level1 steigen (hover-container-elevation
//      steht so in der Spezifikation). Gemessen wird deshalb ausschliesslich der
//      RUHEZUSTAND, mit der Maus in der Ecke.
//
// Und der Grund für den zweiten Teil dieser Datei: Ein Gate, das nur Routen abläuft,
// ist für Dialoge BLIND — sie sind zu. Genau daran ist die erste Messung dieses
// Befundes vorbeigelaufen und hat `Modal.svelte` (Rahmen + shadow-2xl, wirksam für
// elf Dialoge) nicht gesehen. Die Öffner unten sind deshalb Pflicht, und wenn einer
// nicht mehr greift, scheitert dieser Test LAUT statt still zu überspringen.
//
// ── WAS DIESES GATE NICHT SIEHT (bitte lesen, bevor man ihm vertraut) ─────────
// Es prüft, was im Moment der Messung GERENDERT ist. Alles, was an einen Zustand
// gebunden ist, entgeht ihm, wenn dieser Zustand gerade nicht hergestellt ist:
//
//   * Zustandsabhängige Meldungen. `BestelllinkHinweis` erscheint nur, solange
//     keine öffentliche Adresse gesetzt ist. Im Einzellauf dieses Gates war er
//     unsichtbar (grün), im vollen e2e-Lauf sichtbar (rot) — weil ein anderer
//     Test die Einstellung verändert hatte. Gefunden wurde er also durch Zufall,
//     nicht durch Konstruktion.
//   * Die elf selbstgebauten Overlays, die NICHT Modal.svelte benutzen
//     (StudentLockModal, DamageReportModal, WebcamCapture, OmniboxBlockAlert, …).
//     Eine Quelltext-Zählung fand am 04.09.2026 vierzig Kandidaten mit Rahmen und
//     Schatten in derselben Klassenliste; der grösste Teil davon sitzt in genau
//     diesen Overlays. Sie sind bewusst NICHT pauschal geändert worden: Bei
//     `OmniboxBlockAlert` ist der `border-4 border-rose-500` ein Alarmsignal an
//     der Theke, kein Dekor — das ist Einzelfallprüfung, kein Suchen-und-Ersetzen.
//     Der Posten steht in docs/befunde.md.
//
// Wer dieses Gate erweitert, erweitert die Öffnerliste — nicht die Regel.
import { test, expect } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';
import { uiLogin } from './helpers.js';

const hier = dirname(fileURLToPath(import.meta.url));

/** Routenliste aus Router.svelte lesen — eine handgepflegte Liste veraltet still. */
function routen() {
	const quelle = readFileSync(join(hier, '../src/lib/Router.svelte'), 'utf8');
	const block = quelle.match(/const tabToPath = \{([\s\S]*?)\};/);
	if (!block) throw new Error('tabToPath in Router.svelte nicht gefunden — Struktur geändert?');
	return [...block[1].matchAll(/'(\/[a-z0-9/-]+)'/g)].map((m) => m[1]);
}

/** Läuft IM BROWSER. Liefert je Fund Tag, Grösse, Klasse und die gemessenen Werte. */
const FUNDE = () => {
	const deckkraft = (farbe) => {
		const m = farbe.match(/rgba?\(([^)]*)\)/);
		if (!m) return 1;
		const teile = m[1].split(',').map((x) => parseFloat(x));
		return teile.length === 4 ? teile[3] : 1;
	};

	/** Sichtbarer Schlagschatten: nicht inset, Deckkraft > 0, Weichzeichnung > 0. */
	const erhoben = (bs) => {
		if (!bs || bs === 'none') return false;
		return bs.split(/,(?![^(]*\))/).some((teil) => {
			if (teil.includes('inset')) return false;
			const farbe = (teil.match(/rgba?\([^)]*\)/) || [])[0];
			if (farbe && deckkraft(farbe) === 0) return false;
			const laengen = (teil.replace(/rgba?\([^)]*\)/g, '').match(/-?[\d.]+px/g) || []).map(
				parseFloat
			);
			return laengen.length >= 3 && laengen[2] > 0; // [x, y, blur, spread]
		});
	};

	/** Sichtbarer Rahmen an mindestens einer Seite. */
	const umrandet = (st) =>
		['Top', 'Right', 'Bottom', 'Left'].some((seite) => {
			const breite = parseFloat(st[`border${seite}Width`]);
			if (!(breite > 0)) return false;
			if (['none', 'hidden'].includes(st[`border${seite}Style`])) return false;
			return deckkraft(st[`border${seite}Color`]) > 0;
		});

	const funde = [];
	for (const el of document.querySelectorAll('*')) {
		const kasten = el.getBoundingClientRect();
		if (kasten.width < 8 || kasten.height < 8) continue;
		const st = getComputedStyle(el);
		if (st.visibility === 'hidden' || st.display === 'none') continue;
		if (!umrandet(st) || !erhoben(st.boxShadow)) continue;
		funde.push({
			tag: el.tagName.toLowerCase(),
			groesse: `${Math.round(kasten.width)}×${Math.round(kasten.height)}`,
			text: (el.textContent || '').trim().replace(/\s+/g, ' ').slice(0, 40),
			klasse: (el.getAttribute('class') || '').slice(0, 100),
			schatten: st.boxShadow.replace(/rgba\(0, 0, 0, 0\) 0px 0px 0px 0px,?\s*/g, '').slice(0, 60)
		});
	}
	return funde;
};

/** Ruhezustand herstellen: nichts gehovert, Übergänge ausgelaufen. */
async function zurRuhe(page) {
	await page.mouse.move(0, 0);
	await page.waitForTimeout(250);
}

function melde(funde, ort) {
	const zeilen = funde.map(
		(f) =>
			`  <${f.tag}> ${f.groesse} „${f.text}"\n` +
			`      Klasse:   ${f.klasse}\n` +
			`      Schatten: ${f.schatten}`
	);
	return (
		`${funde.length} Fläche(n) auf ${ort} tragen Rahmen UND Schatten zugleich.\n` +
		`Diese Bauform kennt Material 3 bei keinem seiner 84 Bauteile.\n` +
		`Entweder der Rahmen geht (dann ist es eine erhobene Fläche: Dialog, Menü, Snackbar)\n` +
		`oder der Schatten (dann ist es eine umrandete: outlined-*, data-table).\n\n` +
		zeilen.join('\n\n')
	);
}

test.describe('Material 3: Bauform', () => {
	test('keine Fläche trägt Rahmen und Erhebung zugleich — auf allen Routen', async ({ page }) => {
		test.setTimeout(180_000);
		await uiLogin(page);

		/** @type {string[]} */
		const meldungen = [];
		for (const pfad of routen()) {
			await page.goto(pfad);
			await page.waitForTimeout(600);
			await zurRuhe(page);
			const funde = await page.evaluate(FUNDE);
			if (funde.length) meldungen.push(melde(funde, pfad));
		}
		expect(meldungen.join('\n\n' + '─'.repeat(70) + '\n\n'), meldungen.join('\n')).toBe('');
	});

	// Die Öffner sind Pflicht, nicht Kür: Ohne sie prüft dieser Test elf Dialoge nicht,
	// und genau dort sass der Befund, den die erste Messung übersehen hat.
	const DIALOGE = [
		{ name: 'Benutzer anlegen', pfad: '/berechtigungen', knopf: /Benutzer anlegen/ },
		{ name: 'Schüler anlegen', pfad: '/schuelerdatei', knopf: /Neuen Schüler anlegen/ }
	];

	for (const d of DIALOGE) {
		test(`keine Fläche trägt Rahmen und Erhebung zugleich — Dialog „${d.name}"`, async ({
			page
		}) => {
			test.setTimeout(60_000);
			await uiLogin(page);
			await page.goto(d.pfad);
			await page.waitForTimeout(600);

			// Scheitert LAUT, wenn der Öffner nicht mehr greift — ein Gate, das seinen
			// Prüfgegenstand still nicht mehr erreicht, meldet für immer grün.
			const oeffner = page.getByRole('button', { name: d.knopf });
			await expect(
				oeffner,
				`Der Öffner „${d.name}" ist auf ${d.pfad} nicht mehr da. Ohne ihn prüft dieses ` +
					`Gate den Dialog nicht — Beschriftung anpassen, nicht den Test entfernen.`
			).toBeVisible();
			await oeffner.click();

			const dialog = page.getByRole('dialog');
			await expect(dialog).toBeVisible();
			await zurRuhe(page);

			const funde = await page.evaluate(FUNDE);
			expect(funde.length === 0 ? '' : melde(funde, `Dialog „${d.name}"`)).toBe('');
		});
	}
});
