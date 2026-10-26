import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { parse as svelteParse } from 'svelte/compiler';
import { walk } from 'estree-walker';
import { sammleQuelldateien, srcRoot, relPfad } from './hygiene-quellen.js';

// Gate zur Bugklasse „Abschnitt leer statt gescheitert" (docs/sweeps.md, ARCHITEKTUR.md 5.3):
// Eine Seite, deren Liste nicht geladen wurde, zeigt keinen Leerzustand. Er hieße „nichts da",
// beim Bestellbedarf „Bestände ausreichend", im Etiketten-Nachdruck „Für alle Exemplare wurden
// Etiketten gedruckt", während nur der Abruf scheiterte. Jeder Zweig des Markups, der nach einer
// leeren Liste fragt, steht deshalb hinter einer Prüfung des Ladefehlers: in einem Zweig davor
// derselben if-Kette, in seiner eigenen Bedingung oder in einem umgebenden if. Als Leerzustand
// zählen auch die Zweige hinter einer Frage nach der vollen Liste (`{#if x.length > 0}…{:else}`)
// und der `{:else}`-Zweig eines `{#each}`; eine Variable aus `$derived(…length === 0…)` fragt
// wie ihr Ausdruck.
//
// Sieht nicht: einen Ladefehler unter einem Namen ohne fehler/error/nichtGeladen/fehlend, einen
// Leerzustand ohne Frage nach `.length` (`{#if !liste}`, eine Zahl statt der Liste), eine Frage
// nach der vollen Liste hinter `||` oder `!`, eine abgeleitete Variable aus `$derived.by`, aus
// einem Store oder über zwei Schritte, und ob die geprüfte Variable wirklich vom Laden gesetzt
// wird.

const FEHLERNAME = /fehler|error|nichtgeladen|fehlend/i;

/**
 * Ausnahmen mit Grund: Die Liste kommt nicht aus einem Abruf dieses Bauteils, oder das Bauteil
 * steht nur, wenn der Abruf gelang. Das ist ein Bestand bewusster Ausnahmen, KEINE Erlaubnis.
 * @type {Record<string, [number, string]>}
 */
