import { describe, it, expect } from 'vitest';
import { existsSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { parse as svelteParse } from 'svelte/compiler';
import { walk } from 'estree-walker';
import {
	srcRoot,
	sammleQuelldateien,
	relPfad,
	vergleicheMitBestand,
	ohneKommentare
} from './hygiene-quellen.js';
import { ICON_NAMEN } from './components/ui/iconNamen.js';

// Icons kommen aus ui/Icon: Googles Material Symbols, M3 (Styles, Icons): „Use the Material
// Symbols variable font to enable dynamic styling in product“. Eine Stelle für Schrift, Stil,
// Größe und Füllung; ein zweiter Weg (ein Paket, ein Pfad von Hand, die Schrift an anderer
// Stelle) zeichnete mit anderer Strichstärke und altert für sich.
const SVG = /<svg[\s>]/;

// Echte Zeichnungen: Hier entsteht ein Bild, kein Symbol. Diese Liste ist die Ausnahme, nicht
// der Rückstand.
const ZEICHNUNGEN = [
	'src/lib/components/statistik/StatistikVerlauf.svelte', // Verlaufsgraph, viewBox aus den Daten
	// Das Wellen-Motiv der Ausweis-Vorlage „Reis-Welle" als data:-URI, ein Bild auf dem Ausweis.
	'src/lib/designer/welleMotiv.js'
];

// In BuchCoverUpload steckt das <svg> in einer data:-URI als Platzhalterbild, also kein Symbol.
const SVG_BESTAND = ['src/inventur/lib/components/admin/BuchCoverUpload.svelte'];

const ICON_DATEI = 'src/lib/components/ui/Icon.svelte';
const SCHRIFT = join(srcRoot, 'styles/schriften/material-symbols-outlined');
// Loser Wert, der ein Icon nennt: { icon: 'delete' }, icon="delete", symbol: 'warning'.
const LOSER_WERT = /\b(?:icon|symbol)\s*[:=]\s*\{?\s*['"]([a-z0-9_]+)['"]/g;
const ICON_TAG = /<Icon\b[^>]*?\bname="([^"]+)"/g;

/** @returns {[string, string][]} Pfad relativ zu frontend/ und Quelltext */
function quellen() {
	return sammleQuelldateien(srcRoot).map((f) => [relPfad(f), readFileSync(f, 'utf8')]);
}

describe('Icons', () => {
	it('zeichnet keine Symbole von Hand', () => {
		const betroffen = quellen()
			.filter(([, q]) => SVG.test(q))
			.map(([f]) => f)
			.filter((f) => !ZEICHNUNGEN.includes(f))
			.sort();
		const { neu, inzwischenSauber } = vergleicheMitBestand(betroffen, [...SVG_BESTAND].sort());
		expect(neu, `Handgezeichnete Symbole, statt dessen ui/Icon:\n  ${neu.join('\n  ')}`).toEqual(
			[]
		);
		expect(inzwischenSauber, 'bitte aus SVG_BESTAND entfernen').toEqual([]);
	});

	it('hat einen Weg zum Icon: kein Paket, die Schrift nur in styles/icons.css', () => {
		/** @type {string[]} */
		const funde = [];
		for (const [f, q] of quellen()) {
			const code = ohneKommentare(q);
			if (/from\s*['"][^'"]*(?:lucide|heroicons|tabler|phosphor|material-symbols)/.test(code))
				funde.push(`${f}: Icon-Paket`);
			if (/Material Symbols/.test(code)) funde.push(`${f}: Schrift außerhalb von icons.css`);
			if (f !== ICON_DATEI && /data-icon\b|class="icon[\s"]/.test(code))
				funde.push(`${f}: Icon von Hand statt ui/Icon`);
		}
		expect(funde).toEqual([]);
	});

	it('holt die Schrift mit genau der Namensliste', () => {
		expect([...ICON_NAMEN], 'iconNamen.js sortiert und ohne Doppel').toEqual(
			[...new Set(ICON_NAMEN)].sort()
		);
		expect(existsSync(`${SCHRIFT}.woff2`)).toBe(true);
		const geholt = readFileSync(`${SCHRIFT}.namen.txt`, 'utf8').split('\n').filter(Boolean);
		expect(
			geholt,
			'Die Schrift passt nicht zur Liste: node scripts/iconschrift_holen.mjs laufen lassen'
		).toEqual([...ICON_NAMEN]);
	});

	it('nennt nur Icons aus der Liste und jedes Icon der Liste', () => {
		const bekannt = new Set(/** @type {readonly string[]} */ (ICON_NAMEN));
		const navIcon = readFileSync(join(srcRoot, 'lib/components/layout/NavIcon.svelte'), 'utf8');
		const navSchluessel = new Set(
			[...navIcon.matchAll(/^\s*'?([\w-]+)'?: '[a-z0-9_]+',?$/gm)].map((m) => m[1])
		);
		expect(navSchluessel.size, 'NavIcon gelesen').toBeGreaterThan(10);
		/** @type {string[]} */
		const unbekannt = [];
		let gelesen = 0;
		const genutzt = new Set();
		for (const [f, q] of quellen()) {
			const code = ohneKommentare(q);
			for (const [, name] of [...code.matchAll(ICON_TAG), ...code.matchAll(LOSER_WERT)]) {
				gelesen++;
				// Die Ziele der Navigation nennen den Schlüssel, den NavIcon übersetzt.
				const ok = f === 'src/lib/menue.js' ? navSchluessel.has(name) : bekannt.has(name);
				if (!ok) unbekannt.push(`${f}: ${name}`);
			}
			if (!f.endsWith('iconNamen.js'))
				for (const [, n] of code.matchAll(/['"]([a-z0-9_]+)['"]/g))
					if (bekannt.has(n)) genutzt.add(n);
		}
		expect(gelesen, 'der Sammler greift ins Leere').toBeGreaterThan(300);
		expect(unbekannt, 'Unbekannte Icons zeigen nur einen Buchstaben: in iconNamen.js').toEqual([]);
		expect(
			ICON_NAMEN.filter((n) => !genutzt.has(n)),
			'Icons in der Liste, die keine Stelle nennt: aus iconNamen.js nehmen, Schrift neu holen'
		).toEqual([]);
	});

	// Fällt das Programm hinter dem Proxy aus, liefert der Service Worker die Oberfläche aus
	// seinem Vorrat (vite-plugin-pwa, workbox.globPatterns). Liegt die Schrift nicht darin, stehen
	// nach der Sperrfrist von font-display: block die Namen der Icons („delete“) in den Knöpfen.
	it('liegt im Vorrat des Service Workers', () => {
		const konfig = readFileSync(join(srcRoot, '..', 'vite.config.js'), 'utf8');
		const muster = /globPatterns:\s*\[([^\]]*)\]/.exec(konfig)?.[1] ?? '';
		expect(muster, 'workbox.globPatterns in vite.config.js').toMatch(/woff2/);
	});

	// Eine Hülle fester Größe, die kleiner ist als ihr Icon: Es ragt heraus und frisst den Abstand
	// zum Nachbarn, oder die Hülle schneidet es ab. Gelesen am Baum; die Größe des Icons aus
	// groesse, sonst die Vorgabe von 20 px. Ein Icon in einem Bauteil oder Snippet misst der
	// Browser (e2e/icon-regeln.spec.js), nur auf den Seiten, die er öffnet.
	it('steht in keiner Hülle, die kleiner ist als das Icon', () => {
		/** @type {string[]} */
		const funde = [];
		let gesehen = 0;
		for (const [f, q] of quellen()) {
			if (!f.endsWith('.svelte') || !q.includes('<Icon')) continue;
			/** @type {any[]} */
			const stapel = [];
			const traeger = ['RegularElement', 'Component', 'SvelteElement', 'SnippetBlock'];
			walk(/** @type {any} */ (svelteParse(q, { modern: true }).fragment), {
				enter(/** @type {any} */ k) {
					if (k.type === 'Component' && k.name === 'Icon') {
						gesehen++;
						const huelle = stapel.at(-1);
						const px = iconGroesse(k);
						if (huelle?.type === 'RegularElement' && px) {
							const klassen = festeKlassen(huelle);
							const masse = [...klassen.matchAll(/\s(?:size|w|h)-(\d+(?:\.\d+)?)(?=\s)/g)];
							if (masse.some(([, n]) => Number(n) * 4 < px))
								funde.push(
									`${f}:${q.slice(0, k.start).split('\n').length}: ${px} px in „${klassen.trim()}“`
								);
						}
					}
					if (traeger.includes(k.type)) stapel.push(k);
				},
				leave(/** @type {any} */ k) {
					if (traeger.includes(k.type)) stapel.pop();
				}
			});
		}
		expect(gesehen, 'der Baum greift ins Leere').toBeGreaterThan(250);
		expect(funde, 'Hülle so groß wie das Icon oder groesse passend wählen').toEqual([]);
	});
});

/** Die Größe eines Icons in px; 0 für eins im Text (imText), das die Größe des Texts nimmt. @param {any} k */
function iconGroesse(k) {
	const attr = (/** @type {string} */ n) =>
		k.attributes.find((/** @type {any} */ a) => a.type === 'Attribute' && a.name === n);
	if (attr('imText')) return 0;
	const g = attr('groesse');
	if (!g) return 20;
	const teil = Array.isArray(g.value) ? g.value[0] : g.value;
	return Number(teil?.expression?.value ?? teil?.data) || 0;
}

/** Die festen Teile des class-Attributs, ohne Ausdrücke. @param {any} k */
function festeKlassen(k) {
	const a = k.attributes.find(
		(/** @type {any} */ x) => x.type === 'Attribute' && x.name === 'class'
	);
	if (!a || a.value === true) return '';
	const teile = Array.isArray(a.value) ? a.value : [a.value];
	return ` ${teile.map((/** @type {any} */ t) => (t.type === 'Text' ? t.data : ' ')).join('')} `;
}
