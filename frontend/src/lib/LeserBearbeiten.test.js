import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

vi.mock('./apiFetch.js', () => ({
	apiFetch: vi.fn(async () => ({ ok: true, status: 200, json: async () => [] })),
	apiClient: { patch: vi.fn(async () => ({ ok: true, status: 200, json: async () => ({}) })) }
}));

import { apiClient } from './apiFetch.js';
import LeserBearbeiten from './LeserBearbeiten.svelte';
import { snackbars } from './stores/snackbars.svelte.js';
import { bestaetigungStore } from './stores/bestaetigung.svelte.js';
import { uiStore } from './stores/uiStore.svelte.js';

// „Änderungen gespeichert." erschien nie: Die Maske zeigte ihre Meldungen in einer eigenen
// Snackbar und schließt mit dem Speichern (Leserakte, `nachBearbeiten`), die Meldung mit ihr
// (docs/OFFEN.md 5.76).
const MIA = {
	id: 'leser-1',
	art: 'schueler',
	vorname: 'Mia',
	nachname: 'Muster',
	klasse: '07B',
	barcode_id: 'S-1'
};

afterEach(() => {
	for (const toast of [...snackbars.toasts]) snackbars.removeToast(toast.id);
	bestaetigungStore.antworten(null);
	vi.clearAllMocks();
});

describe('Stammdaten bearbeiten: die Meldung nach dem Speichern', () => {
	it('steht in der gemeinsamen Anzeige, auch wenn die Maske danach schließt', async () => {
		// Wie die Leserakte: Nach dem Speichern baut sie die Maske ab.
		const abbau = { maske: () => {} };
		const onSave = vi.fn(() => abbau.maske());
		const screen = render(LeserBearbeiten, { leser: MIA, onClose: () => {}, onSave });
		abbau.maske = () => screen.unmount();

		await fireEvent.input(screen.getByLabelText(/^Vorname/), { target: { value: 'Mila' } });
		await fireEvent.click(screen.getByRole('button', { name: 'Speichern' }));
		await vi.waitFor(() => expect(onSave).toHaveBeenCalledTimes(1));

		expect(apiClient.patch).toHaveBeenCalledWith('/api/schueler/leser-1', { vorname: 'Mila' });
		expect(snackbars.toasts.map((t) => [t.message, t.type])).toEqual([
			['Änderungen gespeichert.', 'success']
		]);
	});
});

// „Zurück" schloss die Maske samt allem Getippten, ebenso ein Wechsel der Seite (docs/OFFEN.md
// 5.80). Wie an der Buchmaske fragt sie jetzt vorher, und nur, wenn etwas ungespeichert ist.
describe('Stammdaten bearbeiten: Ungespeichertes geht nur nach Rückfrage verloren', () => {
	const antworteteMaske = () => new Promise((fertig) => setTimeout(fertig, 0));

	it('„Zurück" ohne Eingabe schließt ohne Frage', async () => {
		const onClose = vi.fn();
		const screen = render(LeserBearbeiten, { leser: MIA, onClose, onSave: () => {} });

		await fireEvent.click(screen.getByRole('button', { name: 'Zurück' }));

		expect(bestaetigungStore.anfrage).toBeNull();
		expect(onClose).toHaveBeenCalledTimes(1);
	});

	it('„Zurück" mit Eingabe fragt; „Bleiben" lässt sie stehen, „Verwerfen" schließt', async () => {
		const onClose = vi.fn();
		const screen = render(LeserBearbeiten, { leser: MIA, onClose, onSave: () => {} });
		const vorname = /** @type {HTMLInputElement} */ (screen.getByLabelText(/^Vorname/));
		await fireEvent.input(vorname, { target: { value: 'Mila' } });

		await fireEvent.click(screen.getByRole('button', { name: 'Zurück' }));
		expect(bestaetigungStore.anfrage).toMatchObject({
			titel: 'Ungespeicherte Änderungen',
			aktion: 'Verwerfen',
			abbruch: 'Bleiben'
		});
		bestaetigungStore.antworten(false);
		await antworteteMaske();
		expect(onClose).not.toHaveBeenCalled();
		expect(vorname.value).toBe('Mila');

		await fireEvent.click(screen.getByRole('button', { name: 'Zurück' }));
		bestaetigungStore.antworten(true);
		await vi.waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
		expect(apiClient.patch).not.toHaveBeenCalled();
	});

	it('hält mit Eingabe den Wechsel der Seite an und gibt den Wächter beim Abbau frei', async () => {
		const screen = render(LeserBearbeiten, { leser: MIA, onClose: () => {}, onSave: () => {} });
		expect(uiStore.verlassenSperre?.(), 'ohne Eingabe').toBe(false);

		await fireEvent.input(screen.getByLabelText(/^Vorname/), { target: { value: 'Mila' } });
		expect(uiStore.verlassenSperre?.()).toBe(true);

		screen.unmount();
		expect(uiStore.verlassenSperre).toBeNull();
	});
});
