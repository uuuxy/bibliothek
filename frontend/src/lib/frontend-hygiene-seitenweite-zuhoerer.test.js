import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { ohneKommentare, relPfad, sammleQuelldateien, srcRoot } from './hygiene-quellen.js';
import { ABGESCHIRMT } from './sperrSchild.js';

// Hinter dem Sperrbildschirm bleibt die Anwendung stehen (App.svelte), damit Ungespeichertes
// die Sperre überlebt. Ein Bauteil, das an window oder document auf Bedienung hört, bekäme
// die Tasten und Klicks des Sperrbildschirms mit: Der Router deutet Escape als „zurück an
// die Theke" und schlösse die offene Maske. sperrSchild.js hält die Ereignisse seiner Liste
// fern. Diese Ratsche verlangt für jedes Ereignis, auf das seitenweit gehört wird, eine
// Entscheidung: abgeschirmt, oder hier mit Grund ausgenommen.
const OHNE_SCHILD = {
	error: 'Fehlerbericht, keine Bedienung',
	unhandledrejection: 'Fehlerbericht, keine Bedienung',
	scroll: 'führt Maße nach, ändert keinen Inhalt',
	resize: 'führt Maße nach, ändert keinen Inhalt',
	beforeunload: 'warnt vor dem Schließen des Fensters — auch hinter der Sperre gewollt',
	popstate: 'der Router stellt hinter der Sperre die Adresse zurück (Router.svelte)',
	offline: 'Netzlage',
	online: 'Netzlage',
	focus: 'idleLock fragt den Server, ob nebenan aufgeschlossen wurde',
	storage: 'Signale der anderen Fenster desselben Browsers',
	mouseover: 'Sprechblase zeigen; der Sperrbildschirm trägt kein data-tip',
	mouseout: 'Sprechblase schließen',
	focusin: 'Sprechblase zeigen; der Sperrbildschirm trägt kein data-tip',
	focusout: 'Sprechblase schließen'
};

// Dateien, die den Namen des Ereignisses aus einer Variablen nehmen.
const NAME_AUS_VARIABLE = {
	'src/lib/stores/idleLock.svelte.js':
		'zählt Bedienung für die Uhr der Sperre; aktivitaet() steigt aus, solange gesperrt ist',
	'src/lib/sperrSchild.js': 'der Schild selbst'
};

/**
 * Ereignisse, auf die eine Quelle an window, document oder body hört.
 * @param {string} quelle
 * @returns {{ namen: string[], variabel: number }}
 */
function seitenweiteZuhoerer(quelle) {
	const text = ohneKommentare(quelle);
	/** @type {string[]} */
	const namen = [];
	let variabel = 0;
	for (const m of text.matchAll(/\b(?:window|document)\.addEventListener\(\s*([^,)]+)/g)) {
		const erstes = m[1].trim();
		const wort = /^(['"`])([A-Za-z]+)\1$/.exec(erstes);
		if (wort) namen.push(wort[2]);
		else variabel++;
	}
	// <svelte:window …>: Das Tag endet am ersten „>" außerhalb geschweifter Klammern —
	// in den Attributen stehen Pfeilfunktionen.
	for (const m of text.matchAll(/<svelte:(?:window|document|body)\b/g)) {
		let tiefe = 0;
		let ende = m.index + m[0].length;
		for (; ende < text.length; ende++) {
			const z = text[ende];
			if (z === '{') tiefe++;
			else if (z === '}') tiefe--;
			else if (z === '>' && tiefe === 0) break;
		}
		const tag = text.slice(m.index, ende);
		// Nur Attribute des Tags selbst, nicht „on…" im Rumpf einer Funktion.
		let ebene = 0;
		let aussen = '';
		for (const z of tag) {
			if (z === '{') ebene++;
			if (ebene === 0) aussen += z;
			if (z === '}') ebene--;
		}
		for (const a of aussen.matchAll(/\bon:?([a-z]+)\s*=/g)) namen.push(a[1]);
	}
	return { namen, variabel };
}

describe('Seitenweite Zuhörer: abgeschirmt oder begründet ausgenommen', () => {
	const dateien = sammleQuelldateien(srcRoot);
	const funde = dateien
		.map((d) => ({ pfad: relPfad(d), ...seitenweiteZuhoerer(readFileSync(d, 'utf8')) }))
		.filter((f) => f.namen.length > 0 || f.variabel > 0);

	it('der Detektor erkennt alle Formen', () => {
		const probe = `
			window.addEventListener('keydown', f);
			document.addEventListener("click", f, true);
			window.addEventListener(\`paste\`, f);
			window.addEventListener(name, f);
			// window.addEventListener('wheel', f);
		`;
		expect(seitenweiteZuhoerer(probe)).toEqual({
			namen: ['keydown', 'click', 'paste'],
			variabel: 1
		});
		const svelte = `<svelte:window
			onkeydown={(e) => {
				if (a > b) onclick = e;
			}}
			onpointerdown={zu}
		/>
		<svelte:document on:paste={p} /><svelte:body onmouseenter={m} />
		<div onclick={x}></div>`;
		expect(seitenweiteZuhoerer(svelte).namen).toEqual([
			'keydown',
			'pointerdown',
			'paste',
			'mouseenter'
		]);
	});

	it('findet die seitenweiten Zuhörer des Bestands — sonst prüfte die Ratsche nichts', () => {
		expect(funde.length).toBeGreaterThan(8);
		const namen = new Set(funde.flatMap((f) => f.namen));
		expect([...namen]).toEqual(expect.arrayContaining(['keydown', 'pointerdown', 'popstate']));
		expect(funde.map((f) => f.pfad)).toEqual(
			expect.arrayContaining(['src/lib/Router.svelte', 'src/lib/components/ui/Select.svelte'])
		);
	});

	it('jedes Ereignis ist abgeschirmt oder mit Grund ausgenommen', () => {
		const offen = funde.flatMap((f) =>
			f.namen
				.filter((n) => !ABGESCHIRMT.includes(n) && !(n in OHNE_SCHILD))
				.map((n) => `${f.pfad}: ${n}`)
		);
		expect(
			offen,
			'Ein Bauteil hört an window oder document auf ein Ereignis, das der Schild der Sperre ' +
				'nicht kennt. Bedienung gehört in ABGESCHIRMT (sperrSchild.js); alles andere mit Grund ' +
				'in OHNE_SCHILD.'
		).toEqual([]);
	});

	it('kein Ereignis steht in beiden Listen, und keine Ausnahme ist verwaist', () => {
		expect(Object.keys(OHNE_SCHILD).filter((n) => ABGESCHIRMT.includes(n))).toEqual([]);
		const benutzt = new Set(funde.flatMap((f) => f.namen));
		expect(Object.keys(OHNE_SCHILD).filter((n) => !benutzt.has(n))).toEqual([]);
	});

	it('der Name des Ereignisses kommt nur in bekannten Dateien aus einer Variablen', () => {
		const variabel = funde.filter((f) => f.variabel > 0).map((f) => f.pfad);
		expect(variabel.filter((p) => !(p in NAME_AUS_VARIABLE))).toEqual([]);
		expect(Object.keys(NAME_AUS_VARIABLE).filter((p) => !variabel.includes(p))).toEqual([]);
	});
});
