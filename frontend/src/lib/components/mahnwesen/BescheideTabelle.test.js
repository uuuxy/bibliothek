import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', () => ({ apiGet: vi.fn(), apiPost: vi.fn() }));

import { bescheideStore } from '../../stores/bescheide.svelte.js';
import BescheideTabelle from './BescheideTabelle.svelte';

// Die Stufe zwischen Mahnliste und Bescheid: „Verlust melden" beendet die Ausleihe, das
// Kind fällt aus der Mahnliste — und stand bis zum 15.09.2026 nirgends mehr. Der Reiter
// „Schadensersatz" gibt ihr eine Zeile mit dem einen nächsten Schritt.
describe('BescheideTabelle', () => {
	it('zeigt die Forderung ohne Brief mit „Bescheid erstellen" und ruft den Dialog für das Kind', async () => {
		bescheideStore.ausstehend = [
			{
				schueler_id: 's1',
				schueler_name: 'Mira Muster',
				klasse: '07B',
				anzahl: 2,
				summe: 20,
				seit: '2026-09-01T00:00:00Z',
				lernmittel: true
			}
		];
		bescheideStore.liste = [];
		const onBescheid = vi.fn();
		const screen = render(BescheideTabelle, { darfSchreiben: true, onBescheid });

		expect(screen.getByText('Bescheid noch nicht erstellt')).toBeTruthy();
		expect(screen.getByText(/2 Forderungen/)).toBeTruthy();
		await screen.getByRole('button', { name: 'Bescheid erstellen' }).click();
		expect(onBescheid).toHaveBeenCalledWith('s1');
	});

	it('bietet für ein Bücherei-Buch keinen Bescheid an', () => {
		bescheideStore.ausstehend = [
			{
				schueler_id: 's2',
				schueler_name: 'Ben Beispiel',
				klasse: '',
				anzahl: 1,
				summe: 5,
				seit: '2026-09-01T00:00:00Z',
				lernmittel: false
			}
		];
		bescheideStore.liste = [];
		const screen = render(BescheideTabelle, { darfSchreiben: true, onBescheid: vi.fn() });
		expect(screen.getByText('Bücherei-Buch, kein Bescheid')).toBeTruthy();
		expect(screen.queryByRole('button', { name: 'Bescheid erstellen' })).toBeNull();
	});

	it('lässt erledigte Briefe weg und zeigt „Übergeben" nur nach Fristablauf', () => {
		bescheideStore.ausstehend = [];
		bescheideStore.liste = [
			{
				id: 'a',
				referenznummer: 'R-1',
				status: 'offen',
				frist_abgelaufen: true,
				frist_bis: '2026-08-01',
				brief_datum: '2026-07-01',
				gesamtbetrag: 10,
				anzahl_positionen: 1,
				schueler_name: 'A'
			},
			{
				id: 'b',
				referenznummer: 'R-2',
				status: 'offen',
				frist_abgelaufen: false,
				frist_bis: '2026-12-01',
				brief_datum: '2026-11-01',
				gesamtbetrag: 10,
				anzahl_positionen: 1,
				schueler_name: 'B'
			},
			{
				id: 'c',
				referenznummer: 'R-3',
				status: 'erledigt',
				frist_bis: '2026-08-01',
				brief_datum: '2026-07-01',
				gesamtbetrag: 10,
				anzahl_positionen: 1,
				schueler_name: 'C'
			}
		];
		const screen = render(BescheideTabelle, { darfSchreiben: true, onBescheid: vi.fn() });
		expect(screen.getByText('R-1')).toBeTruthy();
		expect(screen.getByText('R-2')).toBeTruthy();
		expect(screen.queryByText('R-3')).toBeNull();
		expect(screen.getAllByRole('button', { name: 'Übergeben' })).toHaveLength(1);
		expect(screen.getByText('Frist läuft')).toBeTruthy();
	});
});
