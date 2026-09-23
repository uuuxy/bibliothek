import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/svelte';

vi.mock('../../../../lib/apiFetch.js', () => ({ apiFetch: vi.fn() }));
vi.mock('../../../../lib/stores/bestaetigung.svelte.js', () => ({
	loeschenBestaetigen: vi.fn()
}));
vi.mock('$lib/store.svelte.js', () => ({ showToast: vi.fn() }));

import { apiFetch } from '../../../../lib/apiFetch.js';
import BuchExemplareListe from './BuchExemplareListe.svelte';

// GET /api/buecher/titel/{id}/exemplare antwortet mit dem nackten Array (RespondJSON,
// api/copy_admin.go) — so lesen es die Buchakte (useBookAkte) und das Druck-Center
// (labels.svelte.js). Die Maske „Buch bearbeiten" las seit Juni 2026 `json.data`, bekam
// undefined und zeigte bei jedem Titel „Exemplare (0)" und „Keine Exemplare in der
// Datenbank vorhanden" — auch bei einem Titel mit 30 Exemplaren.
describe('BuchExemplareListe', () => {
	it('zeigt die Exemplare, die der Server liefert', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				json: async () => [
					{ id: 'e1', barcode_id: 'B-000101', ist_ausleihbar: true, ist_verfuegbar: true },
					{ id: 'e2', barcode_id: 'B-000102', ist_ausleihbar: false, ist_verfuegbar: true }
				]
			})
		);
		const screen = render(BuchExemplareListe, { formular: { id: 't-1', stock: 2 } });

		expect(await screen.findByText('B-000101')).toBeTruthy();
		expect(screen.getByText('B-000102')).toBeTruthy();
		expect(screen.getByRole('heading', { name: 'Exemplare (2)' })).toBeTruthy();
		expect(apiFetch).toHaveBeenCalledWith('/api/buecher/titel/t-1/exemplare', expect.anything());
	});
});
