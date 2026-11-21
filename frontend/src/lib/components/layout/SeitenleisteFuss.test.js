import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import SeitenleisteFuss from './SeitenleisteFuss.svelte';

// Eingeklappt steht das Konto nur als Kürzel da. Seine Sprechblase nennt die Rolle wie die
// aufgeklappte Leiste mit ihrem Namen, nicht mit dem gespeicherten Wert („admin“).
describe('SeitenleisteFuss eingeklappt', () => {
	it('nennt Name und Rolle mit dem Namen der Rolle', () => {
		const screen = render(SeitenleisteFuss, {
			collapsed: true,
			benutzer: { vorname: 'Erika', nachname: 'Muster', rolle: 'admin' },
			onLogout: () => {}
		});
		// Die Sprechblase zeigt Name und Rolle, der Bildschirmleser liest sie statt der Initialen.
		expect(
			screen.container.querySelector('[data-tip="Erika Muster · Administrator"]')
		).not.toBeNull();
		expect(screen.getByText('Erika Muster · Administrator')).toBeTruthy();
	});
});
