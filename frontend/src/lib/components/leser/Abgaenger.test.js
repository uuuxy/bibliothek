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
	abonniereNeuladen: (
		/** @type {string} */ name,
		/** @type {() => any} */ holeNeu,
		/** @type {(e: any) => boolean} */ gilt
	) => {
		/** @type {[string, (e: any) => any][]} */
		const zuhoerer = [
			[name, (e) => (gilt(e) ? holeNeu() : undefined)],
			['connected', () => holeNeu()]
		];
		for (const [n, h] of zuhoerer) {
			if (!leitung.hoerer.has(n)) leitung.hoerer.set(n, new Set());
			leitung.hoerer.get(n)?.add(h);
		}
		return () => zuhoerer.forEach(([n, h]) => leitung.hoerer.get(n)?.delete(h));
	}
}));
vi.mock('./abgaengerDienst.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	ladeAbgaenger: vi.fn(async () => ({ abgaenger: [], fenster: { offen: true, von: '', bis: '' } }))
}));

import { ladeAbgaenger } from './abgaengerDienst.js';
import { snackbars } from '../../stores/snackbars.svelte.js';
import Abgaenger from './Abgaenger.svelte';

const abrufe = () => vi.mocked(ladeAbgaenger).mock.calls.length;
const FENSTER = { offen: true, von: '', bis: '' };
/** @param {string} id @param {string} nachname */
const abgaenger = (id, nachname) => ({
	id,
	vorname: 'Mia',
	nachname,
	klasse: '10a',
	barcode_id: id,
	offene_buecher: 1,
	ueberfaellig: 0
});
/** @param {...any} zeilen */
const liste = (...zeilen) => ({ abgaenger: zeilen, fenster: FENSTER });
/** Ein Signal und die Antwort des Holens, das es auslöst. */
const holeNeuAuf = async (/** @type {string} */ name, /** @type {object} */ daten = {}) => {
	const data = JSON.stringify(daten);
	const antworten = [...(leitung.hoerer.get(name) ?? [])].map((h) => h({ data }));
	return (await Promise.all(antworten))[0];
};

