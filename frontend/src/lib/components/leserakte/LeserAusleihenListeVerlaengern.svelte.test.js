import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));
vi.mock('../../../inventur/lib/store.svelte.js', () => ({ melde: vi.fn() }));

import { apiFetch } from '../../apiFetch.js';
import { melde } from '../../../inventur/lib/store.svelte.js';
import { serverAntwort } from '../../serverAntwort.js';
import LeserAusleihenListe from './LeserAusleihenListe.svelte';

// Ging die Antwort auf „Verlängern" verloren, verlängerte der zweite Klick noch einmal: Aus 21
// Tagen wurden 42 (Probe in Chromium, Firefox und WebKit, OFFEN.md 5.106). Die Liste schickt die
// Frist mit, die sie zeigt; steht am Server schon eine andere, kommt die jetzige in die Zeile.
describe('Ausleihliste: Verlängern', () => {
	const buch = () => ({
		id: 'ex-1',
		ausleihe_id: 'a-1',
		barcode_id: 'B-1',
		titel: 'Der Zauberberg',
		autor: 'Mann',
		ist_dauerleihe: false,
		ausgeliehen_am: '2026-09-01T10:00:00Z',
		rueckgabe_frist: '2026-09-28T11:00:00Z'
	});

	// Wie im Programm ein tiefes $state (useLeserakte, akte): Nur dann zeigt die Zeile ein neues Datum.
	const beobachtet = () => {
		const ausleihen = $state([buch()]);
		return ausleihen;
	};

	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
		vi.mocked(melde).mockReset();
	});

	it('schickt die gezeigte Frist mit', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			serverAntwort(200, { neues_rueckgabe_datum: '2026-10-19T11:00:00Z' })
		);
		const screen = render(LeserAusleihenListe, { ausleihen: beobachtet() });

		await fireEvent.click(screen.getByRole('button', { name: 'Ausleihe verlängern' }));

		await waitFor(() => expect(apiFetch).toHaveBeenCalledTimes(1));
		const optionen = /** @type {any} */ (vi.mocked(apiFetch).mock.calls[0][1]);
		expect(JSON.parse(optionen.body)).toEqual({ frist_bisher: '2026-09-28T11:00:00Z' });
	});

	it('zeigt nach 409 die jetzige Frist in der Zeile und nennt sie', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			serverAntwort(409, {
				error: 'Die Frist steht schon auf dem 18.11.2026.',
				rueckgabe_frist: '2026-11-18T11:00:00Z'
			})
		);
		const screen = render(LeserAusleihenListe, { ausleihen: beobachtet() });

		await fireEvent.click(screen.getByRole('button', { name: 'Ausleihe verlängern' }));

		await waitFor(() =>
			expect(melde).toHaveBeenCalledWith('Die Frist steht schon auf dem 18.11.2026.', 'info')
		);
		expect(screen.container.textContent).toContain('18.11.2026');
	});
});
