import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { parse as svelteParse } from 'svelte/compiler';
import { parse as espreeParse } from 'espree';
import { walk } from 'estree-walker';
import { sammleQuelldateien, srcRoot, relPfad } from './hygiene-quellen.js';

// Gate zur Bugklasse „Eine Fehlerantwort wird durch einen eigenen Satz ersetzt" (docs/sweeps.md):
// Im Zweig, der bei einer abgelehnten Antwort läuft (`!res.ok`, das else von `res.ok`, der Rest
// nach `if (res.ok) return`), entsteht die Meldung aus leseApiFehler. Ein eigener Satz dort
// verdeckt den Grund des Servers, etwa „Keine Berechtigung für diese Aktion"; er bleibt als
// Vorgabe für einen leeren Rumpf. Ein Satz zu einem bestimmten Status (`res.status === 409`) ist
// gewollt.
//
// Sieht nicht: einen eigenen Satz, der erst über eine Variable in die Meldung kommt, einen Satz
// mit einer anderen Variablen als dem Status darin, eine Meldung unter einem Namen ohne
// fehler/meldung/error/message und eine Markierung (`fehler = true`), aus der die Seite einen
// festen Text macht.

const MELDER = new Set(['melde', 'addToast', 'showSnackbar']);
const MELDUNGSNAME = /fehler|meldung|error|message/i;

/** @param {string} datei @param {string} quelle @returns {any} */
function baum(datei, quelle) {
	if (datei.endsWith('.svelte')) return svelteParse(quelle, { modern: true });
	return espreeParse(quelle, { ecmaVersion: 'latest', sourceType: 'module' });
}

/** `x.ok` oder `x?.ok`. @param {any} n */
function istOk(n) {
	const m = n?.type === 'ChainExpression' ? n.expression : n;
	return m?.type === 'MemberExpression' && !m.computed && m.property.name === 'ok';
}

/** Läuft der Fehlerzweig, wenn die Bedingung gilt ('dann'), wenn nicht ('sonst'), oder ist sie
 *  keine Prüfung der Antwort (null)? @param {any} t @returns {'dann' | 'sonst' | null} */
function zweigDerAblehnung(t) {
	if (t.type === 'UnaryExpression' && t.operator === '!' && istOk(t.argument)) return 'dann';
	if (istOk(t)) return 'sonst';
	if (t.type === 'BinaryExpression' && istOk(t.left) && typeof t.right.value === 'boolean') {
		const gleich = t.operator === '===' || t.operator === '==';
		return (t.right.value === false) === gleich ? 'dann' : 'sonst';
	}
	// Mit `&&` oder `||` verbunden, läuft der Zweig bei jeder Ablehnung oder bei einem Teil davon
	// (`!res.ok && res.status !== 404`).
	if (t.type !== 'LogicalExpression') return null;
	const seiten = [zweigDerAblehnung(t.left), zweigDerAblehnung(t.right)];
	if (seiten.includes('dann')) return 'dann';
	return seiten.includes('sonst') ? 'sonst' : null;
}

/** Die Knoten, die bei einer abgelehnten Antwort laufen. @param {any} n @param {any} eltern */
function fehlerzweig(n, eltern) {
	const art = zweigDerAblehnung(n.test);
	if (art === 'dann') return [n.consequent];
	if (art !== 'sonst') return [];
	if (n.alternate) return [n.alternate];
	if (n.type !== 'IfStatement' || !Array.isArray(eltern?.body)) return [];
	const letzte = n.consequent.type === 'BlockStatement' ? n.consequent.body.at(-1) : n.consequent;
	if (letzte?.type !== 'ReturnStatement' && letzte?.type !== 'ThrowStatement') return [];
	return eltern.body.slice(eltern.body.indexOf(n) + 1);
}

/** Ob die Bedingung nach einem bestimmten Status fragt (`res.status === 409`). @param {any} t */
function fragtStatus(t) {
	let gefunden = false;
	walk(t, {
		enter(/** @type {any} */ n) {
			if (n.type !== 'BinaryExpression' || (n.operator !== '===' && n.operator !== '==')) return;
			const [a, b] = [n.left, n.right];
			const status = (/** @type {any} */ x) =>
				x.type === 'MemberExpression' && x.property.name === 'status';
			if ((status(a) && typeof b.value === 'number') || (status(b) && typeof a.value === 'number'))
				gefunden = true;
		}
	});
	return gefunden;
}

