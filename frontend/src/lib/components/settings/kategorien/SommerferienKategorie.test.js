import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import SommerferienKategorie from './SommerferienKategorie.svelte';
import { speichereKategorie } from '../../../einstellungenSpeichern.js';

// Die Sommerferien als Einstellung (06.09.2026): Programmjahre stehen als „Programm",
// eigene als „eigener Eintrag" und gewinnen für dasselbe Jahr; aufnehmen, entfernen,
// speichern — gespeichert wird die JSON-Liste der EIGENEN Jahre unter „sommerferien".
vi.mock('../../../einstellungenSpeichern.js', () => ({ speichereKategorie: vi.fn() }));

const DATEN = {
	sommerferien_programm: [
		{ jahr: 2029, von: '2029-07-16', bis: '2029-08-24' },
		{ jahr: 2030, von: '2030-07-22', bis: '2030-08-30' }
	],
	sommerferien: '[{"jahr":2030,"von":"2030-07-01","bis":"2030-08-09"}]'
};

describe('SommerferienKategorie', () => {
	beforeEach(() => vi.mocked(speichereKategorie).mockReset());

	it('zeigt Programm und eigene Einträge in einer Liste — das eigene Jahr gewinnt', () => {
		const { container } = render(SommerferienKategorie, { daten: DATEN });
		const zeilen = [...container.querySelectorAll('tbody tr')].map((tr) =>
			tr.textContent?.replace(/\s+/g, ' ').trim()
		);
		expect(zeilen).toHaveLength(2);
		expect(zeilen[0]).toContain('2029');
		expect(zeilen[0]).toContain('Programm');
		expect(zeilen[1]).toContain('01.07.2030');
		expect(zeilen[1]).toContain('eigener Eintrag');
	});

	it('nimmt ein Jahr auf, entfernt es wieder und speichert nur die eigenen Jahre', async () => {
		const { getByLabelText, getByRole, container } = render(SommerferienKategorie, {
			daten: DATEN
		});
		const aufnehmen = () => getByRole('button', { name: /aufnehmen/ });
		expect(/** @type {HTMLButtonElement} */ (aufnehmen()).disabled).toBe(true);
		await fireEvent.input(getByLabelText('Beginn'), { target: { value: '2031-07-07' } });
		// Ende vor dem Beginn: nicht aufnehmbar.
		await fireEvent.input(getByLabelText('Ende'), { target: { value: '2031-07-01' } });
		expect(/** @type {HTMLButtonElement} */ (aufnehmen()).disabled).toBe(true);
		await fireEvent.input(getByLabelText('Ende'), { target: { value: '2031-08-15' } });
		expect(aufnehmen().textContent).toContain('2031 aufnehmen');
		await fireEvent.click(aufnehmen());
		expect(container.querySelectorAll('tbody tr')).toHaveLength(3);

		await fireEvent.click(getByRole('button', { name: 'Sommerferien speichern' }));
		expect(speichereKategorie).toHaveBeenCalledWith(
			expect.objectContaining({
				felder: {
					sommerferien:
						'[{"jahr":2030,"von":"2030-07-01","bis":"2030-08-09"},{"jahr":2031,"von":"2031-07-07","bis":"2031-08-15"}]'
				}
			})
		);

		await fireEvent.click(getByRole('button', { name: 'Sommerferien 2030 entfernen' }));
		await fireEvent.click(getByRole('button', { name: 'Sommerferien 2031 entfernen' }));
		await fireEvent.click(getByRole('button', { name: 'Sommerferien speichern' }));
		// Ohne eigene Jahre geht "" — die Vorgabe, nicht "[]".
		expect(speichereKategorie).toHaveBeenLastCalledWith(
			expect.objectContaining({ felder: { sommerferien: '' } })
		);
		// Das Programmjahr 2030 steht wieder mit seinen Programmdaten.
		expect(container.querySelectorAll('tbody tr')[1].textContent).toContain('22.07.2030');
	});
});
