import { describe, it, expect, vi, afterEach } from 'vitest';

// Escape an der Theke schließt die Akte, aber nicht mit demselben Tastendruck, der eine Liste,
// einen Dialog oder die Eingabe in einem Feld schließt: Vorher war damit an der Theke die offene
// Maske „Stammdaten bearbeiten" samt Eingaben weg.

vi.mock('../../stores/snackbars.svelte.js', () => ({ snackbars: { addToast: vi.fn() } }));
vi.mock('../../apiFetch.js', () => ({ apiFetch: vi.fn(), apiClient: { post: vi.fn() } }));
vi.mock('../../utils/audio.js', () => ({
	spieleErfolgston: vi.fn(),
	spieleFehlerton: vi.fn()
}));

import { escapeAnDerTheke } from './thekeEscape.js';
import { omniboxStore } from '../../stores/omnibox.svelte.js';
import { uiStore } from '../../stores/uiStore.svelte.js';
import { bestaetigungStore } from '../../stores/bestaetigung.svelte.js';
import { escapeSchliesst } from '../ui/escapeSchliesst.js';

const MIA = { id: 'leser-1', art: 'schueler', vorname: 'Mia' };
const stoppeKamera = vi.fn();

/** @param {string} tag @param {string} [id] */
function element(tag, id = '') {
	const el = document.createElement(tag);
	el.id = id;
	document.body.append(el);
	return el;
}

/**
 * Drückt Escape auf dem Element. Ein Bauteil, das die Taste selbst verarbeitet (offene Liste),
 * ruft preventDefault, bevor sie beim Fenster ankommt.
 * @param {Element} ziel
 * @param {{ verarbeitet?: boolean }} [wie]
 */
function escape(ziel, { verarbeitet = false } = {}) {
	const e = new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true });
	if (verarbeitet) ziel.addEventListener('keydown', (t) => t.preventDefault(), { once: true });
	ziel.dispatchEvent(e);
	return escapeAnDerTheke(e, stoppeKamera);
}

afterEach(() => {
	bestaetigungStore.antworten(null);
	uiStore.verlassenSperre = null;
	omniboxStore.aktiverLeser = null;
	omniboxStore.kameraOffen = false;
	omniboxStore.stoppeZeitgeber();
	document.body.replaceChildren();
	vi.clearAllMocks();
});

describe('Escape an der Theke', () => {
	it('im Scanfeld und neben den Feldern schließt es die Akte und leert das Feld', async () => {
		for (const ziel of [element('input', 'omnibox-input'), document.body]) {
			omniboxStore.aktiverLeser = MIA;
			omniboxStore.eingabe = 'B-1';
			await escape(ziel);
			expect(omniboxStore.aktiverLeser).toBeNull();
			expect(omniboxStore.eingabe).toBe('');
		}
	});

	it('in einem anderen Feld lässt es die Akte stehen', async () => {
		omniboxStore.aktiverLeser = MIA;
		await escape(element('input', 'ort'));
		expect(omniboxStore.aktiverLeser?.id).toBe(MIA.id);
	});

	it('hat ein Bauteil die Taste schon verarbeitet (offene Liste), bleibt die Akte', async () => {
		omniboxStore.aktiverLeser = MIA;
		await escape(element('div'), { verarbeitet: true });
		expect(omniboxStore.aktiverLeser?.id).toBe(MIA.id);
	});

	it('bei offenem Dialog gehört es dem Dialog', async () => {
		omniboxStore.aktiverLeser = MIA;
		const dialog = escapeSchliesst(element('div'), () => {});
		await escape(document.body);
		dialog.destroy();
		expect(omniboxStore.aktiverLeser?.id).toBe(MIA.id);
	});

	it('mit Ungespeichertem fragt es; „Bleiben" lässt die Akte stehen', async () => {
		omniboxStore.aktiverLeser = MIA;
		uiStore.verlassenSperre = () => true;

		const bleiben = escape(document.body);
		expect(bestaetigungStore.anfrage).toMatchObject({
			titel: 'Ungespeicherte Änderungen',
			aktion: 'Verwerfen und weiter'
		});
		bestaetigungStore.antworten(false);
		await bleiben;
		expect(omniboxStore.aktiverLeser?.id).toBe(MIA.id);

		const verwerfen = escape(document.body);
		bestaetigungStore.antworten(true);
		await verwerfen;
		expect(omniboxStore.aktiverLeser).toBeNull();
	});

	it('schließt die Kamera mit', async () => {
		omniboxStore.kameraOffen = true;
		await escape(document.body);
		expect(stoppeKamera).toHaveBeenCalledTimes(1);
	});
});
