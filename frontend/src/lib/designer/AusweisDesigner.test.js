import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render } from '@testing-library/svelte';
import { tick } from 'svelte';
import AusweisDesigner from './AusweisDesigner.svelte';
import { apiFetch } from '../apiFetch.js';
import { authStore } from '../stores/authStore.svelte.js';
import { idStore, resetDesign } from './idDesignerStore.svelte.js';

vi.mock('../apiFetch.js', () => ({ apiFetch: vi.fn() }));
vi.mock('../stores/authStore.svelte.js', () => ({
	authStore: { currentUser: /** @type {any} */ (null) }
}));

// Raster-Fund 24.08.2026: Der Auto-Save ist auf 800 ms entprellt, und ein neuer Lauf des Effekts
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

const schreibende = () =>
	vi.mocked(apiFetch).mock.calls.filter(([, optionen]) => optionen?.method === 'PUT');

/** Ob die Leinwand Zeiger und Fokus abweist; jsdom spiegelt die Eigenschaft nicht ins Attribut. */
const leinwandTraege = () => {
	/** @type {any} */
	let el = document.querySelector('[data-ausweis-vorschau]');
	if (!el) throw new Error('keine Leinwand im Dokument');
	while (el) {
		if (el.inert === true) return true;
		el = el.parentElement;
	}
	return false;
};

/** Länger als die Entprellung des Auto-Save. */
const entprellungAbwarten = () => new Promise((fertig) => setTimeout(fertig, 900));

beforeEach(() => {
	authStore.currentUser = { rolle: 'admin' };
	resetDesign();
	vi.mocked(apiFetch).mockReset();
	vi.mocked(apiFetch).mockImplementation(async (_url, optionen) =>
		// Das GET antwortet wie beim Erststart: nur der Kopf der Schule, die Leinwand nimmt ihre Vorgaben.
		optionen?.method === 'PUT' ? antwort(true, { status: 'gespeichert' }) : antwort(true, {})
	);
});

describe('AusweisDesigner: Auto-Save nur nach einer Änderung', () => {
	// Das Öffnen schickte das unveränderte Design: Wer es nicht ändern darf, las „Speichern
	// fehlgeschlagen", und bei jedem anderen ging ein Schreibbefehl hinaus, ohne dass jemand
	// etwas angefasst hatte.
	it('schreibt beim Öffnen nichts', async () => {
		render(AusweisDesigner);
		await stillhalten();
		await entprellungAbwarten();

		expect(schreibende()).toEqual([]);
	});

	it('schickt eine Änderung nach der Entprellung einmal', async () => {
		render(AusweisDesigner);
		await stillhalten();

		idStore.front.theme = 'smaragd';
		await stillhalten();
		await entprellungAbwarten();

		expect(schreibende()).toHaveLength(1);
		expect(String(schreibende()[0][1]?.body)).toContain('smaragd');
	});
});

describe('AusweisDesigner: ohne das Recht, die Einstellungen zu verwalten', () => {
	beforeEach(() => {
		authStore.currentUser = { rolle: 'mitarbeiter', permissions: ['view_students'] };
	});

	it('schickt auch nach einer Änderung im Speicher nichts', async () => {
		const { unmount } = render(AusweisDesigner);
		await stillhalten();

		idStore.front.theme = 'smaragd';
		await stillhalten();
		await entprellungAbwarten();
		unmount();

		expect(schreibende()).toEqual([]);
	});

	// Material 3, States: „Disabled components can't be focused, dragged, or pressed."
	it('sperrt, was das Design ändert, und lässt Ansehen und Testdruck offen', async () => {
		const screen = render(AusweisDesigner);
		await stillhalten();
		const knopf = (/** @type {string|RegExp} */ name) =>
			/** @type {HTMLButtonElement} */ (screen.getByRole('button', { name }));

		for (const name of ['+ Text', '+ Fläche', '+ Bild(er)', 'Standardwerte wiederherstellen']) {
			expect(knopf(name).disabled, name).toBe(true);
		}
		expect(knopf('Kartendrucker').disabled).toBe(true);
		expect(knopf('Etikettenbogen').disabled).toBe(true);
		expect(knopf(/Vorlage wählen/).disabled).toBe(true);
		expect(
			/** @type {HTMLButtonElement} */ (screen.getByRole('combobox', { name: 'Barcode-Typ' }))
				.disabled
		).toBe(true);

		expect(knopf(/Testdruck/).disabled).toBe(false);
		expect(knopf(/Rückseite/).disabled).toBe(false);
		expect(
			screen.getByText(/Das Design ändern kann, wer die Einstellungen verwalten darf/)
		).toBeTruthy();
		expect(leinwandTraege()).toBe(true);
	});

	it('zeigt dieselbe Bedienung ungesperrt, sobald das Recht da ist', async () => {
		authStore.currentUser = { rolle: 'leitung', permissions: ['view_students', 'manage_settings'] };
		const screen = render(AusweisDesigner);
		await stillhalten();

		expect(
			/** @type {HTMLButtonElement} */ (screen.getByRole('button', { name: '+ Text' })).disabled
		).toBe(false);
		expect(leinwandTraege()).toBe(false);
	});
});

describe('AusweisDesigner: Auto-Save beim Verlassen', () => {
	it('schickt eine noch ausstehende Änderung sofort ab, statt sie zu verwerfen', async () => {
		const { unmount } = render(AusweisDesigner);
		await stillhalten();

		// Eine Änderung, wie sie die Auswahl des Hintergrunds macht — und dann sofort weg,
		// lange vor Ablauf der 800 ms. Smaragd trägt keine Seite als Vorgabe.
		idStore.front.theme = 'smaragd';
		await stillhalten();
		unmount();

		const put = vi.mocked(apiFetch).mock.calls.find(([, optionen]) => optionen?.method === 'PUT');
		expect(put, 'beim Verlassen muss der ausstehende Stand gespeichert werden').toBeTruthy();
		expect(String(put?.[1]?.body)).toContain('smaragd');
	});
});
