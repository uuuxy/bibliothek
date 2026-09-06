// fehlerausgangScanner.js — findet verschluckte Fehlantworten am AST.
//
// Die Bugklasse (Sweep 06.09.2026, docs/sweeps.md): `if (res.ok) { … }` ohne `else`,
// und der Zweig endet nicht mit return/throw. Scheitert der Abruf, läuft der Code
// weiter, als wäre nichts gewesen — der Zustand bleibt auf dem Stand VOR dem Abruf.
//
// Was daraus wurde, ist keine Theorie: die Trefferliste der Theke, die unter dem neuen
// Suchtext die Schüler des alten zeigte; der Ausweis-Designer, dessen Auto-Speicherung
// nach einem fehlgeschlagenen Laden das Vorgabe-Design an alle Arbeitsplätze schrieb;
// „Der Papierkorb ist leer" für einen Papierkorb, der nur nicht geladen werden konnte.
//
// Warum am AST und nicht per Grep: Der erste Grep fand 65 Stellen, davon rund 28
// Kandidaten — der AST fand 26, und zwar die richtigen. Die harmlose Frühe-Rückkehr-Form
// (`if (res.ok) { …; return; }` mit Fehlerbehandlung danach) fällt hier korrekt heraus,
// weil sie sicher endet (sweeps.md, Regel 4: „Grep ist Hypothese, AST ist Befund").

import { readFileSync } from 'node:fs';
import { parse as svelteParse } from 'svelte/compiler';
import { parse as espreeParse } from 'espree';
import { walk } from 'estree-walker';

/** @type {import('espree').Options} */
const ESPREE = { ecmaVersion: 2024, sourceType: 'module', loc: true };

/** Die auswertbaren Programme einer Quelldatei (bei .svelte: instance + module script). */
function programme(datei, quelle) {
	if (datei.endsWith('.svelte')) {
		const baum = svelteParse(quelle, { modern: true });
		return [baum.instance?.content, baum.module?.content].filter(Boolean);
	}
	return [espreeParse(quelle, ESPREE)];
}

/**
 * Endet der Zweig sicher? Dann steht die Fehlerbehandlung DAHINTER, und das Fehlen
 * eines `else` ist kein Verschlucken, sondern eine frühe Rückkehr.
 */
function endetSicher(node) {
	if (!node) return false;
	if (node.type === 'ReturnStatement' || node.type === 'ThrowStatement') return true;
	if (node.type === 'BlockStatement') {
		return node.body.length > 0 && endetSicher(node.body[node.body.length - 1]);
	}
	if (node.type === 'IfStatement') {
		return endetSicher(node.consequent) && endetSicher(node.alternate);
	}
	return false;
}

/**
 * @param {string} datei Pfad (bestimmt auch die Parserwahl)
 * @param {string} [quelle] Inhalt; ohne Angabe wird die Datei gelesen
 * @returns {number[]} Zeilennummern der verschluckten Fehlausgänge
 */
export function findeVerschluckteFehlantworten(datei, quelle) {
	const inhalt = quelle ?? readFileSync(datei, 'utf8');
	/** @type {number[]} */
	const zeilen = [];
	for (const prog of programme(datei, inhalt)) {
		walk(/** @type {any} */ (prog), {
			enter(/** @type {any} */ node) {
				if (node.type !== 'IfStatement' || node.alternate) return;
				const test = node.test;
				if (test?.type !== 'MemberExpression' || test.property?.name !== 'ok') return;
				if (endetSicher(node.consequent)) return;
				zeilen.push(node.loc.start.line);
			}
		});
	}
	return zeilen;
}
