import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, waitFor } from '@testing-library/svelte';

const leitung = vi.hoisted(() => ({
	/** @type {Map<string, Set<(e: any) => void>>} */
	hoerer: new Map(),
	/** @param {string} name @param {object} [daten] */
	sende(name, daten = {}) {
		const data = JSON.stringify(daten);
		for (const h of this.hoerer.get(name) ?? []) h({ data });
	}
}));
vi.mock('../../liveEvents.js', () => ({
	abonniere: (/** @type {string} */ name, /** @type {(e: any) => void} */ handler) => {
		if (!leitung.hoerer.has(name)) leitung.hoerer.set(name, new Set());
		leitung.hoerer.get(name)?.add(handler);
		return () => leitung.hoerer.get(name)?.delete(handler);
	}
}));
vi.mock('./abgaengerDienst.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	ladeAbgaenger: vi.fn(async () => ({ abgaenger: [], fenster: { offen: true, von: '', bis: '' } }))
}));

import { ladeAbgaenger } from './abgaengerDienst.js';
import Abgaenger from './Abgaenger.svelte';

const abrufe = () => vi.mocked(ladeAbgaenger).mock.calls.length;

// Ein zurückgegebenes Buch nimmt den Abgänger von selbst aus der Liste, auch wenn es an einem anderen
// Arbeitsplatz zurückkam. Fiel die Rückgabe in eine Unterbrechung, meldet sie niemand nach.
describe('Abgaenger: die Liste bleibt aktuell', () => {
	beforeEach(() => {
		vi.mocked(ladeAbgaenger).mockClear();
		leitung.hoerer.clear();
	});

	it('holt die Liste nach einer Rückgabe und beim Aufbau der Leitung neu, sonst nicht', async () => {
		render(Abgaenger);
		await waitFor(() => expect(abrufe()).toBe(1));

		leitung.sende('action', { event: 'rueckgabe' });
		expect(abrufe()).toBe(2);
		leitung.sende('action', { event: 'fremdrueckgabe' });
		expect(abrufe()).toBe(3);
		leitung.sende('action', { event: 'ausleihe' });
		expect(abrufe()).toBe(3);

		leitung.sende('connected');
		expect(abrufe()).toBe(4);
	});

	it('hört nach dem Verlassen auf keine der beiden Meldungen mehr', async () => {
		const seite = render(Abgaenger);
		await waitFor(() => expect(abrufe()).toBe(1));

		seite.unmount();

		expect(leitung.hoerer.get('action')?.size).toBe(0);
		expect(leitung.hoerer.get('connected')?.size).toBe(0);
	});
});
