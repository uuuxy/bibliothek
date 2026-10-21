import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { parse as svelteParse } from 'svelte/compiler';
import { parse as espreeParse } from 'espree';
import { walk } from 'estree-walker';
import { sammleQuelldateien, srcRoot, relPfad } from './hygiene-quellen.js';

// Gate zur Bugklasse „Ältere Antwort überschreibt die jüngere" (docs/sweeps.md): Prüft ein Abruf
// seine Laufnummer (`if (nr !== ladeNr) return`) und wartet danach im selben Block noch einmal,
// etwa auf `res.json()`, kann in diesem Warten ein neuer Lauf beginnen oder die Theke geleert
// werden. Dann muss nach dem letzten `await` noch einmal geprüft werden.
//
// Blindheit: nur Prüfungen der Form `if (a !== b) return` und nur Warten im selben Block, nicht
// in einer aufgerufenen Funktion. Eine Prüfung über eine Funktion (`if (ueberholt()) return`)
// sieht das Gate nicht.

const ESPREE = { ecmaVersion: 2024, sourceType: 'module', loc: true };
const FUNKTIONEN = new Set([
	'FunctionExpression',
	'ArrowFunctionExpression',
	'FunctionDeclaration'
]);

/** @param {string} datei @param {string} quelle */
function programme(datei, quelle) {
	if (datei.endsWith('.svelte')) {
		const baum = svelteParse(quelle, { modern: true });
		return [baum.instance?.content, baum.module?.content].filter(Boolean);
	}
	return [espreeParse(quelle, ESPREE)];
}

/** Der Schlüssel einer Laufnummer-Prüfung (`if (a !== b) return`), sonst null.
 * @param {any} anw @param {string} quelle */
function waechter(anw, quelle) {
	if (anw?.type !== 'IfStatement' || anw.alternate) return null;
	const { test, consequent } = anw;
	if (test.type !== 'BinaryExpression' || (test.operator !== '!==' && test.operator !== '!='))
		return null;
	const einfach = (/** @type {any} */ n) =>
		n.type === 'Identifier' || n.type === 'MemberExpression';
	if (!einfach(test.left) || !einfach(test.right)) return null;
	const gibtZurueck =
		consequent.type === 'ReturnStatement' ||
		(consequent.type === 'BlockStatement' &&
			consequent.body.length === 1 &&
			consequent.body[0].type === 'ReturnStatement');
	if (!gibtZurueck) return null;
	const text = (/** @type {any} */ n) => quelle.slice(n.start, n.end);
	return `${text(test.left)} !== ${text(test.right)}`;
}

/** Ob eine Anweisung wartet, ohne in eine innere Funktion zu schauen. @param {any} anw */
function wartet(anw) {
	let gefunden = false;
	walk(anw, {
		enter(/** @type {any} */ n) {
			if (FUNKTIONEN.has(n.type)) this.skip();
			else if (n.type === 'AwaitExpression') gefunden = true;
		}
	});
	return gefunden;
}

/**
 * Stellen, an denen nach einer Laufnummer-Prüfung noch gewartet und danach nicht neu geprüft wird.
 * @param {string} datei @param {string} [quelle]
 * @returns {{ zeilen: number[], waechter: number }}
 */
export function findeSpaetePruefungen(datei, quelle) {
	const inhalt = quelle ?? readFileSync(datei, 'utf8');
	const zeilen = new Set();
	let anzahlWaechter = 0;
	for (const prog of programme(datei, inhalt)) {
		walk(prog, {
			enter(/** @type {any} */ block) {
				if (block.type !== 'BlockStatement') return;
				const anw = block.body;
				const gesehen = new Set();
				anw.forEach((/** @type {any} */ s, /** @type {number} */ i) => {
					const schluessel = waechter(s, inhalt);
					if (!schluessel || gesehen.has(schluessel)) return;
					gesehen.add(schluessel);
					anzahlWaechter++;
					let letztesWarten = -1;
					for (let j = i + 1; j < anw.length; j++) if (wartet(anw[j])) letztesWarten = j;
					if (letztesWarten < 0) return;
					const neuGeprueft = anw
						.slice(letztesWarten + 1)
						.some((/** @type {any} */ t) => waechter(t, inhalt) === schluessel);
					if (!neuGeprueft) zeilen.add(anw[letztesWarten].loc.start.line);
				});
			}
		});
	}
	return { zeilen: [...zeilen], waechter: anzahlWaechter };
}

describe('Gate: Laufnummer nach dem letzten Warten prüfen', () => {
	it('meldet ein Warten nach der Prüfung und lässt das Prüfen danach gelten (Selbstprobe)', () => {
		const spaet = `async function lade() {
			const nr = ++ladeNr;
			const res = await apiFetch('/x');
			if (nr !== ladeNr) return;
			liste = await res.json();
		}`;
		const richtig = `async function lade() {
			const nr = ++ladeNr;
			const res = await apiFetch('/x');
			const daten = await res.json();
			if (nr !== ladeNr) return;
			liste = daten;
		}`;
		expect(findeSpaetePruefungen('probe.js', spaet).zeilen).toEqual([5]);
		expect(findeSpaetePruefungen('probe.js', richtig).zeilen).toEqual([]);
		const inSvelte = `<script>\n\tlet liste = [];\n\tlet ladeNr = 0;\n${spaet}\n</script>`;
		expect(findeSpaetePruefungen('probe.svelte', inSvelte).zeilen).toHaveLength(1);
	});

	it('findet keine Stelle, an der nach der Prüfung noch gewartet wird', () => {
		/** @type {string[]} */
		const funde = [];
		let dateien = 0;
		let waechterGesamt = 0;
		for (const datei of sammleQuelldateien(srcRoot)) {
			dateien++;
			const { zeilen, waechter: anzahl } = findeSpaetePruefungen(datei);
			waechterGesamt += anzahl;
			for (const zeile of zeilen) funde.push(`${relPfad(datei)}:${zeile}`);
		}
		expect(dateien, 'der Sammler greift ins Leere — dieses Gate wäre still grün').toBeGreaterThan(
			200
		);
		expect(
			waechterGesamt,
			'der Detektor findet kaum Prüfungen — er greift ins Leere'
		).toBeGreaterThan(8);
		expect(
			funde,
			'Nach der Prüfung der Laufnummer wird noch gewartet (etwa auf res.json()). Erst lesen, ' +
				'dann prüfen, dann zuweisen: In diesem Warten kann ein neuer Lauf beginnen.'
		).toEqual([]);
	});
});
