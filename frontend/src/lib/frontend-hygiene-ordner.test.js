import { describe, it, expect } from 'vitest';
import { readdirSync, statSync } from 'node:fs';
import { join } from 'node:path';
import { srcRoot } from './hygiene-quellen.js';

// Jede Komponente und jedes Modul liegt im Ordner seines Themas (docs/STIL.md 13). Oben in
// src/lib steht nur, was die ganze Anwendung trägt, und components/ hat keine eigenen Dateien;
// sonst sammelt sich dort wieder alles, was keinen Ordner bekommt.
//
// Sieht nicht: ob eine Datei im richtigen Themenordner liegt, und Tests oben in src/lib.
const OBEN = [
	'Router.svelte',
	'routenTabelle.js',
	'routenAnwenden.js',
	'menue.js',
	'escapeRegel.js',
	'apiFetch.js',
	'liveEvents.js',
	'serverAntwort.js',
	'hygiene-quellen.js',
	'fehlerausgangScanner.js'
];

const lib = join(srcRoot, 'lib');

/** @param {string} ordner */
const dateienIn = (ordner) =>
	readdirSync(ordner).filter(
		(name) => !name.startsWith('.') && statSync(join(ordner, name)).isFile()
	);

describe('Ordner der Oberfläche', () => {
	it('oben in src/lib steht nur der Rahmen der Anwendung', () => {
		const fremd = dateienIn(lib).filter(
			(name) => !name.endsWith('.test.js') && !OBEN.includes(name)
		);
		expect(
			fremd,
			'Diese Dateien gehören in den Ordner ihres Themas unter components/, utils/ oder stores/ ' +
				'(docs/STIL.md 13)'
		).toEqual([]);
	});

	it('components/ hat nur Ordner', () => {
		expect(
			dateienIn(join(lib, 'components')),
			'Jede Komponente liegt im Ordner ihres Themas'
		).toEqual([]);
	});

	it('misst etwas: jede Datei der Liste gibt es', () => {
		const da = dateienIn(lib);
		expect(OBEN.filter((name) => !da.includes(name))).toEqual([]);
	});
});
