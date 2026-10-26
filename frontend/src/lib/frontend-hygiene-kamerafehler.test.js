import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { parse as svelteParse } from 'svelte/compiler';
import { parse as espreeParse } from 'espree';
import { walk } from 'estree-walker';
import { sammleQuelldateien, srcRoot, relPfad } from './hygiene-quellen.js';

// Gate zur Bugklasse „Fehler des Browsers in seinem Wortlaut" (docs/sweeps.md): Verweigert der
// Browser die Kamera, nennt er den Grund am Namen der Ausnahme, sein Text dazu ist englisch und
// je Browser anders („Permission denied", „The request is not allowed by the user agent …").
// fehlertext gäbe ihn so weiter. Jeder Aufruf von getUserMedia steht deshalb in einem try, dessen
// catch kamerafehlertext (utils/kamerafehler.js) ruft.
//
// Sieht nicht: einen Aufruf unter anderem Namen (`const hole = mediaDevices.getUserMedia`) und
// einen catch, der kamerafehlertext ruft, den Satz aber nicht zeigt.

/** @param {string} datei @param {string} quelle @returns {any} */
function baum(datei, quelle) {
	if (datei.endsWith('.svelte')) return svelteParse(quelle, { modern: true });
	return espreeParse(quelle, { ecmaVersion: 'latest', sourceType: 'module' });
}

/** Ruft der Knoten irgendwo kamerafehlertext? @param {any} knoten */
function ruftHelfer(knoten) {
	let gefunden = false;
	walk(knoten, {
		enter(/** @type {any} */ k) {
			if (k.type === 'CallExpression' && k.callee.type === 'Identifier')
				gefunden ||= k.callee.name === 'kamerafehlertext';
		}
	});
	return gefunden;
}

/**
 * Jeder Aufruf von getUserMedia, dessen nächster umgebender try ohne kamerafehlertext im catch
 * ist (oder der in keinem try steht), mit Zeile; dazu die Zahl der Aufrufe.
 * @param {string} datei @param {string} quelle
 */
export function kameraOhneHelfer(datei, quelle) {
	/** @type {string[]} */
	const funde = [];
	let aufrufe = 0;
	/** @type {any[]} die try-Anweisungen, in deren Block der Lauf gerade steht */
	const versuche = [];
	walk(baum(datei, quelle), {
		enter(/** @type {any} */ k, /** @type {any} */ eltern, /** @type {any} */ feld) {
			if (eltern?.type === 'TryStatement' && feld === 'block') versuche.push(eltern);
			const aufruf = k.type === 'CallExpression' ? k.callee : null;
			const ziel = aufruf?.type === 'ChainExpression' ? aufruf.expression : aufruf;
			if (ziel?.type !== 'MemberExpression' || ziel.property?.name !== 'getUserMedia') return;
			aufrufe++;
			const versuch = versuche.findLast((v) => v.handler);
			if (versuch && ruftHelfer(versuch.handler)) return;
			funde.push(`${quelle.slice(0, k.start).split('\n').length}: getUserMedia`);
		},
		leave(/** @type {any} */ _k, /** @type {any} */ eltern, /** @type {any} */ feld) {
			if (eltern?.type === 'TryStatement' && feld === 'block') versuche.pop();
		}
	});
	return { funde, aufrufe };
}

/** Eine Form zur Probe, als Rumpf einer Funktion. @param {string} form */
const probe = (form) => kameraOhneHelfer('probe.js', `async function f() {\n${form}\n}`);

describe('Kamera: der Grund der Verweigerung auf Deutsch', () => {
	const ergebnis = sammleQuelldateien(srcRoot).map((f) => ({
		datei: relPfad(f),
		...kameraOhneHelfer(f, readFileSync(f, 'utf8'))
	}));

	it('findet die Aufrufe der Kamera', () => {
		// Fände der Detektor keinen Aufruf mehr, wäre er blind und das Gate grün.
		expect(ergebnis.reduce((summe, e) => summe + e.aufrufe, 0)).toBeGreaterThanOrEqual(2);
	});

	it('jeder Aufruf von getUserMedia nennt einen Fehler über kamerafehlertext', () => {
		expect(
			ergebnis.flatMap((e) => e.funde.map((f) => `${e.datei}:${f}`)),
			'Fehler der Kamera ohne kamerafehlertext — der Text des Browsers ist englisch'
		).toEqual([]);
	});

	it('Selbstprobe: der Detektor fasst die Formen', () => {
		for (const form of [
			'try {\n\ts = await navigator.mediaDevices.getUserMedia({ video: true });\n} catch (e) {\n\tmelde(fehlertext(e));\n}',
			"try {\n\ts = await navigator.mediaDevices.getUserMedia(c);\n} catch (e) {\n\tfehler = 'Kamerafehler: ' + e.message;\n}",
			'navigator.mediaDevices.getUserMedia(c).catch((e) => melde(e.message));',
			's = await navigator.mediaDevices?.getUserMedia(c);',
			'try {\n\ts = await navigator.mediaDevices.getUserMedia(c);\n} finally {\n\tlaeuft = false;\n}',
			'try {\n\tvorher();\n} catch (e) {\n\tzeige(kamerafehlertext(e));\n}\ns = await navigator.mediaDevices.getUserMedia(c);'
		]) {
			expect(probe(form).funde, form).not.toEqual([]);
		}
		for (const form of [
			'try {\n\ts = await navigator.mediaDevices.getUserMedia(c);\n} catch (e) {\n\tfehler = kamerafehlertext(e);\n}',
			"try {\n\tconst e = await erkenner();\n\ts = await navigator.mediaDevices.getUserMedia(c);\n} catch (error) {\n\tconsole.error('Kamerafehler:', error);\n\tmelde(kamerafehlertext(error));\n} finally {\n\tstarting = false;\n}"
		]) {
			expect(probe(form).funde, form).toEqual([]);
		}
	});
});
