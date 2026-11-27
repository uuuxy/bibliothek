import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';
import InventurStartDialog from './InventurStartDialog.svelte';

// Ein zweiter Klick auf „Starten" vor der Antwort schickte einen zweiten Start. Der
// Server lehnt ihn ab, weil der Bereich schon läuft; nach dem Abschluss stand dann „Für diesen
// Bereich läuft bereits eine Inventur", obwohl keine lief. Gleich in Chromium, Firefox und WebKit.
describe('InventurStartDialog: Start', () => {
	it('schickt keinen zweiten Start, solange der erste läuft', async () => {
		/** @type {() => void} */
		let fertig = () => {};
		const onStart = vi.fn(() => new Promise((weiter) => (fertig = () => weiter(undefined))));
		const k = render(InventurStartDialog, {
			props: {
				open: true,
				ablauf: {
					bereichArt: 'global',
					gewaehlteSignatur: '',
					gewaehltesFach: '',
					gewaehlterJahrgang: '',
					signaturen: [],
					faecher: [],
					fehler: ''
				},
				onClose: vi.fn(),
				onStart
			}
		});
		const knopf = /** @type {HTMLButtonElement} */ (k.getByRole('button', { name: 'Starten' }));

		await fireEvent.click(knopf);
		await fireEvent.click(knopf);

		expect(onStart, 'ein zweiter Start vor der Antwort').toHaveBeenCalledTimes(1);
		expect(knopf.disabled, 'gesperrt, solange der Start läuft').toBe(true);

		fertig();
		await waitFor(() => expect(knopf.disabled).toBe(false));
	});
});

describe('InventurStartDialog: fehlende Angaben', () => {
	/** @param {Record<string, any>} teil @param {any} [onStart] */
	const oeffne = (teil, onStart = vi.fn()) =>
		render(InventurStartDialog, {
			props: {
				open: true,
				ablauf: {
					bereichArt: 'global',
					gewaehlteSignatur: '',
					gewaehltesFach: '',
					gewaehlterJahrgang: '',
					signaturen: [],
					faecher: [],
					fehler: '',
					...teil
				},
				onClose: vi.fn(),
				onStart
			}
		});

	// Eine fehlende Signatur ist ein leeres Feld: Der Klick nennt sie dort und startet nichts.
	it('nennt eine fehlende Signatur beim Klick am Feld', async () => {
		const onStart = vi.fn();
		const k = oeffne({ bereichArt: 'signature' }, onStart);
		const knopf = /** @type {HTMLButtonElement} */ (k.getByRole('button', { name: 'Starten' }));
		expect(knopf.disabled).toBe(false);
		await fireEvent.click(knopf);
		expect(k.getByText('Bitte die Signatur eintragen')).toBeTruthy();
		const feld = k.getByLabelText(/Signatur auswählen/);
		expect(feld.getAttribute('aria-invalid')).toBe('true');
		await waitFor(() => expect(document.activeElement).toBe(feld));
		expect(onStart).not.toHaveBeenCalled();
	});

	// Fach und Klasse sind eine Auswahl: Ohne eins von beiden ist keine Wahl getroffen.
	it('sperrt den Filter ohne Fach und Klasse', () => {
		const k = oeffne({ bereichArt: 'filter' });
		const knopf = /** @type {HTMLButtonElement} */ (k.getByRole('button', { name: 'Starten' }));
		expect(knopf.disabled).toBe(true);
	});
});
