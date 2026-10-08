import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import BookAkteMeta from './BookAkteMeta.svelte';

/** @param {Record<string, any>} zusatz */
function einordnung(zusatz) {
	const nichts = () => {};
	const { container } = render(BookAkteMeta, {
		book: { id: 't-1', title: 'Natura', author: 'Beyer', subject: 'Biologie', ...zusatz },
		borrowers: [],
		exemplare: [],
		coverSrc: '',
		coverFailed: true,
		onCoverError: nichts,
		onCoverLoad: nichts
	});
	return container.textContent ?? '';
}

// Der Kopf der Buchakte nennt den Jahrgang, wie Maske und Titelliste ihn führen: aus „von …
// bis".
describe('Buchakte, Kopf: der Jahrgang', () => {
	it('nennt ein Jahr und eine Spanne', () => {
		const einJahr = einordnung({ jahrgangVon: 7, jahrgangBis: 7 });
		expect(einJahr).toContain('Biologie · Jahrgang 7');
		expect(einJahr).not.toContain('Jahrgang 7–');
		expect(einordnung({ jahrgangVon: 7, jahrgangBis: 10 })).toContain('Jahrgang 7–10');
	});

	it('nennt ohne „von … bis" keinen Jahrgang', () => {
		const text = einordnung({ jahrgangVon: 0, jahrgangBis: 0 });
		expect(text).toContain('Biologie');
		expect(text).not.toContain('Jahrgang');
	});
});
