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
	eingangsjahrgaenge: [5, 7],
	sommerferien: { jahr: 2027, von: '2027-06-28', bis: '2027-08-06', bekannt: true }
};
// Der Rahmen, mit dem ein neuer Büchertausch beginnt (Migration 101): Donnerstag vor den
// Sommerferien, 4. Stunde — vom Server aus der Ferientabelle.
const RAHMEN = {
	erster_tag: '',
	startstunde: 1,
	letzter_tag: '2027-06-24',
	letzte_stunde: 4,
	stunden_je_tag: 6
};

// Ein gespeicherter Plan ohne Stempel ist ein Entwurf (Migration 100): „Veröffentlichen"
// steht bereit, der Hinweis sagt „nicht im Portal". Mit Stempel verschwindet der Knopf.
const PLAN = {
	id: 'p1',
	art: 'rueckgabe',
	erster_tag: '2027-06-28',
	startstunde: 1,
	letzter_tag: '2027-06-28',
	letzte_stunde: 1,
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

// Der Anker des Büchertauschs (Peter, 06.09.2026: „es endet immer am gleichen Tag —
// Donnerstags vor den Ferien zur vierten Stunde"): Ein neuer Plan kommt mit dem
// vorbelegten letzten Tag aus der Ferientabelle, die Vorschau schickt ihn als
// letzter_tag, und der Satz unter dem Rahmen nennt Ferien und gerechneten Beginn. Fehlt
// das Jahr in der Tabelle, sagt der Satz das und das Feld bleibt leer.
describe('LmfPlan: Anker am Ende des Büchertauschs', () => {
	beforeEach(() => vi.mocked(apiFetch).mockReset());

	/** @param {any} stand @returns {any[]} die Körper der Vorschau-Aufrufe */
	function antworte(stand) {
		/** @type {any[]} */
		const gesendet = [];
		vi.mocked(apiFetch).mockImplementation(async (url, init) => {
			if (init?.method === 'PUT') {
				gesendet.push(JSON.parse(String(init.body)));
				return new Response(
					JSON.stringify({
						plan: { erster_tag: '2027-06-24', startstunde: 4 },
						zeilen: [{ datum: '2027-06-24', stunde: 4 }],
						ausfaelle: []
					}),
					{ status: 200 }
				);
			}
			return new Response(JSON.stringify(stand), { status: 200 });
		});
		return gesendet;
	}

	it('belegt den letzten Tag vor, schickt ihn als letzter_tag und nennt den Beginn', async () => {
		const gesendet = antworte({ ...STAND, vorschlag: { ...STAND.vorschlag, rahmen: RAHMEN } });
		render(LmfPlan);
		const feld = /** @type {HTMLInputElement} */ (await screen.findByLabelText('Letzter Tag'));
		expect(feld.value).toBe('2027-06-24');
		expect(screen.getByText('Ende am letzten Tag')).toBeTruthy();
		expect(screen.queryByLabelText('Erster Tag')).toBeNull();
		const hinweis = await screen.findByTestId('lmf-zeitraum-hinweis');
		expect(hinweis.textContent).toContain('Sommerferien 2027: 28.06.27 bis 06.08.27');
		await vi.waitFor(() => expect(gesendet.length).toBeGreaterThan(0));
		expect(gesendet[0].letzter_tag).toBe('2027-06-24');
		expect(gesendet[0].letzte_stunde).toBe(4);
		expect(gesendet[0].vorschau).toBe(true);
		await vi.waitFor(() =>
			expect(screen.getByTestId('lmf-zeitraum-hinweis').textContent).toContain(
				'Der Plan beginnt Donnerstag, 24.06.27 in der 4. Stunde'
			)
		);
	});

	it('nennt das fehlende Ferienjahr und lässt den Tag leer (Gegenprobe)', async () => {
		const gesendet = antworte({
			...STAND,
			sommerferien: { jahr: 2031, von: '', bis: '', bekannt: false },
			vorschlag: { ...STAND.vorschlag, rahmen: { ...RAHMEN, letzter_tag: '' } }
		});
		render(LmfPlan);
		const feld = /** @type {HTMLInputElement} */ (await screen.findByLabelText('Letzter Tag'));
		expect(feld.value).toBe('');
		expect(screen.getByTestId('lmf-zeitraum-hinweis').textContent).toContain(
			'Sommerferien 2031 sind im Programm noch nicht hinterlegt'
		);
		// Ohne Anker keine Vorschau und kein Speichern.
		await new Promise((r) => setTimeout(r, 400));
		expect(gesendet).toHaveLength(0);
		expect(
			/** @type {HTMLButtonElement} */ (screen.getByRole('button', { name: 'Plan speichern' }))
				.disabled
		).toBe(true);
	});
});
