import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { parse } from 'svelte/compiler';
import { srcRoot, sammleQuelldateien, relPfad } from './hygiene-quellen.js';
import geteilt from './utils/satzzeichen.faelle.json';

// Ratsche: Die Texte der Oberfläche setzen ihre Satzzeichen wie die Meldungen des Servers
// (STIL.md 11, pruefung/meldung_ratsche_test.go): ein einzelner Satz ohne Punkt, mehrere Sätze
// mit Punkt, kein Gedankenstrich. M3, Content design, Grammar and punctuation: „Avoid using
// periods to end single sentences“, „Use periods on: Multiple sentences“, „Em dashes are best
// avoided in UX writing“. Setzt das Programm Sätze zusammen, nimmt es verbindeSaetze (format.js).
//
// Ein Text ist ein String im Code, ein Attributwert oder ein Textlauf im Markup samt seinen
// Inline-Elementen. Als Satz zählt er, wenn er groß oder mit einer Zahl beginnt und ein
// Leerzeichen und einen Kleinbuchstaben trägt. Ein Satz endet mitten im Text an . ! ? nach zwei
// Buchstaben, einer Klammer, einem Anführungszeichen oder einem Platzhalter, vor einem großen
// Anfang oder einem Platzhalter; der Doppelpunkt und eine Abkürzung („bzw.“) enden keinen Satz.
//
// Sieht den Punkt am Ende nicht bei einem Satz, den ein Block ({#if}) im selben Element
// fortsetzt, und bei einem String, der mit Leerzeichen beginnt: Beide hängen an einem Satz davor.
//
// Die Abkürzungen und die Fälle für das Satzende teilt die Prüfung mit dem Server
// (utils/satzzeichen.faelle.json, pruefung/meldung_ratsche_test.go): eine Regel für beide.
const ABK = new Set(geteilt.abkuerzungen);
const INLINE = new Set(
	'a abbr b bdi code em i kbd mark q s small span strong sub sup time u'.split(' ')
);
// Eine Ordnungszahl vor einem Wort ist kein Satzende.
const AUSNAHMEN = ['src/lib/components/lmfplan/LmfPlanRahmen.svelte: {…}. Stunde'];

