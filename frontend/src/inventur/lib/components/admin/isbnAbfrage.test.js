import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('../../../../lib/apiFetch.js', () => ({ apiFetch: vi.fn() }));
vi.mock('../../../../lib/stores/bestaetigung.svelte.js', () => ({ bestaetigen: vi.fn() }));
vi.mock('$lib/store.svelte.js', () => ({ appState: { bookToEdit: null }, showToast: vi.fn() }));

import { apiFetch } from '../../../../lib/apiFetch.js';
import { bestaetigen } from '../../../../lib/stores/bestaetigung.svelte.js';
import { erzeugeIsbnAbfrage } from './isbnAbfrage.svelte.js';

const A = '9783791504650';
const B = '9783551551672';
/** Was die Katalogdienste je ISBN kennen; B nennt nur den Titel. */
const DIENSTE = {
	[A]: {
		title: 'Buch A',
		subtitle: 'Untertitel A',
		author: 'Autor A',
		verlag: 'Verlag A',
		jahr: '2001',
		grade: '7',
		preis: 12.5
	},
	[B]: { title: 'Buch B', preis: 0 }
};

/** @param {number} status @param {any} koerper */
const antwort = (status, koerper) =>
	/** @type {any} */ ({ ok: status < 400, status, json: async () => koerper });
/** @param {string} isbn */
const abfragenZu = (isbn) =>
	vi.mocked(apiFetch).mock.calls.filter(([url]) => String(url) === `/api/lookup/${isbn}`).length;

/** Eine neue Maske, in der zur ISBN A schon abgefragt ist. */
async function nachAbfrageA() {
	const formular = /** @type {any} */ ({ id: null, isbn: A, title: '', author: '', gradeLevel: 5 });
	const abfrage = erzeugeIsbnAbfrage(
		() => formular,
		() => undefined
	);
	await abfrage.nachschlagen(false);
	expect(formular).toMatchObject({ title: 'Buch A', author: 'Autor A', gradeLevel: 7 });
	return { formular, abfrage };
}

beforeEach(() => {
	vi.clearAllMocks();
	vi.mocked(apiFetch).mockImplementation(async (url) => {
		const u = String(url);
		if (!u.startsWith('/api/lookup/')) return antwort(200, { data: { vorhanden: null } });
		const daten = DIENSTE[u.slice('/api/lookup/'.length)];
		return daten ? antwort(200, { data: daten }) : antwort(404, {});
	});
});

// Die ISBN ändert sich in derselben Maske, etwa durch einen zweiten Scan: Was die erste
// Abfrage eingetragen hat, gehört zum ersten Buch und darf nicht unter der zweiten Nummer
// stehen bleiben. Was jemand von Hand geändert hat, bleibt.
describe('isbnAbfrage: eine andere ISBN in derselben Maske', () => {
	it('ersetzt die Angaben der ersten Abfrage, auch die, die der zweiten fehlen', async () => {
		const { formular, abfrage } = await nachAbfrageA();

		formular.isbn = B;
		await abfrage.nachschlagen(false);

		expect(formular.title).toBe('Buch B');
		expect([formular.author, formular.verlag, formular.erscheinungsjahr]).toEqual([
			'',
			undefined,
			undefined
		]);
		expect(formular.gradeLevel, 'die Vorgabe der Maske').toBe(5);
	});

	it('lässt stehen, was jemand nach der Abfrage geändert hat', async () => {
		const { formular, abfrage } = await nachAbfrageA();
		formular.author = 'Von Hand';

		formular.isbn = B;
		await abfrage.nachschlagen(false);

		expect([formular.title, formular.author]).toEqual(['Buch B', 'Von Hand']);
	});

	it('ein von Hand geänderter Titel bleibt, und zur neuen ISBN wird nicht geladen', async () => {
		const { formular, abfrage } = await nachAbfrageA();
		formular.title = 'Mein Titel';

		formular.isbn = B;
		await abfrage.nachschlagen(false);

		expect([formular.title, formular.author]).toEqual(['Mein Titel', '']);
		expect(abfragenZu(B)).toBe(0);
	});

	it('kennt die zweite ISBN niemand, stehen die Angaben der ersten nicht mehr da', async () => {
		const { formular, abfrage } = await nachAbfrageA();

		formular.isbn = '9783000000003';
		await abfrage.nachschlagen(false);

		expect([formular.title, formular.author, formular.gradeLevel]).toEqual(['', '', 5]);
		expect(abfrage.ausgang?.text).toContain('nichts bekannt');
	});

	it('der Knopf zur selben ISBN nimmt nichts zurück; danach gilt der Stand vor der ersten Abfrage', async () => {
		const { formular, abfrage } = await nachAbfrageA();

		await abfrage.nachschlagen(true);
		expect(formular.title).toBe('Buch A');
		expect(abfragenZu(A)).toBe(2);

		formular.isbn = '9783000000003';
		await abfrage.nachschlagen(false);
		expect([formular.title, formular.author, formular.gradeLevel]).toEqual(['', '', 5]);
	});

	it('zeigt die Maske inzwischen einen anderen Titel, bleibt er unberührt', async () => {
		/** @type {any} */
		let inDerMaske = { id: null, isbn: A, title: '', author: '' };
		const abfrage = erzeugeIsbnAbfrage(
			() => inDerMaske,
			() => undefined
		);
		await abfrage.nachschlagen(false);

		// Derselbe Autor wie in der ersten Abfrage: Er gehört hier zum vorhandenen Titel.
		inDerMaske = { id: 'titel-2', isbn: B, title: 'Vorhandener Titel', author: 'Autor A' };
		await abfrage.nachschlagen(false);

		expect(inDerMaske).toEqual({
			id: 'titel-2',
			isbn: B,
			title: 'Vorhandener Titel',
			author: 'Autor A'
		});
	});
});

