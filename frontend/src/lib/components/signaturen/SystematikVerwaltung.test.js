import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', async (original) => ({
	...(await original()),
	apiGet: vi.fn(),
	apiPost: vi.fn(),
	apiPut: vi.fn(),
	apiDelete: vi.fn()
}));

vi.mock('../../stores/bestaetigung.svelte.js', async (original) => ({
	...(await original()),
	loeschenBestaetigen: vi.fn(async () => true)
}));

import { apiGet, apiPost, apiPut, apiDelete } from '../../apiFetch.js';
import SystematikVerwaltung from './SystematikVerwaltung.svelte';

/** @param {HTMLElement} feld */
const beschreibung = (feld) =>
	document.getElementById(feld.getAttribute('aria-describedby') ?? '')?.textContent;

/** Ein echter Klick legt den Fokus auf den Knopf; fireEvent allein tut das nicht. @param {HTMLElement} knopf */
async function klicke(knopf) {
	knopf.focus();
	await fireEvent.click(knopf);
}

// „Anlegen" und „Sichern" bleiben bei leerem Feld bedienbar: Der Klick nennt am Feld, was
// fehlt, führt dorthin und schickt nichts ab.
describe('Sachgruppen: leeres Feld', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		vi.mocked(apiGet).mockResolvedValue([{ id: 's1', kuerzel: 'Deu', bezeichnung: 'Deutsch' }]);
	});

	it('„Anlegen" nennt erst das fehlende Kürzel, dann die fehlende Bezeichnung', async () => {
		const k = render(SystematikVerwaltung);
		const anlegen = k.getByRole('button', { name: 'Anlegen' });
		const kuerzel = k.getByLabelText('Kürzel');
		const bezeichnung = k.getByLabelText('Bezeichnung');

		await klicke(anlegen);
		expect(kuerzel.getAttribute('aria-invalid')).toBe('true');
		expect(beschreibung(kuerzel)).toBe('Bitte ein Kürzel eintragen.');
		expect(document.activeElement).toBe(kuerzel);

		await fireEvent.input(kuerzel, { target: { value: 'Mat' } });
		await klicke(anlegen);
		expect(kuerzel.getAttribute('aria-invalid')).toBeNull();
		expect(bezeichnung.getAttribute('aria-invalid')).toBe('true');
		expect(beschreibung(bezeichnung)).toBe('Bitte eine Bezeichnung eintragen.');
		expect(document.activeElement).toBe(bezeichnung);
		expect(apiPost).not.toHaveBeenCalled();
	});

	// Die Gegenprobe: Nach dem Anlegen sind die Felder leer und tragen keinen Fehler.
	it('leert nach dem Anlegen die Felder ohne Fehlertext', async () => {
		vi.mocked(apiPost).mockResolvedValue({});
		const k = render(SystematikVerwaltung);
		const kuerzel = /** @type {HTMLInputElement} */ (k.getByLabelText('Kürzel'));
		await fireEvent.input(kuerzel, { target: { value: 'Mat' } });
		await fireEvent.input(k.getByLabelText('Bezeichnung'), { target: { value: 'Mathematik' } });
		await klicke(k.getByRole('button', { name: 'Anlegen' }));

		await waitFor(() => expect(apiPost).toHaveBeenCalledTimes(1));
		await waitFor(() => expect(kuerzel.value).toBe(''));
		expect(kuerzel.getAttribute('aria-invalid')).toBeNull();
		expect(k.getByLabelText('Bezeichnung').getAttribute('aria-invalid')).toBeNull();
	});

	it('„Sichern" nennt am geleerten Feld, was fehlt, und schickt nichts ab', async () => {
		const k = render(SystematikVerwaltung);
		await fireEvent.click(await k.findByRole('button', { name: 'Ändern' }));
		const kuerzel = k.getByLabelText('Kürzel bearbeiten');
		await fireEvent.input(kuerzel, { target: { value: '  ' } });

		await klicke(k.getByRole('button', { name: 'Sichern' }));
		expect(kuerzel.getAttribute('aria-invalid')).toBe('true');
		expect(beschreibung(kuerzel)).toBe('Bitte ein Kürzel eintragen.');
		expect(document.activeElement).toBe(kuerzel);
		expect(apiPut).not.toHaveBeenCalled();
	});
});

// Eine offene Zeile der Tabelle übersteht das Nachladen der Liste, das ein Anlegen oben auslöst.
describe('Sachgruppen: offene Zeile beim Nachladen', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		vi.mocked(apiGet).mockResolvedValue([{ id: 's1', kuerzel: 'Deu', bezeichnung: 'Deutsch' }]);
		vi.mocked(apiPost).mockResolvedValue({});
	});

	it('lässt die getippte Änderung stehen, wenn oben eine Sachgruppe angelegt wird', async () => {
		const k = render(SystematikVerwaltung);
		await fireEvent.click(await k.findByRole('button', { name: 'Ändern' }));
		await fireEvent.input(k.getByLabelText('Bezeichnung bearbeiten'), {
			target: { value: 'Deutsch als Zweitsprache' }
		});

		await fireEvent.input(k.getByLabelText('Kürzel'), { target: { value: 'Mat' } });
		await fireEvent.input(k.getByLabelText('Bezeichnung'), { target: { value: 'Mathematik' } });
		await klicke(k.getByRole('button', { name: 'Anlegen' }));
		await waitFor(() => expect(apiGet).toHaveBeenCalledTimes(2));

		const feld = /** @type {HTMLInputElement} */ (
			await k.findByLabelText('Bezeichnung bearbeiten')
		);
		expect(feld.value).toBe('Deutsch als Zweitsprache');
	});
});

// Ein zweiter Auftrag zum Löschen vor der Antwort schickte ein zweites DELETE: Nach „Sachgruppe
// gelöscht." stand „Sachgruppe nicht gefunden", in Chromium, Firefox und WebKit (OFFEN.md 5.106).
describe('Sachgruppen: Löschen', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		vi.mocked(apiGet).mockResolvedValue([{ id: 's1', kuerzel: 'Deu', bezeichnung: 'Deutsch' }]);
	});

	it('schickt kein zweites DELETE, solange das erste läuft', async () => {
		/** @type {() => void} */
		let fertig = () => {};
		vi.mocked(apiDelete).mockImplementation(
			() => new Promise((weiter) => (fertig = () => weiter(null)))
		);
		const k = render(SystematikVerwaltung);
		const knopf = /** @type {HTMLButtonElement} */ (
			await k.findByRole('button', { name: 'Löschen' })
		);

		await fireEvent.click(knopf);
		await waitFor(() => expect(apiDelete).toHaveBeenCalledTimes(1));
		await fireEvent.click(knopf);
		await new Promise((weiter) => setTimeout(weiter));

		expect(apiDelete, 'ein zweites DELETE vor der Antwort').toHaveBeenCalledTimes(1);
		expect(knopf.disabled, 'gesperrt, solange das Löschen läuft').toBe(true);

		fertig();
		await waitFor(() => expect(knopf.disabled).toBe(false));
	});
});
