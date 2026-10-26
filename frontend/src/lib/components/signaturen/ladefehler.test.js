import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';

// Kommt eine Liste der Signaturen-Seite nicht, steht ihr Grund da: „Noch kein Buch trägt eine
// Signatur", „Unter … steht kein Buch" und „Noch keine Sachgruppen" wären falsche Auskünfte,
// und die letzte lüde dazu ein, vorhandene Sachgruppen noch einmal anzulegen.
vi.mock('../../apiFetch.js', async (original) => ({
	.../** @type {any} */ (await original()),
	apiGet: vi.fn(),
	apiPost: vi.fn()
}));

import { apiGet } from '../../apiFetch.js';
import Signaturen from './Signaturen.svelte';
import SignaturRegal from './SignaturRegal.svelte';
import SystematikVerwaltung from './SystematikVerwaltung.svelte';

const GRUND = 'Die Datenbank antwortet nicht';

describe('Signaturen: eine Liste kommt nicht', () => {
	beforeEach(() => {
		vi.mocked(apiGet).mockReset();
		vi.mocked(apiGet).mockRejectedValue(new Error(GRUND));
	});

	it('die Signaturen zeigen den Grund statt „Noch kein Buch trägt eine Signatur"', async () => {
		render(Signaturen);
		await vi.waitFor(() => expect(screen.getByText('Signaturen nicht geladen')).toBeTruthy());
		expect(screen.getAllByText(GRUND).length).toBeGreaterThan(0);
		expect(screen.queryByText(/Noch kein Buch trägt eine Signatur/)).toBeNull();
	});

	it('das Regal zeigt den Grund statt „steht kein Buch" und lädt neu', async () => {
		render(SignaturRegal, { props: { signatur: 'LMF Deu 7' } });
		await vi.waitFor(() => expect(screen.getByText('Regal nicht geladen')).toBeTruthy());
		expect(screen.getByText(GRUND)).toBeTruthy();
		expect(screen.queryByText(/steht kein Buch/)).toBeNull();

		vi.mocked(apiGet).mockResolvedValue({ buecher: [], gekappt: false });
		await fireEvent.click(screen.getByRole('button', { name: 'Erneut versuchen' }));
		await vi.waitFor(() => expect(screen.getByText(/steht kein Buch/)).toBeTruthy());
	});

	it('die Sachgruppen zeigen den Grund statt „Noch keine Sachgruppen"', async () => {
		render(SystematikVerwaltung);
		await vi.waitFor(() => expect(screen.getByText('Sachgruppen nicht geladen')).toBeTruthy());
		expect(screen.getByText(GRUND)).toBeTruthy();
		expect(screen.queryByText(/Noch keine Sachgruppen/)).toBeNull();
	});
});
