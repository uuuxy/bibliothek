import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import AnliegenWidget from './AnliegenWidget.svelte';
import { apiFetch } from '../../apiFetch.js';

vi.mock('../../apiFetch.js', () => ({ apiFetch: vi.fn() }));
vi.mock('../../stores/toastStore.svelte.js', () => ({ toastStore: { addToast: vi.fn() } }));

// Die Art eines Anliegens wird gewählt und nie vorbelegt: Mit der Vorbelegung „Buchwunsch"
// stand ein Problem als Wunsch in der Liste der Bibliothek, sobald niemand umschaltete.

const aufbau = () => {
	const onaktualisiert = vi.fn();
	return { ...render(AnliegenWidget, { anliegen: [], onaktualisiert }), onaktualisiert };
};

/** Der Rumpf der letzten Anfrage an die Tür für Anliegen. */
function gesendet() {
	const ruf = vi.mocked(apiFetch).mock.calls.findLast(([url]) => url === '/api/anliegen');
	return ruf ? JSON.parse(/** @type {any} */ (ruf[1]).body) : null;
}

describe('AnliegenWidget', () => {
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({ ok: true, json: async () => ({ id: 'neu' }), headers: new Headers() })
		);
	});

	it('zeigt vor der Wahl zwei Knöpfe und kein Formular', () => {
		const s = aufbau();

		expect(s.getByRole('button', { name: 'Buchwunsch' })).toBeTruthy();
		expect(s.getByRole('button', { name: 'Problem melden' })).toBeTruthy();
		expect(s.queryByRole('radio'), 'eine Auswahl mit Vorbelegung').toBeNull();
		expect(s.queryByLabelText('Welches Buch?')).toBeNull();
		expect(s.queryByRole('button', { name: 'Absenden' })).toBeNull();
	});

	it('schickt nach „Problem melden" eine Meldung und verlangt die Beschreibung', async () => {
		const s = aufbau();
		await fireEvent.click(s.getByRole('button', { name: 'Problem melden' }));

		await fireEvent.input(s.getByLabelText('Welches Buch?'), {
			target: { value: 'Markl Biologie 2' }
		});
		const absenden = /** @type {HTMLButtonElement} */ (s.getByRole('button', { name: 'Absenden' }));
		expect(absenden.disabled, 'ohne Beschreibung des Problems').toBe(true);

		await fireEvent.input(s.getByLabelText('Was stimmt nicht?'), {
			target: { value: 'falsche Auflage' }
		});
		await fireEvent.click(absenden);

		await vi.waitFor(() => expect(gesendet()).toBeTruthy());
		expect(gesendet()).toEqual({
			art: 'meldung',
			titel_text: 'Markl Biologie 2',
			klasse: '',
			kommentar: 'falsche Auflage'
		});
		// Nach dem Absenden steht wieder die Wahl da, und die eigene Liste wird neu gelesen.
		await vi.waitFor(() =>
			expect(s.queryByRole('button', { name: 'Problem melden' })).toBeTruthy()
		);
		expect(s.onaktualisiert).toHaveBeenCalled();
	});

	it('schickt nach „Buchwunsch" einen Wunsch, die Anmerkung bleibt freiwillig', async () => {
		const s = aufbau();
		await fireEvent.click(s.getByRole('button', { name: 'Buchwunsch' }));

		await fireEvent.input(s.getByLabelText('Welches Buch?'), { target: { value: 'Neues Buch' } });
		await fireEvent.click(s.getByRole('button', { name: 'Absenden' }));

		await vi.waitFor(() => expect(gesendet()).toBeTruthy());
		expect(gesendet().art).toBe('wunsch');
		expect(gesendet().titel_text).toBe('Neues Buch');
	});

	it('führt mit „Abbrechen" zurück zur Wahl, ohne etwas zu senden', async () => {
		const s = aufbau();
		await fireEvent.click(s.getByRole('button', { name: 'Buchwunsch' }));
		await fireEvent.click(s.getByRole('button', { name: 'Abbrechen' }));

		await vi.waitFor(() =>
			expect(s.queryByRole('button', { name: 'Problem melden' })).toBeTruthy()
		);
		expect(s.queryByLabelText('Welches Buch?')).toBeNull();
		expect(gesendet()).toBeNull();
	});
});