// Die Katalogdienste antworten nach Sekunden, und in der Zeit wird weiter gescannt und getippt.
// Ein zweiter Scan ersetzt die ISBN, und seine Eingabetaste schließt sich dem laufenden Ablauf
// an: Die Antwort zur ersten ISBN gehört zum ersten Buch und darf nicht unter der zweiten
// Nummer stehen. Ein Feld, in das inzwischen jemand geschrieben hat, überschreibt sie nicht.
describe('isbnAbfrage: Eingaben, während die Abfrage läuft', () => {
	/** Hält die Antwort der Katalogdienste zur ISBN A an, bis der Test sie freigibt. */
	function haltAn() {
		let gibFrei = () => {};
		const warte = new Promise((r) => (gibFrei = () => r(undefined)));
		const sonst = /** @type {any} */ (vi.mocked(apiFetch).getMockImplementation());
		vi.mocked(apiFetch).mockImplementation(async (url) => {
			if (String(url) === `/api/lookup/${A}`) await warte;
			return sonst(url);
		});
		return gibFrei;
	}
	const neueMaske = () => /** @type {any} */ ({ id: null, isbn: A, title: '', author: '' });

	it('zeigt das zweite Buch und fragt zu seiner ISBN', async () => {
		const gibFrei = haltAn();
		const formular = neueMaske();
		const abfrage = erzeugeIsbnAbfrage(
			() => formular,
			() => undefined
		);

		const erster = abfrage.nachschlagen(false);
		await vi.waitFor(() => expect(abfragenZu(A)).toBe(1));
		formular.isbn = B;
		const zweiter = abfrage.nachschlagen(false);
		gibFrei();
		await Promise.all([erster, zweiter]);

		expect(formular).toMatchObject({ isbn: B, title: 'Buch B', author: '' });
		expect(abfragenZu(B)).toBe(1);
	});

	it('eine geleerte ISBN bekommt die Angaben nicht', async () => {
		const gibFrei = haltAn();
		const formular = neueMaske();
		const abfrage = erzeugeIsbnAbfrage(
			() => formular,
			() => undefined
		);

		const lauf = abfrage.nachschlagen(false);
		await vi.waitFor(() => expect(abfragenZu(A)).toBe(1));
		formular.isbn = '';
		gibFrei();
		await lauf;

		expect(formular).toMatchObject({ isbn: '', title: '', author: '' });
	});

	// Dieselbe Regel wie beim Listenpreis: Was jemand eingetragen hat, bleibt.
	it('ein Feld, das während der Abfrage getippt wird, bleibt; die übrigen füllt die Antwort', async () => {
		const gibFrei = haltAn();
		const formular = neueMaske();
		const abfrage = erzeugeIsbnAbfrage(
			() => formular,
			() => undefined
		);

		const lauf = abfrage.nachschlagen(false);
		await vi.waitFor(() => expect(abfragenZu(A)).toBe(1));
		formular.author = 'Von Hand';
		gibFrei();
		await lauf;

		expect(formular).toMatchObject({ title: 'Buch A', author: 'Von Hand', verlag: 'Verlag A' });
	});

	it('der Knopf ersetzt, was vor dem Klick dastand, nicht was danach getippt wird', async () => {
		const gibFrei = haltAn();
		const formular = { ...neueMaske(), title: 'Alter Titel', author: 'Alter Autor' };
		const abfrage = erzeugeIsbnAbfrage(
			() => formular,
			() => undefined
		);

		const lauf = abfrage.nachschlagen(true);
		await vi.waitFor(() => expect(abfragenZu(A)).toBe(1));
		formular.author = 'Von Hand';
		gibFrei();
		await lauf;

		expect(formular).toMatchObject({ title: 'Buch A', author: 'Von Hand' });
	});
});

