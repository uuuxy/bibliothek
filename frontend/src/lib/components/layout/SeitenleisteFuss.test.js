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
		expect(screen.getByTitle('Erika Muster · Administrator')).toBeTruthy();
	});
});
