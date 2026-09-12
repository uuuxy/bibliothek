import { describe, it, expect, vi } from 'vitest';
import {
	naechsterIndex,
	tippsprungIndex,
	tastenBefehl,
	tippsprungSammler
} from './selectTastatur.js';

// Die Tastaturbedienung des Auswahlfelds — die Regeln, nicht das Markup.
//
// Ein nachgebautes Auswahlfeld muss ohne Maus alles können, was ein natives kann.
// Bis zum 12.09.2026 lagen diese Regeln als if-Kette in Select.svelte und waren nur
// über das gerenderte Bauteil prüfbar; ein vertauschter Zweig fiel dort niemandem auf.

const optionen = [
	{ value: 'a', label: 'Anna' },
	{ value: 'b', label: 'Bernd', disabled: true },
	{ value: 'c', label: 'Cem' }
];

describe('naechsterIndex', () => {
	it('überspringt gesperrte Einträge', () => {
		expect(naechsterIndex(optionen, 0, 1)).toBe(2);
		expect(naechsterIndex(optionen, 2, -1)).toBe(0);
	});

	it('läuft am Ende um', () => {
		expect(naechsterIndex(optionen, 2, 1)).toBe(0);
		expect(naechsterIndex(optionen, 0, -1)).toBe(2);
	});

	// Ohne die Zählschranke liefe die Suche nach dem nächsten freien Eintrag endlos.
	// Nach einer vollen Runde steht der Index wieder da, wo er war — die Auswahl
	// wandert also nicht auf einen gesperrten Eintrag.
	it('bleibt stehen, wenn alles gesperrt ist', () => {
		const alleGesperrt = [
			{ value: 'a', label: 'A', disabled: true },
			{ value: 'b', label: 'B', disabled: true }
		];
		expect(naechsterIndex(alleGesperrt, 0, 1)).toBe(0);
	});

	it('kommt mit einer leeren Liste zurecht', () => {
		expect(naechsterIndex([], -1, 1)).toBe(-1);
	});
});

describe('tippsprungIndex', () => {
	it('springt zum ersten passenden Eintrag', () => {
		expect(tippsprungIndex(optionen, 'c')).toBe(2);
		expect(tippsprungIndex(optionen, 'be')).toBe(1);
	});

	it('meldet -1 ohne Treffer — der Index bleibt dann stehen', () => {
		expect(tippsprungIndex(optionen, 'x')).toBe(-1);
	});
});

describe('tastenBefehl', () => {
	it('öffnet geschlossen mit Enter, Leertaste und den Pfeilen', () => {
		for (const taste of ['Enter', ' ', 'ArrowDown', 'ArrowUp']) {
			expect(tastenBefehl(taste, false)).toEqual({ tat: 'oeffnen', verhindern: true });
		}
	});

	it('lässt geschlossen alles andere dem Browser', () => {
		for (const taste of ['a', 'Tab', 'Escape', 'F5']) {
			expect(tastenBefehl(taste, false)).toEqual({ tat: 'nichts', verhindern: false });
		}
	});

	it('bedient das offene Feld', () => {
		expect(tastenBefehl('ArrowDown', true)).toMatchObject({ tat: 'wandern', richtung: 1 });
		expect(tastenBefehl('ArrowUp', true)).toMatchObject({ tat: 'wandern', richtung: -1 });
		expect(tastenBefehl('Home', true)).toMatchObject({ tat: 'springen', ziel: 'anfang' });
		expect(tastenBefehl('End', true)).toMatchObject({ tat: 'springen', ziel: 'ende' });
		expect(tastenBefehl('Enter', true)).toMatchObject({ tat: 'waehlen' });
		expect(tastenBefehl(' ', true)).toMatchObject({ tat: 'waehlen' });
		expect(tastenBefehl('Escape', true)).toMatchObject({ tat: 'schliessen' });
	});

	// Tab schließt, ohne den Fokus zurückzuholen — und darf NICHT verhindert werden,
	// sonst säße die Tastaturbedienung im Feld fest.
	it('lässt Tab weiterwandern', () => {
		expect(tastenBefehl('Tab', true)).toEqual({ tat: 'verlassen', verhindern: false });
	});

	it('macht aus einem Zeichen einen Tippsprung, aus F5 nichts', () => {
		expect(tastenBefehl('c', true)).toEqual({ tat: 'tippen', zeichen: 'c', verhindern: false });
		expect(tastenBefehl('F5', true)).toEqual({ tat: 'nichts', verhindern: false });
	});

	// Die Pfeiltasten scrollen sonst die Seite, die Leertaste ebenso.
	it('verhindert die Browser-Deutung genau bei den Steuertasten', () => {
		const verhindert = ['ArrowDown', 'ArrowUp', 'Home', 'End', 'Enter', ' ', 'Escape'];
		for (const taste of verhindert) {
			expect(tastenBefehl(taste, true).verhindern, taste).toBe(true);
		}
		for (const taste of ['Tab', 'c', 'F5']) {
			expect(tastenBefehl(taste, true).verhindern, taste).toBe(false);
		}
	});
});

describe('tippsprungSammler', () => {
	it('sammelt Zeichen und vergisst sie nach der Pause', () => {
		vi.useFakeTimers();
		try {
			const sammler = tippsprungSammler(600);
			expect(sammler.zeichen('b', optionen)).toBe(1); // „b" → Bernd
			expect(sammler.zeichen('e', optionen)).toBe(1); // „be" → weiter Bernd
			// Nach der Pause beginnt die Eingabe neu: „c" allein trifft Cem.
			vi.advanceTimersByTime(700);
			expect(sammler.zeichen('c', optionen)).toBe(2);
		} finally {
			vi.useRealTimers();
		}
	});

	// Die Bugklasse „Timer überlebt den Abbau" (sweeps.md, 11.09.2026): Ein Timer, den
	// niemand räumt, läuft nach dem Ende der Komponente weiter.
	it('räumt seinen Timer beim Abbau', () => {
		vi.useFakeTimers();
		try {
			const sammler = tippsprungSammler();
			sammler.zeichen('a', optionen);
			expect(vi.getTimerCount()).toBe(1);
			sammler.abbauen();
			expect(vi.getTimerCount()).toBe(0);
		} finally {
			vi.useRealTimers();
		}
	});
});
