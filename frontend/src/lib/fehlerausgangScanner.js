// fehlerausgangScanner.js — findet verschluckte Fehlantworten am AST.
//
// Die Bugklasse (docs/sweeps.md): `if (res.ok) { … }` ohne `else`, und der Zweig endet
// nicht mit return/throw. Scheitert der Abruf, läuft der Code weiter, als wäre nichts
// gewesen, und der Zustand bleibt auf dem Stand vor dem Abruf: eine Trefferliste der
// Theke, die unter dem neuen Suchtext die Schüler des alten zeigt, oder „Der Papierkorb
// ist leer" für einen Papierkorb, der nur nicht geladen werden konnte.
//
// Am AST und nicht per Grep: Die harmlose Frühe-Rückkehr-Form (`if (res.ok) { …; return; }`
// mit Fehlerbehandlung danach) fällt hier heraus, weil sie sicher endet (sweeps.md,
// Regel 4: „Grep ist Hypothese, AST ist Befund").

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

/** Ist der Ausdruck `irgendwas.ok`? */
function istOkZugriff(node) {
	return node?.type === 'MemberExpression' && node.property?.name === 'ok';
}

/**
 * Fragt die Bedingung nach `.ok`, auch als ein Glied einer UND-Kette? Die Wettlauf-Form der
 * Suchfelder (`if (res.ok && nr === ladeNr)`) gehört zur selben Bugklasse, und eine Ratsche
 * prüft über alle Formen, nicht über das eine Beispiel (sweeps.md, Regel 6).
 */
function fragtNachOk(node) {
	if (!node) return false;
	if (istOkZugriff(node)) return true;
	if (node.type === 'LogicalExpression' && node.operator === '&&') {
		return fragtNachOk(node.left) || fragtNachOk(node.right);
	}
	return false;
}

/**
 * Ist das ein neutraler Ersatzwert — einer, den die Oberfläche als „nichts da" zeigt?
 *
 * `[]`, `{}`, `null`, `undefined`, `''` und `0`. Genau diese machen aus einem Ladefehler
 * ein leeres Ergebnis: „Keine Treffer" statt „konnte nicht geladen werden". Ein
 * Ersatzwert, der etwas AUSSAGT (`'fehlgeschlagen'`, `'error'`), ist Fehlerbehandlung und
 * gehört nicht in diese Liste.
 */
function istNeutralerErsatz(node) {
	if (!node) return false;
	if (node.type === 'ArrayExpression') return node.elements.length === 0;
	if (node.type === 'ObjectExpression') return node.properties.length === 0;
	if (node.type === 'Identifier') return node.name === 'undefined';
	if (node.type === 'Literal') return node.value === null || node.value === '' || node.value === 0;
	return false;
}

/**
 * Setzt der Fehlerzweig des Dreisatzes einen neutralen Wert? Er steht bei `res.ok ? … : []`
 * im zweiten Zweig, bei der Verneinung `!res.ok ? [] : …` im ersten.
 */
function fehlerzweigIstNeutral(node) {
	const { test } = node;
	if (fragtNachOk(test)) return istNeutralerErsatz(node.alternate);
	const verneint =
		test.type === 'UnaryExpression' && test.operator === '!' && fragtNachOk(test.argument);
	return verneint && istNeutralerErsatz(node.consequent);
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
				// Form 1: `if (res.ok …) { … }` ohne else, ohne sicheres Ende.
				if (node.type === 'IfStatement' && !node.alternate) {
					if (!fragtNachOk(node.test)) return;
					if (endetSicher(node.consequent)) return;
					zeilen.push(node.loc.start.line);
					return;
				}
				// Form 2: `res.ok ? … : []` — der Fehlerzweig ist da, setzt aber einen neutralen
				// Wert. Auf dem Bildschirm ist das nicht von „nichts gefunden" zu unterscheiden:
				// „Der Papierkorb ist leer" für einen Papierkorb, der nicht geladen werden konnte.
				// Die Bedingung zählt in jeder Form, auch als UND-Kette und verneint.
				if (node.type === 'ConditionalExpression' && fehlerzweigIstNeutral(node)) {
					zeilen.push(node.loc.start.line);
				}
			}
		});
	}
	return zeilen;
}
