import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import BookExemplarStatusEditor from './components/BookExemplarStatusEditor.svelte';
import { apiClient } from './apiFetch.js';

vi.mock('./apiFetch.js', () => ({
	apiClient: { put: vi.fn() },
	apiFetch: vi.fn()
}));

vi.mock('../inventur/lib/store.svelte.js', () => ({ showToast: vi.fn() }));

/**
 * Der Beschädigungsgrad im Status-Editor (Migration 127, Anforderungsliste Nr. 2).
 *
 * Die eine Regel, die hier hängt: Das Feld darf nur mitreisen, wenn sein Wert BEKANNT
 * ist oder ein Mensch etwas eingetragen hat. Schickte der Editor immer eine 0 mit,
 * löschte jedes Speichern an einem Exemplar, dessen Grad die Antwort nicht enthielt,
 * einen erfassten Wasserschaden — und der Ersatzbetrag stiege beim nächsten Verlust
 * still auf den vollen Zeitwert (Bugklasse Upsert-Blanking).
 */

const ERFOLG = /** @type {any} */ ({ ok: true, json: async () => ({}) });

function feldWert(screen) {
	return screen.getByLabelText(/Wertverlust durch Beschädigung/);
}

beforeEach(() => {
	vi.mocked(apiClient.put).mockReset().mockResolvedValue(ERFOLG);
});

describe('Beschädigungsgrad im Status-Editor', () => {
	it('schickt den eingetragenen Grad mit', async () => {
		const ex = {
			id: 'ex-1',
			ist_ausleihbar: true,
			zustand_notiz: '',
			zustand_abwertung_prozent: 0
		};
		const screen = render(BookExemplarStatusEditor, { props: { ex, onDone: () => {} } });

		await fireEvent.input(feldWert(screen), { target: { value: '20' } });
		await fireEvent.click(screen.getByRole('button', { name: 'Speichern' }));

		expect(apiClient.put).toHaveBeenCalledWith(
			'/api/buecher/exemplare/ex-1/status',
			expect.objectContaining({ zustand_abwertung_prozent: 20 })
		);
	});

	it('lässt das Feld weg, wenn der Grad unbekannt ist und niemand etwas eingetragen hat', async () => {
		// Kein zustand_abwertung_prozent in der Antwort: ein alter Stand oder eine andere
		// Lese-Tür. Dann darf der Editor den Wert am Server nicht anfassen.
		const ex = { id: 'ex-2', ist_ausleihbar: false, zustand_notiz: 'gesperrt' };
		const screen = render(BookExemplarStatusEditor, { props: { ex, onDone: () => {} } });

		await fireEvent.click(screen.getByRole('button', { name: 'Speichern' }));

		const [, koerper] = vi.mocked(apiClient.put).mock.calls[0];
		expect(koerper).not.toHaveProperty('zustand_abwertung_prozent');
	});

	it('schickt eine 0 mit, wenn der Grad bekannt ist — Zurücknehmen muss möglich sein', async () => {
		const ex = {
			id: 'ex-3',
			ist_ausleihbar: true,
			zustand_notiz: '',
			zustand_abwertung_prozent: 30
		};
		const screen = render(BookExemplarStatusEditor, { props: { ex, onDone: () => {} } });

		await fireEvent.input(feldWert(screen), { target: { value: '0' } });
		await fireEvent.click(screen.getByRole('button', { name: 'Speichern' }));

		expect(apiClient.put).toHaveBeenCalledWith(
			'/api/buecher/exemplare/ex-3/status',
			expect.objectContaining({ zustand_abwertung_prozent: 0 })
		);
	});

	it('zeigt das Feld auch bei „Verfügbar" — der Abschlag überlebt den Status', () => {
		const ex = {
			id: 'ex-4',
			ist_ausleihbar: true,
			zustand_notiz: '',
			zustand_abwertung_prozent: 20
		};
		const screen = render(BookExemplarStatusEditor, { props: { ex, onDone: () => {} } });

		// Die Notiz ist bei „Verfügbar" ausgeblendet, der Wertverlust nicht.
		expect(screen.queryByLabelText('Notiz zum Status')).toBeNull();
		expect(feldWert(screen).value).toBe('20');
	});
});
