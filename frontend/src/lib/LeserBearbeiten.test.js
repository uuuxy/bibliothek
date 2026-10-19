import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

vi.mock('./apiFetch.js', () => ({
	apiFetch: vi.fn(async () => ({ ok: true, status: 200, json: async () => [] })),
	apiClient: { patch: vi.fn(async () => ({ ok: true, status: 200, json: async () => ({}) })) }
}));

import { apiClient } from './apiFetch.js';
import LeserBearbeiten from './LeserBearbeiten.svelte';
import { snackbars } from './stores/snackbars.svelte.js';

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
