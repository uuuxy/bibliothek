import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('./snackbars.svelte.js', () => ({ snackbars: { addToast: vi.fn() } }));
vi.mock('../apiFetch.js', () => ({ apiGet: vi.fn(), apiPost: vi.fn() }));

import { apiGet, apiPost } from '../apiFetch.js';
import { bescheideStore } from './bescheide.svelte.js';

const apiGetMock = vi.mocked(apiGet);
const apiPostMock = vi.mocked(apiPost);

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
		apiGetMock.mockImplementation(async (url) =>
			url === '/api/bescheide/ausstehend'
				? [{ schueler_id: 's1' }, { schueler_id: 's2' }]
				: [{ id: 'laeuft', status: 'offen', frist_abgelaufen: false }]
		);
		await bescheideStore.lade();
		expect(bescheideStore.beiDerSchule).toBe(3);

		apiGetMock.mockRejectedValue(new Error('503'));
		await bescheideStore.lade();

		expect(bescheideStore.ausstehend).toHaveLength(2);
		expect(bescheideStore.beiDerSchule).toBe(3);
	});

	it('merkt sich den Grund, wenn ein Abruf scheitert, und vergisst ihn beim nächsten Erfolg', async () => {
		apiGetMock.mockRejectedValue(new Error('Die Datenbank antwortet nicht'));
		await bescheideStore.lade();
		expect(bescheideStore.ladefehler).toBe('Die Datenbank antwortet nicht');

		apiGetMock.mockResolvedValue([]);
		await bescheideStore.lade();
		expect(bescheideStore.ladefehler).toBe('');
	});

	// Geladen wird bei jedem Besuch und nach jeder Übergabe. Die Antwort auf den älteren Abruf
	// kann zuletzt ankommen; dann zählt sie nicht mehr.
	it('lässt eine ältere Antwort, die zuletzt kommt, den jüngeren Stand nicht überschreiben', async () => {
		/** @type {((liste: unknown[]) => void)[]} */
		const antworten = [];
		apiGetMock.mockImplementation((url) =>
			url === '/api/bescheide'
				? new Promise((fertig) => antworten.push(fertig))
				: Promise.resolve([])
		);

		const aelter = bescheideStore.lade();
		const juenger = bescheideStore.lade();
		antworten[1]([{ id: 'b1', status: 'uebergeben' }]);
		await juenger;
		antworten[0]([{ id: 'b1', status: 'offen', frist_abgelaufen: true }]);
		await aelter;

		expect(bescheideStore.liste).toEqual([{ id: 'b1', status: 'uebergeben' }]);
		expect(bescheideStore.laedt).toBe(false);
	});

	// Ein Doppelklick auf „Übergeben" schickte zwei Anfragen. Der Server bucht nur die erste, auf
	// die zweite antwortet er „Übergabe nicht möglich", und die Meldung stand neben „gebucht".
	it('schickt die Übergabe bei einem zweiten Klick vor der Antwort nicht noch einmal', async () => {
		/** @type {(wert: unknown) => void} */
		let antworte = () => {};
		apiPostMock.mockReset();
		apiPostMock.mockImplementation(() => new Promise((fertig) => (antworte = fertig)));
		apiGetMock.mockResolvedValue([]);

		const erster = bescheideStore.uebergebe('b1');
		const zweiter = bescheideStore.uebergebe('b1');
		expect(apiPostMock).toHaveBeenCalledTimes(1);
		expect(bescheideStore.uebergibt).toBe('b1');

		antworte({ id: 'b1', status: 'uebergeben' });
		await Promise.all([erster, zweiter]);
		expect(bescheideStore.uebergibt).toBe('');
	});

	it('gibt den Knopf auch nach einer Ablehnung wieder frei', async () => {
		apiPostMock.mockReset();
		apiPostMock.mockRejectedValue(
			Object.assign(new Error('Übergabe nicht möglich'), { handled: true })
		);
		await bescheideStore.uebergebe('b1');
		expect(bescheideStore.uebergibt).toBe('');
		apiPostMock.mockResolvedValue({ id: 'b2', status: 'uebergeben' });
		await bescheideStore.uebergebe('b2');
		expect(apiPostMock).toHaveBeenLastCalledWith('/api/bescheide/b2/uebergeben', {});
	});
});