const AUSNAHMEN = {
	'src/lib/components/buchakte/AusleiherListe.svelte': [1, 'Buchakte nennt, was nicht geladen ist'],
	'src/lib/components/buchakte/BuchakteAusleiher.svelte': [
		1,
		'Buchakte nennt, was nicht geladen ist'
	],
	'src/lib/components/buchakte/BuchakteExemplare.svelte': [
		1,
		'Buchakte nennt, was nicht geladen ist'
	],
	'src/lib/components/buchakte/BuchakteHistorie.svelte': [
		1,
		'Buchakte nennt, was nicht geladen ist'
	],
	'src/lib/components/buchakte/BuchakteVormerkungen.svelte': [
		1,
		'die Vormerkungen: Buchakte nennt, was nicht geladen ist'
	],
	'src/lib/components/leserakte/LeserAusleihenKarte.svelte': [
		1,
		'Leserakte nennt, was nicht geladen ist (ListenNichtGeladen)'
	],
	'src/lib/components/leserakte/LeserVormerkungenKarte.svelte': [
		1,
		'steht nur mit Vormerkungen; Leserakte nennt, was nicht geladen ist'
	],
	'src/inventur/lib/components/admin/Bestandstabelle.svelte': [
		1,
		'steht nur, wenn die Titelliste geladen ist (TitelVerwaltung)'
	],
	'src/lib/components/bestellungen/BestellDetailExemplare.svelte': [
		1,
		'steht nur, wenn die Bestellung geladen ist (BestellDetail)'
	],
	'src/lib/components/bestellungen/BestellDetailPositionen.svelte': [
		1,
		'steht nur, wenn die Bestellung geladen ist (BestellDetail)'
	],
	'src/lib/components/lmfplan/LmfPlanReihenfolge.svelte': [
		1,
		'steht nur, wenn der Plan geladen ist (LmfPlan)'
	],
	'src/inventur/lib/components/KlassenSuchfeld.svelte': [
		1,
		'Treffer des Suchtextes; den Ladefehler zeigt KlassenUebersicht'
	],
	'src/lib/components/inventur/FehlbestandBericht.svelte': [
		1,
		'steht nur mit Einträgen; leer wird er im Browser'
	],
	'src/inventur/lib/components/admin/KlassensatzZusammenfassung.svelte': [
		1,
		'die Auswahl im Browser, nichts geladen'
	],
	'src/lib/components/bestellungen/Warenkorb.svelte': [
		1,
		'der Warenkorb im Browser, nichts geladen'
	],
	'src/lib/components/etiketten/EtikettVorschau.svelte': [
		1,
		'die gewählten Etiketten im Browser; den Ladefehler zeigt EtikettBarcodeSchritt'
	],
	'src/lib/components/einstellungen/MailVorlagenPlatzhalter.svelte': [
		1,
		'feste Angaben (mailVorlagenInfo.js), nichts geladen'
	],
	'src/lib/components/leserakte/AusleihQuittung.svelte': [
		1,
		'die Ausleihen kommen mit der Akte; ohne Akte steht die Quittung nicht (Leserakte)'
	],
	'src/lib/components/leserakte/LeserLoeschenDialog.svelte': [
		1,
		'die Löschfrage, kein Leerzustand; die Ausleihen kommen mit der Akte, und der Server lehnt das Löschen mit offenen Ausleihen ab'
	],
	'src/lib/monitor/FolieBeliebt.svelte': [
		1,
		'steht nur mit geladenem Stand; ein Fehlschlag behält den alten (monitorTakt)'
	],
	'src/lib/monitor/FolieNeuEingetroffen.svelte': [
		1,
		'steht nur mit geladenem Stand; ein Fehlschlag behält den alten (monitorTakt)'
	],
	'src/lib/components/ui/SelectListe.svelte': [
		1,
		'allgemeines Bauteil; den Ladefehler zeigt der Aufrufer'
	],
	'src/lib/components/ui/KlassenVersandDialog.svelte': [
		1,
		'öffnet nur mit geladenen Klassen: „Alle anmahnen" erst mit Kindern (MahnwesenAktionen), der Versand der Abgänger ohne Liste gesperrt'
	]
};

/** Ist der Ausdruck `<x>.length`? @param {any} x */
const laenge = (x) =>
	x?.type === 'MemberExpression' && !x.computed && x.property?.name === 'length';

/** @typedef {{ leer: Set<string>, voll: Set<string> }} Namen abgeleitete Variablen, die fragen */
/** @type {Namen} */
const KEINE_NAMEN = { leer: new Set(), voll: new Set() };

/** Ist k ein Name für sich, nicht das Feld hinter einem Punkt? @param {any} k @param {any} eltern */
const name = (k, eltern) =>
	k.type === 'Identifier' &&
	!(eltern?.type === 'MemberExpression' && eltern.property === k && !eltern.computed);

/** Fragt der Ausdruck nach einer leeren Liste? @param {any} test @param {Namen} [namen] */
function fragtLeer(test, namen = KEINE_NAMEN) {
	let ja = false;
	walk(test, {
		enter(/** @type {any} */ k, /** @type {any} */ eltern) {
			const verneint = eltern?.type === 'UnaryExpression' && eltern.operator === '!';
			if (name(k, eltern) && namen.leer.has(k.name) && !verneint) ja = true;
			if (name(k, eltern) && namen.voll.has(k.name) && verneint) ja = true;
			if (k.type === 'UnaryExpression' && k.operator === '!' && laenge(k.argument)) ja = true;
			if (k.type !== 'BinaryExpression') return;
			const gleich = k.operator === '===' || k.operator === '==';
			if (gleich && laenge(k.left) && k.right.value === 0) ja = true;
			if (gleich && laenge(k.right) && k.left.value === 0) ja = true;
			if (k.operator === '<' && laenge(k.left) && k.right.value === 1) ja = true;
		}
	});
	return ja;
}

