import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/svelte';
import PortalMeldungen from './PortalMeldungen.svelte';

// Die eigenen Meldungen der Lehrkraft mit ihrem Stand, unter der Suche des Portals. Der Knopf
// „Problem melden" steht darüber in der Zeile unter dem Suchfeld (KollegiumPortal.test.js).

/** @param {Record<string, any>} [props] */
const aufbau = (props = {}) =>
	render(PortalMeldungen, { anliegen: [], onaktualisiert: vi.fn(), ...props });

describe('PortalMeldungen', () => {
	it('zeigt ohne Meldungen nichts, auch keinen Knopf', () => {
		const s = aufbau();

		expect(s.queryByRole('heading', { name: 'Deine Meldungen' })).toBeNull();
		expect(s.queryByRole('button', { name: 'Problem melden' })).toBeNull();
	});

	// Wünsche aus der Zeit des Buchwunschs stehen weiter in der Liste, bis sie abgehakt sind.
	it('zeigt die eigenen Meldungen mit Art, Stand und der Antwort der Bibliothek', () => {
		const s = aufbau({
			anliegen: [
				{
					id: 'a1',
					art: 'meldung',
					titel_text: 'Markl Biologie 2',
					klasse: '8G3',
					erstellt_am: 'x'
				},
				{
					id: 'a2',
					art: 'wunsch',
					titel_text: 'Natura 2',
					klasse: '7G1',
					erstellt_am: 'x',
					erledigt_am: 'y',
					erledigt_notiz: 'liegt bereit'
				}
			]
		});

		expect(s.getByRole('heading', { name: 'Deine Meldungen' })).toBeTruthy();
		expect(s.getByText('Meldung:')).toBeTruthy();
		expect(s.getByText('Wunsch:')).toBeTruthy();
		expect(s.getByText('Offen')).toBeTruthy();
		expect(s.getByText('Erledigt')).toBeTruthy();
		expect(s.getByText('Bibliothek: „liegt bereit"')).toBeTruthy();
	});
});
