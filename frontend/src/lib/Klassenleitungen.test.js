import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('./apiFetch.js', async (original) => ({
	.../** @type {any} */ (await original()),
	apiFetch: vi.fn(),
	apiGet: vi.fn(),
	apiPost: vi.fn(),
	apiDelete: vi.fn()
}));
vi.mock('./stores/snackbars.svelte.js', () => ({ snackbars: { addToast: vi.fn() } }));

import { apiFetch, apiGet } from './apiFetch.js';
import Klassenleitungen from './Klassenleitungen.svelte';

// Scheitert das Laden, steht weder „Noch keine Zuordnungen" noch das Formular da: Hinzufügen
// ersetzt die Zuordnung derselben Klasse, und die stünde gerade nicht in der Liste.
describe('Klassenleitungen', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({ ok: true, json: async () => ['05F'] })
		);
	});

	it('zeigt nach einem gescheiterten Laden den Fehler statt einer leeren Liste', async () => {
		vi.mocked(apiGet).mockRejectedValueOnce(new Error('Netz weg'));
		const screen = render(Klassenleitungen);

		await waitFor(() => expect(screen.getByRole('alert')).toBeTruthy());
		expect(screen.queryByText('Noch keine Zuordnungen vorhanden.')).toBeNull();
		expect(screen.queryByRole('button', { name: 'Hinzufügen' })).toBeNull();
	});

	it('lädt auf „Erneut versuchen" neu und zeigt dann Liste und Formular', async () => {
		vi.mocked(apiGet).mockRejectedValueOnce(new Error('Netz weg'));
		vi.mocked(apiGet).mockResolvedValueOnce([
			{ klasse: '05F', lehrer_email: 'lehrkraft@schule.example' }
		]);
		const screen = render(Klassenleitungen);

		await fireEvent.click(await screen.findByRole('button', { name: 'Erneut versuchen' }));

		expect(await screen.findByText('lehrkraft@schule.example')).toBeTruthy();
		expect(screen.getByRole('button', { name: 'Hinzufügen' })).toBeTruthy();
		expect(screen.queryByRole('alert')).toBeNull();
	});
});
