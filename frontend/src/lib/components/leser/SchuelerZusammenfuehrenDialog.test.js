import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import SchuelerZusammenfuehrenDialog from './SchuelerZusammenfuehrenDialog.svelte';
import { apiFetch } from '../../apiFetch.js';

vi.mock('../../apiFetch.js', () => ({
	apiFetch: vi.fn(),
	apiClient: { post: vi.fn() },
	extractApiError: vi.fn(async (/** @type {any} */ res) => `Fehler ${res.status}`)
}));
vi.mock('../../stores/toastStore.svelte.js', () => ({ toastStore: { addToast: vi.fn() } }));

// Raster-Funde 02.09.2026 (Frontend-Prüfer, Funde 2 und 3) an der Kandidatensuche des
// Zusammenführen-Dialogs. Die Suche ist ein Schreibpfad in eine Liste, aus der ein
// UNUMKEHRBARER Schritt folgt — wer hier den falschen Kandidaten sieht, führt den
// falschen Datensatz zusammen. Deshalb gilt an der Liste, was an der Theke gilt:
// nur die jüngste Antwort schreibt, ein Fehler heißt „Fehler“ und nicht „nichts da“,
// und ein neu geöffneter Dialog zeigt nie die Liste des vorigen Schülers.

const PROFIL = { id: 'quelle', vorname: 'Lena', nachname: 'Alt', klasse: '07A', barcode_id: 'S-1' };
const KANDIDAT = (id, nachname) => ({
	id,
	vorname: 'Lena',
	nachname,
	klasse: '07A',
	barcode_id: `S-${id}`
});

/** Ein Versprechen, das der Test von außen einlöst — so bestimmt der Test die Antwort-Reihenfolge. */
function offen() {
	/** @type {(v: any) => void} */
	let einloesen = () => {};
	/** @type {(e: any) => void} */
	let verwerfen = () => {};
	const promise = new Promise((res, rej) => {
		einloesen = res;
		verwerfen = rej;
	});
	return { promise, einloesen, verwerfen };
}

const antwort = (liste) => ({ ok: true, status: 200, json: async () => liste });

/** Microtasks durchlaufen lassen (fetch → json → Zustand → DOM). */
const stillhalten = async () => {
	for (let i = 0; i < 5; i++) await Promise.resolve();
};

/** Tippt in das Suchfeld und lässt die Entprellung (250 ms) ablaufen. */
async function tippe(screen, text) {
	await fireEvent.input(screen.getByLabelText('Anderen Datensatz suchen'), {
		target: { value: text }
	});
	await vi.advanceTimersByTimeAsync(250);
}

function oeffne() {
	return render(SchuelerZusammenfuehrenDialog, {
		props: { open: true, profile: PROFIL, onMerged: vi.fn() }
	});
}

beforeEach(() => {
	vi.useFakeTimers();
	vi.mocked(apiFetch).mockReset();
});
afterEach(() => {
	vi.useRealTimers();
});

describe('SchuelerZusammenfuehrenDialog: Kandidatensuche', () => {
	it('lässt eine späte Antwort auf die ältere Eingabe die Liste nicht mehr überschreiben', async () => {
		const langsam = offen();
		const schnell = offen();
		vi.mocked(apiFetch)
			.mockReturnValueOnce(/** @type {any} */ (langsam.promise))
			.mockReturnValueOnce(/** @type {any} */ (schnell.promise));
		const screen = oeffne();

		await tippe(screen, 'Al');
		await tippe(screen, 'Alt123');
		expect(apiFetch).toHaveBeenCalledTimes(2);
		expect(vi.mocked(apiFetch).mock.calls[1][0]).toContain('q=Alt123');

		schnell.einloesen(antwort([KANDIDAT('k2', 'Richtig')]));
		await stillhalten();
		expect(screen.getByText(/Richtig/)).toBeTruthy();

		// Die Antwort auf „Al“ trudelt danach ein — sie gehört zu einer Eingabe, die es
		// nicht mehr gibt, und darf die Liste nicht auf den falschen Kandidaten drehen.
		langsam.einloesen(antwort([KANDIDAT('k1', 'Falsch')]));
		await stillhalten();
		expect(screen.getByText(/Richtig/)).toBeTruthy();
		expect(screen.queryByText(/Falsch/)).toBeNull();
	});

	it('zeigt einen Serverfehler als Fehler — nicht als „Kein anderer Datensatz gefunden“', async () => {
		vi.mocked(apiFetch).mockResolvedValue(/** @type {any} */ ({ ok: false, status: 500 }));
		const screen = oeffne();

		await tippe(screen, 'Al');
		await stillhalten();

		expect(screen.getByText('Fehler 500')).toBeTruthy();
		expect(screen.queryByText('Kein anderer Datensatz gefunden.')).toBeNull();
	});

	it('fängt einen Netzwerkfehler in der Suche ab und sagt es', async () => {
		vi.mocked(apiFetch).mockRejectedValue(new Error('Netz weg'));
		const screen = oeffne();

		await tippe(screen, 'Al');
		await stillhalten();

		expect(screen.getByText(/Netzwerkfehler/)).toBeTruthy();
		expect(screen.queryByText('Kein anderer Datensatz gefunden.')).toBeNull();
	});

	it('lässt eine Antwort, die nach dem Schließen eintrifft, nicht im neu geöffneten Dialog landen', async () => {
		const spaet = offen();
		vi.mocked(apiFetch).mockReturnValueOnce(/** @type {any} */ (spaet.promise));
		const screen = oeffne();

		await tippe(screen, 'Al');
		expect(apiFetch).toHaveBeenCalledTimes(1);

		// Zu, wieder auf — an der Theke ist das der nächste Schüler.
		await screen.rerender({ open: false, profile: PROFIL, onMerged: vi.fn() });
		await screen.rerender({ open: true, profile: PROFIL, onMerged: vi.fn() });
		spaet.einloesen(antwort([KANDIDAT('k1', 'Vorig')]));
		await stillhalten();

		expect(screen.queryByText(/Vorig/)).toBeNull();
		expect(
			/** @type {HTMLInputElement} */ (screen.getByLabelText('Anderen Datensatz suchen')).value
		).toBe('');
	});

	it('lässt einen beim Schließen laufenden Entprell-Timer keine Suche mehr auslösen', async () => {
		const screen = oeffne();
		await fireEvent.input(screen.getByLabelText('Anderen Datensatz suchen'), {
			target: { value: 'Al' }
		});
		await screen.rerender({ open: false, profile: PROFIL, onMerged: vi.fn() });
		await vi.advanceTimersByTimeAsync(300);
		expect(apiFetch).not.toHaveBeenCalled();
	});
});
