import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import SicherungKategorie from './SicherungKategorie.svelte';
import { speichereKategorie } from '../einstellungenSpeichern.js';

// Der Schalter der eigenen Nachtsicherung: Vorgabe aus, gespeichert wird „sicherung_an“ mit dem
// Stand beim Öffnen wie in jeder Kategorie (einstellungenSpeichern.js).
vi.mock('../einstellungenSpeichern.js', async (original) => ({
	.../** @type {any} */ (await original()),
	speichereKategorie: vi.fn()
}));

describe('SicherungKategorie', () => {
	beforeEach(() => {
		vi.mocked(speichereKategorie).mockReset();
	});

	it('steht ohne gespeicherte Einstellung auf aus und speichert das Einschalten', async () => {
		const { getByRole } = render(SicherungKategorie, { daten: {} });
		const schalter = getByRole('switch', { name: 'Nächtliche Sicherung umschalten' });
		expect(schalter.getAttribute('aria-checked')).toBe('false');

		await fireEvent.click(schalter);
		await fireEvent.click(getByRole('button', { name: 'Sicherung speichern' }));

		expect(speichereKategorie).toHaveBeenCalledWith(
			expect.objectContaining({ geladen: { sicherung_an: false }, felder: { sicherung_an: true } })
		);
	});

	it('zeigt eine eingeschaltete Sicherung als an', () => {
		const { getByRole } = render(SicherungKategorie, { daten: { sicherung_an: true } });
		expect(
			getByRole('switch', { name: 'Nächtliche Sicherung umschalten' }).getAttribute('aria-checked')
		).toBe('true');
	});
});
