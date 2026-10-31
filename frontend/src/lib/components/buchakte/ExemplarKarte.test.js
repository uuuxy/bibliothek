import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, waitFor, cleanup } from '@testing-library/svelte';

vi.mock('../../apiFetch.js', async (original) => ({
	...(await original()),
	apiFetch: vi.fn(),
	apiClient: { put: vi.fn() }
}));

import { apiClient, apiFetch } from '../../apiFetch.js';
import ExemplarKarte from './ExemplarKarte.svelte';
import { serverAntwort } from '../../serverAntwort.js';

// Was nicht zum Bestand zählt, heißt nicht „Gesperrt". Ein ausgesondertes Exemplar hat keinen
// Papierkorb: Die Tür weist es als schon ausgebucht ab.
describe('Exemplarkarte: Zustand', () => {
	/** @param {Record<string, any>} ex */
	const karte = (ex) =>
		render(ExemplarKarte, {
			ex: { id: 'e1', barcode_id: 'B-0041873', ist_verfuegbar: true, ...ex },
			gewaehlt: false,
			darfBearbeiten: true,
			darfLoeschen: true,
			onToggleSelect: () => {},
			onDelete: () => {}
		});
	/** @param {ReturnType<typeof karte>} k */
	const wort = (k) => k.container.querySelector('[data-chip]')?.textContent?.trim();

	it('nennt ein bestelltes Exemplar bestellt und bietet das Löschen an', () => {
		const k = karte({ ist_ausleihbar: false, ist_ausgesondert: false, im_bestand: false });

		expect(wort(k)).toBe('Bestellt');
		expect(k.queryByRole('button', { name: 'Exemplar löschen' })).not.toBeNull();
	});

	it('nennt ein ausgesondertes Exemplar ausgesondert und bietet kein Löschen an', () => {
		const k = karte({ ist_ausleihbar: false, ist_ausgesondert: true, im_bestand: false });

		expect(wort(k)).toBe('Ausgesondert');
		expect(k.queryByRole('button', { name: 'Exemplar löschen' })).toBeNull();
		expect(k.queryByRole('button', { name: 'Status ändern' })).not.toBeNull();
	});

	it('nennt ein nicht ausleihbares Exemplar im Bestand gesperrt', () => {
		const k = karte({ ist_ausleihbar: false, ist_ausgesondert: false, im_bestand: true });

		expect(wort(k)).toBe('Gesperrt');
	});
});

// Ein Helfer darf den Bestand lesen. Stift und Papierkorb brächten ihm nur eine Ablehnung des Servers.
describe('Exemplarkarte: Knöpfe nach Rechten', () => {
	const ex = { id: 'e1', barcode_id: 'B-1', ist_ausleihbar: true, ist_verfuegbar: true };
	const karte = (/** @type {Record<string, boolean>} */ rechte) =>
		render(ExemplarKarte, {
			ex,
			gewaehlt: false,
			...rechte,
			onToggleSelect: vi.fn(),
			onDelete: vi.fn()
		});
	const knopf = (/** @type {any} */ k, /** @type {string} */ name) =>
		k.queryByRole('button', { name }) !== null;

	it('zeigt ohne Rechte nur die Angaben', () => {
		const k = karte({});
		expect(knopf(k, 'Status ändern')).toBe(false);
		expect(knopf(k, 'Exemplar löschen')).toBe(false);
		expect(knopf(k, 'Barcode zuweisen oder ändern')).toBe(false);
		expect(k.getByText('B-1')).toBeTruthy();
	});

	it('bietet das Ändern mit edit_books an, das Löschen erst mit delete_books', () => {
		const nurAendern = karte({ darfBearbeiten: true });
		expect(knopf(nurAendern, 'Status ändern')).toBe(true);
		expect(knopf(nurAendern, 'Exemplar löschen')).toBe(false);
		cleanup();

		const nurLoeschen = karte({ darfLoeschen: true });
		expect(knopf(nurLoeschen, 'Status ändern')).toBe(false);
		expect(knopf(nurLoeschen, 'Exemplar löschen')).toBe(true);
	});
});

