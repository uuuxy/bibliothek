import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

vi.mock('./apiFetch.js', () => ({
	apiClient: { post: vi.fn() },
	apiFetch: vi.fn()
}));

import { apiClient } from './apiFetch.js';
import StudentCreateModal from './StudentCreateModal.svelte';

// „Neuen Leser anlegen" fragt zuerst nach der Art, und an ihr hängt, WAS hinausgeht.
//
// Ginge bei einer Lehrkraft eine Klasse oder ein Geburtsdatum mit, stünde sie in den
// Klassenlisten und im LUSD-Abgleich — ein Import, der sie nicht kennt, machte sie zur
// Abgängerin und anonymisierte nach der Karenz ihren Namen (Migration 123).
describe('Neuen Leser anlegen', () => {
	beforeEach(() => {
		vi.mocked(apiClient.post).mockReset();
		vi.mocked(apiClient.post).mockResolvedValue(
			/** @type {any} */ ({ ok: true, json: async () => ({ status: 'success' }) })
		);
	});

	/** @param {string} art */
	async function anlegen(art, vorname = 'Katrin', nachname = 'Wendland') {
		const screen = render(StudentCreateModal, {
			open: true,
			klassen: ['07A'],
			onclose: vi.fn(),
			onsuccess: vi.fn()
		});
		if (art !== 'schueler') {
			await fireEvent.click(screen.getByLabelText(art === 'liv' ? 'LiV' : 'Lehrkraft'));
		}
		await fireEvent.input(screen.getByLabelText('Vorname *'), { target: { value: vorname } });
		await fireEvent.input(screen.getByLabelText('Nachname *'), { target: { value: nachname } });
		await fireEvent.click(screen.getByText('Speichern'));
		return screen;
	}

	it('schickt für eine Lehrkraft die Art — und weder Klasse noch Geburtsdatum', async () => {
		await anlegen('lehrkraft');

		expect(vi.mocked(apiClient.post)).toHaveBeenCalledTimes(1);
		const [pfad, koerper] = vi.mocked(apiClient.post).mock.calls[0];
		expect(pfad).toBe('/api/schueler');
		expect(/** @type {any} */ (koerper).art).toBe('lehrkraft');
		expect(
			/** @type {any} */ (koerper).klasse,
			'mit einer Klasse stünde die Lehrkraft in den Klassenlisten und im LUSD-Abgleich'
		).toBe('');
		expect(/** @type {any} */ (koerper).geburtsdatum).toBeNull();
	});

	it('nennt die LiV als eigene Art', async () => {
		await anlegen('liv');
		expect(/** @type {any} */ (vi.mocked(apiClient.post).mock.calls[0][1]).art).toBe('liv');
	});

	it('sagt einer Lehrkraft, dass hier kein Zugang entsteht', async () => {
		const screen = render(StudentCreateModal, {
			open: true,
			klassen: [],
			onclose: vi.fn(),
			onsuccess: vi.fn()
		});
		await fireEvent.click(screen.getByLabelText('Lehrkraft'));
		expect(screen.container.textContent ?? '').toContain('kein Zugang zum Programm');
	});

	it('verlangt vom Schüler weiterhin Klasse und Geburtsdatum', async () => {
		const screen = await anlegen('schueler', 'Lena', 'Hoffmann');

		expect(
			vi.mocked(apiClient.post),
			'ein Schüler ohne Klasse darf gar nicht erst abgeschickt werden'
		).not.toHaveBeenCalled();
		expect(screen.container.textContent ?? '').toContain('Klasse');
	});
});