/** Ob ein Ausdruck nur aus eigenen Worten und dem Status der Antwort besteht. @param {any} a
 *  @returns {boolean} */
function eigenerSatz(a) {
	switch (a?.type) {
		case 'Literal':
			return typeof a.value === 'string';
		case 'TemplateLiteral':
			return a.expressions.every(teilEinesSatzes);
		case 'BinaryExpression':
			return a.operator === '+' && teilEinesSatzes(a.left) && teilEinesSatzes(a.right);
		case 'ConditionalExpression':
		case 'LogicalExpression':
			return eigenerSatz(a.consequent ?? a.left) && eigenerSatz(a.alternate ?? a.right);
		case 'CallExpression':
			return a.callee.name === 'String' && a.arguments.every(teilEinesSatzes);
		default:
			return false;
	}
}

/** @param {any} a */
function teilEinesSatzes(a) {
	if (a.type === 'Literal') return true;
	if (a.type === 'MemberExpression' && ['status', 'statusText'].includes(a.property?.name))
		return true;
	return eigenerSatz(a);
}

/** Der Text, der an diesem Knoten zur Meldung wird, sonst null. @param {any} n */
function meldungAn(n) {
	if (n.type === 'NewExpression' && /(Error|Fehler)$/.test(n.callee.name ?? ''))
		return n.arguments[0];
	if (n.type === 'ThrowStatement') return n.argument;
	if (n.type === 'CallExpression') {
		const name = n.callee.type === 'MemberExpression' ? n.callee.property.name : n.callee.name;
		return MELDER.has(name) ? n.arguments[0] : null;
	}
	if (n.type === 'AssignmentExpression' && n.operator === '=') {
		const ziel = n.left.type === 'MemberExpression' ? n.left.property.name : n.left.name;
		return MELDUNGSNAME.test(ziel ?? '') ? n.right : null;
	}
	if (n.type === 'Property' && !n.computed && MELDUNGSNAME.test(n.key.name ?? '')) return n.value;
	return null;
}

/**
 * Jeder eigene Satz in einem Fehlerzweig, mit Zeile; dazu die Zahl der Fehlerzweige.
 * @param {string} datei @param {string} quelle
 */
export function eigeneSaetzeBeiAblehnung(datei, quelle) {
	/** @type {string[]} */
	const funde = [];
	let zweige = 0;
	walk(baum(datei, quelle), {
		enter(/** @type {any} */ n, /** @type {any} */ eltern) {
			if (n.type !== 'IfStatement' && n.type !== 'ConditionalExpression') return;
			const zweig = fehlerzweig(n, eltern);
			if (zweig.length > 0) zweige++;
			/** @type {[number, number][]} */
			const gewollt = [];
			for (const teil of zweig) {
				walk(teil, {
					enter(/** @type {any} */ k) {
						if (k.type === 'IfStatement' && fragtStatus(k.test))
							gewollt.push([k.consequent.start, k.consequent.end]);
						const satz = meldungAn(k);
						if (!satz || !eigenerSatz(satz)) return;
						if (gewollt.some(([von, bis]) => k.start >= von && k.end <= bis)) return;
						const zeile = quelle.slice(0, k.start).split('\n').length;
						funde.push(`${zeile}: ${quelle.slice(satz.start, satz.end)}`);
					}
				});
			}
		}
	});
	return { funde: [...new Set(funde)], zweige };
}

/** Eine Form zur Probe, als Rumpf einer Funktion. @param {string} form */
const probe = (form) => eigeneSaetzeBeiAblehnung('probe.js', `async function f() {\n${form}\n}`);

