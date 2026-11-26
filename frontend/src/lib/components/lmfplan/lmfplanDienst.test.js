import { describe, it, expect, vi } from 'vitest';
import {
	artErklaerung,
	entwurfAus,
	ladePdf,
	ladePlan,
	ladeStand,
	speicherePlan,
	veroeffentlichePlan
} from './lmfplanDienst.js';
import { apiFetch } from '../../apiFetch.js';
import { serverAntwort } from '../../serverAntwort.js';

vi.mock('../../apiFetch.js', async (importOriginal) => ({
	.../** @type {any} */ (await importOriginal()),
	apiFetch: vi.fn()
}));

/** @typedef {import('./lmfplanDienst.js').PlanStand} PlanStand */

describe('entwurfAus: „Nicht im Plan“', () => {
	it('sortiert nach Jahrgang als Zahl, nicht nach dem ersten Zeichen', () => {
		/** @type {PlanStand} */
		const stand = {
			plan: null,
			zeilen: [],
			ausgelassen: [],
			vorbei: false,
			vorschlag: { quelle: 'regel', zeilen: [], ausgelassen: ['10R1', '9H2', '5F'] },
			klassen: ['12T', '7G1']
		};

		// Als Text sortiert stünde „10R1“ vor „5F“.
		expect(entwurfAus(stand).ausgelassen).toEqual(['5F', '7G1', '9H2', '10R1', '12T']);
	});

	it('lässt die Liste des Vorschlags unverändert stehen', () => {
		const ausgelassen = ['9H2', '5F'];
		/** @type {PlanStand} */
		const stand = {
			plan: null,
			zeilen: [],
			ausgelassen: [],
			vorbei: false,
			vorschlag: { quelle: 'regel', zeilen: [], ausgelassen },
			klassen: []
		};

		entwurfAus(stand);

		expect(ausgelassen).toEqual(['9H2', '5F']);
	});
});

describe('artErklaerung', () => {
	it('zählt die Eingangsjahrgänge mit Komma und „und“ auf', () => {
		expect(artErklaerung('ausgabe', [5])).toContain('Klassen in Jahrgang 5 bekommen');
		expect(artErklaerung('ausgabe', [5, 7])).toContain('in Jahrgang 5 und 7 bekommen');
		expect(artErklaerung('ausgabe', [5, 7, 11])).toContain('in Jahrgang 5, 7 und 11 bekommen');
	});
});

// Lehnt der Server ab, steht sein Grund in der Meldung, etwa ein fehlendes Recht; der eigene Satz
// bleibt für eine Antwort ohne Rumpf.
describe('Abrufe des Plans: der Grund des Servers', () => {
	it.each([
		['ladePlan', () => ladePlan()],
		['ladeStand', () => ladeStand('rueckgabe')],
		['ladePdf', () => ladePdf()]
	])('%s nennt den Grund, den der Server angibt', async (_name, abruf) => {
		vi.mocked(apiFetch).mockResolvedValueOnce(
			serverAntwort(403, { error: 'Keine Berechtigung für diese Aktion' })
		);

		await expect(abruf()).rejects.toThrow('Keine Berechtigung für diese Aktion');
	});

	it('nennt ohne Rumpf den eigenen Satz', async () => {
		vi.mocked(apiFetch).mockResolvedValueOnce(serverAntwort(502));

		await expect(ladeStand('rueckgabe')).rejects.toThrow('LMF-Plan konnte nicht geladen werden');
	});
});

// Ein veröffentlichter Rückgabe-Plan verschiebt die Fristen der Klassen; die Meldung nennt, wie viele
// Ausleihen das traf, und schweigt, wenn es keine waren.
// Ein Entwurf heißt Entwurf: Portal und PDF zeigen ihn erst nach dem Veröffentlichen. Ein
// veröffentlichter Plan nennt beim Speichern die Fristen, die ihm gefolgt sind.
describe('speicherePlan: die Meldung', () => {
	/** @type {any} */
	const entwurf = { zeilen: [] };

	it('nennt einen unveröffentlichten Plan einen Entwurf', async () => {
		vi.mocked(apiFetch).mockResolvedValueOnce(
			serverAntwort(200, { plan: { veroeffentlicht_am: null } })
		);
		expect(await speicherePlan('rueckgabe', entwurf)).toEqual({
			ok: true,
			meldung: 'Entwurf gespeichert'
		});
	});

	it('nennt beim veröffentlichten Plan die angepassten Fristen', async () => {
		vi.mocked(apiFetch).mockResolvedValueOnce(
			serverAntwort(200, {
				plan: { veroeffentlicht_am: '2026-11-20T10:00:00Z' },
				fristen_angepasst: 2
			})
		);
		expect(await speicherePlan('rueckgabe', entwurf)).toEqual({
			ok: true,
			meldung: 'Plan gespeichert · Frist von 2 Ausleihen angepasst'
		});
	});
});

describe('veroeffentlichePlan: die Meldung', () => {
	it('nennt die Zahl der angepassten Fristen', async () => {
		vi.mocked(apiFetch).mockResolvedValueOnce(serverAntwort(200, { fristen_angepasst: 3 }));

		expect(await veroeffentlichePlan('rueckgabe')).toEqual({
			ok: true,
			meldung: 'Plan veröffentlicht · Frist von 3 Ausleihen angepasst'
		});
	});

	it('ohne angepasste Frist nur die Veröffentlichung', async () => {
		vi.mocked(apiFetch).mockResolvedValueOnce(serverAntwort(200, { fristen_angepasst: 0 }));

		expect(await veroeffentlichePlan('ausgabe')).toEqual({
			ok: true,
			meldung: 'Plan veröffentlicht.'
		});
	});
});
