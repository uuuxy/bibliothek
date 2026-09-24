import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import StudentKontoStatus from './StudentKontoStatus.svelte';

// Der Knopf beim Konto-Status richtet sich nach dem einen Prädikat (sperrStatus.js). Seit dem
// 24.09.2026 hebt er jede Sperre am Leser auf — auch die, die das Programm den Ehemaligen
// setzt; bis dahin stand bei einem gesperrten Ehemaligen „Schüler sperren". Einen Kollegen
// sperrt niemand: Der Knopf bleibt an seiner Stelle, verschlossen, mit dem Satz warum.
describe('StudentKontoStatus', () => {
	it('bietet bei der Sperre der Ehemaligen das Aufheben an', async () => {
		const onLock = vi.fn();
		const screen = render(StudentKontoStatus, {
			profile: {
				art: 'schueler',
				ist_gesperrt: true,
				block_reason: 'Automatisierte Abgänger-Sperre (offene Vorgänge)'
			},
			onLock
		});
		expect(screen.getByText('Gesperrt')).toBeTruthy();
		await fireEvent.click(screen.getByRole('button', { name: /Sperre aufheben/ }));
		expect(onLock).toHaveBeenCalledOnce();
	});

	it('bietet bei einem freien Schüler das Sperren an', () => {
		const screen = render(StudentKontoStatus, { profile: { art: 'schueler' }, onLock: vi.fn() });
		expect(screen.getByRole('button', { name: /Schüler sperren/ })).toBeTruthy();
	});

	it('verschließt den Knopf bei einem Kollegen und sagt warum', () => {
		const onLock = vi.fn();
		const screen = render(StudentKontoStatus, {
			profile: { art: 'lehrkraft', is_manually_blocked: true, block_reason: 'alt' },
			onLock
		});
		const knopf = /** @type {HTMLButtonElement} */ (
			screen.getByRole('button', { name: /Kollegen sperren/ })
		);
		expect(knopf.disabled).toBe(true);
		expect(screen.getByText('Kollegen werden nicht gesperrt.')).toBeTruthy();
		expect(screen.getByText('Aktiv')).toBeTruthy();
		// click() statt fireEvent: Wie im Browser erreicht ein Klick auf einen verschlossenen
		// Knopf keinen Handler; fireEvent schickt das Ereignis in jsdom trotzdem durch.
		knopf.click();
		expect(onLock).not.toHaveBeenCalled();
	});
});
