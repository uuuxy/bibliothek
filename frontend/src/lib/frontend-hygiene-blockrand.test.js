import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { parse } from 'svelte/compiler';
import { srcRoot, sammleQuelldateien, relPfad } from './hygiene-quellen.js';

// Ratsche: Svelte nimmt den Leerraum am Rand eines Blocks ({#if}, {#each}) weg. Steht davor Text
// oder ein Ausdruck und beginnt der Block mit Leerraum und einem Wort, klebt das Wort am Nachbarn:
// „2 Exemplare· 0,00 €", „Titel oben suchenoder …". Ein geschütztes Leerzeichen (&nbsp;) bleibt stehen;
// {' '} verbietet ESLint (svelte/no-useless-mustaches).
//
// Sieht nicht: {#await}, {#key} und Snippets; Leerraum aus einem Ausdruck.

const RAND = /[ \t\r\n]/;

/** @param {any} n */
const klebtAmEnde = (n) =>
	n && ((n.type === 'Text' && /[^ \t\r\n]$/.test(n.data)) || n.type === 'ExpressionTag');
/** @param {any} n */
const klebtAmAnfang = (n) =>
	n && ((n.type === 'Text' && /^[^ \t\r\n.,;:!?)]/.test(n.data)) || n.type === 'ExpressionTag');

/**
 * Die Stellen einer Datei, an denen ein Wort am Rand eines Blocks seinen Abstand verliert.
 * @param {string} quelle
 * @returns {string[]} Zeile und Wort
 */
export function funde(quelle) {
	const ast = parse(quelle, { modern: true });
	/** @type {string[]} */
	const out = [];
	/** @param {any} n */
	const zeile = (n) => quelle.slice(0, n.start).split('\n').length;
	/** @param {any[]} knoten */
	const folge = (knoten) => {
		knoten.forEach((n, i) => {
			if (n.type !== 'IfBlock' && n.type !== 'EachBlock') return;
			const zweige = [n.consequent ?? n.body, n.alternate].filter(Boolean);
			for (const z of zweige) {
				const inhalt = z.nodes.filter((/** @type {any} */ x) => x.type !== 'Comment');
				const erster = inhalt[0];
				const letzter = inhalt.at(-1);
				if (
					klebtAmEnde(knoten[i - 1]) &&
					erster?.type === 'Text' &&
					RAND.test(erster.data[0] ?? '') &&
					/^[ \t\r\n]+[^ \t\r\n.,;:!?)]/.test(erster.data)
				)
					out.push(`${zeile(erster)}: „${erster.data.trim().slice(0, 30)}"`);
				if (
					klebtAmAnfang(knoten[i + 1]) &&
					letzter?.type === 'Text' &&
					/[^ \t\r\n][ \t\r\n]+$/.test(letzter.data)
				)
					out.push(`${zeile(letzter)}: „…${letzter.data.trim().slice(-30)}"`);
			}
		});
	};
	/** @param {any} n */
	const gehe = (n) => {
		for (const f of [n.fragment, n.consequent, n.alternate, n.body]) {
			if (!f?.nodes) continue;
			folge(f.nodes);
			f.nodes.forEach(gehe);
		}
	};
	gehe(ast);
	return out;
}

describe('Wörter am Rand eines Blocks behalten ihren Abstand', () => {
	it('Selbstprobe: erkennt die geklebten Formen und lässt die richtigen gelten', () => {
		expect(funde('{n} Exemplare{#if p}\n\t· {summe}{/if}')).toHaveLength(1);
		expect(funde('Titel suchen{#if b}\n\toder hinzufügen{/if}')).toHaveLength(1);
		expect(funde('{#if a}Ende \n{/if}Wort')).toHaveLength(1);
		expect(funde('{id}{#if s}&nbsp;· SN {s}{/if}')).toEqual([]);
		expect(funde('<p>{#if a}\n\tText\n{/if}</p>')).toEqual([]);
		expect(funde('Satz{#if a}\n\t, weiter{/if}')).toEqual([]);
	});

	it('keine Datei verliert ein Leerzeichen am Rand eines Blocks', () => {
		const gefunden = sammleQuelldateien(srcRoot)
			.filter((d) => d.endsWith('.svelte'))
			.flatMap((d) => funde(readFileSync(d, 'utf8')).map((f) => `${relPfad(d)}:${f}`));
		expect(
			gefunden,
			'Wort klebt am Nachbarn: Svelte nimmt Leerraum am Rand eines Blocks weg, &nbsp; setzen'
		).toEqual([]);
	});
});
