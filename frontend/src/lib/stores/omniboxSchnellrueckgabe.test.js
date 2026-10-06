import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

// Die Schnellrückgabe der Theke (docs/HANDBUCH.md, Ausleihe): Ein Stapel vom Rückgabetisch wird nur
// zurückgenommen. Ohne sie lädt die erste Rückgabe den Leser des Buchs, und das nächste freie
// Buch geht an ihn.
//
// Der Server leiht nur aus, wenn die Anfrage einen Leser nennt. Geprüft wird deshalb an der
// Anfrage, dass im Modus keiner mitgeht, und am Zustand, dass keiner geladen wird.

vi.mock('./toastStore.svelte.js', () => ({ toastStore: { addToast: vi.fn() } }));
vi.mock('../apiFetch.js', () => ({ apiFetch: vi.fn(), apiClient: { post: vi.fn() } }));
vi.mock('../audio.js', () => ({
	playSoundSuccess: vi.fn(),
	playSoundError: vi.fn(),
	playSuccessBeep: vi.fn(),
	playErrorBeep: vi.fn()
}));
vi.mock('./buchBarcodes.svelte.js', () => ({
	buchBarcodes: {
		istBuch: () => false,
		bereitstellen: vi.fn(),
		laden: vi.fn(),
		auffrischen: vi.fn(),
		anzahl: 0,
		geholtAm: 0
	}
}));

import { apiClient } from '../apiFetch.js';
import { playSoundError } from '../audio.js';
import { toastStore } from './toastStore.svelte.js';
import { createOmniboxStore } from './omnibox.svelte.js';
import { thekeLeeren } from './thekeLeeren.js';
import { escapeSchliesst } from '../components/ui/escapeSchliesst.js';
import { omniboxStore } from './omnibox.svelte.js';

const MIA = { id: 'leser-1', art: 'schueler', vorname: 'Mia', nachname: 'Muster', klasse: '07B' };
const BUCH = { titel: 'Natura 2' };

/** @param {Record<string, unknown>} data @returns {any} */
const antwort = (data) => ({ ok: true, status: 200, json: async () => data });
/** @param {string} text @returns {any} */
const ablehnung = (text) => ({
	ok: false,
	status: 409,
	headers: new Headers(),
	text: async () => JSON.stringify({ error: text })
});

/** @type {ReturnType<typeof createOmniboxStore>[]} */
const stores = [];
function neueTheke() {
	const store = createOmniboxStore();
	stores.push(store);
	return store;
}
// Ein Zeitgeber, der die Datei überlebt, reißt den ganzen Lauf (omniboxZeitgeber.test.js).
afterEach(() => {
	while (stores.length) stores.pop()?.stoppeZeitgeber();
	omniboxStore.stoppeZeitgeber();
});

/** @param {ReturnType<typeof createOmniboxStore>} store @param {string} code */
async function scanne(store, code) {
	store.queryVal = code;
	await store.submitAction(null, null);
}

/** Der Rumpf der letzten Anfrage an die Theken-Tür. @returns {any} */
const letzteAnfrage = () => vi.mocked(apiClient.post).mock.calls.at(-1)?.[1];

