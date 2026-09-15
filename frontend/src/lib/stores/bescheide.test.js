import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('./toastStore.svelte.js', () => ({ toastStore: { addToast: vi.fn() } }));
vi.mock('../apiFetch.js', () => ({ apiGet: vi.fn(), apiPost: vi.fn() }));

import { apiGet } from '../apiFetch.js';
import { bescheideStore } from './bescheide.svelte.js';

const apiGetMock = vi.mocked(apiGet);

// Die Zahl am Reiter „Schadensersatz" ist die Menge dessen, was bei der Schule liegt:
// Kinder mit Forderung ohne Brief, Briefe mit laufender oder abgelaufener Frist, Rückgaben
// nach der Übergabe. Bis zum 15.09.2026 zählte sie nur die abgelaufenen Fristen — ein
// frischer Brief stand mit „0" am Reiter, obwohl eine Zeile darin wartete.
describe('bescheideStore', () => {
	beforeEach(() => {
		apiGetMock.mockReset();
	});

	it('lädt Briefe und Forderungen ohne Brief und zählt, was bei der Schule liegt', async () => {
		apiGetMock.mockImplementation(async (url) => {
			if (url === '/api/bescheide') {
				return [
					{ id: 'laeuft', status: 'offen', frist_abgelaufen: false },
					{ id: 'abgelaufen', status: 'offen', frist_abgelaufen: true },
					{ id: 'weg', status: 'uebergeben' },
					{ id: 'zurueck', status: 'uebergeben', rueckgabe_nach_uebergabe: true },
					{ id: 'bezahlt', status: 'erledigt' }
				];
			}
			if (url === '/api/bescheide/ausstehend')
				return [{ schueler_id: 's1' }, { schueler_id: 's2' }];
			throw new Error('unerwartete URL ' + url);
		});

		await bescheideStore.lade();

		expect(bescheideStore.geladen).toBe(true);
		expect(bescheideStore.ausstehend).toHaveLength(2);
		// 2 ohne Brief + laeuft + abgelaufen + zurueck; weg und bezahlt zählen nicht.
		expect(bescheideStore.beiDerSchule).toBe(5);
		// Der Reiter zeigt alles außer erledigt — der bezahlte Brief bleibt in der Akte.
		expect(bescheideStore.zeilen.map((b) => b.id)).toEqual([
			'laeuft',
			'abgelaufen',
			'weg',
			'zurueck'
		]);
	});

	it('behält die alten Listen, wenn ein Abruf scheitert — statt leer zu behaupten', async () => {
		apiGetMock.mockRejectedValue(new Error('503'));
		await bescheideStore.lade();
		expect(bescheideStore.ausstehend).toHaveLength(2);
		expect(bescheideStore.beiDerSchule).toBe(5);
	});
});
