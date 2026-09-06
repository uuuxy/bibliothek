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
	klassen: ['09H1'],
	nur_rueckgabe: ['09H1'],
	eingangsjahrgaenge: [5, 7]
};

// Ein gespeicherter Plan ohne Stempel ist ein Entwurf (Migration 100): „Veröffentlichen"
// steht bereit, der Hinweis sagt „nicht im Portal". Mit Stempel verschwindet der Knopf.
const PLAN = {
	id: 'p1',
	art: 'rueckgabe',
	erster_tag: '2027-06-28',
	startstunde: 1,
	stunden_je_tag: 6,
	freie_tage: [],
	veroeffentlicht_am: null
};
const ZEILEN = [
	{ position: 1, datum: '2027-06-28', stunde: 1, fest: false, klassen: ['09H1'], vermerk: '' }
];

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

describe('LmfPlan: Entwurf und Veröffentlichung', () => {
	beforeEach(() => vi.mocked(apiFetch).mockReset());

	/** @param {any} stand */
	function antworte(stand) {
		vi.mocked(apiFetch).mockImplementation(async (url, init) => {
			if (init?.method === 'PUT')
				return new Response(JSON.stringify({ zeilen: [], ausfaelle: [] }), { status: 200 });
			return new Response(JSON.stringify(stand), { status: 200 });
		});
	}

	it('bietet „Veröffentlichen“ für einen Entwurf an und nennt ihn Entwurf', async () => {
		antworte({ ...STAND, plan: PLAN, zeilen: ZEILEN, vorschlag: undefined });
		render(LmfPlan);
		expect(await screen.findByRole('button', { name: 'Veröffentlichen' })).toBeTruthy();
		expect(screen.getByTestId('lmf-plan-hinweis').textContent).toContain('Entwurf vom 28.06.27');
		expect(screen.getByTestId('lmf-plan-hinweis').textContent).toContain('nicht im Portal');
		// Die Markierungen aus dem Stand: 09H1 hat Schüler und gibt nur zurück.
		expect(screen.getByText('nur Rückgabe')).toBeTruthy();
		expect(screen.queryByText('ohne Schüler')).toBeNull();
	});

	it('zeigt bei einem veröffentlichten Plan keinen Veröffentlichen-Knopf mehr (Gegenprobe)', async () => {
		antworte({
			...STAND,
			plan: { ...PLAN, veroeffentlicht_am: '2027-05-20T10:00:00+02:00' },
			zeilen: ZEILEN,
			klassen: [],
			vorschlag: undefined
		});
		render(LmfPlan);
		// Erst wenn der Hinweis da ist, ist der Stand geladen — die Knöpfe stehen schon vorher.
		expect(await screen.findByTestId('lmf-plan-hinweis')).toBeTruthy();
		expect(screen.getByRole('button', { name: 'Plan speichern' })).toBeTruthy();
		expect(screen.queryByRole('button', { name: 'Veröffentlichen' })).toBeNull();
		expect(screen.getByTestId('lmf-plan-hinweis').textContent).toContain(
			'veröffentlicht am 20.05.27'
		);
		// Ohne Schüler in 09H1 (klassen leer): die Zeile sagt es.
		expect(screen.getByText('ohne Schüler')).toBeTruthy();
	});
});
