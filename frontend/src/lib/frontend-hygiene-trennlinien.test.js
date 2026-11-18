import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { srcRoot, sammleQuelldateien, relPfad, ohneKommentare } from './hygiene-quellen.js';

// Ratsche: Trennlinien nach Material 3 (Divider, Specs): Farbe „Outline variant", eine Linie von
// 1 dp. Eine Trennlinie mit eigener Deckkraft (divide-outline-variant/40) oder in einer anderen
// Rolle steht neben den übrigen blasser oder dunkler da; eine dickere wiegt mehr als eine Linie.
//
// Sieht nicht: Linien aus Klassen in Ausdrücken, aus <style>-Blöcken und CSS-Dateien, Linien als
// Fläche (h-px mit bg-…) und Rahmen ganzer Bauteile (Karte, Textfeld): Deren Farbe halten
// frontend-hygiene-karten und die Gates der Textfelder.

const DIVIDE_FARBE = /(?<![\w-])divide-(?!x\b|y\b|x-|y-)([a-z][\w-]*)/g;
const BLASS = /(?<![\w-])(?:border|divide)-outline-variant\/\d+/g;
const DICK = /(?<![\w-])divide-[xy]-(?:[2-9]|\[)/g;

describe('Trennlinien nach M3', () => {
	const dateien = sammleQuelldateien(srcRoot).filter((f) => f.endsWith('.svelte'));

	it('zieht jede Trennlinie in outline-variant, voll und 1 px stark', () => {
		/** @type {string[]} */
		const funde = [];
		for (const f of dateien) {
			const quelle = ohneKommentare(readFileSync(f, 'utf8'));
			for (const [klasse, farbe] of quelle.matchAll(DIVIDE_FARBE))
				if (farbe !== 'outline-variant') funde.push(`${relPfad(f)}: ${klasse}`);
			for (const [klasse] of quelle.matchAll(BLASS)) funde.push(`${relPfad(f)}: ${klasse}`);
			for (const [klasse] of quelle.matchAll(DICK)) funde.push(`${relPfad(f)}: ${klasse}`);
		}
		expect(funde, 'Trennlinie abseits von M3 (Divider: Outline variant, 1 dp)').toEqual([]);
	});

	it('findet Trennlinien', () => {
		const anzahl = dateien.filter((f) =>
			/divide-outline-variant/.test(readFileSync(f, 'utf8'))
		).length;
		expect(anzahl).toBeGreaterThan(10);
	});
});