/**
 * Fragt der Ausdruck nach einer vollen Liste, für sich oder als Glied eines `&&`? Die Zweige
 * dahinter sind dann ihr Leerzustand.
 * @param {any} test
 * @param {Namen} [namen]
 * @returns {boolean}
 */
function fragtVoll(test, namen = KEINE_NAMEN) {
	if (test.type === 'LogicalExpression' && test.operator === '&&')
		return fragtVoll(test.left, namen) || fragtVoll(test.right, namen);
	if (test.type === 'Identifier' && namen.voll.has(test.name)) return true;
	const verneint = test.type === 'UnaryExpression' && test.operator === '!';
	if (verneint && test.argument.type === 'Identifier' && namen.leer.has(test.argument.name))
		return true;
	if (laenge(test)) return true;
	if (test.type !== 'BinaryExpression') return false;
	const { operator: op, left: l, right: r } = test;
	if (laenge(l) && op === '>' && r.value === 0) return true;
	if (laenge(l) && op === '>=' && r.value === 1) return true;
	if (laenge(l) && (op === '!==' || op === '!=') && r.value === 0) return true;
	return laenge(r) && op === '<' && l.value === 0;
}

/** Nennt der Ausdruck einen Ladefehler? @param {any} test */
function fragtFehler(test) {
	let ja = false;
	walk(test, {
		enter(/** @type {any} */ k) {
			if (k.type === 'Identifier' && FEHLERNAME.test(k.name)) ja = true;
			if (k.type === 'MemberExpression' && !k.computed && FEHLERNAME.test(k.property?.name ?? ''))
				ja = true;
		}
	});
	return ja;
}

/**
 * Beginnt der Zweig mit einer Frage nach dem Ladefehler? Dann steht sein Leerzustand dahinter
 * (`{:else}{#if ladefehler}…{:else}<p>Keine</p>{/if}`).
 * @param {any} zweig
 */
function beginntMitFehlerpruefung(zweig) {
	const erster = zweig?.nodes?.find(
		(/** @type {any} */ n) => n.type !== 'Comment' && !(n.type === 'Text' && !n.data.trim())
	);
	return erster?.type === 'IfBlock' && fragtFehler(erster.test);
}

/**
 * Die Variablen aus `$derived(…)`, deren Ausdruck nach einer leeren oder vollen Liste fragt.
 * @param {any} ast
 * @returns {Namen}
 */
function abgeleiteteNamen(ast) {
	/** @type {Namen} */
	const namen = { leer: new Set(), voll: new Set() };
	if (!ast.instance) return namen;
	walk(ast.instance.content, {
		enter(/** @type {any} */ k) {
			if (k.type !== 'VariableDeclarator' || k.id.type !== 'Identifier') return;
			const aufruf = k.init?.type === 'CallExpression' ? k.init : null;
			if (aufruf?.callee.type !== 'Identifier' || aufruf.callee.name !== '$derived') return;
			const ausdruck = aufruf.arguments[0];
			if (!ausdruck) return;
			// Eine Zahl (`$derived(x.length)`) fragt nichts; erst ein Vergleich macht die Frage.
			if (fragtLeer(ausdruck)) namen.leer.add(k.id.name);
			else if (fragtVoll(ausdruck) && !laenge(ausdruck)) namen.voll.add(k.id.name);
		}
	});
	return namen;
}

/**
 * Jeder Leerzustand ohne Prüfung des Ladefehlers davor, mit Zeile; dazu die Zahl der
 * Leerzustände.
 * @param {string} quelle Svelte-Quelltext
 */