// Der Server trägt beim Speichern nichts nach: Was die Dienste wissen, steht nach der Abfrage
// in der Maske, auch Untertitel und Listenpreis, und wer speichert, wartet auf sie.
describe('isbnAbfrage: Untertitel, Listenpreis und das Speichern', () => {
	it('trägt Untertitel und Ladenpreis ein', async () => {
		const { formular } = await nachAbfrageA();
		expect([formular.untertitel, formular.listenpreis]).toEqual(['Untertitel A', 12.5]);
	});

	it('ein eingetragener Listenpreis bleibt, auch wenn der Knopf neu lädt', async () => {
		const formular = /** @type {any} */ ({ id: null, isbn: A, title: '', listenpreis: 20 });
		const abfrage = erzeugeIsbnAbfrage(
			() => formular,
			() => undefined
		);

		await abfrage.nachschlagen(true);

		expect([formular.title, formular.listenpreis]).toEqual(['Buch A', 20]);
	});

	it('ein Preis von 0 füllt nichts, und eine andere ISBN nimmt den Preis der ersten zurück', async () => {
		const { formular, abfrage } = await nachAbfrageA();

		formular.isbn = B;
		await abfrage.nachschlagen(false);

		expect([formular.title, formular.untertitel, formular.listenpreis]).toEqual([
			'Buch B',
			undefined,
			undefined
		]);
	});

	it('ruht() wartet auf den laufenden Ablauf und meldet die Frage nach einem vorhandenen Titel', async () => {
		const formular = /** @type {any} */ ({ id: null, isbn: A, title: '' });
		const abfrage = erzeugeIsbnAbfrage(
			() => formular,
			() => undefined
		);
		expect(await abfrage.ruht(), 'ohne Ablauf').toBe(false);

		abfrage.nachschlagen(false);
		expect(formular.title, 'noch unterwegs').toBe('');
		expect(await abfrage.ruht()).toBe(false);
		expect(formular.title).toBe('Buch A');

		vi.mocked(apiFetch).mockImplementation(async () =>
			antwort(200, { data: { vorhanden: { id: 't1', title: 'Buch A' }, meldung: 'Vergeben.' } })
		);
		formular.isbn = B;
		abfrage.nachschlagen(false);
		expect(await abfrage.ruht(), 'die ISBN ist vergeben').toBe(true);
	});
});

// Der Katalog trägt die ISBN in der anderen Länge. „Anderes Buch" heißt: weiter in dieser
// Maske, mit den Angaben der Katalogdienste — und zu dieser ISBN keine zweite Frage.
describe('isbnAbfrage: die ISBN steht in der anderen Länge im Katalog', () => {
	it('nach „Anderes Buch" lädt die Maske die Angaben und fragt nicht noch einmal', async () => {
		vi.mocked(apiFetch).mockImplementation(async (url) => {
			const u = String(url);
			if (u.startsWith('/api/lookup/')) return antwort(200, { data: DIENSTE[A] });
			return antwort(200, {
				data: {
					vorhanden: null,
					andereForm: { id: 't-9', title: 'Aus Littera', ohneExemplar: false, isbn: '3791504657' },
					meldung: 'Im Katalog steht diese ISBN in zehnstelliger Form.'
				}
			});
		});
		vi.mocked(bestaetigen).mockResolvedValue(false);
		const formular = /** @type {any} */ ({ id: null, isbn: A, title: '' });
		const abfrage = erzeugeIsbnAbfrage(
			() => formular,
			() => undefined
		);

		expect(await abfrage.nachschlagen(false), 'die Maske führt nicht zum anderen Titel').toBe(
			false
		);
		expect(formular.title).toBe('Buch A');

		formular.title = '';
		await abfrage.nachschlagen(false);
		expect(bestaetigen).toHaveBeenCalledTimes(1);
		expect(formular.title).toBe('Buch A');
	});
});