// Ein zurückgegebenes Buch nimmt den Abgänger von selbst aus der Liste, auch wenn es an einem anderen
// Arbeitsplatz zurückkam. Fiel die Rückgabe in eine Unterbrechung, meldet sie niemand nach.
describe('Abgaenger: die Liste bleibt aktuell', () => {
	beforeEach(() => {
		vi.mocked(ladeAbgaenger).mockClear();
		leitung.hoerer.clear();
		for (const t of [...snackbars.toasts]) snackbars.removeToast(t.id);
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

	it('eine unlesbare Meldung löst kein Laden aus', async () => {
		const fehler = vi.spyOn(console, 'error').mockImplementation(() => {});
		try {
			render(Abgaenger);
			await waitFor(() => expect(abrufe()).toBe(1));

			for (const h of leitung.hoerer.get('action') ?? []) h({ data: '{kaputt' });

			expect(abrufe()).toBe(1);
			expect(fehler).toHaveBeenCalled();
		} finally {
			fehler.mockRestore();
		}
	});

	it('hört nach dem Verlassen auf keine der beiden Meldungen mehr', async () => {
		const seite = render(Abgaenger);
		await waitFor(() => expect(abrufe()).toBe(1));

		seite.unmount();

		expect(leitung.hoerer.get('action')?.size).toBe(0);
		expect(leitung.hoerer.get('connected')?.size).toBe(0);
	});

	// Sonst stünde jemand mit erledigter Rückgabe wieder in der Liste.
	it('eine ältere Antwort überschreibt die neuere nicht', async () => {
		vi.mocked(ladeAbgaenger).mockResolvedValueOnce(liste(abgaenger('a', 'Lange')));
		const seite = render(Abgaenger);
		await seite.findByText('Mia Lange');

		/** @type {(antwort: any) => void} */
		let aeltereAntwort = () => {};
		vi.mocked(ladeAbgaenger).mockReturnValueOnce(
			new Promise((fertig) => (aeltereAntwort = fertig))
		);
		leitung.sende('action', { event: 'rueckgabe' });
		vi.mocked(ladeAbgaenger).mockResolvedValueOnce(liste());
		leitung.sende('action', { event: 'rueckgabe' });
		await seite.findByText('Alle Abgänger entlastet!');

		aeltereAntwort(liste(abgaenger('a', 'Lange')));
		await new Promise((fertig) => setTimeout(fertig));

		expect(seite.queryByText('Mia Lange')).toBeNull();
		expect(seite.getByText('Alle Abgänger entlastet!')).toBeTruthy();
	});

	it('eine späte Absage nennt keinen Fehler, wenn die jüngere Anfrage kam', async () => {
		vi.mocked(ladeAbgaenger).mockResolvedValueOnce(liste(abgaenger('a', 'Lange')));
		const seite = render(Abgaenger);
		await seite.findByText('Mia Lange');

		/** @type {(grund: Error) => void} */
		let aeltereScheitert = () => {};
		vi.mocked(ladeAbgaenger).mockReturnValueOnce(
			new Promise((_, nein) => (aeltereScheitert = nein))
		);
		leitung.sende('action', { event: 'rueckgabe' });
		vi.mocked(ladeAbgaenger).mockResolvedValueOnce(liste(abgaenger('b', 'Neumann')));
		leitung.sende('action', { event: 'rueckgabe' });
		await seite.findByText('Mia Neumann');

		aeltereScheitert(new Error('Probe'));
		await new Promise((fertig) => setTimeout(fertig));

		expect(seite.getByText('Mia Neumann')).toBeTruthy();
		expect(snackbars.toasts).toHaveLength(0);
	});

	// An der Liste wird gerade gearbeitet: Ein Kasten „nicht geladen" anstelle der Liste nähme sie
	// weg, nur weil das Auffrischen scheiterte.
	it('scheitert das Auffrischen, bleibt die Liste stehen, und die Leitung holt nach', async () => {
		vi.mocked(ladeAbgaenger).mockResolvedValueOnce(liste(abgaenger('a', 'Lange')));
		const seite = render(Abgaenger);
		await seite.findByText('Mia Lange');

		vi.mocked(ladeAbgaenger).mockRejectedValueOnce(new Error('Datenbank kurz weg'));
		expect(await holeNeuAuf('action', { event: 'rueckgabe' })).toBe(false);

		expect(seite.getByText('Mia Lange')).toBeTruthy();
		expect(seite.queryByRole('alert')).toBeNull();
		expect(snackbars.toasts.map((t) => [t.message, t.type])).toEqual([
			['Die Liste wurde nicht aktualisiert: Datenbank kurz weg', 'error']
		]);

		vi.mocked(ladeAbgaenger).mockResolvedValueOnce(liste());
		expect(await holeNeuAuf('connected')).toBe(true);
		expect(await seite.findByText('Alle Abgänger entlastet!')).toBeTruthy();
	});

	it('fehlt das Recht, geht auch die Liste von vorhin', async () => {
		vi.mocked(ladeAbgaenger).mockResolvedValueOnce(liste(abgaenger('a', 'Lange')));
		const seite = render(Abgaenger);
		await seite.findByText('Mia Lange');

		vi.mocked(ladeAbgaenger).mockRejectedValueOnce(
			Object.assign(new Error('Keine Berechtigung'), { status: 403 })
		);
		await holeNeuAuf('action', { event: 'rueckgabe' });

		expect((await seite.findByRole('alert')).textContent).toContain('Keine Berechtigung');
		expect(seite.queryByText('Mia Lange')).toBeNull();
		expect(snackbars.toasts).toHaveLength(0);
	});

	it('scheitert der erste Abruf, nennt der Kasten den Grund, keine Meldung', async () => {
		vi.mocked(ladeAbgaenger).mockRejectedValueOnce(new Error('Datenbank kurz weg'));
		const seite = render(Abgaenger);

		const kasten = await seite.findByRole('alert');
		expect(kasten.textContent).toContain('Abgängerliste nicht geladen');
		expect(kasten.textContent).toContain('Datenbank kurz weg');
		expect(snackbars.toasts).toHaveLength(0);
	});
});
