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
		// Die Schul-E-Mail ist seit dem 16.09.2026 Pflicht — ohne sie kommt das Formular
		// nicht bis zum Server, und der Test prüfte nur noch seine eigene Vorprüfung.
		if (art !== 'schueler') {
			await fireEvent.input(screen.getByLabelText('Schul-E-Mail *'), {
				target: { value: `${vorname}.${nachname}@schule.invalid`.toLowerCase() }
			});
		}
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

	it('schickt die Schul-E-Mail einer Lehrkraft mit', async () => {
		await anlegen('lehrkraft');
		const koerper = /** @type {any} */ (vi.mocked(apiClient.post).mock.calls[0][1]);
		expect(koerper.email).toBe('katrin.wendland@schule.invalid');
	});

	// Die Gegenrichtung — dass beim Schüler KEINE Adresse mitgeht — steht bewusst nicht
	// hier, sondern im Server-Test (api/leser_kollegium_konto_pg_test.go). Hier wäre sie
	// teuer zu bekommen: Ein Schüler braucht zusätzlich Klasse und Geburtsdatum, sonst
	// kommt das Formular gar nicht erst los. Und sie ist dort auch besser aufgehoben — der
	// Server weist eine Adresse an einem Schüler ausdrücklich ab, das Formular schickt sie
	// nur nicht mit.

	it('lässt eine Lehrkraft ohne Schul-E-Mail gar nicht erst los', async () => {
		const screen = render(StudentCreateModal, {
			open: true,
			klassen: [],
			onclose: vi.fn(),
			onsuccess: vi.fn()
		});
		await fireEvent.click(screen.getByLabelText('Lehrkraft'));
		await fireEvent.input(screen.getByLabelText('Vorname *'), { target: { value: 'Ohne' } });
		await fireEvent.input(screen.getByLabelText('Nachname *'), { target: { value: 'Mail' } });
		await fireEvent.click(screen.getByText('Speichern'));

		expect(vi.mocked(apiClient.post)).not.toHaveBeenCalled();
		expect(screen.container.textContent ?? '').toContain('Schul-E-Mail-Adresse fehlt');
	});

	// UMGEKEHRT seit dem 16.09.2026 (Peter): Aus der Schul-E-Mail entsteht der Zugang gleich
	// mit. Vorher stand hier ausdrücklich „kein Zugang zum Programm" — das war richtig,
	// solange die Adresse fehlte, und führte genau zu dem Doppeleintrag, den die Pflicht
	// jetzt abschafft. Was der Dialog NICHT vergibt, ist eine Rolle.
	it('sagt einer Lehrkraft, dass hier ihr Zugang entsteht — aber keine Rolle', async () => {
		const screen = render(StudentCreateModal, {
			open: true,
			klassen: [],
			onclose: vi.fn(),
			onsuccess: vi.fn()
		});
		await fireEvent.click(screen.getByLabelText('Lehrkraft'));
		const text = screen.container.textContent ?? '';
		expect(text).toContain('Zugang zu „Mein Portal');
		expect(text, 'eine Rolle vergibt der Administrator eigens').toContain('Rolle');
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
