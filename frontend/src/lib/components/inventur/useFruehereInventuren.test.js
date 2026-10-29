import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('./inventurApi.js', () => ({
	ladeAbgeschlosseneInventuren: vi.fn(),
	loescheAbgeschlosseneInventur: vi.fn()
}));
vi.mock('../../stores/snackbars.svelte.js', () => ({ snackbars: { addToast: vi.fn() } }));

import { ladeAbgeschlosseneInventuren, loescheAbgeschlosseneInventur } from './inventurApi.js';
import { snackbars } from '../../stores/snackbars.svelte.js';
import { useFruehereInventuren } from './useFruehereInventuren.svelte.js';

const lauf = (/** @type {string} */ id) => ({ session_id: id, label: `Lauf ${id}` });

// Die Liste wächst seitenweise: Die nächste Seite hängt sich an, mit dem Zeichen der vorigen.
describe('Frühere Inventuren: Mehr laden', () => {
	beforeEach(() => vi.clearAllMocks());

	it('hängt die nächste Seite an und fragt mit dem Zeichen der vorigen', async () => {
		vi.mocked(ladeAbgeschlosseneInventuren)
			.mockResolvedValueOnce({ laeufe: [lauf('a'), lauf('b')], weitere: 3, naechste_seite: 'z1' })
			.mockResolvedValueOnce({ laeufe: [lauf('c')], weitere: 0, naechste_seite: '' });
		const frueher = useFruehereInventuren({ frueher: '' });

		await frueher.lade();
		expect(frueher.weitere).toBe(3);
		await frueher.ladeMehr();

		expect(vi.mocked(ladeAbgeschlosseneInventuren).mock.calls).toEqual([[], ['z1']]);
		expect(frueher.laeufe.map((l) => l.session_id)).toEqual(['a', 'b', 'c']);
		expect(frueher.weitere).toBe(0);
		// Ohne Zeichen gibt es keine nächste Seite, und es geht keine Anfrage hinaus.
		await frueher.ladeMehr();
		expect(ladeAbgeschlosseneInventuren).toHaveBeenCalledTimes(2);
	});

	it('meldet eine gescheiterte Seite und behält die Liste', async () => {
		vi.mocked(ladeAbgeschlosseneInventuren)
			.mockResolvedValueOnce({ laeufe: [lauf('a')], weitere: 1, naechste_seite: 'z1' })
			.mockRejectedValueOnce(new Error('Die Datenbank antwortet nicht'));
		const frueher = useFruehereInventuren({ frueher: '' });

		await frueher.lade();
		await frueher.ladeMehr();

		expect(snackbars.addToast).toHaveBeenCalledWith('Die Datenbank antwortet nicht', 'error');
		expect(frueher.laeufe.map((l) => l.session_id)).toEqual(['a']);
		expect(frueher.weitere).toBe(1);
		expect(frueher.laedtMehr).toBe(false);
	});
});

// Eine gelöschte Inventur verschwindet aus der Liste und schließt ihren Bericht. Lehnt der Server
// ab, bleibt sie stehen; kennt er sie nicht mehr, ist sie schon weg.
describe('Frühere Inventuren: löschen', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		vi.mocked(ladeAbgeschlosseneInventuren).mockResolvedValue({
			laeufe: [lauf('a'), lauf('b')],
			weitere: 0,
			naechste_seite: ''
		});
	});

	it.each([
		['gelöscht', { ok: true, status: 204, data: {} }, 'Inventur „Lauf a“ gelöscht.', 'success'],
		[
			'schon weg',
			{
				ok: false,
				status: 404,
				error: 'Keine abgeschlossene Inventur mit dieser Kennung',
				data: {}
			},
			'Keine abgeschlossene Inventur mit dieser Kennung',
			'warning'
		]
	])('nimmt die Inventur aus der Liste, wenn sie %s ist', async (_name, antwort, meldung, art) => {
		vi.mocked(loescheAbgeschlosseneInventur).mockResolvedValue(antwort);
		const nachLoeschen = vi.fn();
		const frueher = useFruehereInventuren({ frueher: '' }, nachLoeschen);
		await frueher.lade();

		await frueher.loesche(lauf('a'));

		expect(loescheAbgeschlosseneInventur).toHaveBeenCalledWith('a');
		expect(frueher.laeufe.map((l) => l.session_id)).toEqual(['b']);
		expect(nachLoeschen).toHaveBeenCalledWith('a');
		expect(snackbars.addToast).toHaveBeenCalledWith(meldung, art);
	});

	it('lässt die Inventur stehen, wenn der Server ablehnt', async () => {
		vi.mocked(loescheAbgeschlosseneInventur).mockResolvedValue({
			ok: false,
			status: 403,
			error: 'Keine Berechtigung für diese Aktion',
			data: {}
		});
		const nachLoeschen = vi.fn();
		const frueher = useFruehereInventuren({ frueher: '' }, nachLoeschen);
		await frueher.lade();

		await frueher.loesche(lauf('a'));

		expect(frueher.laeufe.map((l) => l.session_id)).toEqual(['a', 'b']);
		expect(nachLoeschen).not.toHaveBeenCalled();
		expect(snackbars.addToast).toHaveBeenCalledWith('Keine Berechtigung für diese Aktion', 'error');
	});
});
