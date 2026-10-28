import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { parse as svelteParse } from 'svelte/compiler';
import { walk } from 'estree-walker';
import { sammleQuelldateien, srcRoot, relPfad } from './hygiene-quellen.js';

// Gate zur Bugklasse „Grund ohne Rolle" (docs/sweeps.md): Eine Zeile oder ein Kasten in den
// Fehlerrollen, die einen Fehler zeigen, erscheinen nach einem Klick oder einem gescheiterten
// Abruf, ohne dass der Fokus zu ihnen wandert. Ohne role="alert" liest ein Bildschirmleser nichts
// vor. M3, Text fields, Accessibility: „Text field error messages should be given an “alert” role
// in accessibility labels"; LadeFehler trägt dieselbe Rolle. Als Rolle zählen auch status und
// aria-live, am Element oder einem um es herum.
//
// Sieht nicht: einen Fehler ohne die Klassen text-error, text-on-error-container oder
// bg-error-container, oder mit ihnen erst aus einem Ausdruck in der Klasse; einen Fehler unter
// einem Namen ohne fehler/error, einen festen Satz in der Fehlerfarbe und eine Rolle an einem
// Element, das schon vor dem Fehler steht und sich nicht ändert.

const FEHLERNAME = /fehler|error/i;
const FEHLERFARBE = /(^|\s)(text-error|text-on-error-container|bg-error-container)(\s|$)/;

/**
 * Ausnahmen mit Grund. Das ist ein Bestand bewusster Ausnahmen, KEINE Erlaubnis.
 * @type {Record<string, [number, string]>}
 */
const AUSNAHMEN = {
	'src/lib/components/layout/NachbuchMeldungen.svelte': [
		1,
		'eine Liste früherer Meldungen, kein neuer Fehler; mit Rolle läse das Öffnen jede vor'
	]
};

/** Der feste Text eines Attributs, ohne Ausdrücke. @param {any} attribut */
const text = (attribut) =>
	Array.isArray(attribut?.value)
		? attribut.value.map((/** @type {any} */ t) => (t.type === 'Text' ? t.data : ' ')).join('')
		: '';

/** @param {any} element @param {string} name */
const attribut = (element, name) =>
	element.attributes?.find((/** @type {any} */ a) => a.type === 'Attribute' && a.name === name);

/** Wird das Element vorgelesen, sobald es erscheint? @param {any} element */
const vorgelesen = (element) =>
	['alert', 'status'].includes(text(attribut(element, 'role')).trim()) ||
	attribut(element, 'aria-live') !== undefined;

/** Zeigt das Element einen Ausdruck, der einen Fehler trägt? @param {any} element @param {string} quelle */
function zeigtFehler(element, quelle) {
	let ja = false;
	walk(element.fragment, {
		enter(/** @type {any} */ k) {
			if (k.type === 'ExpressionTag')
				ja ||= FEHLERNAME.test(quelle.slice(k.expression.start, k.expression.end));
		}
	});
	return ja;
}

/**
 * Jede Fehlerzeile ohne Rolle an sich oder einem Vorfahren, mit Zeile; dazu die Zahl der
 * Fehlerzeilen überhaupt.
 * @param {string} quelle
 */
export function fehlerzeilenOhneRolle(quelle) {
	/** @type {string[]} */
	const funde = [];
	let zeilen = 0;
	/** @type {any[]} */
	const vorfahren = [];
	walk(/** @type {any} */ (svelteParse(quelle, { modern: true })).fragment, {
		enter(/** @type {any} */ k) {
			if (k.type !== 'RegularElement') return;
			if (FEHLERFARBE.test(text(attribut(k, 'class'))) && zeigtFehler(k, quelle)) {
				zeilen++;
				if (!vorgelesen(k) && !vorfahren.some(vorgelesen))
					funde.push(`${quelle.slice(0, k.start).split('\n').length}: <${k.name}>`);
			}
			vorfahren.push(k);
		},
		leave(/** @type {any} */ k) {
			if (k.type === 'RegularElement') vorfahren.pop();
		}
	});
	return { funde, zeilen };
}

describe('Fehlerzeilen werden vorgelesen', () => {
	const ergebnis = sammleQuelldateien(srcRoot)
		.filter((f) => f.endsWith('.svelte'))
		.map((f) => ({ datei: relPfad(f), ...fehlerzeilenOhneRolle(readFileSync(f, 'utf8')) }));

	it('findet die Fehlerzeilen', () => {
		// Fände der Detektor keine mehr, wäre er blind und das Gate grün.
		expect(ergebnis.reduce((summe, e) => summe + e.zeilen, 0)).toBeGreaterThanOrEqual(25);
	});

	it('jede Fehlerzeile trägt eine Rolle', () => {
		expect(
			ergebnis
				.filter((e) => e.funde.length !== (AUSNAHMEN[e.datei]?.[0] ?? 0))
				.flatMap((e) => e.funde.map((f) => `${e.datei}:${f}`)),
			'Fehlerzeile ohne role="alert": Ein Bildschirmleser liest sie nicht vor'
		).toEqual([]);
	});

	it('jede Ausnahme trifft noch zu', () => {
		const ohne = Object.keys(AUSNAHMEN).filter(
			(datei) => ergebnis.find((e) => e.datei === datei)?.funde.length !== AUSNAHMEN[datei][0]
		);
		expect(ohne, 'aus AUSNAHMEN austragen oder die Zahl anpassen').toEqual([]);
	});

	it('Selbstprobe: der Detektor fasst die Formen', () => {
		for (const form of [
			'{#if fehler}<p class="text-xs text-error">{fehler}</p>{/if}',
			'<span class="font-semibold text-error">Speichern fehlgeschlagen: {ablage.speicherfehler}</span>',
			'<div class="py-4"><p class="text-error">{s.error}</p></div>',
			'<div class="rounded-xl bg-error-container p-4 text-on-error-container">{exportFehler}</div>',
			'<p class="text-error" role="presentation"><strong>{klassenLadefehler(x)}</strong></p>'
		]) {
			expect(fehlerzeilenOhneRolle(form).funde, form).not.toEqual([]);
		}
		for (const form of [
			'<p role="alert" class="text-xs text-error">{fehler}</p>',
			'<div role="alert"><p class="text-error">{fehler}</p></div>',
			'<p aria-live="polite" class="text-error">{fehler}</p>',
			'<span class="text-error">Gesperrt</span>',
			'<p class="text-error/80">{fehler}</p>',
			'<span class="text-error">{formatEuro(summe)}</span>'
		]) {
			expect(fehlerzeilenOhneRolle(form).funde, form).toEqual([]);
		}
	});
});
