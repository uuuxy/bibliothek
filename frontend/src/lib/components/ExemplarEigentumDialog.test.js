import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

vi.mock('../apiFetch.js', () => ({ apiPut: vi.fn() }));
vi.mock('../stores/toastStore.svelte.js', () => ({ toastStore: { addToast: vi.fn() } }));

import { apiPut } from '../apiFetch.js';
import ExemplarEigentumDialog from './ExemplarEigentumDialog.svelte';

// Das Eigentum markierter Exemplare (docs/OFFEN.md 4.24, Stufe 3). Zwei Fehler wären teuer:
// eine Änderung ohne Grund (sie wirkt auf Etikett, Bestandsbücher und Schadensersatz, und
// niemand wüsste später, warum) und „Vorgabe" als Wort statt als leerer Wert an den Server.

const aendern = (/** @type {any} */ getByRole) => getByRole('button', { name: 'Eigentum ändern' });
// Der rohe DOM-Text trägt zwischen zwei Ausdrücken einen Umbruch; der Browser zeigt ein Leerzeichen.
const text = (/** @type {any} */ getByRole) =>
	(getByRole('dialog').textContent || '').replace(/\s+/g, ' ');

describe('ExemplarEigentumDialog', () => {
	beforeEach(() => {
		vi.mocked(apiPut).mockReset();
		vi.mocked(apiPut).mockResolvedValue({ geaendert: 2 });
	});

	it('gibt die Aktion erst mit Auswahl und Grund frei', async () => {
		const { getByRole, getByLabelText } = render(ExemplarEigentumDialog, {
			open: true,
			exemplarIds: ['e1', 'e2'],
			onclose: () => {},
			onGeaendert: () => {}
		});
		expect(text(getByRole)).toContain('Gilt für 2 Exemplare.');
		expect(aendern(getByRole).disabled).toBe(true);

		await fireEvent.click(getByLabelText('Land'));
		expect(aendern(getByRole).disabled).toBe(true);

		await fireEvent.input(getByLabelText(/Grund der Änderung/), { target: { value: '   ' } });
		expect(aendern(getByRole).disabled).toBe(true);

		await fireEvent.input(getByLabelText(/Grund der Änderung/), {
			target: { value: 'Klassensatz aus LMF-Mitteln' }
		});
		expect(aendern(getByRole).disabled).toBe(false);
	});

	it('schickt Kennungen, Eigentum und getrimmten Grund und lädt danach neu', async () => {
		const onGeaendert = vi.fn();
		const onclose = vi.fn();
		const { getByRole, getByLabelText } = render(ExemplarEigentumDialog, {
			open: true,
			exemplarIds: ['e1', 'e2'],
			onclose,
			onGeaendert
		});
		await fireEvent.click(getByLabelText('Schulträger'));
		await fireEvent.input(getByLabelText(/Grund der Änderung/), {
			target: { value: '  aus Mitteln des Kreises  ' }
		});
		await fireEvent.click(aendern(getByRole));

		expect(apiPut).toHaveBeenCalledWith('/api/exemplare/eigentum', {
			exemplar_ids: ['e1', 'e2'],
			eigentum: 'schultraeger',
			grund: 'aus Mitteln des Kreises'
		});
		expect(onGeaendert).toHaveBeenCalledTimes(1);
		expect(onclose).toHaveBeenCalledTimes(1);
	});

	it('schickt „Vorgabe" als leeres Eigentum', async () => {
		const { getByRole, getByLabelText } = render(ExemplarEigentumDialog, {
			open: true,
			exemplarIds: ['e1'],
			onclose: () => {},
			onGeaendert: () => {}
		});
		expect(text(getByRole)).toContain('Gilt für 1 Exemplar.');
		await fireEvent.click(getByLabelText('Vorgabe'));
		await fireEvent.input(getByLabelText(/Grund der Änderung/), {
			target: { value: 'irrtümlich gesetzt' }
		});
		await fireEvent.click(aendern(getByRole));

		expect(vi.mocked(apiPut).mock.calls[0][1]).toEqual({
			exemplar_ids: ['e1'],
			eigentum: '',
			grund: 'irrtümlich gesetzt'
		});
	});
});