describe('Schnellrückgabe', () => {
	beforeEach(() => vi.clearAllMocks());

	it('nimmt zurück, lädt keinen Leser und nennt ihn in der Meldung', async () => {
		const theke = neueTheke();
		theke.schalteSchnellrueckgabe(true);
		vi.mocked(apiClient.post).mockResolvedValue(
			antwort({ type: 'rueckgabe', book: BUCH, student: MIA })
		);

		await scanne(theke, 'B-1');

		expect(theke.activeStudent, 'die Rückgabe hat einen Leser geladen').toBeNull();
		expect(theke.schnellrueckgabe).toBe(true);
		expect(toastStore.addToast).toHaveBeenCalledWith(
			'„Natura 2" zurückgegeben, war bei Mia Muster (07B).',
			'success'
		);
	});

	it('schickt auch beim zweiten Buch keinen Leser mit', async () => {
		const theke = neueTheke();
		theke.schalteSchnellrueckgabe(true);
		vi.mocked(apiClient.post).mockResolvedValue(
			antwort({ type: 'rueckgabe', book: BUCH, student: MIA })
		);

		await scanne(theke, 'B-1');
		await scanne(theke, 'B-2');

		expect(letzteAnfrage().query).toBe('B-2');
		expect(letzteAnfrage().active_leser_id, 'mit Leser leiht der Server aus').toBeUndefined();
	});

	it('ohne den Modus lädt die Rückgabe den Leser wie bisher', async () => {
		const theke = neueTheke();
		vi.mocked(apiClient.post).mockResolvedValue(
			antwort({ type: 'rueckgabe', book: BUCH, student: MIA })
		);

		await scanne(theke, 'B-1');
		await scanne(theke, 'B-2');

		expect(theke.activeStudent?.id).toBe('leser-1');
		expect(letzteAnfrage().active_leser_id).toBe('leser-1');
		expect(toastStore.addToast).toHaveBeenCalledWith(
			'„Natura 2" erfolgreich zurückgegeben.',
			'success'
		);
	});

	it('Einschalten lässt den geladenen Leser und den gemerkten Ausweis fallen', async () => {
		const theke = neueTheke();
		theke.activeStudent = MIA;
		theke.offlineAusweis = 'S-10001';

		theke.schalteSchnellrueckgabe(true);

		expect(theke.activeStudent).toBeNull();
		expect(theke.offlineAusweis).toBe('');
		vi.mocked(apiClient.post).mockResolvedValue(
			antwort({ type: 'rueckgabe', book: BUCH, student: MIA })
		);
		await scanne(theke, 'B-1');
		expect(letzteAnfrage().active_leser_id).toBeUndefined();
	});

	it('ein gescannter Ausweis beendet sie und lädt den Leser', async () => {
		const theke = neueTheke();
		theke.schalteSchnellrueckgabe(true);
		vi.mocked(apiClient.post).mockResolvedValue(antwort({ type: 'student', student: MIA }));

		await scanne(theke, 'S-10001');

		expect(theke.schnellrueckgabe).toBe(false);
		expect(theke.activeStudent?.id).toBe('leser-1');
	});

	it('ein ohne Netz gemerkter Ausweis beendet sie', async () => {
		const theke = neueTheke();
		theke.schalteSchnellrueckgabe(true);
		vi.mocked(apiClient.post).mockRejectedValue(new TypeError('Failed to fetch'));

		await scanne(theke, 'S-10001');

		expect(theke.offlineAusweis).toBe('S-10001');
		expect(theke.schnellrueckgabe, 'die folgenden Bücher gehören dem Ausweis').toBe(false);
	});

	it('ein von außen gesetzter Leser beendet sie', () => {
		const theke = neueTheke();
		theke.schalteSchnellrueckgabe(true);

		theke.activeStudent = MIA;

		expect(theke.schnellrueckgabe).toBe(false);
	});

	it('ein Buch, das nicht verliehen ist, ist im Modus zu hören', async () => {
		const theke = neueTheke();
		theke.schalteSchnellrueckgabe(true);
		vi.mocked(apiClient.post).mockResolvedValue(
			ablehnung('Dieses Buchexemplar ist aktuell nicht ausgeliehen')
		);

		await scanne(theke, 'B-FREI');

		expect(playSoundError).toHaveBeenCalledTimes(1);
		expect(theke.screenFlash).toBe('error');
		expect(theke.errorMessage).toContain('nicht ausgeliehen');
		expect(theke.schnellrueckgabe).toBe(true);
	});

	it('Escape beendet sie, außer es schließt gerade einen Hinweis', () => {
		const theke = neueTheke();
		theke.schalteSchnellrueckgabe(true);
		const hinweis = escapeSchliesst(document.createElement('div'), () => {});

		theke.escapeGedrueckt();
		expect(theke.schnellrueckgabe, 'das Escape galt dem Hinweis').toBe(true);

		hinweis.destroy();
		theke.escapeGedrueckt();
		expect(theke.schnellrueckgabe).toBe(false);
	});

	it('das Leeren der Theke beendet sie', () => {
		omniboxStore.schalteSchnellrueckgabe(true);

		thekeLeeren();

		expect(omniboxStore.schnellrueckgabe).toBe(false);
	});
});
