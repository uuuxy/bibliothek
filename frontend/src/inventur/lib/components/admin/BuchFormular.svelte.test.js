import { describe, it, expect, vi, beforeAll, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';

// Ersetzt wird nur der Erkenner (er braucht eine Kamera) und der Server; der Weg vom
// Kamera-Fenster über die Maske bis zur Frage läuft echt.
vi.mock('$lib/components/scanner/barcode_detector.js', async (importOriginal) => ({
	...(await importOriginal()),
	createBarcodeDetector: async () => ({
		name: 'test',
		detector: { detect: async () => [{ rawValue: '9783060130764' }] }
	})
}));
vi.mock('../../../../lib/apiFetch.js', async (importOriginal) => ({
	...(await importOriginal()),
	apiFetch: vi.fn()
}));
vi.mock('../../../../lib/stores/bestaetigung.svelte.js', () => ({ bestaetigen: vi.fn() }));
vi.mock('$lib/store.svelte.js', () => ({ appState: { bookToEdit: null }, showToast: vi.fn() }));

import { apiFetch } from '../../../../lib/apiFetch.js';
import { bestaetigen } from '../../../../lib/stores/bestaetigung.svelte.js';
import { appState } from '$lib/store.svelte.js';
import BuchFormular from './BuchFormular.svelte';
import { leeresBuchFormular } from './buch_form_optionen.js';

const GESCANNT = '9783060130764';
const KATALOG = '/api/books/vorhanden';
const DIENSTE = '/api/lookup/';

/** @param {number} status @param {any} koerper */
const antwort = (status, koerper) =>
	/** @type {any} */ ({ ok: status < 400, status, json: async () => koerper });

/**
 * @param {boolean} vergeben — der eigene Katalog kennt die gescannte ISBN
 * @param {number} [dienste] Status, mit dem die Katalogdienste antworten
 */
function server(vergeben, dienste = 200) {
	vi.mocked(apiFetch).mockImplementation(async (url) => {
		const u = String(url);
		if (u.startsWith(KATALOG)) {
			return antwort(200, {
				data: vergeben
					? {
							vorhanden: { id: 'titel-1', title: 'Green Line 3', ohneExemplar: false },
							meldung: 'Diese ISBN trägt schon der Titel „Green Line 3“.'
						}
					: { vorhanden: null }
			});
		}
		if (u.startsWith(DIENSTE)) return antwort(dienste, { data: { title: 'Green Line 3' } });
		return antwort(200, []);
	});
}
/** @param {string} anfang */
const aufrufe = (anfang) =>
	vi.mocked(apiFetch).mock.calls.filter(([url]) => String(url).startsWith(anfang));

/** @param {any} formular @param {{ wirdGescannt?: boolean }} [zusatz] */
function maske(formular, zusatz = {}) {
	return render(BuchFormular, {
		formular,
		onClose: () => {},
		onSave: () => {},
		onCoverUpload: () => {},
		onCoverNeuHolen: () => {},
		onAssignClass: () => {},
		...zusatz
	});
}

beforeAll(() => {
	// Lücken von jsdom, die nur dieser Weg braucht: eine Kamera und ein Video, das spielt.
	Object.defineProperty(navigator, 'mediaDevices', {
		configurable: true,
		value: { getUserMedia: async () => ({ getTracks: () => [] }) }
	});
	HTMLMediaElement.prototype.play = vi.fn().mockResolvedValue(undefined);
	Object.defineProperty(HTMLMediaElement.prototype, 'readyState', {
		configurable: true,
		get: () => 2
	});
});

beforeEach(() => {
	vi.clearAllMocks();
	appState.bookToEdit = null;
});

// Der Kamera-Scan trägt die ISBN ohne Verlassen des Feldes ein. Auch er fragt in einer neuen
// Maske zuerst den eigenen Katalog — sonst lüde er Angaben für ein Buch, das es schon gibt.
describe('BuchFormular: Kamera-Scan in einer neuen Maske', () => {
	it('die gescannte ISBN trägt schon ein Titel: Frage statt Nachschlagen', async () => {
		server(true);
		vi.mocked(bestaetigen).mockResolvedValue(true);
		const formular = $state(leeresBuchFormular());
		const screen = maske(formular);

		await fireEvent.click(screen.getByRole('button', { name: 'Scan ISBN' }));

		await waitFor(() => expect(bestaetigen).toHaveBeenCalledTimes(1), { timeout: 4000 });
		expect(formular.isbn).toBe(GESCANNT);
		expect(String(aufrufe(KATALOG)[0][0])).toBe(`${KATALOG}?isbn=${GESCANNT}`);
		expect(appState.bookToEdit).toEqual({ id: 'titel-1' });
		expect(aufrufe(DIENSTE)).toHaveLength(0);
	});

	it('die gescannte ISBN ist frei: die Angaben werden geladen', async () => {
		server(false);
		const formular = $state(leeresBuchFormular());
		const screen = maske(formular);

		await fireEvent.click(screen.getByRole('button', { name: 'Scan ISBN' }));

		await waitFor(() => expect(formular.title).toBe('Green Line 3'), { timeout: 4000 });
		expect(aufrufe(KATALOG)).toHaveLength(1);
		expect(String(aufrufe(DIENSTE)[0][0])).toBe(`${DIENSTE}${GESCANNT}`);
		expect(bestaetigen).not.toHaveBeenCalled();
	});

	// Eine Abfrage für beide Wege: Nach dem Scan steht derselbe Satz am ISBN-Feld wie nach
	// dem Tippen.
	it('die Katalogdienste antworten nicht: der Grund steht am ISBN-Feld', async () => {
		server(false, 502);
		const formular = $state(leeresBuchFormular());
		const screen = maske(formular);

		await fireEvent.click(screen.getByRole('button', { name: 'Scan ISBN' }));

		const eingabe = screen.getByLabelText('ISBN');
		await waitFor(() => expect(eingabe.getAttribute('aria-invalid')).toBe('true'), {
			timeout: 4000
		});
		const hinweis = document.getElementById(eingabe.getAttribute('aria-describedby') ?? '');
		expect(hinweis?.textContent).toContain('Die Katalogdienste sind nicht erreichbar.');
		expect(formular.title).toBe('');
	});
});

// Der Knopf „Scanner" der Titelliste öffnet dieselbe Maske wie „Neues Buch", nur mit
// eingeschalteter Kamera. Ein eigenes Fenster mit eigenem Speichern gibt es nicht mehr.
describe('BuchFormular: geöffnet mit eingeschalteter Kamera', () => {
	it('das Kamera-Fenster steht sofort, der Scan trägt die ISBN ein und schließt es', async () => {
		server(false);
		const formular = $state(leeresBuchFormular());
		const screen = maske(formular, { wirdGescannt: true });

		expect(screen.getByRole('heading', { name: 'ISBN scannen' })).toBeTruthy();
		expect(screen.getByRole('heading', { name: 'Neues Buch' })).toBeTruthy();

		await waitFor(() => expect(formular.isbn).toBe(GESCANNT), { timeout: 4000 });
		await waitFor(() => expect(screen.queryByRole('heading', { name: 'ISBN scannen' })).toBeNull());
		await waitFor(() => expect(formular.title).toBe('Green Line 3'));
	});

	it('ohne diesen Wunsch bleibt die Kamera aus', () => {
		const formular = $state(leeresBuchFormular());
		const screen = maske(formular);
		expect(screen.queryByRole('heading', { name: 'ISBN scannen' })).toBeNull();
	});
});
