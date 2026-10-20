import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { parse } from 'svelte/compiler';
import {
	srcRoot,
	relPfad,
	sammleQuelldateien,
	sammleTestdateien,
	ohneKommentare
} from './hygiene-quellen.js';

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
// Der zweite Teil unten prüft die Namen, die eine Datei erklärt, mit derselben Liste.
//
// Sieht nicht: Wörter außerhalb der Liste und die Namen unter e2e/.
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

/** @type {Record<string, string[]>} */
const BESTAND = {};

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
			const erlaubt = BESTAND[pfad] ?? [];
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

// Ratsche: Namen innerhalb der Dateien sind deutsch (docs/STIL.md, Abschnitt 13), Stufe 2 des
// Umbaus (OFFEN 5.66). Gezählt werden die Namen, die eine Quelldatei erklärt (let, const, var,
// function, class), ohne Kommentare, zerlegt an Großbuchstaben, gegen dieselbe Liste wie die
// Pfade und dazu Wörter, die in Namen häufig sind und in Pfaden nicht vorkamen. Wörter von Svelte
// und dem Web (value, event, submit, open) stehen nicht darin, ebenso wenig die Bauteile von M3.
//
// NAMEN_BESTAND nennt je Datei, wie viele Erklärungen noch ein englisches Wort tragen; die Zahl
// kann nur sinken.
//
// Sieht nicht: Props, Schlüssel von Objekten, Parameter, Methoden der Stores und Namen in Tests.
const ENGLISCH_IN_NAMEN = new Set([
	...ENGLISCH,
	'handle',
	'error',
	'errors',
	'selected',
	'loading',
	'loaded',
	'load',
	'save',
	'saving',
	'deleting',
	'remove',
	'add',
	'query',
	'editing',
	'update',
	'users'
]);

const ERKLAERUNG = /\b(?:let|const|var|function|class)\s+([A-Za-z_$][\w$]*)/g;

/** Trägt der Name ein englisches Wort? @param {string} name */
function englischImNamen(name) {
	return (name.match(/[A-Z]?[a-z]+|[A-Z]+(?![a-z])/g) ?? []).some((w) =>
		ENGLISCH_IN_NAMEN.has(w.toLowerCase())
	);
}

/** @type {Record<string, number>} */
const NAMEN_BESTAND = {
	// apiDelete heißt wie die HTTP-Methode, wie apiGet, apiPost und apiPut daneben.
	'src/lib/apiFetch.js': 1
};

describe('Namen innerhalb der Dateien der Oberfläche sind deutsch', () => {
	/** @type {Map<string, number>} */
	const zaehlung = new Map();
	// Gelesene Erklärungen insgesamt: Die Selbstprobe hängt so nicht am Bestand, der sinken soll.
	let erklaerungen = 0;
	for (const datei of sammleQuelldateien(srcRoot)) {
		const text = ohneKommentare(readFileSync(datei, 'utf8'));
		let n = 0;
		for (const m of text.matchAll(ERKLAERUNG)) {
			erklaerungen++;
			if (englischImNamen(m[1])) n++;
		}
		if (n > 0) zaehlung.set(relPfad(datei), n);
	}

	it('liest überhaupt Dateien und erkennt einen englischen Namen', () => {
		expect(erklaerungen).toBeGreaterThan(3000);
		expect(englischImNamen('handleDeleteClick')).toBe(true);
		expect(englischImNamen('selectedTitel')).toBe(true);
		expect(englischImNamen('ladeLeser')).toBe(false);
		expect(englischImNamen('eventQuelle')).toBe(false);
	});

	it('kennt keine Datei mit mehr englischen Namen als im Bestand', () => {
		const neu = [...zaehlung]
			.filter(([pfad, n]) => n > (NAMEN_BESTAND[pfad] ?? 0))
			.map(([pfad, n]) => `${pfad}: ${n} (Bestand ${NAMEN_BESTAND[pfad] ?? 0})`);
		expect(
			neu,
			'Englisches Wort in einem erklärten Namen. Deutsch benennen nach docs/STIL.md, Abschnitte 2 ' +
				'und 13.'
		).toEqual([]);
	});

	it('führt keinen Bestand, der inzwischen kleiner ist', () => {
		const veraltet = Object.entries(NAMEN_BESTAND)
			.filter(([pfad, n]) => (zaehlung.get(pfad) ?? 0) < n)
			.map(([pfad, n]) => `${pfad}: ${n} → ${zaehlung.get(pfad) ?? 0}`);
		expect(veraltet, 'Diese Einträge im Bestand senken oder austragen.').toEqual([]);
	});
});