// Lehnt der Server einen Barcode ab, steht sein Grund ungekürzt am Feld und ist dessen
// Beschreibung: Ein Screenreader liest ihn mit dem Feld vor.
describe('Exemplarkarte: abgelehnter Barcode', () => {
	const grund = 'dieser Barcode wird bereits von einem anderen Exemplar verwendet';

	const karte = () =>
		render(ExemplarKarte, {
			ex: { id: 'e1', barcode_id: 'AUTO-000123', ist_ausleihbar: true, ist_verfuegbar: true },
			gewaehlt: false,
			darfBearbeiten: true,
			darfLoeschen: true,
			onToggleSelect: () => {},
			onDelete: () => {}
		});

	/** @param {ReturnType<typeof karte>} k */
	async function trageEin(k) {
		await fireEvent.click(k.getByRole('button', { name: 'Barcode scannen' }));
		const feld = /** @type {HTMLInputElement} */ (k.getByLabelText('Barcode des Exemplars'));
		await fireEvent.input(feld, { target: { value: 'B-0041873' } });
		await fireEvent.keyDown(feld, { key: 'Enter' });
		return feld;
	}

	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('nennt den Grund des Servers als Beschreibung des Felds', async () => {
		vi.mocked(apiClient.put).mockResolvedValue(serverAntwort(409, { error: grund }));
		const feld = await trageEin(karte());

		await waitFor(() => expect(feld.getAttribute('aria-invalid')).toBe('true'));
		const beschreibung = document.getElementById(feld.getAttribute('aria-describedby') ?? '');
		expect(beschreibung?.textContent).toBe(grund);
		expect(beschreibung?.hasAttribute('title'), 'der Grund steht nur in einer Sprechblase').toBe(
			false
		);
	});

	// Die Gegenprobe: Ein angenommener Barcode schließt das Feld und hinterlässt keinen Fehler.
	it('zeigt nach einem angenommenen Barcode die Nummer und keinen Fehler', async () => {
		vi.mocked(apiClient.put).mockResolvedValue(/** @type {any} */ ({ ok: true }));
		const k = karte();
		await trageEin(k);

		await waitFor(() => expect(k.queryByLabelText('Barcode des Exemplars')).toBeNull());
		expect(k.container.textContent).toContain('B-0041873');
		expect(k.container.textContent).not.toContain(grund);
	});
});

// Ein Klick auf „Speichern" bei leerem Feld nennt am Feld, was fehlt, führt dorthin und schickt
// nichts ab. Ein Fehler des Servers gilt nur für den Wert, den er abgelehnt hat.
describe('Exemplarkarte: leerer Barcode', () => {
	const karte = () =>
		render(ExemplarKarte, {
			ex: { id: 'e1', barcode_id: 'AUTO-000123', ist_ausleihbar: true, ist_verfuegbar: true },
			gewaehlt: false,
			darfBearbeiten: true,
			darfLoeschen: true,
			onToggleSelect: () => {},
			onDelete: () => {}
		});

	/** @param {HTMLElement} feld */
	const beschreibung = (feld) =>
		document.getElementById(feld.getAttribute('aria-describedby') ?? '')?.textContent;

	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('nennt am Feld, was fehlt, statt den Klick zu schlucken', async () => {
		const k = karte();
		await fireEvent.click(k.getByRole('button', { name: 'Barcode scannen' }));
		const feld = k.getByLabelText('Barcode des Exemplars');
		const speichern = k.getByRole('button', { name: 'Speichern' });
		speichern.focus();
		await fireEvent.click(speichern);

		expect(feld.getAttribute('aria-invalid')).toBe('true');
		expect(beschreibung(feld)).toBe(
			'Bitte einen Barcode scannen oder eintragen. Gespeichert wird erst mit ihm.'
		);
		expect(document.activeElement).toBe(feld);
		expect(apiClient.put).not.toHaveBeenCalled();

		await fireEvent.input(feld, { target: { value: 'B-0041873' } });
		expect(feld.getAttribute('aria-invalid')).toBeNull();
	});

	it('nimmt den Grund des Servers weg, sobald eine interne ID im Feld steht', async () => {
		vi.mocked(apiClient.put).mockResolvedValue(serverAntwort(409, { error: 'schon vergeben' }));
		vi.mocked(apiFetch).mockResolvedValue(
			/** @type {any} */ ({ ok: true, json: async () => ({ next_barcode: 'B-0099999' }) })
		);
		const k = karte();
		await fireEvent.click(k.getByRole('button', { name: 'Barcode scannen' }));
		const feld = /** @type {HTMLInputElement} */ (k.getByLabelText('Barcode des Exemplars'));
		await fireEvent.input(feld, { target: { value: 'B-0041873' } });
		await fireEvent.click(k.getByRole('button', { name: 'Speichern' }));
		await waitFor(() => expect(beschreibung(feld)).toBe('schon vergeben'));

		await fireEvent.click(k.getByRole('button', { name: 'Interne ID generieren' }));

		await waitFor(() => expect(feld.value).toBe('B-0099999'));
		expect(feld.getAttribute('aria-invalid')).toBeNull();
	});
});

