import { describe, it, expect, vi, beforeEach } from 'vitest';
import { useStudentProfile } from './useStudentProfile.svelte.js';
import { apiFetch } from './apiFetch.js';

// Die Akte lädt vier Dinge nebeneinander: Stammdaten, Vormerkungen, Gebühren, Bescheide.
//
// Bis zum Rasterdurchgang am 06.09.2026 stand dort dreimal `if (res.ok)` ohne `else` und
// ohne Sequenznummer. Fiel genau eine der drei Anfragen aus — 500, oder 429 vom
// Rate-Limiter, es sind drei parallele Anfragen je Akte —, behielt dieser Teil die Werte
// des VORHER geöffneten Schülers, während Kopf und Ausleihen schon zum neuen gehörten.
//
// Bei den Gebühren ist das nicht nur Anzeige: StudentGebuehrenCard schreibt auf die
// Fall-ID der Zeile („Zahlung verbucht", „Storno"). Ein Klick auf der Akte von B hätte
// die Zahlung einem Schadensfall von A gutgeschrieben — Kopf, Ausdruck und Audit zeigen B.
vi.mock('./apiFetch.js', () => ({ apiFetch: vi.fn() }));

/** @param {any} body */
const ok = (body) => ({ ok: true, json: async () => body });
const fehler = { ok: false, status: 500, json: async () => ({}) };

describe('useStudentProfile.fetchProfile', () => {
	beforeEach(() => vi.mocked(apiFetch).mockReset());

	it('lässt die Gebühren des vorigen Schülers nicht stehen', async () => {
		vi.mocked(apiFetch).mockImplementation(async (url) => {
			const u = String(url);
			if (u.includes('schadensfaelle')) return /** @type {any} */ (ok({ data: [{ id: 'f-A' }] }));
			if (u.includes('vormerkungen')) return /** @type {any} */ (ok([]));
			return /** @type {any} */ (ok({ id: 'A', vorname: 'Anna' }));
		});
		const st = useStudentProfile();
		await st.fetchProfile('A');
		expect(st.gebuehren).toEqual([{ id: 'f-A' }]);

		// Schüler B: die Gebühren-Anfrage scheitert.
		vi.mocked(apiFetch).mockImplementation(async (url) => {
			const u = String(url);
			if (u.includes('schadensfaelle')) return /** @type {any} */ (fehler);
			if (u.includes('vormerkungen')) return /** @type {any} */ (ok([]));
			return /** @type {any} */ (ok({ id: 'B', vorname: 'Ben' }));
		});
		await st.fetchProfile('B');
		expect(st.profile?.id).toBe('B');
		expect(st.gebuehren, 'Gebühren von A stehen unter dem Namen von B').toEqual([]);
	});

	// Dieselbe Klasse für die Bescheide (seit 12.09.2026 in der Akte): Ein Bescheid nennt
	// Referenznummer, Betrag und Frist. Der von A unter dem Namen von B wäre die Auskunft,
	// mit der Eltern in der Bibliothek stehen.
	it('lässt die Bescheide des vorigen Schülers nicht stehen', async () => {
		/** @param {string} id */
		const antworten = (id, bescheideOk) =>
			vi.mocked(apiFetch).mockImplementation(async (url) => {
				const u = String(url);
				if (u.includes('bescheide')) {
					return /** @type {any} */ (
						bescheideOk ? ok({ data: [{ id: 'bescheid-' + id }] }) : fehler
					);
				}
				if (u.includes('schadensfaelle')) return /** @type {any} */ (ok({ data: [] }));
				if (u.includes('vormerkungen')) return /** @type {any} */ (ok([]));
				return /** @type {any} */ (ok({ id, vorname: id }));
			});

		const st = useStudentProfile();
		antworten('A', true);
		await st.fetchProfile('A');
		expect(st.bescheide).toEqual([{ id: 'bescheid-A' }]);

		antworten('B', false);
		await st.fetchProfile('B');
		expect(st.profile?.id).toBe('B');
		expect(st.bescheide, 'der Bescheid von A steht unter dem Namen von B').toEqual([]);
	});

	it('lässt die überholte Antwort nicht gewinnen', async () => {
		/** @type {() => void} */
		let loesenA = () => {};
		const aKam = new Promise((r) => (loesenA = /** @type {any} */ (r)));
		vi.mocked(apiFetch).mockImplementation(async (url) => {
			const u = String(url);
			const istA = u.includes('/A');
			if (istA) await aKam;
			if (u.includes('schadensfaelle')) return /** @type {any} */ (ok({ data: [] }));
			if (u.includes('vormerkungen')) return /** @type {any} */ (ok([]));
			return /** @type {any} */ (ok({ id: istA ? 'A' : 'B', vorname: istA ? 'Anna' : 'Ben' }));
		});
		const st = useStudentProfile();
		const langsam = st.fetchProfile('A');
		await st.fetchProfile('B');
		expect(st.profile?.id).toBe('B');
		loesenA();
		await langsam;
		expect(st.profile?.id, 'die ältere Antwort hat die jüngere überschrieben').toBe('B');
		expect(st.loading).toBe(false);
	});

	it('schließt beim Schülerwechsel jedes offene Blatt', () => {
		const st = useStudentProfile();
		st.showEditModal = true;
		st.showDamageModal = true;
		st.showLockModal = true;
		st.showDeleteConfirm = true;
		st.showWebcam = true;
		st.schliesseAlleBlaetter();
		expect([
			st.showEditModal,
			st.showDamageModal,
			st.showLockModal,
			st.showDeleteConfirm,
			st.showWebcam
		]).toEqual([false, false, false, false, false]);
	});
});
