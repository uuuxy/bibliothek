import { describe, it, expect, vi, beforeEach } from 'vitest';

// Derselbe Scan gibt an der Theke zurück oder leiht aus. Ob eine Ausleihe entstand, ist am Ton
// zu hören: dieselben zwei Töne, in umgekehrter Folge.

/** @type {{ freq: number, zeit: number }[]} */
let toene = [];

class AttrappenKontext {
	currentTime = 0;
	destination = {};
	createOscillator() {
		return {
			type: '',
			connect: vi.fn(),
			start: vi.fn(),
			stop: vi.fn(),
			frequency: {
				/** @param {number} freq @param {number} zeit */
				setValueAtTime: (freq, zeit) => toene.push({ freq, zeit })
			}
		};
	}
	createGain() {
		return {
			connect: vi.fn(),
			gain: {
				setValueAtTime: vi.fn(),
				linearRampToValueAtTime: vi.fn(),
				exponentialRampToValueAtTime: vi.fn()
			}
		};
	}
}

vi.stubGlobal('AudioContext', AttrappenKontext);

import { playSoundSuccess } from './audio.js';

beforeEach(() => {
	toene = [];
});

describe('Ton für Gebuchtes', () => {
	it('steigt bei Rückgabe und geladenem Leser', () => {
		playSoundSuccess();

		expect(toene.map((t) => t.freq)).toEqual([880, 1320]);
	});

	it('fällt bei einer Ausleihe', () => {
		playSoundSuccess('ausleihe');

		expect(toene.map((t) => t.freq)).toEqual([1320, 880]);
	});

	// Die Ausleihe dauert nicht länger als die Rückgabe: Der nächste Scan wartet auf keinen Ton.
	it('setzt beide Töne zu denselben Zeiten', () => {
		playSoundSuccess();
		const rueckgabe = toene.map((t) => t.zeit);
		toene = [];

		playSoundSuccess('ausleihe');

		expect(toene.map((t) => t.zeit)).toEqual(rueckgabe);
	});
});
