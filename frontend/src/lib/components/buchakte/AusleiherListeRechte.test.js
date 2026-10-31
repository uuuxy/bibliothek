import { describe, it, expect, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import { appState } from '../../../inventur/lib/store.svelte.js';
import { authStore } from '../../stores/authStore.svelte.js';
import AusleiherListe from './AusleiherListe.svelte';

// Der Name führt über einen Scan an die Theke. Ohne perform_actions lehnte der Server diesen Scan
// ab, und die Liste bot einen Knopf an, der nichts öffnet.
describe('Ausleiher-Liste: Recht zu scannen', () => {
	const zeile = {
		schueler_name: 'Kim',
		schueler_nachname: 'Beispiel',
		klasse: '07a',
		schueler_barcode: 'S-1',
		exemplar_barcode: 'B-1',
		ausgeliehen_am: '2026-09-01T10:00:00Z',
		rueckgabe_frist: '2026-11-01T10:00:00Z',
		ist_dauerleihe: false
	};
	const fmtDate = (/** @type {string} */ d) => new Date(d).toLocaleDateString('de-DE');
	const zeige = () => render(AusleiherListe, { zeilen: [zeile], onBack: () => {}, fmtDate });

	beforeEach(() => {
		appState.triggerStudentScan = '';
	});

	it('zeigt den Namen ohne das Recht als Text, nicht als Knopf', () => {
		authStore.currentUser = { rolle: 'mitarbeiter', permissions: ['view_students'] };
		const { queryByRole, getByText } = zeige();
		expect(queryByRole('button')).toBeNull();
		expect(getByText(/Kim\s+Beispiel/)).toBeTruthy();
		expect(getByText('(07a)')).toBeTruthy();
	});

	it('legt den Ausweis mit dem Recht in den Scan-Kanal', async () => {
		authStore.currentUser = { rolle: 'mitarbeiter', permissions: ['perform_actions'] };
		const { getByRole } = zeige();
		await fireEvent.click(getByRole('button', { name: /Kim\s+Beispiel/ }));
		expect(appState.triggerStudentScan).toBe('S-1');
	});
});