// Ein zweiter Scan, bevor der Server geantwortet hat, hängt sich an den Feldinhalt. Sein Enter
// schickt nichts ab: Gespeichert würde sonst „B-0041873B-0041873", und das Etikett fände an der
// Theke niemand.
describe('Exemplarkarte: zweiter Scan während des Speicherns', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('schickt den Barcode einmal und nimmt bis zur Antwort keinen zweiten an', async () => {
		/** @type {(antwort: any) => void} */
		let antworte = () => {};
		vi.mocked(apiClient.put).mockReturnValue(
			/** @type {any} */ (new Promise((r) => (antworte = r)))
		);
		const k = render(ExemplarKarte, {
			ex: { id: 'e1', barcode_id: 'AUTO-000123', ist_ausleihbar: true, ist_verfuegbar: true },
			gewaehlt: false,
			darfBearbeiten: true,
			darfLoeschen: true,
			onToggleSelect: () => {},
			onDelete: () => {}
		});
		await fireEvent.click(k.getByRole('button', { name: 'Barcode scannen' }));
		const feld = k.getByLabelText('Barcode des Exemplars');
		await fireEvent.input(feld, { target: { value: 'B-0041873' } });
		await fireEvent.keyDown(feld, { key: 'Enter' });

		await fireEvent.input(feld, { target: { value: 'B-0041873B-0041873' } });
		await fireEvent.keyDown(feld, { key: 'Enter' });
		await fireEvent.click(k.getByRole('button', { name: 'Speichern' }));

		expect(apiClient.put).toHaveBeenCalledTimes(1);
		expect(apiClient.put).toHaveBeenCalledWith('/api/buecher/exemplare/e1/barcode', {
			barcode: 'B-0041873'
		});
		expect(k.getByRole('button', { name: 'Speichern' }).hasAttribute('disabled')).toBe(true);

		antworte({ ok: true });
		await waitFor(() => expect(k.queryByLabelText('Barcode des Exemplars')).toBeNull());
		expect(k.container.textContent).toContain('B-0041873');
		expect(k.container.textContent).not.toContain('B-0041873B-0041873');
	});

	// Lehnt der Server ab, ist der Knopf wieder frei: Die Sperre gilt nur, solange gespeichert wird.
	it('gibt das Speichern nach einer Ablehnung wieder frei', async () => {
		vi.mocked(apiClient.put).mockResolvedValue(serverAntwort(409, { error: 'schon vergeben' }));
		const k = render(ExemplarKarte, {
			ex: { id: 'e1', barcode_id: 'AUTO-000123', ist_ausleihbar: true, ist_verfuegbar: true },
			gewaehlt: false,
			darfBearbeiten: true,
			darfLoeschen: true,
			onToggleSelect: () => {},
			onDelete: () => {}
		});
		await fireEvent.click(k.getByRole('button', { name: 'Barcode scannen' }));
		const feld = k.getByLabelText('Barcode des Exemplars');
		await fireEvent.input(feld, { target: { value: 'B-0041873' } });
		await fireEvent.keyDown(feld, { key: 'Enter' });
		await waitFor(() => expect(feld.getAttribute('aria-invalid')).toBe('true'));

		await fireEvent.input(feld, { target: { value: 'B-0041874' } });
		await fireEvent.keyDown(feld, { key: 'Enter' });

		expect(apiClient.put).toHaveBeenCalledTimes(2);
		expect(apiClient.put).toHaveBeenLastCalledWith('/api/buecher/exemplare/e1/barcode', {
			barcode: 'B-0041874'
		});
	});
});
