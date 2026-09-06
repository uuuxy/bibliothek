import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/svelte';
import DeletedStudentList from './DeletedStudentList.svelte';
import { apiFetch } from '../../apiFetch.js';
import { toastStore } from '../../stores/toastStore.svelte.js';

vi.mock('../../apiFetch.js', () => ({ apiFetch: vi.fn() }));
vi.mock('../../stores/toastStore.svelte.js', () => ({
	toastStore: { addToast: vi.fn() }
}));

/**
 * Der Papierkorb, Rasterdurchgang 06.09.2026 (Frage 5, stille Fehler; Frage 10, Rückweg).
 *
 * Drei Zusicherungen, die vorher keine waren:
 *  1. An einer anonymisierten Zeile steht KEIN Wiederherstellen-Knopf. Der Server
 *     antwortet dort mit 409; ein Knopf, der nur scheitern kann, ist eine Falle.
 *  2. Ein gescheiterter Abruf zeigt den Fehler — nicht „Der Papierkorb ist leer".
 *     Diese Zeile war eine falsche Auskunft über gelöschte Schülerdaten.
 *  3. Ein abgelehntes Wiederherstellen sagt, warum. Vorher passierte NICHTS.
 */

const NORMAL = {
	id: 's-1',
	barcode_id: 'B-1',
	vorname: 'Anna',
	nachname: 'Muster',
	klasse: '07H1',
	deleted_at: '2026-09-01T10:00:00Z',
	anonymized_at: null
};

const ANONYM = {
	id: 's-2',
	barcode_id: 'ANON-s-2',
	vorname: 'a1b2c3d4',
	nachname: 'Anonym',
	klasse: '',
	deleted_at: '2026-01-05T10:00:00Z',
	anonymized_at: '2026-07-04T02:00:00Z'
};

/** @param {any} body @param {{ok?: boolean, status?: number}} opts */
function antwort(body, opts = {}) {
	const ok = opts.ok ?? true;
	return {
		ok,
		status: opts.status ?? (ok ? 200 : 409),
		json: async () => body,
		text: async () => (typeof body === 'string' ? body : JSON.stringify(body))
	};
}

beforeEach(() => {
	vi.mocked(apiFetch).mockReset();
	vi.mocked(toastStore.addToast).mockReset();
});

describe('DeletedStudentList', () => {
	it('bietet Wiederherstellen nur an, wo es gehen kann', async () => {
		vi.mocked(apiFetch).mockResolvedValue(/** @type {any} */ (antwort([NORMAL, ANONYM])));
		render(DeletedStudentList, { props: {} });

		await waitFor(() => expect(screen.getAllByRole('row')).toHaveLength(3)); // Kopf + 2

		// Genau EIN Knopf für zwei Zeilen — der der wiederherstellbaren.
		const knoepfe = screen.getAllByRole('button', { name: 'Wiederherstellen' });
		expect(knoepfe).toHaveLength(1);

		// Und die andere Zeile sagt, warum sie keinen hat.
		expect(screen.getByText(/anonymisiert \(DSGVO\)/i)).toBeTruthy();
	});

	it('nennt einen Ladefehler einen Ladefehler, nicht einen leeren Papierkorb', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ (
				antwort({ error: 'Datenbank nicht erreichbar' }, { ok: false, status: 500 })
			)
		);
		render(DeletedStudentList, { props: {} });

		await waitFor(() => expect(screen.getByText('Datenbank nicht erreichbar')).toBeTruthy());
		expect(screen.queryByText('Der Papierkorb ist leer.')).toBeNull();
	});

	it('meldet ein abgelehntes Wiederherstellen, statt es zu verschlucken', async () => {
		vi.mocked(apiFetch).mockImplementation(async (_url, opts) => {
			if (opts?.method === 'POST') {
				return /** @type {any} */ (
					antwort({ error: 'dieser Datensatz ist bereits anonymisiert (DSGVO)' }, { ok: false })
				);
			}
			return /** @type {any} */ (antwort([NORMAL]));
		});
		render(DeletedStudentList, { props: {} });

		const knopf = await screen.findByRole('button', { name: 'Wiederherstellen' });
		knopf.click();

		await waitFor(() =>
			expect(vi.mocked(toastStore.addToast)).toHaveBeenCalledWith(
				'dieser Datensatz ist bereits anonymisiert (DSGVO)',
				'error'
			)
		);
	});
});
