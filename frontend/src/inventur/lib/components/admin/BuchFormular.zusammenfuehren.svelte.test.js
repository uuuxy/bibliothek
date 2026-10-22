import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

vi.mock('../../../../lib/apiFetch.js', async (importOriginal) => ({
	...(await importOriginal()),
	apiFetch: vi.fn()
}));
vi.mock('$lib/store.svelte.js', () => ({ appState: { bookToEdit: null }, melde: vi.fn() }));
vi.mock('../../../../lib/stores/bestaetigung.svelte.js', async (importOriginal) => ({
	...(await importOriginal()),
	verwerfenBestaetigen: vi.fn()
}));

import { apiFetch } from '../../../../lib/apiFetch.js';
import { verwerfenBestaetigen } from '../../../../lib/stores/bestaetigung.svelte.js';
import { uiStore } from '../../../../lib/stores/uiStore.svelte.js';
import BuchFormular from './BuchFormular.svelte';
import { leeresBuchFormular } from './buchFormularRegeln.js';

// „Mit anderem Titel zusammenführen" in der Maske eines Titels: nur mit dem Recht (die
// Titelverwaltung reicht onZusammengefuehrt nur mit delete_books herein), vorher die Frage nach
// Ungespeichertem, und nach dem Neuladen zeigt die Maske den Titel ganz neu.

const KNOPF = 'Mit anderem Titel zusammenführen';
/** @param {string} id */
const titel = (id) => ({ ...leeresBuchFormular(), id, title: 'Natur und Technik', author: '' });

/** @param {Record<string, any>} zusatz */
function maske(zusatz = {}) {
	return render(BuchFormular, {
		formular: titel('t-1'),
		onClose: () => {},
		onSave: () => {},
		onCoverUpload: () => {},
		onCoverNeuHolen: () => {},
		onAssignClass: () => {},
		...zusatz
	});
}
const exemplarAbrufe = () =>
	vi.mocked(apiFetch).mock.calls.filter(([url]) => String(url).endsWith('/t-1/exemplare')).length;

beforeEach(() => {
	vi.mocked(apiFetch).mockReset();
	vi.mocked(apiFetch).mockResolvedValue(
		/** @type {any} */ ({ ok: true, status: 200, json: async () => [] })
	);
	vi.mocked(verwerfenBestaetigen).mockReset();
	uiStore.verlassenSperre = null;
});

describe('Titel zusammenführen in der Maske', () => {
	it('gibt es nur, wenn die Titelverwaltung es mit dem Recht hereinreicht', () => {
		expect(maske().queryByRole('button', { name: KNOPF })).toBeNull();
		expect(
			maske({ onZusammengefuehrt: vi.fn() }).getByRole('button', { name: KNOPF })
		).toBeTruthy();
	});

	it('fragt vor dem Öffnen nach Ungespeichertem und öffnet nur nach dem Verwerfen', async () => {
		uiStore.verlassenSperre = () => true;
		const screen = maske({ onZusammengefuehrt: vi.fn() });

		vi.mocked(verwerfenBestaetigen).mockResolvedValueOnce(false);
		await fireEvent.click(screen.getByRole('button', { name: KNOPF }));
		expect(screen.queryByRole('heading', { name: KNOPF })).toBeNull();

		vi.mocked(verwerfenBestaetigen).mockResolvedValueOnce(true);
		await fireEvent.click(screen.getByRole('button', { name: KNOPF }));
		expect(await screen.findByRole('heading', { name: KNOPF })).toBeTruthy();
	});

	// Nach dem Zusammenführen lädt die Titelverwaltung den Titel neu, oft mit derselben Kennung.
	// Die Exemplare laden beim Aufbau; ohne Neuaufbau stünde die alte Liste da.
	it('baut den Inhalt neu auf, wenn der Titel neu geladen in die Maske kommt', async () => {
		const screen = maske({ onZusammengefuehrt: vi.fn() });
		await vi.waitFor(() => expect(exemplarAbrufe()).toBe(1));

		await screen.rerender({ formular: titel('t-1') });

		await vi.waitFor(() => expect(exemplarAbrufe()).toBe(2));
	});
});
