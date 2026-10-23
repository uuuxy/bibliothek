import { describe, it, expect, vi, beforeEach } from 'vitest';
import { erzeugePlaner } from './lmfplanPlaner.svelte.js';
import { apiFetch } from './apiFetch.js';
import { serverAntwort } from './serverAntwort.js';

// Der LMF-Plan meldet seine Änderungen über die SSE-Leitung (api/lernmittel/lmf_plan_live.go), damit
// das Kollegium im Portal und die zweite Bibliothekskraft am anderen PC den neuen Stand
// sehen, ohne neu zu laden (07.09.2026).
//
// Für den PLANER ist stilles Nachladen aber nicht immer richtig: `lade()` setzt den
// Entwurf auf den Server-Stand zurück. Wer gerade eine Reihenfolge zusammengezogen hat
// und noch nicht gespeichert hat, verlöre sie in dem Moment, in dem nebenan jemand
// speichert — ohne ein Wort, mitten in der Arbeit. Deshalb entscheidet `fremdesSignal`:
// still nachladen, solange hier nichts Ungespeichertes steht, sonst stehen lassen und
// hinweisen.
vi.mock('./apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));
vi.mock('../inventur/lib/store.svelte.js', () => ({ melde: vi.fn() }));
vi.mock('./stores/bestaetigung.svelte.js', () => ({ bestaetigen: vi.fn(async () => true) }));

const STAND = {
	plan: null,
	zeilen: [],
	ausgelassen: [],
	vorbei: false,
	vorschlag: { quelle: 'regel', zeilen: [{ klassen: ['09H1'], vermerk: '' }], ausgelassen: [] },
	klassen: ['09H1', '10R2'],
	eingangsjahrgaenge: [5, 7],
	sommerferien: { jahr: 2027, von: '2027-06-28', bis: '2027-08-06', bekannt: true }
};

/** @param {any} body */
const ok = (body) => /** @type {any} */ ({ ok: true, json: async () => body });

describe('LMF-Planer: fremdes Live-Signal', () => {
	beforeEach(() => {
		vi.mocked(apiFetch).mockReset();
		vi.mocked(apiFetch).mockImplementation(async (/** @type {any} */ url) => {
			if (String(url).startsWith('/api/lmf-plan/')) return ok(STAND);
			return ok({});
		});
	});

	it('lädt still nach, solange nichts Ungespeichertes offen ist', async () => {
		const planer = erzeugePlaner();
		await planer.lade();
		const rufeVorher = vi.mocked(apiFetch).mock.calls.length;

		planer.fremdesSignal();
		await vi.waitFor(() =>
			expect(vi.mocked(apiFetch).mock.calls.length, 'kein Nachladen ausgelöst').toBeGreaterThan(
				rufeVorher
			)
		);
		expect(planer.zustand.fremdeAenderung, 'ohne offene Arbeit braucht es keinen Hinweis').toBe(
			false
		);
	});

	it('wirft ungespeicherte Arbeit NICHT weg, sondern weist hin', async () => {
		const planer = erzeugePlaner();
		await planer.lade();

		// Dieselbe Bewegung wie im Planer: eine Klasse in die Reihenfolge ziehen.
		planer.zustand.entwurf.zeilen.push({ klassen: ['10R2'], vermerk: '', fest: null });
		expect(planer.hatUngespeichertes(), 'die Probe selbst muss greifen').toBe(true);
		const rufeVorher = vi.mocked(apiFetch).mock.calls.length;
		const zeilenVorher = JSON.stringify(planer.zustand.entwurf.zeilen);

		planer.fremdesSignal();
		await Promise.resolve();

		expect(vi.mocked(apiFetch).mock.calls.length, 'hat trotzdem nachgeladen').toBe(rufeVorher);
		expect(JSON.stringify(planer.zustand.entwurf.zeilen), 'die offene Arbeit ist weg').toBe(
			zeilenVorher
		);
		expect(planer.zustand.fremdeAenderung, 'niemand erfährt von der fremden Änderung').toBe(true);
	});
});

// Ohne Live-Leitung kommt das Signal nicht an. Dann entscheidet der Server: Der Planer
// schickt den Stempel des geladenen Plans mit, und ist der Plan inzwischen ein anderer,
// lehnt der Server das Speichern ab (409). Die Arbeit bleibt stehen, der Hinweis erscheint.
describe('LMF-Planer: Speichern auf einem überholten Stand', () => {
	const MIT_PLAN = {
		...STAND,
		plan: {
			id: 'p1',
			art: 'rueckgabe',
			erster_tag: '2027-06-14',
			startstunde: 2,
			letzter_tag: '2027-06-24',
			letzte_stunde: 4,
			stunden_je_tag: 6,
			freie_tage: [],
			veroeffentlicht_am: null,
			aktualisiert_am: '2027-01-11T09:30:00.123456Z'
		}
	};
	/** @type {any} */
	let gesendet;
	beforeEach(() => {
		gesendet = null;
		vi.mocked(apiFetch).mockReset();
		vi.mocked(apiFetch).mockImplementation(async (/** @type {any} */ url, /** @type {any} */ o) => {
			if (o?.method !== 'PUT') return ok(MIT_PLAN);
			gesendet = JSON.parse(o.body);
			if (gesendet.vorschau) return ok({ zeilen: [], ausfaelle: [] });
			return serverAntwort(409, {
				error: 'Der Plan wurde inzwischen an einem anderen Platz geändert.'
			});
		});
	});

	it('schickt den Stempel mit, behält die Arbeit und weist hin', async () => {
		const planer = erzeugePlaner();
		await planer.lade();
		planer.zustand.entwurf.zeilen.push({ klassen: ['10R2'], vermerk: '', fest: null });
		const zeilenVorher = JSON.stringify(planer.zustand.entwurf.zeilen);

		await planer.speichern();

		expect(gesendet?.aktualisiert_am, 'der Stempel des geladenen Plans').toBe(
			MIT_PLAN.plan.aktualisiert_am
		);
		expect(planer.zustand.fremdeAenderung, 'kein Hinweis auf den anderen Platz').toBe(true);
		expect(JSON.stringify(planer.zustand.entwurf.zeilen), 'die offene Arbeit ist weg').toBe(
			zeilenVorher
		);
	});

	it('Veröffentlichen speichert zuerst und weist ebenso hin', async () => {
		const planer = erzeugePlaner();
		await planer.lade();
		await planer.veroeffentlichen();
		expect(planer.zustand.fremdeAenderung, 'kein Hinweis auf den anderen Platz').toBe(true);
	});

	it('ohne Plan geht ein leerer Stempel mit', async () => {
		vi.mocked(apiFetch).mockImplementation(async (/** @type {any} */ url, /** @type {any} */ o) => {
			if (o?.method === 'PUT') gesendet = JSON.parse(o.body);
			return ok(o?.method === 'PUT' ? { plan: { veroeffentlicht_am: null } } : STAND);
		});
		const planer = erzeugePlaner();
		await planer.lade();
		await planer.speichern();
		expect(gesendet?.aktualisiert_am).toBe('');
	});
});
