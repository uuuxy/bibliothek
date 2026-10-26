import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import PortalUeberblick from './PortalUeberblick.svelte';

// Kamen die eigenen Reservierungen nicht, steht der Grund da. „Zurzeit wartet keine Reservierung"
// hieße, es gebe keine, und lüde dazu ein, denselben Satz noch einmal zu reservieren.
describe('PortalUeberblick: die eigenen Reservierungen kommen nicht', () => {
	it('zeigt den Grund statt „Zurzeit wartet keine Reservierung" und lädt auf Wunsch neu', async () => {
		const onerneut = vi.fn();
		render(PortalUeberblick, {
			props: { reservierungen: [], ladefehler: 'Keine Verbindung zum Server.', onerneut }
		});

		expect(screen.getByText('Deine Reservierungen konnten nicht geladen werden')).toBeTruthy();
		expect(screen.getByText('Keine Verbindung zum Server.')).toBeTruthy();
		expect(screen.queryByText(/Zurzeit wartet keine Reservierung/)).toBeNull();

		await fireEvent.click(screen.getByRole('button', { name: 'Erneut versuchen' }));
		expect(onerneut).toHaveBeenCalled();
	});

	it('sagt ohne Fehler und ohne Reservierungen, dass keine wartet', () => {
		render(PortalUeberblick, { props: { reservierungen: [] } });
		expect(screen.getByText(/Zurzeit wartet keine Reservierung/)).toBeTruthy();
	});
});
