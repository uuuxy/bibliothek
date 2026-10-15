import { describe, it, expect } from 'vitest';
import { srcRoot, relPfad, sammleQuelldateien, sammleTestdateien } from './hygiene-quellen.js';

// Ratsche: Datei- und Ordnernamen der Oberfläche sind deutsch (docs/STIL.md, Abschnitt 13).
//
// Dieselbe Sache heißt im Go-Code und in der Oberfläche gleich: Wer in der Leserakte einen
// Fehler sucht, findet die Tür in api/leser und soll die Komponente nicht unter Student suchen.
// Gezählt werden die Wörter eines Pfads unter src/, zerlegt an Großbuchstaben, Unterstrichen und
// Bindestrichen, gegen eine feste Liste englischer Wörter. Lehnwörter (Barcode, Cover, Scan,
// Import, Upload), die Bauteile von M3 ohne deutschen Namen (Button, Chip, Dialog, Snackbar,
// Switch, Select, Radio, Badge, Tooltip) und Wörter von Svelte und dem Web (stores, actions,
// route, fetch) stehen nicht darin.
//
// BESTAND nennt je Pfad, was noch umzubenennen ist. Ein neuer Pfad mit einem englischen Wort ist
// rot, ein Pfad aus dem Bestand mit einem Wort mehr ebenso; wer einen Pfad umbenennt, trägt ihn
// aus.
//
// Sieht nicht: Namen innerhalb der Dateien, Wörter außerhalb der Liste und die Namen unter e2e/.
const ENGLISCH = new Set([
	'active',
	'alert',
	'area',
	'assign',
	'assignment',
	'audit',
	'book',
	'books',
	'borrowed',
	'borrowers',
	'camera',
	'capture',
	'card',
	'cart',
	'catalog',
	'chart',
	'checklist',
	'class',
	'config',
	'consistency',
	'create',
	'creation',
	'damage',
	'danger',
	'dashboard',
	'data',
	'dates',
	'delete',
	'deleted',
	'detector',
	'directory',
	'edit',
	'editor',
	'extend',
	'face',
	'flash',
	'form',
	'graduates',
	'grid',
	'history',
	'idle',
	'indicator',
	'inventory',
	'item',
	'keyboard',
	'label',
	'labels',
	'list',
	'lock',
	'login',
	'management',
	'manager',
	'media',
	'menu',
	'metadata',
	'modal',
	'order',
	'overdue',
	'page',
	'panel',
	'peek',
	'permission',
	'permissions',
	'picker',
	'preview',
	'print',
	'printer',
	'profile',
	'promote',
	'properties',
	'queue',
	'receipt',
	'recommendations',
	'report',
	'result',
	'results',
	'screen',
	'search',
	'selector',
	'session',
	'settings',
	'sheet',
	'sidebar',
	'sort',
	'sorting',
	'staging',
	'stats',
	'student',
	'students',
	'summary',
	'supplier',
	'tab',
	'tabs',
	'table',
	'templates',
	'toast',
	'toolbar',
	'unified',
	'user',
	'view',
	'widget',
	'workspace'
]);

const BESTAND = {
	'src/lib/TabelleSortKopf.test.js': ['sort'],
	'src/lib/components/ui/CoverPeek.svelte': ['peek'],
	'src/lib/components/ui/TabelleSortKopf.svelte': ['sort']
};

/** @param {string} name */
const stamm = (name) => name.replace(/\.(test|spec)\.js$|\.svelte\.js$|\.svelte$|\.js$/, '');

/** Die englischen Wörter eines Pfads unter src/, jedes einmal.
 * @param {string} pfad */
function englischeWoerter(pfad) {
	const teile = pfad.split('/').slice(1);
	/** @type {string[]} */
	const gefunden = [];
	teile.forEach((teil, i) => {
		const name = i === teile.length - 1 ? stamm(teil) : teil;
		for (const wort of name.match(/[A-Z]?[a-z]+|[A-Z]+(?![a-z])/g) ?? []) {
			const klein = wort.toLowerCase();
			if (ENGLISCH.has(klein) && !gefunden.includes(klein)) gefunden.push(klein);
		}
	});
	return gefunden;
}

describe('Datei- und Ordnernamen der Oberfläche sind deutsch', () => {
	const pfade = [...sammleQuelldateien(srcRoot), ...sammleTestdateien(srcRoot)].map(relPfad);
	/** @type {Map<string, string[]>} */
	const aktuell = new Map();
	for (const pfad of pfade) {
		const woerter = englischeWoerter(pfad);
		if (woerter.length > 0) aktuell.set(pfad, woerter);
	}

	it('liest überhaupt Dateien und erkennt ein englisches Wort', () => {
		expect(pfade.length).toBeGreaterThan(600);
		expect(englischeWoerter('src/lib/components/students/StudentLockModal')).toEqual([
			'students',
			'student',
			'lock',
			'modal'
		]);
		expect(englischeWoerter('src/lib/components/ui/BestaetigungsDialog')).toEqual([]);
	});

	it('kennt keinen neuen Pfad mit einem englischen Wort', () => {
		/** @type {string[]} */
		const neu = [];
		for (const [pfad, woerter] of aktuell) {
			const erlaubt = BESTAND[/** @type {keyof typeof BESTAND} */ (pfad)] ?? [];
			const zuviel = woerter.filter((w) => !erlaubt.includes(w));
			if (zuviel.length > 0) neu.push(`${pfad}: ${zuviel.join(', ')}`);
		}
		expect(
			neu,
			'Englisches Wort im Namen. Deutsch benennen nach docs/STIL.md, Abschnitt 13 (Wörterbuch in ' +
				'Abschnitt 2), oder ein Lehnwort aus der Liste nehmen.'
		).toEqual([]);
	});

	it('führt nichts, was es nicht mehr gibt oder inzwischen deutsch heißt', () => {
		const veraltet = Object.entries(BESTAND)
			.filter(([pfad, woerter]) => {
				const jetzt = aktuell.get(pfad) ?? [];
				return woerter.some((w) => !jetzt.includes(w));
			})
			.map(([pfad]) => pfad);
		expect(veraltet, 'Diese Einträge im Bestand austragen oder kürzen.').toEqual([]);
	});
});