export function leerOhneFehlerpruefung(quelle) {
	/** @type {string[]} */
	const funde = [];
	let leerzustaende = 0;
	/** @type {any[]} die Bedingungen der if-Blöcke um den Lauf herum */
	const umgebend = [];
	const zeile = (/** @type {number} */ stelle) => quelle.slice(0, stelle).split('\n').length;
	const ast = svelteParse(quelle, { modern: true });
	const namen = abgeleiteteNamen(ast);
	walk(/** @type {any} */ (ast.fragment), {
		enter(/** @type {any} */ k) {
			if (k.type === 'EachBlock' && k.fallback) {
				leerzustaende++;
				const stelle = k.fallback.nodes[0]?.start ?? k.start;
				const geprueft = umgebend.some(fragtFehler) || beginntMitFehlerpruefung(k.fallback);
				if (!geprueft) funde.push(`${zeile(stelle)}: {#each …}{:else}`);
			}
			if (k.type !== 'IfBlock') return;
			if (!k.elseif) {
				// Eine Kette von vorn: if, else if, else if … Hinter der Frage nach der vollen Liste
				// ist jeder weitere Zweig ihr Leerzustand.
				let geprueft = umgebend.some(fragtFehler);
				let nachVoll = false;
				const melde = (/** @type {number} */ stelle, /** @type {string} */ zweig) => {
					leerzustaende++;
					if (!geprueft) funde.push(`${zeile(stelle)}: ${zweig}`);
				};
				for (let glied = k; glied;) {
					geprueft ||= fragtFehler(glied.test);
					if (fragtLeer(glied.test, namen) || nachVoll)
						melde(glied.start, `{#if ${quelle.slice(glied.test.start, glied.test.end)}}`);
					nachVoll ||= fragtVoll(glied.test, namen);
					const naechstes = glied.alternate?.nodes?.find(
						(/** @type {any} */ n) => n.type === 'IfBlock'
					);
					if (naechstes?.elseif) {
						glied = naechstes;
						continue;
					}
					if (nachVoll && glied.alternate?.nodes?.length) {
						geprueft ||= beginntMitFehlerpruefung(glied.alternate);
						melde(glied.alternate.nodes[0].start, '{:else}');
					}
					glied = null;
				}
			}
			umgebend.push(k.test);
		},
		leave(/** @type {any} */ k) {
			if (k.type === 'IfBlock') umgebend.pop();
		}
	});
	return { funde, leerzustaende };
}

