// Rundgang: drei Fehlerarten, die man im Markup NICHT sieht — auf JEDER Route.
//
// Warum es diesen Test gibt: Die Fehler, die in diesem Projekt zuletzt von Hand gefunden
// wurden, standen alle nirgends in der Quelle. Sie ergaben sich erst aus dem gerenderten
// Bild:
//   * Das Buch-Karussell (10.08.2026): 2.816 px Inhalt auf 910 px Fläche, zehn von
//     sechzehn Büchern ausserhalb — und die Blätterpfeile auf opacity:0, sichtbar nur bei
//     :hover, am Tablet also nie.
//   * Die Aktionsspalte der Lieferantentabelle: der Knopf lag IM DOM, aber ausserhalb des
//     sichtbaren Bereichs. Playwright scrollte hin und meldete grün.
//
// Beides sind Messwerte, keine Klassennamen. Ein Grep findet sie prinzipiell nicht, und
// eine Spec pro Seite findet sie nur dort, wo jemand hingeschaut hat. Deshalb: alle
// Routen, drei Zusagen, automatisch mitwachsend.
//
// Die Routenliste wird aus Router.svelte GELESEN, nicht hier gepflegt. Eine
// handgepflegte Liste veraltet still — genau das ist am 10.08. passiert, als zwei Gates
// weiter /lehrer-portal besuchten und deshalb den Kiosk zweimal massen.
import { test, expect } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';
import { uiLogin, gehZu } from './helpers.js';

const hier = dirname(fileURLToPath(import.meta.url));
const ROUTER = join(hier, '../src/lib/Router.svelte');

/** Liest die Pfade aus dem tabToPath-Block in Router.svelte. */
function routen() {
	const quelle = readFileSync(ROUTER, 'utf8');
	const block = quelle.match(/const tabToPath = \{([\s\S]*?)\};/);
	if (!block) throw new Error('tabToPath in Router.svelte nicht gefunden — Struktur geändert?');
	return [...block[1].matchAll(/'(\/[a-z0-9/-]+)'/g)].map((m) => m[1]);
}

const ROUTEN = routen();

/**
 * Sammelt die drei Befunde einer Seite.
 *
 * Bewusst als EINE Funktion im Browser statt drei Durchläufen: Der Zustand einer Seite
 * (offene Menüs, laufende Animationen) soll für alle drei Messungen derselbe sein.
 */
const BEFUNDE = () => {
	const BEDIENELEMENTE = 'button, a[href], input, select, textarea, [role="button"], [role="tab"]';

	/** @param {Element} el */
	const beschreibe = (el) => {
		const text = (el.textContent || '').trim().slice(0, 40);
		const kennung =
			el.id || el.getAttribute('aria-label') || el.getAttribute('title') || text || '(namenlos)';
		return `<${el.tagName.toLowerCase()}> „${kennung}"`;
	};

	// 1. Scrollt die SEITE waagerecht? Eine Arbeitsfläche darf das nie — waagerechtes
	//    Scrollen gehört in einen Container, der es ausdrücklich anbietet.
	const wurzel = document.documentElement;
	const seitenUeberlauf =
		wurzel.scrollWidth > wurzel.clientWidth + 1
			? `${wurzel.scrollWidth} px Inhalt auf ${wurzel.clientWidth} px Fenster`
			: null;

	/** @type {string[]} */
	const unsichtbar = [];
	/** @type {string[]} */
	const abgeschnitten = [];

	for (const el of document.querySelectorAll(BEDIENELEMENTE)) {
		const kasten = el.getBoundingClientRect();
		if (kasten.width === 0 || kasten.height === 0) continue; // display:none & Co.

		const stil = getComputedStyle(el);
		if (stil.visibility === 'hidden') continue; // ausdrücklich verborgen, kein Zwitter

		// 2. Belegt Platz, ist aber durchsichtig: das Hover-Muster. Auf einem Tablet gibt
		//    es kein :hover — dort ist so ein Knopf dauerhaft unsichtbar und trotzdem da.
		if (parseFloat(stil.opacity) < 0.05) {
			unsichtbar.push(beschreibe(el));
			continue;
		}

		// 3. Liegt vollständig ausserhalb eines Vorfahren, der bei overflow:hidden
		//    abschneidet. Dann ist es unerreichbar — auch mit Scrollen, denn hidden bietet
		//    keines an. Playwright klickt so ein Element trotzdem und meldet grün.
		for (let v = el.parentElement; v; v = v.parentElement) {
			const vs = getComputedStyle(v);
			if (vs.overflowX !== 'hidden' && vs.overflowX !== 'clip') continue;
			const vk = v.getBoundingClientRect();
			if (vk.width === 0) break;
			if (kasten.right <= vk.left + 1 || kasten.left >= vk.right - 1) {
				abgeschnitten.push(`${beschreibe(el)} (abgeschnitten von <${v.tagName.toLowerCase()}>)`);
			}
			break; // nur der NÄCHSTE abschneidende Vorfahr zählt
		}
	}

	return {
		seitenUeberlauf,
		unsichtbar,
		abgeschnitten,
		geprueft: document.querySelectorAll(BEDIENELEMENTE).length
	};
};

/**
 * Misst erst, wenn die Seite steht.
 *
 * Ohne das misst der Test Animationen: Diese Oberfläche blendet Ansichten mit
 * `animate-fade-in` ein, und während der Blende steht JEDES Bedienelement auf opacity 0 —
 * der Test hätte auf jeder Route Dutzende „unsichtbare" Knöpfe gemeldet. Gewartet wird
 * auf zwei gleiche Messungen hintereinander, wie in control-hoehen.spec.js.
 */
async function stabileBefunde(/** @type {import('@playwright/test').Page} */ page) {
	let vorher = null;
	for (let i = 0; i < 25; i++) {
		const jetzt = await page.evaluate(BEFUNDE);
		if (vorher && JSON.stringify(vorher) === JSON.stringify(jetzt)) return jetzt;
		vorher = jetzt;
		await page.waitForTimeout(80);
	}
	return vorher;
}

test('Keine Route scrollt waagerecht, versteckt Bedienelemente hinter :hover oder schneidet sie ab', async ({
	page
}) => {
	await uiLogin(page);

	/** @type {string[]} */
	const maengel = [];
	let bedienelementeGesamt = 0;

	for (const pfad of ROUTEN) {
		await gehZu(page, pfad);
		const b = await stabileBefunde(page);
		bedienelementeGesamt += b.geprueft;

		if (b.seitenUeberlauf) {
			maengel.push(`${pfad}: die Seite scrollt waagerecht — ${b.seitenUeberlauf}`);
		}
		for (const e of b.unsichtbar) {
			maengel.push(
				`${pfad}: ${e} belegt Platz, steht aber auf opacity 0 (nur bei :hover sichtbar)`
			);
		}
		for (const e of b.abgeschnitten) {
			maengel.push(`${pfad}: ${e} liegt ausserhalb und ist nicht erreichbar`);
		}
	}

	// Zwei Gegenproben gegen einen stillen Nulllauf: Ohne sie wäre der Test auch grün,
	// wenn die Routenliste leer wäre oder jede Seite als leere Hülle geladen hätte.
	expect(ROUTEN.length, 'Routenliste aus Router.svelte ist leer').toBeGreaterThan(10);
	expect(
		bedienelementeGesamt,
		'über alle Routen wurden kaum Bedienelemente gefunden — laden die Seiten überhaupt?'
	).toBeGreaterThan(100);

	expect(maengel, `Befunde des Rundgangs über ${ROUTEN.length} Routen`).toEqual([]);
});
