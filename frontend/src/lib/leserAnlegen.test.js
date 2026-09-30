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
/**
 * Wählt die Art in der Auswahlliste „Art des Lesers" (seit dem 30.09.2026 eine Liste mit
 * sieben Arten statt drei Knöpfen).
 * @param {any} screen
 * @param {string} wort
 */
async function waehleArt(screen, wort) {
	await fireEvent.click(screen.getByRole('combobox', { name: 'Art des Lesers' }));
	await fireEvent.click(screen.getByRole('option', { name: wort }));
}

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
			await waehleArt(screen, art === 'liv' ? 'LiV' : 'Lehrkraft');
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
		await waehleArt(screen, 'Lehrkraft');
		await fireEvent.input(screen.getByLabelText('Vorname *'), { target: { value: 'Ohne' } });
		await fireEvent.input(screen.getByLabelText('Nachname *'), { target: { value: 'Mail' } });
		await fireEvent.click(screen.getByText('Speichern'));

		expect(vi.mocked(apiClient.post)).not.toHaveBeenCalled();
		expect(screen.container.textContent ?? '').toContain('Schul-E-Mail-Adresse fehlt');
	});

	// UMGEKEHRT seit dem 16.09.2026 : Aus der Schul-E-Mail entsteht der Zugang gleich
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
		await waehleArt(screen, 'Lehrkraft');
		const text = screen.container.textContent ?? '';
		expect(text).toContain('Zugang zu „Mein Portal');
		expect(text, 'eine Rolle vergibt der Administrator eigens').toContain('Rolle');
	});

	// Praktikum und Fachbereich bekommen keinen Zugang zu „Mein Portal" (30.09.2026). Bis dahin
	// ließ sich ein Praktikant ohne Schuladresse gar nicht anlegen: Die Adresse war für jeden
	// Kollegen Pflicht.
	it('legt ein Praktikum ohne Schul-E-Mail an und schickt keine mit', async () => {
		const screen = render(StudentCreateModal, {
			open: true,
			klassen: [],
			onclose: vi.fn(),
			onsuccess: vi.fn()
		});
		await waehleArt(screen, 'Praktikum');
		await fireEvent.input(screen.getByLabelText('Vorname *'), { target: { value: 'Paul' } });
		await fireEvent.input(screen.getByLabelText('Nachname *'), { target: { value: 'Praktikant' } });
		expect(
			/** @type {HTMLInputElement} */ (screen.getByLabelText('Schul-E-Mail')).disabled,
			'die Adresse ist beim Praktikum verschlossen'
		).toBe(true);
		expect(screen.container.textContent ?? '').toContain('keinen Zugang zu „Mein Portal“');
		await fireEvent.click(screen.getByText('Speichern'));

		expect(vi.mocked(apiClient.post)).toHaveBeenCalledTimes(1);
		const koerper = /** @type {any} */ (vi.mocked(apiClient.post).mock.calls[0][1]);
		expect(koerper.art).toBe('praktikum');
		expect(koerper.email).toBe('');
	});

	// Eine vorher getippte Adresse geht nicht mit, wenn die Art danach auf Fachbereich wechselt:
	// Der Server wiese sie ab.
	it('schickt beim Fachbereich auch eine vorher getippte Adresse nicht mit', async () => {
		const screen = render(StudentCreateModal, {
			open: true,
			klassen: [],
			onclose: vi.fn(),
			onsuccess: vi.fn()
		});
		await waehleArt(screen, 'Lehrkraft');
		await fireEvent.input(screen.getByLabelText('Schul-E-Mail *'), {
			target: { value: 'erdkunde@schule.invalid' }
		});
		await waehleArt(screen, 'Fachbereich');
		await fireEvent.input(screen.getByLabelText('Vorname *'), { target: { value: 'Fachbereich' } });
		await fireEvent.input(screen.getByLabelText('Nachname *'), { target: { value: 'Erdkunde' } });
		await fireEvent.click(screen.getByText('Speichern'));

		const koerper = /** @type {any} */ (vi.mocked(apiClient.post).mock.calls[0][1]);
		expect(koerper.art).toBe('fachbereich');
		expect(koerper.email).toBe('');
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
