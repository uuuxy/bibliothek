import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import BuchKarte from './BuchKarte.svelte';

// Zwei Kommentare widersprachen sich (OFFEN.md 5.9): BuchKarte.svelte sagte, der Stift
// öffne trotz stopPropagation auch die Akte, weil der Flächen-Handler „als direkter
// Listener vor Sveltes Delegation feuert"; e2e/cover-aendern.spec.js sagte, der Stift
// öffne die Titel-Verwaltung, nicht die Akte. Gemessen am gebauten Bauteil: Welche der
// beiden Rückrufe feuern bei einem Klick auf den Stift?
const buch = {
	id: 'b-1',
	isbn: '9783123456789',
	title: 'Mathematik Neue Wege 9',
	author: 'Lergenmüller',
	subject: 'Mathematik',
	coverUrl: '',
	gesamt: 28,
	verfuegbar: 27
};

describe('BuchKarte — wohin die Klicks gehen', () => {
	it('der Stift öffnet NUR die Bearbeitung, nicht auch die Akte', async () => {
		const akte = vi.fn();
		const bearbeiten = vi.fn();
		const { getByRole } = render(BuchKarte, { book: buch, onclick: akte, onEditClick: bearbeiten });

		await fireEvent.click(getByRole('button', { name: 'Buch schnell bearbeiten' }));

		expect(bearbeiten).toHaveBeenCalledTimes(1);
		expect(akte).not.toHaveBeenCalled();
	});

	it('der Titel öffnet die Akte über die Fläche — genau einmal', async () => {
		const akte = vi.fn();
		const { getByRole } = render(BuchKarte, { book: buch, onclick: akte });

		await fireEvent.click(getByRole('button', { name: buch.title }));

		expect(akte).toHaveBeenCalledTimes(1);
	});
});
