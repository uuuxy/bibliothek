import { describe, it, expect, vi, beforeEach } from 'vitest';
import { erzeugeReservierungsListen } from './klassensatzReservierung.svelte.js';
import { apiFetch } from '../../apiFetch.js';

vi.mock('../../apiFetch.js', () => ({ apiFetch: vi.fn() }));

const OFFEN = '/api/reservierungen/klassensatz/offen';
const EIGENE = '/api/reservierungen/klassensatz/eigene';

/** Lässt die beiden Abrufe von `lade()` auslaufen; die Methode wartet nicht auf sie. */
const ausgelaufen = () => new Promise((fertig) => setTimeout(fertig));

/**
 * Antwortet je Tür mit dem gegebenen Körper.
 * @param {Record<string, any>} jeTuer Pfad → Antwort von apiFetch
 */
function antworte(jeTuer) {
	vi.mocked(apiFetch).mockImplementation(async (pfad) => jeTuer[String(pfad)]);
}

/** @param {any} daten */
const ok = (daten) => ({ ok: true, json: async () => daten });

describe('Reservierungs-Listen des Portals', () => {
	// Mit Block: Eine Funktion als Rückgabe riefe Vitest nach dem Test als Aufräumer auf, und
	// mockReset gibt die Attrappe selbst zurück.
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
	});

	it('lädt die Warteschlange und die eigenen Reservierungen, jede von ihrer Tür', async () => {
		antworte({
			[OFFEN]: ok([
				{ titel_id: 't1', klasse: '07A' },
				{ titel_id: 't2', klasse: '08B' }
			]),
			[EIGENE]: ok([{ titel_id: 't2', klasse: '08B' }])
		});
		const listen = erzeugeReservierungsListen();

		listen.lade();
		await ausgelaufen();

		expect(listen.offene).toHaveLength(2);
		expect(listen.eigene).toEqual([{ titel_id: 't2', klasse: '08B' }]);
		expect(listen.warteschlangeFuer('t1')).toEqual([{ titel_id: 't1', klasse: '07A' }]);
	});

	// Die Listen sind Zusatz: Was keine Liste ist oder nicht ankommt, lässt den Stand stehen,
	// statt die Anzeige beim Filtern abstürzen zu lassen.
	it.each([
		['eine abgelehnte Antwort', { ok: false, status: 403, json: async () => [{ titel_id: 'x' }] }],
		['ein Körper, der keine Liste ist', ok({ error: 'kaputt' })]
	])('übernimmt %s nicht', async (_was, antwort) => {
		antworte({ [OFFEN]: ok([{ titel_id: 't1', klasse: '07A' }]), [EIGENE]: ok([]) });
		const listen = erzeugeReservierungsListen();
		listen.lade();
		await ausgelaufen();

		antworte({ [OFFEN]: antwort, [EIGENE]: antwort });
		listen.lade();
		await ausgelaufen();

		expect(listen.offene).toEqual([{ titel_id: 't1', klasse: '07A' }]);
		expect(listen.eigene).toEqual([]);
	});

	it('bleibt bei einem Netzfehler benutzbar', async () => {
		vi.mocked(apiFetch).mockRejectedValue(new Error('Netz weg'));
		const listen = erzeugeReservierungsListen();

		listen.lade();
		await ausgelaufen();

		expect(listen.offene).toEqual([]);
		expect(listen.warteschlangeFuer('t1')).toEqual([]);
	});
});