/** @param {string} t */
function satzende(t) {
	for (const m of t.matchAll(/(?:([\p{L}\p{N}]+)|[)“"}])[.!?]\s+[\p{Lu}„"{]/gu)) {
		if (m[1] && (m[1].length < 2 || ABK.has(`${m[1]}.`))) continue;
		return true;
	}
	return false;
}

/** @param {string} roh @param {boolean} [ohneSchlusspunkt] @returns {string[]} */
function verstoesse(roh, ohneSchlusspunkt = false) {
	const t = roh.replace(/\s+/g, ' ').trim();
	if (!/^[\p{Lu}\d„"{]/u.test(t) || !t.includes(' ') || !/\p{Ll}/u.test(t)) return [];
	const funde = [];
	if (/—| – /.test(t)) funde.push('Gedankenstrich');
	const mehrere = satzende(t);
	const ende = t.replace(/([\p{L}“")])([.!?])\{…\}$/u, '$1$2');
	const letztes = ende.split(' ').pop() ?? '';
	const abkuerzung = ABK.has(letztes) || /^\p{L}\.$/u.test(letztes);
	if (!mehrere && !ohneSchlusspunkt && /[^.]\.$/.test(ende) && !abkuerzung)
		funde.push('ein Satz mit Punkt');
	if (mehrere && !/[.!?…:]$/.test(ende)) funde.push('mehrere Sätze ohne Punkt');
	return funde;
}

// Klammern nur um ein Kürzel oder eine Quelle, M3 (Grammar and punctuation): „Parentheses can be
// used to define acronyms or jargon or when referencing a source. They shouldn’t be used when
// adding a side note or an afterthought of a sentence.“ Kein Schrägstrich zwischen Wörtern, Googles
// developer documentation style guide (Slashes): „Avoid using slashes, except in code.“ Ein Kürzel
// trägt zwei Großbuchstaben und kein Leerzeichen (SMTP, HSchG), eine Quelle beginnt mit Art., §
// oder DIN. Was in Anführungszeichen steht, ist ein Beispiel aus den Daten („LMF Deu 7 / Bie“).
const KUERZEL = /^(?=(?:[^\p{Lu}]*\p{Lu}){2})[\p{L}\d.-]+$|^(?:Art\.|§|DIN) /u;
// Code: ein Pfad, eine Adresse, ein Dateityp, und was weder Leerzeichen noch Großbuchstaben trägt.
const CODE = /^\.{0,2}\/|^data:|https?:\/\/|^[^\s\p{Lu}]+$/u;
// Eine Webadresse ohne Schema (kmk.org/service/ferienregelung) ist Code und, in Klammern, eine Quelle.
const ADRESSE = /[\w-]+(?:\.[\w-]+)*\.[a-z]{2,}\/[^\s()]*/gu;

/** @param {string} roh @returns {string[]} */
function nebenzeichen(roh) {
	const t = roh.replace(/\s+/g, ' ').trim();
	// CSS: eine Eigenschaft, eine Funktion, ein Selektor mit :not( oder eine Media-Query.
	if (/^[a-z-]+: |^[a-z-]+\(|:not\(|^\([a-z-]+(?::| [<>])/.test(t)) return [];
	const funde = [...t.matchAll(/\(([^()]*)\)/g)]
		.filter((m) => !KUERZEL.test(m[1]) && m[1].replace(ADRESSE, '') !== '')
		.map((m) => `Klammer (${m[1]})`);
	const ohneZitat = t.replace(/„[^“]*“|"[^"]*"/g, '„…“').replace(ADRESSE, '…');
	if (!CODE.test(ohneZitat) && /[\p{L}}]\s?\/\s?[\p{L}{]/u.test(ohneZitat))
		funde.push('Schrägstrich');
	return funde;
}

/**
 * Die Texte einer Datei, je mit der Angabe, ob ihr Punkt am Ende ungeprüft bleibt.
 * @param {string} quelle @param {boolean} svelte @returns {[string, boolean][]}
 */
function texte(quelle, svelte) {
	const ast = parse(svelte ? quelle : `<script>${quelle}</script>`, { modern: true });
	/** @type {[string, boolean][]} */
	const out = [];
	const verbraucht = new WeakSet();
	/** @param {any} n */
	const textVon = (n) =>
		n.type === 'Literal' && typeof n.value === 'string'
			? n.value
			: n.type === 'TemplateLiteral'
				? n.quasis.map((/** @type {any} */ q) => q.value.cooked ?? '').join('{…}')
				: '{…}';
	/** @param {any} n @returns {any[]} */
	const kette = (n) =>
		n.type === 'BinaryExpression' && n.operator === '+'
			? [...kette(n.left), ...kette(n.right)]
			: [n];
	/** @param {any} n @param {any} eltern */
	const js = (n, eltern) => {
		if (!n || typeof n !== 'object') return;
		if (Array.isArray(n)) return n.forEach((x) => js(x, eltern));
		if (typeof n.type !== 'string' || /^Import|TaggedTemplate/.test(n.type)) return;
		if (n.type === 'CallExpression' && n.callee.object?.name === 'console') return;
		if (
			n.type === 'BinaryExpression' &&
			n.operator === '+' &&
			eltern?.type !== 'BinaryExpression'
		) {
			const teile = kette(n);
			if (teile.some((t) => /^(Literal|TemplateLiteral)$/.test(t.type))) {
				out.push([teile.map(textVon).join(''), false]);
				teile.forEach((t) => verbraucht.add(t));
			}
		}
		const literal = n.type === 'Literal' && typeof n.value === 'string';
		if ((literal || n.type === 'TemplateLiteral') && !verbraucht.has(n))
			out.push([textVon(n), /^\s/.test(textVon(n))]);
		for (const [schluessel, wert] of Object.entries(n)) {
			if (schluessel === 'key' && n.type === 'Property' && !n.computed) continue;
			if (wert && typeof wert === 'object' && schluessel !== 'loc') js(wert, n);
		}
	};
	/** @param {any[]} knoten @returns {string} */
	const lauf = (knoten) =>
		knoten
			.map((k) => {
				if (k.type === 'Text') return k.data;
				if (k.type === 'ExpressionTag') return (js(k.expression, k), '{…}');
				if (k.type === 'RegularElement' && INLINE.has(k.name))
					return (attribute(k), lauf(k.fragment.nodes));
				return '';
			})
			.join('');
	/** @param {any} el */
	const attribute = (el) => {
		for (const a of el.attributes ?? []) {
			if (a.type !== 'Attribute' || a.name === 'class' || a.value === true) {
				if (a.expression) js(a.expression, a);
				continue;
			}
			const werte = Array.isArray(a.value) ? a.value : [a.value];
			const text = lauf(werte);
			if (werte.some((w) => w.type === 'Text')) out.push([text, false]);
		}
	};
	/** @param {any[]} knoten @param {boolean} imBlock */
	const fragment = (knoten, imBlock) => {
		const gemischt =
			knoten.some((k) => k.type.endsWith('Block')) &&
			knoten.some((k) => k.type === 'Text' && k.data.trim());
		/** @type {any[]} */
		let gruppe = [];
		const schliesse = () => {
			const text = lauf(gruppe);
			if (text.trim()) out.push([text, gemischt || imBlock]);
			gruppe = [];
		};
		for (const k of knoten) {
			if (
				k.type === 'Text' ||
				k.type === 'ExpressionTag' ||
				(k.type === 'RegularElement' && INLINE.has(k.name))
			) {
				gruppe.push(k);
				continue;
			}
			schliesse();
			besuche(k, imBlock || (gemischt && k.type.endsWith('Block')));
		}
		schliesse();
	};
	/** @param {any} k @param {boolean} imBlock */
	const besuche = (k, imBlock) => {
		if (k.attributes) attribute(k);
		for (const [schluessel, wert] of Object.entries(k)) {
			if (
				!wert ||
				typeof wert !== 'object' ||
				schluessel === 'attributes' ||
				schluessel === 'metadata'
			)
				continue;
			if (wert.type === 'Fragment') fragment(wert.nodes, imBlock);
			else if (typeof wert.type === 'string' && schluessel !== 'fragment') js(wert, k);
		}
	};
	if (ast.instance) js(ast.instance.content, null);
	if (ast.module) js(ast.module.content, null);
	fragment(ast.fragment.nodes, false);
	return out;
}

describe('Satzzeichen in den Texten der Oberfläche', () => {
	const gefunden = new Set();
	/** @type {string[]} */
	const funde = [];
	/** @type {string[]} */
	const zeichenFunde = [];
	let anzahl = 0;
	for (const datei of sammleQuelldateien(srcRoot)) {
		for (const [roh, ohneSchlusspunkt] of texte(
			readFileSync(datei, 'utf8'),
			datei.endsWith('.svelte')
		)) {
			anzahl++;
			const ort = `${relPfad(datei)}: ${roh.replace(/\s+/g, ' ').trim()}`;
			if (AUSNAHMEN.includes(ort)) {
				gefunden.add(ort);
				continue;
			}
			for (const v of verstoesse(roh, ohneSchlusspunkt)) funde.push(`${v}: ${ort.slice(0, 160)}`);
			for (const v of nebenzeichen(roh)) zeichenFunde.push(`${v}: ${ort.slice(0, 160)}`);
		}
	}

	it('liest die Texte aller Dateien', () => {
		expect(anzahl, 'Der Leser findet kaum Texte').toBeGreaterThan(3000);
		expect([...gefunden], 'Ausnahme ohne Fundstelle').toEqual(AUSNAHMEN);
	});

	it('setzt sie wie der Server: ein Satz ohne Punkt, mehrere mit, kein Gedankenstrich', () => {
		expect(
			funde,
			'Satzzeichen nach M3 (STIL.md 11): einen einzelnen Satz ohne Punkt, mehrere mit; statt des ' +
				'Gedankenstrichs ein Komma, ein Doppelpunkt oder ein zweiter Satz. Sätze aus Teilen ' +
				'verbindet verbindeSaetze (utils/format.js).'
		).toEqual([]);
	});

	it('setzt Klammern nur um Kürzel und Quellen, keinen Schrägstrich zwischen Wörtern', () => {
		expect(
			zeichenFunde,
			'Nach M3 gehört ein Nebengedanke in den Satz, eine Zahl in den Text („12 Exemplare“) oder ' +
				'als Badge in den Reiter, eine Einheit rechts ins Feld (Snippet nachlaufend). Statt des ' +
				'Schrägstrichs „oder“, „und“, bei Zahlen „von“.'
		).toEqual([]);
	});

	it('Selbstprobe: Klammern und Schrägstriche', () => {
		for (const gut of ['Postausgang (SMTP)', 'Gemäß (§ 153 HSchG)', 'DSGVO-Auskunft (Art. 15)'])
			expect(nebenzeichen(gut), gut).toEqual([]);
		for (const gut of ['„LMF Deu 7 / Bie“ ist ein Fach', 'image/jpeg', 'transform: scale({…});'])
			expect(nebenzeichen(gut), gut).toEqual([]);
		for (const gut of ['input:not([type="file"])', '(width < 80rem)', 'calc({…}px + 1rem)'])
			expect(nebenzeichen(gut), gut).toEqual([]);
		for (const gut of ['Plan (kmk.org/service/x)', 'Steht unter kmk.org/service/x bereit'])
			expect(nebenzeichen(gut), gut).toEqual([]);
		expect(nebenzeichen('Notiz (optional)')).toEqual(['Klammer (optional)']);
		expect(nebenzeichen('Forderungen ({…})')).toEqual(['Klammer ({…})']);
		expect(nebenzeichen('Zahlungsfrist (Tage)')).toEqual(['Klammer (Tage)']);
		for (const schlecht of ['Schüler/in', 'Klasse / Kurs', '{…} / {…} erfasst', 'N/A'])
			expect(nebenzeichen(schlecht), schlecht).toEqual(['Schrägstrich']);
	});

	it('Selbstprobe: der Detektor fasst die Formen', () => {
		expect(verstoesse('Buch gespeichert.')).toEqual(['ein Satz mit Punkt']);
		expect(verstoesse('Buch gespeichert')).toEqual([]);
		expect(verstoesse('Netzwerkfehler — die Suche')).toEqual(['Gedankenstrich']);
		expect(verstoesse('Der Name genügt – @{…} ergänzt')).toEqual(['Gedankenstrich']);
		expect(verstoesse('Erster Satz. Zweiter Satz')).toEqual(['mehrere Sätze ohne Punkt']);
		expect(verstoesse('Eingebucht. {…} brauchen ein Etikett')).toEqual([
			'mehrere Sätze ohne Punkt'
		]);
		expect(verstoesse('Erster Satz. Zweiter Satz.{…}')).toEqual([]);
		expect(verstoesse('Max. Ausleihen je Leser')).toEqual([]);
		expect(verstoesse('Ein Etikett ist schon am Buch, z. B.')).toEqual([]);
		expect(verstoesse('Kein Cover: Die DNB kennt den Titel nicht')).toEqual([]);
		expect(verstoesse(' Diese Seite bleibt erreichbar.', true)).toEqual([]);
		const markup = texte('<p>Klicke <strong>Speichern</strong>, um zu sichern.</p>', true);
		expect(markup.map(([t]) => t)).toContain('Klicke Speichern, um zu sichern.');
		expect(texte("const a = 'Erster Teil. ' + b + ' Ende.';", false).map(([t]) => t)).toContain(
			'Erster Teil. {…} Ende.'
		);
	});

	it('findet das Satzende wie der Server (satzzeichen.faelle.json)', () => {
		expect(ABK.size, 'Die geteilte Liste ist leer').toBeGreaterThan(20);
		expect(geteilt.faelle.length, 'Die geteilten Fälle fehlen').toBeGreaterThan(5);
		for (const { text, mehrere } of geteilt.faelle) expect(satzende(text), text).toBe(mehrere);
	});
});
