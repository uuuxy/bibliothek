import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import { authStore } from '../stores/authStore.svelte.js';
import BestandImport from '../components/admin/BestandImport.svelte';
import ListenImport from '../components/admin/ListenImport.svelte';
import LitteraImport from '../components/admin/LitteraImport.svelte';
import LusdImport from '../components/leser/LusdImport.svelte';

// Wer eine abgelehnte Liste korrigiert, wählt dieselbe Datei noch einmal. Bliebe sie im Feld, meldete
// der Browser keine Änderung, und auch nach einem geglückten Import ließe sie sich nicht mehr wählen.
describe('Dateifelder der Importe: dieselbe Datei lässt sich noch einmal wählen', () => {
	beforeEach(() => {
		authStore.currentUser = /** @type {any} */ ({ rolle: 'admin', permissions: ['*'] });
	});

	it.each([
		['Bestands-Import', BestandImport],
		['Listenimport', ListenImport],
		['Littera-Import', LitteraImport],
		['LUSD-Import', LusdImport]
	])('%s leert das Feld, die Datei bleibt gewählt', async (_name, Bauteil) => {
		const { container } = render(/** @type {any} */ (Bauteil));
		const feld = /** @type {HTMLInputElement} */ (container.querySelector('input[type="file"]'));
		const geleert = vi.fn();
		Object.defineProperty(feld, 'value', { configurable: true, set: geleert });

		await fireEvent.change(feld, {
			target: { files: [new File(['x'], 'liste.csv', { type: 'text/csv' })] }
		});

		expect(geleert).toHaveBeenCalledWith('');
		expect(container.textContent).toContain('liste.csv');
	});
});
