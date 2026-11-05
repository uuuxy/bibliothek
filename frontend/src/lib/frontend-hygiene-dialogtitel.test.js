import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { parse as svelteParse } from 'svelte/compiler';
import { walk } from 'estree-walker';
import { sammleQuelldateien, srcRoot, relPfad } from './hygiene-quellen.js';

// Der Titel eines Dialogs hat eine Form: die Klasse `dialog-titel` aus styles/komponenten.css
// (Headline small, Regular, On surface; material-web md.comp.dialog.headline), im Vollbild-Dialog
// `dialog-titel-vollbild` (Title large). Geprüft wird die erste Überschrift in jedem <Dialog> und
// in jedem Element mit role="dialog" oder "alertdialog". Daneben steht keine Klasse für Größe,
// Stärke oder Farbe: Tailwinds Utilities liegen in einer späteren Schicht und schlügen die Klasse.
//
// Sieht nicht: einen Titel ohne Überschrift-Element und einen Dialog, der weder ui/Dialog nutzt
// noch eine Rolle trägt.

/** Ausnahmen mit Grund, KEINE Erlaubnis. @type {Record<string, string>} */
const AUSNAHMEN = {
	'src/lib/components/omnibox/OmniboxSperrHinweis.svelte':
		'Alarm der Theke: unterbricht den Scan und ist bewusst laut gebaut',
	'src/lib/components/omnibox/OmniboxVormerkungHinweis.svelte':
		'Alarm der Theke: unterbricht den Scan und ist bewusst laut gebaut'
};

const UEBERSTIMMT = /^(?!text-(left|center|right|start|end)$)(text-|font-|leading-|tracking-)/;

/** @param {any} knoten @param {string} name */
function attribut(knoten, name) {
	return knoten.attributes?.find(
		(/** @type {any} */ a) => a.type === 'Attribute' && a.name === name
	);
}

/** Der feste Text eines Attributs; null, wenn ein Ausdruck darin steht. @param {any} a */
function festerWert(a) {
	if (!a || a.value === true) return '';
	const teile = Array.isArray(a.value) ? a.value : [a.value];
	if (teile.some((/** @type {any} */ t) => t.type !== 'Text')) return null;
	return teile.map((/** @type {any} */ t) => t.data).join('');
}

/** @param {any} wurzel */
function ersteUeberschrift(wurzel) {
	/** @type {any} */
	let fund = null;
	walk(wurzel, {
		enter(/** @type {any} */ k) {
			if (fund) return this.skip();
			if (k !== wurzel && k.type === 'RegularElement' && /^h[1-6]$/.test(k.name)) fund = k;
		}
	});
	return fund;
}

/** @param {string} quelle @returns {{ zeile: number, fehler: string }[]} */
function pruefeDatei(quelle) {
	const ast = svelteParse(quelle, { modern: true });
	/** @type {{ zeile: number, fehler: string }[]} */
	const funde = [];
	walk(/** @type {any} */ (ast.fragment), {
		enter(/** @type {any} */ k) {
			const istDialog =
				(k.type === 'Component' && k.name === 'Dialog') ||
				(k.type === 'RegularElement' &&
					['dialog', 'alertdialog'].includes(festerWert(attribut(k, 'role')) ?? ''));
			if (!istDialog) return;
			const titel = ersteUeberschrift(k);
			if (!titel) return;
			const soll =
				festerWert(attribut(k, 'size')) === 'voll' ? 'dialog-titel-vollbild' : 'dialog-titel';
			const klasse = festerWert(attribut(titel, 'class'));
			const zeile = quelle.slice(0, titel.start).split('\n').length;
			if (klasse === null) {
				funde.push({ zeile, fehler: 'Klasse mit Ausdruck' });
				return;
			}
			const klassen = klasse.split(/\s+/).filter(Boolean);
			if (!klassen.includes(soll)) funde.push({ zeile, fehler: `ohne ${soll}` });
			const fremd = klassen.filter((c) => UEBERSTIMMT.test(c));
			if (fremd.length) funde.push({ zeile, fehler: `überstimmt mit ${fremd.join(' ')}` });
		}
	});
	return funde;
}

describe('Dialogtitel', () => {
	it('tragen dialog-titel und keine eigene Größe, Stärke oder Farbe', () => {
		const verstoesse = [];
		let geprueft = 0;
		for (const datei of sammleQuelldateien(srcRoot)) {
			if (!datei.endsWith('.svelte')) continue;
			const rel = relPfad(datei);
			if (AUSNAHMEN[rel]) continue;
			const quelle = readFileSync(datei, 'utf8');
			if (!/<Dialog\b|role="(alert)?dialog"/.test(quelle)) continue;
			geprueft++;
			for (const f of pruefeDatei(quelle)) verstoesse.push(`${rel}:${f.zeile} ${f.fehler}`);
		}
		// Nicht-leer-Garantie: Fände die Suche keine Dialoge mehr, wäre das Gate stumm grün.
		expect(geprueft).toBeGreaterThan(20);
		expect(verstoesse).toEqual([]);
	});

	it('erkennt einen Titel in alter Form', () => {
		const alt = '<Dialog open><h3 class="text-lg font-bold text-on-surface">X</h3></Dialog>';
		expect(pruefeDatei(alt).map((f) => f.fehler)).toEqual([
			'ohne dialog-titel',
			'überstimmt mit text-lg font-bold text-on-surface'
		]);
		const voll =
			'<Dialog size="voll">{#snippet header()}<h2 class="dialog-titel">X</h2>{/snippet}</Dialog>';
		expect(pruefeDatei(voll).map((f) => f.fehler)).toEqual(['ohne dialog-titel-vollbild']);
	});
});
