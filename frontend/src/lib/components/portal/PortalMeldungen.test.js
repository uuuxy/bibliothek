import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import PortalMeldungen from './PortalMeldungen.svelte';

// Die eigenen Meldungen und „Problem melden" ohne Buch, unter der Suche des Portals. Einen
// Buchwunsch gibt es nicht mehr: Ein Buch für eine Klasse wird über die Suche reserviert.
// Der Zustand des Formulars gehört dem Portal; das Zusammenspiel prüft KollegiumPortal.test.js.

const zu = { open: false, worum: '', klasse: '', text: '', sending: false };

/** @param {Record<string, any>} [props] */
const aufbau = (props = {}) => {
	const rufe = { onoeffnen: vi.fn(), onsenden: vi.fn(), onabbrechen: vi.fn() };
	return {
		...render(PortalMeldungen, {
			anliegen: [],
			form: zu,
			onaktualisiert: vi.fn(),
			...rufe,
			...props
		}),
		...rufe
	};
};

describe('PortalMeldungen', () => {
	it('zeigt einen Knopf „Problem melden" und kein Formular', async () => {
		const s = aufbau();

		expect(s.queryByRole('button', { name: 'Buchwunsch' })).toBeNull();
		expect(s.queryByRole('button', { name: 'Absenden' })).toBeNull();

		await fireEvent.click(s.getByRole('button', { name: 'Problem melden' }));
		expect(s.onoeffnen).toHaveBeenCalled();
	});

	it('fragt ohne Buch, worum es geht, und kennzeichnet die Pflichtfelder', async () => {
		const s = aufbau({ form: { ...zu, open: true } });

		expect(s.getByLabelText('Worum geht es? *')).toBeTruthy();
		expect(s.getByLabelText('Klasse / Kurs')).toBeTruthy();
		expect(s.getByLabelText('Was stimmt nicht? *')).toBeTruthy();
		const absenden = /** @type {HTMLButtonElement} */ (s.getByRole('button', { name: 'Absenden' }));
		expect(absenden.disabled, 'ohne Gegenstand und Beschreibung').toBe(true);

		await fireEvent.click(s.getByRole('button', { name: 'Abbrechen' }));
		expect(s.onabbrechen).toHaveBeenCalled();
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