describe('Fehlerantwort: der Grund des Servers statt eines eigenen Satzes', () => {
	const dateien = sammleQuelldateien(srcRoot);
	const ergebnis = dateien.map((f) => ({
		datei: relPfad(f),
		...eigeneSaetzeBeiAblehnung(f, readFileSync(f, 'utf8'))
	}));

	it('findet die Prüfungen der Antworten', () => {
		expect(dateien.length).toBeGreaterThan(300);
		// Fände der Detektor keine Fehlerzweige mehr, wäre er blind und das Gate grün.
		expect(ergebnis.reduce((summe, e) => summe + e.zweige, 0)).toBeGreaterThan(120);
	});

	it('kein Fehlerzweig meldet einen eigenen Satz statt des Grundes vom Server', () => {
		expect(
			ergebnis.flatMap((e) => e.funde.map((f) => `${e.datei}:${f}`)),
			'Eigener Satz statt des Grundes vom Server — leseApiFehler(res, satz) aus apiFetch.js nehmen'
		).toEqual([]);
	});

	it('Selbstprobe: der Detektor fasst die Formen', () => {
		for (const form of [
			"if (!res.ok) throw new Error('Fehler beim Laden');",
			"if (!res.ok) {\n\tif (res.status === 401) throw new Error('UNAUTHORIZED');\n\tthrow new Error('Buch konnte nicht geladen werden');\n}",
			'if (!res.ok) throw new Error(`HTTP ${res.status}`);',
			'if (!res.ok) throw new Error(String(res.status));',
			"if (res.ok) {\n\tdaten = await res.json();\n} else {\n\tmelde('Fehler beim Speichern', 'error');\n}",
			"if (res.ok) return res.json();\nsnackbars.addToast('Ein Fehler ist aufgetreten.', 'error');",
			"if (!antwort.ok) {\n\tfehler = 'Schulbücher konnten nicht geladen werden.';\n\treturn;\n}",
			"if (!res.ok) this.fehler = res.status === 403 ? 'Kein Recht.' : `Nicht abrufbar (${res.status}).`;",
			"return apiFetch(u).then((r) => (r.ok ? r.json() : Promise.reject(new Error('Laden gescheitert'))));",
			"if (!res?.ok || lauf !== meinLauf) {\n\tladefehler = 'Fehler ' + res.status;\n\treturn;\n}",
			"if (res.ok === false) throw 'Abruf fehlgeschlagen';",
			"if (res.ok && daten) zeige(daten);\nelse return { ok: false, meldung: 'Speichern fehlgeschlagen.' };",
			"if (!res.ok) {\n\tif (res.status === 404) melde('Nicht da', 'info');\n\telse melde('Fehler', 'error');\n}",
			'if (!res.ok && res.status !== 404) {\n\tthis.fehler = `Quittieren fehlgeschlagen (${res.status}).`;\n\treturn;\n}',
			"if (res.ok || zwischengespeichert) zeige();\nelse melde('Laden gescheitert', 'error');"
		]) {
			expect(probe(form).funde, form).not.toEqual([]);
		}
		const knopf =
			'<button onclick={async () => { const r = await apiFetch(u); if (!r.ok) melde("Fehler", "error"); }}>x</button>';
		expect(eigeneSaetzeBeiAblehnung('probe.svelte', knopf).funde).not.toEqual([]);

		for (const form of [
			"if (!res.ok) throw new Error(await leseApiFehler(res, 'Fehler beim Laden'));",
			"if (!res.ok) {\n\tif (res.status === 401) throw new Error('UNAUTHORIZED');\n\tthrow new Error(await leseApiFehler(res));\n}",
			"if (!res.ok) this.fehler = res.status === 403 ? 'Kein Recht.' : await leseApiFehler(res, 'Nicht abrufbar.');",
			"if (!r.ok) {\n\tsnackbars.addToast(r.error || 'Fund konnte nicht verbucht werden.', 'error');\n\treturn;\n}",
			'if (!res.ok) throw new Error(meldung);',
			'if (!res.ok) {\n\tfehler = true;\n\treturn;\n}',
			"if (res.ok) melde('Gespeichert', 'success');",
			"const farbe = res.ok ? 'text-primary' : 'text-error';",
			"if (!res.ok) melde(`Nicht gelöscht: ${await leseApiFehler(res)}`, 'error');"
		]) {
			expect(probe(form).funde, form).toEqual([]);
		}
	});
});
