import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor } from '@testing-library/svelte';

vi.mock('../../../../lib/apiFetch.js', () => ({
	apiFetch: vi.fn(),
	apiPut: vi.fn(),
	apiClient: { post: vi.fn() },
	extractApiError: vi.fn(async (/** @type {any} */ res) => `Fehler ${res.status}`)
}));
vi.mock('$lib/store.svelte.js', () => ({ showToast: vi.fn() }));

import { apiFetch } from '../../../../lib/apiFetch.js';
import BuchEingabefelder from './BuchEingabefelder.svelte';
import { erzeugeDnbSchlagwortVorschlag } from '../../../../lib/utils/dnbSchlagwortVorschlag.svelte.js';
import { leeresBuchFormular } from './buch_form_optionen.js';
import { erzeugeIsbnAbfrage } from './isbnAbfrage.svelte.js';

// Die Felder stehen in der Reihenfolge der Arbeit: die ISBN, mit der die Aufnahme beginnt,
// dann die Angaben, die ihre Abfrage bringt, dann unter „An der Schule", was die Schule
// entscheidet. Die Wahl Bibliothek oder Lernmittel steht über allem, was von ihr abhängt.
const TITELANGABEN = [
	'buch-isbn',
	'buch-medientyp',
	'buch-titel',
	'buch-untertitel',
	'buch-autor',
	'buch-verlag',
	'buch-jahr',
	'buch-auflage',
	'buch-listenpreis',
	'buch-schlagworte'
];

beforeEach(() => {
	vi.clearAllMocks();
	vi.mocked(apiFetch).mockResolvedValue(
		/** @type {any} */ ({ ok: true, status: 200, json: async () => [] })
	);
});

/** @param {any} formular */
function maske(formular) {
	const dnbVorschlag = erzeugeDnbSchlagwortVorschlag();
	const abfrage = erzeugeIsbnAbfrage(
		() => formular,
		() => dnbVorschlag
	);
	return render(BuchEingabefelder, { formular, dnbVorschlag, abfrage });
}

/** Die Bedienelemente mit Kennung, wie sie im Dokument aufeinander folgen.
 * @param {HTMLElement} container */
const reihenfolge = (container) =>
	[...container.querySelectorAll('input[id], textarea[id], button[role="combobox"][id]')].map(
		(e) => e.id
	);

describe('BuchEingabefelder: Reihenfolge und Gruppen', () => {
	it('ein Lernmittel: ISBN zuerst, dann die Angaben zum Buch, dann die Schule', async () => {
		const formular = $state({ ...leeresBuchFormular(), id: 't-1', istLernmittel: true });
		const screen = maske(formular);
		await waitFor(() => expect(screen.container.querySelector('#buch-standort')).toBeTruthy());
		expect(reihenfolge(screen.container)).toEqual([
			...TITELANGABEN,
			'buch-signatur',
			'buch-standort',
			'buch-fach',
			'buch-schulzweig',
			'buch-klasse',
			'buch-jahrgang-von',
			'buch-jahrgang-bis',
			'buch-mehrjahresband'
		]);
	});

	it('ein Bibliotheksbuch zeigt weder Schulzweig noch Mehrjahresband', async () => {
		const formular = $state({ ...leeresBuchFormular(), id: 't-1' });
		const screen = maske(formular);
		await waitFor(() => expect(screen.container.querySelector('#buch-standort')).toBeTruthy());
		expect(reihenfolge(screen.container)).toEqual([
			...TITELANGABEN,
			'buch-signatur',
			'buch-standort',
			'buch-fach',
			'buch-klasse',
			'buch-jahrgang-von',
			'buch-jahrgang-bis'
		]);
	});

	// Bestand und Zähldatum stehen bei den Exemplaren (BuchExemplareListe), nicht hier.
	it('führt den Bestand nicht zwischen den Angaben zum Titel', () => {
		const formular = $state({ ...leeresBuchFormular(), id: 't-1' });
		const screen = maske(formular);
		expect(screen.container.querySelector('#buch-bestand')).toBeNull();
		expect(screen.container.querySelector('#buch-zaehldatum')).toBeNull();
	});

	it('trägt eine Überschrift nur über der zweiten Gruppe', () => {
		const formular = $state({ ...leeresBuchFormular(), id: 't-1', istLernmittel: true });
		const screen = maske(formular);
		expect(screen.getAllByRole('heading').map((h) => h.textContent?.trim())).toEqual([
			'An der Schule'
		]);
	});
});

describe('BuchEingabefelder: Bibliothek oder Lernmittel', () => {
	it('die Wahl schaltet, was von ihr abhängt, und nimmt das Mehrjahresband mit zurück', async () => {
		const formular = $state(leeresBuchFormular());
		const screen = maske(formular);
		const wahl = screen.getByRole('group', { name: 'Art des Buchs' });
		const knopf = (/** @type {string} */ name) =>
			/** @type {HTMLElement} */ (
				[...wahl.querySelectorAll('button')].find((b) => b.textContent?.trim() === name)
			);
		expect(knopf('Bibliothek').getAttribute('aria-pressed')).toBe('true');
		expect(screen.queryByLabelText('Schulzweig')).toBeNull();
		expect(screen.queryByRole('checkbox', { name: 'Mehrjahresband' })).toBeNull();

		await fireEvent.click(knopf('Lernmittel'));
		expect(formular.istLernmittel).toBe(true);
		expect(screen.getByLabelText('Schulzweig')).toBeTruthy();
		// Eine Angabe, die mit „Speichern" gilt, ist ein Kästchen und kein Schalter.
		await fireEvent.click(screen.getByRole('checkbox', { name: 'Mehrjahresband' }));
		expect(formular.mehrjahresband).toBe(true);
		expect(screen.queryAllByRole('switch')).toHaveLength(0);

		// Der Server weist das Mehrjahresband am Bibliotheksbuch ab.
		await fireEvent.click(knopf('Bibliothek'));
		expect(formular.istLernmittel).toBe(false);
		expect(formular.mehrjahresband).toBe(false);
		expect(screen.queryByRole('checkbox', { name: 'Mehrjahresband' })).toBeNull();
	});
});

describe('BuchEingabefelder: Signatur', () => {
	it('ist am neuen Bibliotheksbuch Pflicht: Stern an der Beschriftung, Fehler am leeren Feld', async () => {
		const formular = $state(leeresBuchFormular());
		const screen = maske(formular);
		const feld = /** @type {HTMLInputElement} */ (screen.getByLabelText('Signatur (Buchrücken) *'));
		expect(feld.id).toBe('buch-signatur');
		expect(feld.required).toBe(true);
		expect(feld.getAttribute('aria-invalid')).toBe('true');

		await fireEvent.input(feld, { target: { value: 'Sk' } });
		expect(feld.getAttribute('aria-invalid')).toBeNull();
	});

	it('ist am Lernmittel und am vorhandenen Titel frei', async () => {
		const neu = $state({ ...leeresBuchFormular(), istLernmittel: true });
		const lernmittel = maske(neu);
		const feld = /** @type {HTMLInputElement} */ (
			lernmittel.getByLabelText('Signatur (Buchrücken)')
		);
		expect(feld.required).toBe(false);
		expect(feld.getAttribute('aria-invalid')).toBeNull();
		lernmittel.unmount();

		const vorhanden = $state({ ...leeresBuchFormular(), id: 't-1' });
		const alt = maske(vorhanden);
		expect(alt.getByLabelText('Signatur (Buchrücken)')).toBeTruthy();
	});
});
