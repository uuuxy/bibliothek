import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('../apiFetch.js', () => ({ apiPut: vi.fn(), apiFetch: vi.fn() }));
vi.mock('../stores/toastStore.svelte.js', () => ({ toastStore: { addToast: vi.fn() } }));

import { apiFetch, apiPut } from '../apiFetch.js';
import { toastStore } from '../stores/toastStore.svelte.js';
import ExemplarStandortDialog from './ExemplarStandortDialog.svelte';

// Der Standort markierter Exemplare (docs/OFFEN.md 5.53). Was hier schiefgehen kann: eine
// Änderung ohne Wahl (ein leeres Feld löschte den Standort aller markierten Exemplare),
// „entfernen" als Wort statt als leerer Wert an den Server, und ein Scan ins offene Feld,
// der zum Standort würde.

const aendern = (/** @type {any} */ getByRole) => getByRole('button', { name: 'Standort ändern' });
const feld = (/** @type {any} */ getByLabelText) =>
	/** @type {HTMLInputElement} */ (getByLabelText('Standort'));

/** @param {any} [zusatz] */
const dialog = (zusatz = {}) =>
	render(ExemplarStandortDialog, {
		open: true,
		exemplarIds: ['e1', 'e2'],
		onclose: () => {},
		onGeaendert: () => {},
		...zusatz
	});

describe('ExemplarStandortDialog', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		vi.mocked(apiPut).mockResolvedValue({ geaendert: 2 });
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({
				ok: true,
				json: async () => [
					{ standort: 'Lehrerschrank', anzahl: 3 },
					{ standort: 'Bibliothek, Regal 3B', anzahl: 1 }
				]
			})
		);
	});

	it('gibt die Aktion erst mit einem Standort oder mit „Standort entfernen" frei', async () => {
		const { getByRole, getByLabelText } = dialog();
		expect(getByRole('dialog').textContent).toContain('Gilt für 2 Exemplare.');
		expect(feld(getByLabelText).value).toBe('');
		expect(aendern(getByRole).disabled).toBe(true);

		await fireEvent.input(feld(getByLabelText), { target: { value: '   ' } });
		expect(aendern(getByRole).disabled).toBe(true);

		await fireEvent.input(feld(getByLabelText), { target: { value: 'Lehrerschrank' } });
		expect(aendern(getByRole).disabled).toBe(false);

		await fireEvent.input(feld(getByLabelText), { target: { value: '' } });
		expect(aendern(getByRole).disabled).toBe(true);

		await fireEvent.click(getByLabelText('Standort entfernen'));
		expect(aendern(getByRole).disabled).toBe(false);
		expect(feld(getByLabelText).disabled).toBe(true);
	});

	it('schickt Kennungen und getrimmten Standort und lädt danach neu', async () => {
		const onGeaendert = vi.fn();
		const onclose = vi.fn();
		const { getByRole, getByLabelText } = dialog({ onclose, onGeaendert });
		await fireEvent.input(feld(getByLabelText), { target: { value: '  Bibliothek, Regal 3B ' } });
		await fireEvent.click(aendern(getByRole));

		expect(apiPut).toHaveBeenCalledWith('/api/exemplare/standort', {
			exemplar_ids: ['e1', 'e2'],
			standort: 'Bibliothek, Regal 3B'
		});
		await waitFor(() => expect(onclose).toHaveBeenCalledTimes(1));
		expect(onGeaendert).toHaveBeenCalledTimes(1);
		expect(toastStore.addToast).toHaveBeenCalledWith('Standort geändert: 2 Exemplare.', 'success');
	});

	it('schickt „entfernen" als leeren Standort, auch wenn im Feld noch etwas steht', async () => {
		vi.mocked(apiPut).mockResolvedValue({ geaendert: 1 });
		const { getByRole, getByLabelText } = dialog({ exemplarIds: ['e1'] });
		expect(getByRole('dialog').textContent).toContain('Gilt für 1 Exemplar.');
		await fireEvent.input(feld(getByLabelText), { target: { value: 'Keller' } });
		await fireEvent.click(getByLabelText('Standort entfernen'));
		await fireEvent.click(aendern(getByRole));

		expect(vi.mocked(apiPut).mock.calls[0][1]).toEqual({ exemplar_ids: ['e1'], standort: '' });
		await waitFor(() =>
			expect(toastStore.addToast).toHaveBeenCalledWith('Standort entfernt: 1 Exemplar.', 'success')
		);
	});

	it('bleibt nach einem Fehler des Servers offen und meldet keinen Erfolg', async () => {
		vi.mocked(apiPut).mockImplementation(() => Promise.reject(new Error('404')));
		const onGeaendert = vi.fn();
		const onclose = vi.fn();
		const { getByRole, getByLabelText } = dialog({ onclose, onGeaendert });
		await fireEvent.input(feld(getByLabelText), { target: { value: 'Keller' } });
		await fireEvent.click(aendern(getByRole));

		await waitFor(() => expect(aendern(getByRole).disabled).toBe(false));
		expect(onclose).not.toHaveBeenCalled();
		expect(onGeaendert).not.toHaveBeenCalled();
		expect(toastStore.addToast).not.toHaveBeenCalled();
	});

	it('schlägt die Standorte aus dem Bestand vor, den häufigsten zuerst', async () => {
		const { getByLabelText, baseElement } = dialog();
		await waitFor(() => expect(baseElement.querySelectorAll('datalist option')).toHaveLength(2));
		expect(apiFetch).toHaveBeenCalledWith('/api/exemplare/standorte');
		const liste = /** @type {HTMLDataListElement} */ (baseElement.querySelector('datalist'));
		expect(feld(getByLabelText).getAttribute('list')).toBe(liste.id);
		expect(
			[...liste.querySelectorAll('option')].map((o) => [o.value, o.textContent?.trim()])
		).toEqual([
			['Lehrerschrank', '3 Exemplare'],
			['Bibliothek, Regal 3B', '1 Exemplar']
		]);
	});

	// Ein Handscanner tippt in das Feld mit dem Fokus und endet mit Enter; im Dialog bestätigte
	// das Enter die Änderung (ui/EingabeDialog).
	it('nimmt einen Scan ins offene Feld nicht als Standort', async () => {
		const { getByRole, getByLabelText } = dialog();
		const eingabe = feld(getByLabelText);
		/** @param {string} key @param {number} zeit */
		const druecke = (key, zeit) => {
			const e = new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true });
			Object.defineProperty(e, 'timeStamp', { value: zeit });
			eingabe.dispatchEvent(e);
			return e;
		};
		[...'B-90231'].forEach((key, i) => {
			druecke(key, 1000 + i * 5);
			eingabe.value += key;
			eingabe.dispatchEvent(new Event('input', { bubbles: true }));
		});
		const enter = druecke('Enter', 1040);

		expect(enter.defaultPrevented).toBe(true);
		await waitFor(() => expect(eingabe.value).toBe(''));
		expect(aendern(getByRole).disabled).toBe(true);
		expect(apiPut).not.toHaveBeenCalled();
	});
});
