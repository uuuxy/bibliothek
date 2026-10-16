import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import ts from 'typescript';
import { parse } from 'svelte/compiler';
import { srcRoot, relPfad, sammleQuelldateien } from './hygiene-quellen.js';

// Ein Objektliteral mit Gettern trägt /** @satisfies {object} */. In JS gibt TypeScript einem
// Objektliteral ohne Kontexttyp einen offenen Typ: „object literals have an open-ended type (an
// index signature) that allows adding and looking up properties that were not defined
// originally" (TypeScript Handbook, „Type Checking JavaScript Files"). svelte-check meldet dann
// keinen Zugriff auf einen Namen, den der Store nicht hat: Gelesen liefert er undefined,
// geschrieben legt er still ein Feld an. @satisfies gibt dem Literal einen Kontexttyp und schließt
// seinen Typ.
//
// Sieht nicht: Objektliterale ohne Getter und Literale in Tests.

/**
 * Die Objektliterale mit Gettern in einem Stück JS, je mit Zeile und ob ihr Typ geschlossen ist.
 * @param {string} text
 * @param {string} name
 */
function literaleMitGettern(text, name) {
	const quelle = ts.createSourceFile(name, text, ts.ScriptTarget.Latest, true, ts.ScriptKind.JS);
	/** @type {{ zeile: number, geschlossen: boolean }[]} */
	const literale = [];
	/** @param {ts.Node} knoten */
	const besuche = (knoten) => {
		if (
			ts.isObjectLiteralExpression(knoten) &&
			knoten.properties.some(ts.isGetAccessorDeclaration)
		) {
			const huelle = knoten.parent;
			literale.push({
				zeile: quelle.getLineAndCharacterOfPosition(knoten.getStart()).line + 1,
				geschlossen:
					ts.isParenthesizedExpression(huelle) &&
					ts.getJSDocTags(huelle).some(ts.isJSDocSatisfiesTag)
			});
		}
		ts.forEachChild(knoten, besuche);
	};
	besuche(quelle);
	return literale;
}

/**
 * Die Skripte eines Bauteils, je mit der Zahl der Zeilen davor.
 * @param {string} text
 */
function skripte(text) {
	const baum = parse(text, { modern: true });
	/** @type {{ text: string, zeilenDavor: number }[]} */
	const teile = [];
	for (const skript of [baum.module, baum.instance]) {
		if (!skript) continue;
		// acorn setzt start und end, der Typ Program von estree nennt sie nicht.
		const { start, end } = /** @type {any} */ (skript.content);
		teile.push({
			text: text.slice(start, end),
			zeilenDavor: text.slice(0, start).split('\n').length - 1
		});
	}
	return teile;
}

describe('Objektliterale mit Gettern haben einen geschlossenen Typ', () => {
	/** @type {string[]} */
	const offen = [];
	let gelesen = 0;
	for (const datei of sammleQuelldateien(srcRoot)) {
		const text = readFileSync(datei, 'utf8');
		const teile = datei.endsWith('.svelte') ? skripte(text) : [{ text, zeilenDavor: 0 }];
		for (const teil of teile) {
			for (const literal of literaleMitGettern(teil.text, datei)) {
				gelesen++;
				if (!literal.geschlossen)
					offen.push(`${relPfad(datei)}:${teil.zeilenDavor + literal.zeile}`);
			}
		}
	}

	it('liest die Stores der Oberfläche', () => {
		// Untergrenze: Ein Sammler, der ins Leere läuft, bliebe sonst grün.
		expect(gelesen).toBeGreaterThanOrEqual(30);
	});

	it('kennt kein Objektliteral mit Gettern ohne @satisfies', () => {
		expect(
			offen,
			'Objektliteral mit Gettern ohne /** @satisfies {object} */ (…): svelte-check sähe keinen ' +
				'Zugriff auf einen Namen, den es nicht gibt.'
		).toEqual([]);
	});
});
