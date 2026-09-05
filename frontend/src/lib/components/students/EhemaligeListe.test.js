import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import EhemaligeListe from './EhemaligeListe.svelte';
import { apiFetch } from '../../apiFetch.js';

vi.mock('../../apiFetch.js', () => ({ apiFetch: vi.fn() }));

// Der Reiter „Ehemalige / Archiv" muss die WEGGEGANGENEN laden (status=ehemalige) —
// nicht die Abgängerliste, die er bis zum 05.09.2026 eingebettet hatte und die seitdem
// die Abschlussklassen meint. Geprüft wird die Tür, durch die er geht, und was er zeigt.
const ehemalige = [
	{
		id: 'e1',
		vorname: 'Wera',
		nachname: 'Weggegangen',
		barcode_id: 'S-EHEM-1',
		abgaenger_jahr: 2026,
		ausgeliehen_count: 1,
		ueberfaellig_count: 1,
		ist_gesperrt: true
	}
];

/** Antwort wie apiFetch sie liefert — nur das, was die Komponente liest.
 *  @param {any[]} daten @returns {any} */
const antwort = (daten) => ({ ok: true, json: async () => daten });

describe('EhemaligeListe', () => {
	beforeEach(() => vi.mocked(apiFetch).mockReset());

	it('lädt über /api/schueler?status=ehemalige und zeigt Abgang, Name, offene Bücher', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort(ehemalige));
		const { findByRole, getByText } = render(EhemaligeListe, { onSelect: () => {} });

		expect(
			await findByRole('button', { name: 'Profil von Wera Weggegangen anzeigen' })
		).toBeTruthy();
		expect(vi.mocked(apiFetch).mock.calls[0][0]).toBe('/api/schueler?status=ehemalige');
		expect(getByText('2026')).toBeTruthy();
		expect(getByText(/1 überfällig/)).toBeTruthy();
		expect(getByText('Sperre aktiv')).toBeTruthy();
	});

	it('Klick auf die Zeile öffnet das Profil', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort(ehemalige));
		const onSelect = vi.fn();
		const { findByRole } = render(EhemaligeListe, { onSelect });
		await fireEvent.click(await findByRole('button', { name: /Wera Weggegangen/ }));
		expect(onSelect).toHaveBeenCalledWith(ehemalige[0]);
	});

	it('Suche läuft auf dem Server mit demselben Status', async () => {
		vi.mocked(apiFetch).mockResolvedValue(antwort([]));
		const { getByRole, findByText } = render(EhemaligeListe, { onSelect: () => {} });
		await findByText('Keine Ehemaligen im Archiv.');

		await fireEvent.input(getByRole('searchbox', { name: 'Ehemalige suchen' }), {
			target: { value: 'Wera' }
		});
		await findByText('Keine Ehemaligen gefunden.');
		// Die Suche ist entprellt (250 ms) — auf den Aufruf warten, nicht auf den Text.
		await vi.waitFor(() => {
			const urls = vi.mocked(apiFetch).mock.calls.map((c) => c[0]);
			expect(urls).toContain('/api/schueler?status=ehemalige&q=Wera');
		});
	});
});
