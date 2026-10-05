import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/svelte';
import PortalTrefferkarte from './PortalTrefferkarte.svelte';

// Die Abzeichen am Treffer sagen, ob ein Klassensatz reichen kann. Die Obergrenze der
// Reservierung zählt bestellte Exemplare mit; steht nur der Bestand im Haus da, nennt der
// Treffer weniger, als sich reservieren lässt.

const zu = { open: false, worum: '', klasse: '', text: '', sending: false };

/** @param {Record<string, any>} book */
const aufbau = (book) =>
	render(PortalTrefferkarte, {
		book: { id: 't1', titel: 'Natura 2', ...book },
		form: { open: false, success: '', anzahl: 1 },
		meldung: zu,
		warteschlange: [],
		ontoggle: vi.fn(),
		onsenden: vi.fn(),
		onmelden: vi.fn(),
		onmeldungsenden: vi.fn(),
		onmeldungabbrechen: vi.fn()
	});

describe('PortalTrefferkarte', () => {
	it('nennt neben dem Bestand im Haus, was bestellt ist', () => {
		const s = aufbau({ gesamt: 10, verfuegbar: 10, im_zulauf: 20 });

		expect(s.getByText('10 von 10 verfügbar')).toBeTruthy();
		expect(s.getByText('20 bestellt')).toBeTruthy();
	});

	it('nennt den Zulauf auch, wenn im Haus gerade nichts frei ist', () => {
		const s = aufbau({ gesamt: 10, verfuegbar: 0, im_zulauf: 20 });

		expect(s.getByText('nicht verfügbar (10 im Bestand)')).toBeTruthy();
		expect(s.getByText('20 bestellt')).toBeTruthy();
	});

	it('schreibt ohne Zulauf nichts von einer Bestellung', () => {
		const s = aufbau({ gesamt: 10, verfuegbar: 10, im_zulauf: 0 });

		expect(s.getByText('10 von 10 verfügbar')).toBeTruthy();
		expect(s.queryByText(/bestellt/)).toBeNull();
	});

	it('nennt einen nur bestellten Titel einmal', () => {
		const s = aufbau({ gesamt: 0, verfuegbar: 0, im_zulauf: 2 });

		expect(s.getAllByText('2 bestellt')).toHaveLength(1);
		expect(s.queryByText(/verfügbar/)).toBeNull();
	});
});