// Ratsche: Props sind deutsch (docs/STIL.md, Abschnitt 13), Stufe 3 des Umbaus (OFFEN 5.66).
// Gezählt werden die Schlüssel in `let { … } = $props()` jeder Komponente, mit derselben Liste wie
// die Namen. Englisch bleiben nach Abschnitt 13 die Ereignisse (on…), `class` und `children`, die
// Svelte und das Web vorgeben.
//
// PROPS_BESTAND nennt je Komponente, wie viele Props noch ein englisches Wort tragen; die Zahl
// kann nur sinken. Wer eine Prop umbenennt, zieht jeden Rufer mit, auch Tests.
/** @type {Record<string, number>} */
const PROPS_BESTAND = {};

/** Die Props einer Komponente, wie sie `let { … } = $props()` nennt. @param {string} text */
function propsAus(text) {
	const baum = parse(text, { modern: true });
	/** @type {string[]} */
	const namen = [];
	for (const s of /** @type {any[]} */ (baum.instance?.content.body ?? [])) {
		for (const d of s.declarations ?? []) {
			if (d.init?.type !== 'CallExpression' || d.init.callee?.name !== '$props') continue;
			if (d.id?.type !== 'ObjectPattern') continue;
			for (const p of d.id.properties) if (p.key?.name) namen.push(p.key.name);
		}
	}
	return namen;
}

/** Trägt die Prop ein englisches Wort, das nicht Svelte oder das Web vorgibt? @param {string} name */
const englischeProp = (name) =>
	!/^on[A-Z]/.test(name) && name !== 'class' && name !== 'children' && englischImNamen(name);

describe('Props der Bauteile sind deutsch', () => {
	/** @type {Map<string, number>} */
	const zaehlung = new Map();
	let gelesen = 0;
	for (const datei of sammleQuelldateien(srcRoot)) {
		if (!datei.endsWith('.svelte')) continue;
		const props = propsAus(readFileSync(datei, 'utf8'));
		gelesen += props.length;
		const n = props.filter(englischeProp).length;
		if (n > 0) zaehlung.set(relPfad(datei), n);
	}

	it('liest überhaupt Props und erkennt eine englische', () => {
		expect(gelesen).toBeGreaterThan(800);
		expect(propsAus('<script>let { book, onclick, class: k } = $props();</script>')).toEqual([
			'book',
			'onclick',
			'class'
		]);
		expect(['book', 'onclick', 'class', 'children', 'titel'].filter(englischeProp)).toEqual([
			'book'
		]);
	});

	it('kennt kein Bauteil mit mehr englischen Props als im Bestand', () => {
		const neu = [...zaehlung]
			.filter(([pfad, n]) => n > (PROPS_BESTAND[pfad] ?? 0))
			.map(([pfad, n]) => `${pfad}: ${n} (Bestand ${PROPS_BESTAND[pfad] ?? 0})`);
		expect(neu, 'Englisches Wort in einer Prop. Deutsch benennen nach docs/STIL.md 13.').toEqual(
			[]
		);
	});

	it('führt keinen Bestand, der inzwischen kleiner ist', () => {
		const veraltet = Object.entries(PROPS_BESTAND)
			.filter(([pfad, n]) => (zaehlung.get(pfad) ?? 0) < n)
			.map(([pfad, n]) => `${pfad}: ${n} → ${zaehlung.get(pfad) ?? 0}`);
		expect(veraltet, 'Diese Einträge im Bestand senken oder austragen.').toEqual([]);
	});
});