describe('Leerzustand nur nach der Prüfung des Ladefehlers', () => {
	const ergebnis = sammleQuelldateien(srcRoot)
		.filter((f) => f.endsWith('.svelte'))
		.map((f) => ({ datei: relPfad(f), ...leerOhneFehlerpruefung(readFileSync(f, 'utf8')) }));

	it('findet die Leerzustände', () => {
		// Fände der Detektor keine mehr, wäre er blind und das Gate grün.
		expect(ergebnis.reduce((summe, e) => summe + e.leerzustaende, 0)).toBeGreaterThan(40);
	});

	it('kein Leerzustand steht ohne Prüfung des Ladefehlers davor', () => {
		const neu = ergebnis
			.filter((e) => e.funde.length > (AUSNAHMEN[e.datei]?.[0] ?? 0))
			.map((e) => `${e.datei}: ${e.funde.join(' · ')}`);
		expect(
			neu,
			'Leerzustand ohne Ladefehler davor — bei gescheitertem Abruf ui/LadeFehler an seiner Stelle zeigen'
		).toEqual([]);
	});

	it('der Bestand der Ausnahmen ist aktuell', () => {
		const ueberholt = Object.entries(AUSNAHMEN)
			.filter(
				([datei, [anzahl]]) => (ergebnis.find((e) => e.datei === datei)?.funde.length ?? 0) < anzahl
			)
			.map(([datei]) => datei);
		expect(ueberholt, 'Ausnahme ohne Fund — austragen').toEqual([]);
	});

	it('Selbstprobe: der Detektor fasst die Formen', () => {
		for (const form of [
			'{#if laeuft}<Ladekreis />{:else if liste.length === 0}<p>Nichts offen.</p>{/if}',
			'{#if fehler}<p>{fehler}</p>{/if}\n{#if !liste.length}<p>Nichts da</p>{/if}',
			'{#if offen && treffer.length == 0}<p>Keine Treffer</p>{/if}',
			'{#if zeilen.length < 1}<p>Leer</p>{:else}<Tabelle />{/if}',
			'{#if laedt}<Ladekreis />{:else}{#if m.liste.length === 0}<p>Nichts offen.</p>{/if}{/if}',
			'{#if treffer.length > 0}<Liste />{:else if gesucht}<p>Keine Bücher gefunden</p>{/if}',
			'{#if zeilen.length}<Tabelle />{:else}<p>Leer</p>{/if}',
			'{#if laedt}<Ladekreis />{:else if bereit && zeilen.length !== 0}<Tabelle />{:else}<p>Leer</p>{/if}',
			'{#each liste as x (x)}<p>{x}</p>{:else}<p>Nichts da</p>{/each}',
			'{#each liste as x (x)}<p>{x}</p>{:else}<p>Nichts</p>{#if fehler}<p>{fehler}</p>{/if}{/each}',
			'<script>const leer = $derived(a.length === 0 && b.length === 0);</script>{#if leer}<p>Keine</p>{/if}',
			'<script>const hatDaten = $derived(zeilen.length > 0);</script>{#if hatDaten}<Tabelle />{:else}<p>Leer</p>{/if}',
			'<script>const hatDaten = $derived(zeilen.length > 0);</script>{#if !hatDaten}<p>Leer</p>{/if}'
		]) {
			expect(leerOhneFehlerpruefung(form).funde, form).not.toEqual([]);
		}
		for (const form of [
			'{#if ladefehler}<LadeFehler />{:else if liste.length === 0}<p>Nichts</p>{/if}',
			'{#if !fehler && liste.length === 0}<p>Nichts</p>{/if}',
			'{#if laedt}<Ladekreis />{:else if exemplareNichtGeladen}<LadeFehler />{:else if x.length === 0}<p>Leer</p>{/if}',
			'{#if !store.fehler}{#if liste.length === 0}<p>Nichts</p>{/if}{/if}',
			'{#if fehlendeListen.length === 0 && gebuehren.length === 0}<p>Keine</p>{/if}',
			'{#if klassen.length === 1}<p>Eine Klasse</p>{/if}',
			'{#if ladefehler}<LadeFehler />{:else if treffer.length > 0}<Liste />{:else}<p>Keine</p>{/if}',
			'{#if treffer.length > 0}<Liste />{:else if ladefehler}<LadeFehler />{:else}<p>Keine</p>{/if}',
			'{#if fehler}<LadeFehler />{:else}{#each liste as x (x)}<p>{x}</p>{:else}<p>Nichts</p>{/each}{/if}',
			'{#if auswahl.length > 0}<Knopf />{/if}',
			'{#each liste as x (x)}<p>{x}</p>{/each}',
			'{#each liste as x (x)}<p>{x}</p>{:else}{#if ladefehler}<LadeFehler />{:else}<p>Nichts</p>{/if}{/each}',
			'{#if liste.length}<Tabelle />{:else}<!-- leer -->\n{#if fehler}<LadeFehler />{:else}<p>Leer</p>{/if}{/if}',
			'<script>const leer = $derived(a.length === 0);</script>{#if leer && ladefehler}<LadeFehler />{:else if leer}<p>Keine</p>{/if}',
			'<script>const leer = $derived(a.length === 0);</script>{#if !leer}<Tabelle />{/if}',
			'<script>const leer = $derived(a.length === 0);</script>{#if x.leer}<p>Feld</p>{/if}',
			'<script>const anzahl = $derived(a.filter(f).length);</script>{#if anzahl}<p>{anzahl}</p>{:else if a.length}<p>Alle gut</p>{/if}'
		]) {
			expect(leerOhneFehlerpruefung(form).funde, form).toEqual([]);
		}
	});
});
