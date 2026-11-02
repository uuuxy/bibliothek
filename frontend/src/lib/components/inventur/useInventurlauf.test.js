import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('./inventurApi.js', () => ({
	starteLauf: vi.fn(),
	ladeOffeneSessions: vi.fn(async () => []),
	brichAb: vi.fn()
}));
vi.mock('../../apiFetch.js', () => ({ apiFetch: vi.fn() }));

import { starteLauf } from './inventurApi.js';
import { useInventurlauf } from './useInventurlauf.svelte.js';

const GESTARTET = /** @type {any} */ ({
	ok: true,
	data: { session_id: 'lauf-1', erwartet: 3, fehlend: 3, label: 'Mathematik' }
});

/** @param {{ fach?: string, jahrgang?: string }} wahl */
function teilInventur({ fach = '', jahrgang = '' }) {
	const lauf = useInventurlauf();
	lauf.bereichArt = 'filter';
	lauf.gewaehltesFach = fach;
	lauf.gewaehlterJahrgang = jahrgang;
	return lauf;
}

// Eine Teil-Inventur („nur Mathe, Kl. 5") braucht ein Fach oder eine Klasse, nicht beides.
describe('Inventur starten: nach Fach oder Klasse', () => {
	beforeEach(() => vi.clearAllMocks());

	it('startet mit dem Fach allein', async () => {
		vi.mocked(starteLauf).mockResolvedValue(GESTARTET);
		const lauf = teilInventur({ fach: 'Mathematik' });

		await lauf.starteInventur();

		expect(starteLauf).toHaveBeenCalledWith({ type: 'filter', subject: 'Mathematik' });
		expect(lauf.status).toBe('active');
	});

	it('startet mit der Klasse allein', async () => {
		vi.mocked(starteLauf).mockResolvedValue(GESTARTET);
		const lauf = teilInventur({ jahrgang: '7' });

		await lauf.starteInventur();

		expect(starteLauf).toHaveBeenCalledWith({ type: 'filter', grade: 7 });
	});

	it('ohne beides: ein Hinweis, kein Start', async () => {
		const lauf = teilInventur({});

		await lauf.starteInventur();

		expect(starteLauf).not.toHaveBeenCalled();
		expect(lauf.fehler).toBe('Bitte wähle mindestens ein Fach oder eine Klasse.');
	});
});
