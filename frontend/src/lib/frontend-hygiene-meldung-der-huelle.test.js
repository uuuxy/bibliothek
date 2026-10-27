import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { parse as svelteParse } from 'svelte/compiler';
import { parse as espreeParse } from 'espree';
import { walk } from 'estree-walker';
import { sammleQuelldateien, srcRoot, relPfad } from './hygiene-quellen.js';

// Gate zur Bugklasse „Eigener Satz statt des Grundes vom Server" (docs/sweeps.md), die Form im
// catch: apiGet, apiPost, apiPut und apiDelete melden jeden Fehler selbst, mit seinem Grund
// (apiFetch.js). Setzt der catch danach einen eigenen Satz, ersetzt der diese Meldung, denn die
// Snackbar zeigt eine zur Zeit: „Vermerken nicht möglich." statt „Keine Berechtigung für diese
// Aktion". Erlaubt ist ein Satz, der den gefangenen Fehler nennt (fehlertext(e)): Er trägt den
// Grund und sagt dazu, was folgt.
//
// Sieht nicht: eine Hülle unter anderem Namen, einen catch in einer anderen Funktion als der, die
// die Hülle ruft, und einen Satz, der erst über eine Variable in die Meldung kommt.

const HUELLEN = new Set(['apiGet', 'apiPost', 'apiPut', 'apiDelete']);
const MELDER = new Set(['melde', 'addToast', 'showSnackbar']);

/** @param {string} datei @param {string} quelle @returns {any} */
function baum(datei, quelle) {
	if (datei.endsWith('.svelte')) return svelteParse(quelle, { modern: true });
	return espreeParse(quelle, { ecmaVersion: 'latest', sourceType: 'module' });
}

/** Ob der Knoten den Namen enthält. @param {any} knoten @param {string} name */
function nennt(knoten, name) {
	let ja = false;
	walk(knoten, {
		enter(/** @type {any} */ n) {
			if (n.type === 'Identifier' && n.name === name) ja = true;
		}
	});
	return ja;
}

/**
 * Jeder eigene Satz in einem catch nach einer Hülle, mit Zeile; dazu die Zahl dieser catch.
 * @param {string} datei @param {string} quelle
 */
export function saetzeNachDerHuelle(datei, quelle) {
	/** @type {string[]} */
	const funde = [];
	let faenger = 0;
	walk(baum(datei, quelle), {
		enter(/** @type {any} */ n) {
			if (n.type !== 'TryStatement' || !n.handler) return;
			let ruft = false;
			walk(n.block, {
				enter(/** @type {any} */ m) {
					if (m.type === 'CallExpression' && HUELLEN.has(m.callee.name)) ruft = true;
				}
			});
			if (!ruft) return;
			faenger++;
			const fehler = n.handler.param?.type === 'Identifier' ? n.handler.param.name : null;
			walk(n.handler.body, {
				enter(/** @type {any} */ k) {
					if (k.type !== 'CallExpression') return;
					const name =
						k.callee.type === 'MemberExpression' ? k.callee.property.name : k.callee.name;
					const satz = k.arguments[0];
					if (!MELDER.has(name) || !satz || (fehler && nennt(satz, fehler))) return;
					const zeile = quelle.slice(0, k.start).split('\n').length;
					funde.push(`${zeile}: ${quelle.slice(satz.start, satz.end)}`);
				}
			});
		}
	});
	return { funde, faenger };
}

/** Eine Form zur Probe, als Rumpf einer Funktion. @param {string} form */
const probe = (form) => saetzeNachDerHuelle('probe.js', `async function f() {\n${form}\n}`);

describe('Meldung der Hülle: kein eigener Satz darüber', () => {
	const dateien = sammleQuelldateien(srcRoot);
	const ergebnis = dateien.map((f) => ({
		datei: relPfad(f),
		...saetzeNachDerHuelle(f, readFileSync(f, 'utf8'))
	}));

	it('findet die catch nach den Hüllen', () => {
		// Fände der Detektor keine mehr, wäre er blind und das Gate grün.
		expect(ergebnis.reduce((summe, e) => summe + e.faenger, 0)).toBeGreaterThan(30);
	});

	it('kein catch nach einer Hülle setzt einen eigenen Satz über ihre Meldung', () => {
		expect(
			ergebnis.flatMap((e) => e.funde.map((f) => `${e.datei}:${f}`)),
			'Eigener Satz über der Meldung der Hülle — weglassen oder den Grund nennen (fehlertext(e))'
		).toEqual([]);
	});

	it('Selbstprobe: der Detektor fasst die Formen', () => {
		for (const form of [
			"try { await apiPost(u, d); } catch { snackbars.addToast('Vermerken nicht möglich.', 'error'); }",
			"try { await apiGet(u); } catch (e) { melde('Laden gescheitert', 'error'); }",
			"try { await apiPut(u, d); } catch (err) { if (!err.handled) snackbars.addToast('Fehler beim Speichern', 'error'); }",
			'try { x = await apiDelete(u); } catch { showSnackbar(`Nicht gelöscht (${u})`); }'
		]) {
			expect(probe(form).funde, form).not.toEqual([]);
		}
		for (const form of [
			"try { await apiPut(u, d); } catch (e) { snackbars.addToast(`Nicht gespeichert: ${fehlertext(e)}`, 'error'); }",
			'try { await apiPost(u, d); } catch { /* apiPost meldet */ }',
			"try { await apiFetch(u); } catch { melde('Netzwerkfehler', 'error'); }",
			"try { await apiPost(u, d); melde('Gespeichert', 'success'); } catch { return; }"
		]) {
			expect(probe(form).funde, form).toEqual([]);
		}
	});
});
