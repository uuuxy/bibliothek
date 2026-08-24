import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/svelte';
import { tick } from 'svelte';
import StudentIdDesigner from './StudentIdDesigner.svelte';
import { apiFetch } from './apiFetch.js';
import { idStore } from './designer/idDesignerStore.svelte.js';

vi.mock('./apiFetch.js', () => ({ apiFetch: vi.fn() }));

// Raster-Fund 24.08.2026: Der Auto-Save ist auf 800 ms entprellt, und der Effekt-Abbau
// löscht den Timer. Beim Tippen ist das die Entprellung selbst — beim VERLASSEN des
// Bildschirms verwarf es die letzte Änderung: Die Anzeige stand noch auf „Speichert…",
// gespeichert wurde nie, und der nächste Ladevorgang holte den alten Stand zurück.
// Besonders tückisch, weil das Design zentral für alle Arbeitsplätze gilt: Der Verlust
// fällt erst auf, wenn ein ANDERER Arbeitsplatz den alten Stand druckt.

/** @param {boolean} ok @param {any} daten */
const antwort = (ok, daten) => /** @type {any} */ ({ ok, json: async () => daten });

/** Microtasks (Laden → applyDesign → Effekt-Neuplanung) durchlaufen lassen. */
const stillhalten = async () => {
	await tick();
	await new Promise((fertig) => setTimeout(fertig, 0));
	await tick();
};

describe('StudentIdDesigner: Auto-Save beim Verlassen', () => {
	it('schickt eine noch ausstehende Änderung sofort ab, statt sie zu verwerfen', async () => {
		vi.mocked(apiFetch).mockImplementation(async (url, optionen) => {
			if (optionen?.method === 'PUT') return antwort(true, { status: 'gespeichert' });
			if (String(url).includes('einstellungen')) return antwort(false, {});
			return antwort(true, {}); // GET /api/ausweis-layout: Erststart → Defaults
		});

		const { unmount } = render(StudentIdDesigner);
		await stillhalten();

		// Eine Änderung, wie sie der Farb-Umschalter macht — und dann sofort weg,
		// lange vor Ablauf der 800 ms.
		idStore.front.theme = 'bg-white text-black border-slate-200 FLUSH-MARKE';
		await stillhalten();
		unmount();

		const put = vi.mocked(apiFetch).mock.calls.find(([, optionen]) => optionen?.method === 'PUT');
		expect(put, 'beim Verlassen muss der ausstehende Stand gespeichert werden').toBeTruthy();
		expect(String(put?.[1]?.body)).toContain('FLUSH-MARKE');
	});
});
