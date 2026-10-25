import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render } from '@testing-library/svelte';
import DruckCenter from './DruckCenter.svelte';
import { authStore } from '../../stores/authStore.svelte.js';

// Die Reiter laden beim Öffnen; hier zählt nur, welche es gibt.
vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn(() => Promise.resolve({ ok: false, status: 403, json: async () => ({}) })),
	apiGet: vi.fn(() => Promise.resolve(null)),
	apiPost: vi.fn(() => Promise.resolve(null))
}));

// Der Menüpunkt öffnet mit jedem Recht eines Reiters (menue.js). Dahinter steht nur, was die
// Rolle öffnen darf: Ein Reiter, dessen Routen mit 403 antworten, wäre eine Tür ins Leere.
describe('Druck-Center', () => {
	beforeEach(() => {
		authStore.currentUser = /** @type {any} */ ({ rolle: 'mitarbeiter', permissions: [] });
	});

	/** @param {string[]} rechte */
	const reiter = (rechte) => {
		authStore.currentUser = /** @type {any} */ ({ rolle: 'mitarbeiter', permissions: rechte });
		const screen = render(DruckCenter);
		return screen.queryAllByRole('tab').map((t) => t.textContent?.trim().replace(/\s*\d+$/, ''));
	};

	it('zeigt mit edit_books nur die fehlenden Etiketten', () => {
		expect(reiter(['edit_books'])).toEqual(['Fehlende Etiketten']);
	});

	it('zeigt die Buch-Etiketten nur mit view_books und edit_books', () => {
		// Gedruckt wird mit view_books, vermerkt mit edit_books; der Helfer ab Werk hat nur das
		// erste und sieht das Druck-Center nicht (menue.test.js).
		expect(reiter(['view_books', 'edit_books'])).toEqual(['Buch-Etiketten', 'Fehlende Etiketten']);
		expect(reiter(['view_books'])).toEqual([]);
	});

	it('zeigt mit view_students Ausweise und Klassendruck', () => {
		expect(reiter(['view_students'])).toEqual(['Schülerausweise', 'Klassenweise drucken']);
	});
});
