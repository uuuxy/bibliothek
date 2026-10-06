import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import BookExemplarStatusEditor from './BookExemplarStatusEditor.svelte';
import { apiClient } from '../apiFetch.js';

vi.mock('../apiFetch.js', () => ({
	apiClient: { put: vi.fn() },
	apiFetch: vi.fn()
}));

vi.mock('../../inventur/lib/store.svelte.js', () => ({ showToast: vi.fn() }));

// Der Status, mit dem der Editor öffnet, kommt aus den zwei Merkmalen des Exemplars
// (ausleihbar, ausgesondert) und nicht aus dem Wortlaut der Notiz: Wer an einem gesperrten
// Exemplar nur die Notiz oder den Wertverlust ändert, sondert es sonst aus.

const ERFOLG = /** @type {any} */ ({ ok: true, json: async () => ({}) });

/** @param {any} ex */
async function oeffneUndSpeichere(ex) {
	const screen = render(BookExemplarStatusEditor, { props: { ex, onDone: () => {} } });
	// ui/Select ist ein Knopf mit der Rolle combobox; er zeigt die gewählte Zeile als Text.
	const gewaehlt = (
		screen.getByRole('combobox', { name: 'Status des Exemplars' }).textContent || ''
	).trim();
	await fireEvent.click(screen.getByRole('button', { name: 'Speichern' }));
	const [, koerper] = vi.mocked(apiClient.put).mock.calls[0];
	return { gewaehlt, koerper };
}

beforeEach(() => {
	vi.mocked(apiClient.put).mockReset().mockResolvedValue(ERFOLG);
});

describe('Status-Editor: der Status beim Öffnen', () => {
	it('lässt ein gesperrtes Exemplar gesperrt, auch wenn die Notiz „verloren“ enthält', async () => {
		const { gewaehlt, koerper } = await oeffneUndSpeichere({
			id: 'ex-1',
			ist_ausleihbar: false,
			ist_ausgesondert: false,
			zustand_notiz: 'CD verloren, Buch vollständig'
		});

		expect(koerper).toMatchObject({
			ist_ausleihbar: false,
			ist_ausgesondert: false,
			zustand_notiz: 'CD verloren, Buch vollständig'
		});
		expect(gewaehlt).toBe('Gesperrt (Defekt/Reserviert)');
	});

	it('öffnet ein ausgesondertes Exemplar als „Verloren“ und lässt es ausgesondert', async () => {
		const { gewaehlt, koerper } = await oeffneUndSpeichere({
			id: 'ex-2',
			ist_ausleihbar: false,
			ist_ausgesondert: true,
			zustand_notiz: 'bei der Inventur nicht gefunden'
		});

		expect(gewaehlt).toBe('Verloren');
		expect(koerper).toMatchObject({ ist_ausleihbar: false, ist_ausgesondert: true });
	});

	it('öffnet ein ausleihbares Exemplar als „Verfügbar“', async () => {
		const { gewaehlt, koerper } = await oeffneUndSpeichere({
			id: 'ex-3',
			ist_ausleihbar: true,
			ist_ausgesondert: false,
			zustand_notiz: ''
		});

		expect(gewaehlt).toBe('Verfügbar');
		expect(koerper).toMatchObject({ ist_ausleihbar: true, ist_ausgesondert: false });
	});
});
