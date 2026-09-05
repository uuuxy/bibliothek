import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import LmfPlan from './LmfPlan.svelte';
import { apiFetch } from './apiFetch.js';

// Scheitert das Laden, darf der Planer NICHT erscheinen.
//
// Sonst stünde nach einem Netzfehler „Noch kein Plan" da, der Planer böte die
// Regel-Reihenfolge an — und ein Klick auf „Plan speichern" ersetzte den echten Plan des
// Schuljahres durch diesen Entwurf und stellte die Fristen aller Klassen auf den
// Stichtag zurück. Dieselbe Klasse wie an den Einstellungen am 31.08.2026; das Bauteil
// dafür gab es schon (ui/LadeFehler.svelte), nur hier nicht.
//
// Gemockt wird apiFetch, nicht der Dienst: So läuft der echte Lesepfad (lmfplanDienst)
// mit, und der Fehler entsteht dort, wo er im Betrieb entsteht.
vi.mock('./apiFetch.js', () => ({ apiFetch: vi.fn() }));
vi.mock('../inventur/lib/store.svelte.js', () => ({ showToast: vi.fn() }));

const STAND = {
	plan: null,
	zeilen: [],
	ausgelassen: [],
	vorbei: false,
	vorschlag: { quelle: 'regel', zeilen: [{ klassen: ['09H1'], vermerk: '' }], ausgelassen: [] },
	klassen: ['09H1']
};

describe('LmfPlan: gescheitertes Laden', () => {
	beforeEach(() => vi.mocked(apiFetch).mockReset());

	it('zeigt den Fehler statt eines leeren Plans — und keinen Speichern-Knopf', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({ ok: false, status: 500, json: async () => ({}) })
		);
		render(LmfPlan);
		expect(await screen.findByText('Plan nicht geladen')).toBeTruthy();
		expect(screen.queryByText(/Noch kein Plan/)).toBeNull();
		expect(screen.queryByRole('button', { name: 'Plan speichern' })).toBeNull();
		expect(screen.getByRole('button', { name: 'Erneut versuchen' })).toBeTruthy();
	});

	it('zeigt den Planer, wenn das Laden gelingt (Gegenprobe)', async () => {
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({ ok: true, json: async () => STAND })
		);
		render(LmfPlan);
		expect(await screen.findByText(/Noch kein Plan/)).toBeTruthy();
		expect(screen.getByRole('button', { name: 'Plan speichern' })).toBeTruthy();
		expect(screen.queryByText('Plan nicht geladen')).toBeNull();
	});
});
