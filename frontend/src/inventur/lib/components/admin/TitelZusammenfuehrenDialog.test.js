import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

vi.mock('../../../../lib/apiFetch.js', () => ({
	apiFetch: vi.fn(),
	apiClient: { post: vi.fn() },
	leseApiFehler: vi.fn(async (/** @type {any} */ res) => (await res.json()).error)
}));
vi.mock('../../../../lib/stores/snackbars.svelte.js', () => ({
	snackbars: { addToast: vi.fn() }
}));

import { apiFetch, apiClient } from '../../../../lib/apiFetch.js';
import { snackbars } from '../../../../lib/stores/snackbars.svelte.js';
import TitelZusammenfuehrenDialog from './TitelZusammenfuehrenDialog.svelte';

// „Mit anderem Titel zusammenführen": suchen, antippen, eine Zeile Vorschau des Servers,
// bestätigen. Der Titel mit ISBN bleibt. Material 3, Dialogs: „Disable confirming actions until
// a choice is made. Dismissive actions are never disabled."

const dieser = { id: 'gross', title: 'Natur und Technik - Biologie 7/10', isbn: '' };
const anderer = {
	id: 'klein',
	title: 'Natur und Technik - Biologie 7/10',
	isbn: '9783060158423',
	verlag: 'Cornelsen',
	istLernmittel: false,
	gesamt: 1,
	verfuegbar: 1,
	imZulauf: 0
};
// Ein zweiter Treffer ohne ISBN: Mit ihm bleibt dieser Titel.
const ohneISBN = { ...anderer, id: 'liste', title: 'Natur und Technik Biologie 7-10', isbn: '' };
const vorschau = {
	titel_id: 'klein',
	titel: anderer.title,
	isbn: '9783060158423',
	exemplare_danach: 59,
	quelle_id: 'gross',
	quelle_titel: dieser.title,
	quelle_isbn: '',
	exemplare: 58,
	uebernommen: [],
	abweichend: []
};
/** @param {number} status @param {any} koerper @returns {any} */
const antwort = (status, koerper) => ({ ok: status < 400, status, json: async () => koerper });

/** Die Suche findet diesen und den anderen Titel; jede Vorschau bekommt vorschauAntwort. */
function server(vorschauAntwort = antwort(200, vorschau)) {
	vi.mocked(apiFetch).mockImplementation(async (url) => {
		const u = String(url);
		if (u.startsWith('/api/books?q=')) {
			return antwort(200, u.includes('bestand=ohne') ? [] : [anderer, dieser, ohneISBN]);
		}
		if (u.includes('/zusammenfuehren?quelle_id=')) return vorschauAntwort;
		throw new Error(`unerwarteter Aufruf ${u}`);
	});
}

async function bisZurVorschau() {
	const onZusammengefuehrt = vi.fn();
	const screen = render(TitelZusammenfuehrenDialog, {
		props: { open: true, titel: dieser, onZusammengefuehrt }
	});
	await fireEvent.input(screen.getByLabelText('Anderen Titel suchen'), {
		target: { value: 'Natur' }
	});
	await vi.advanceTimersByTimeAsync(250);
	expect(screen.getByRole('button', { name: 'Zusammenführen' }).hasAttribute('disabled')).toBe(
		true
	);
	await fireEvent.click(screen.getByRole('button', { name: /9783060158423/ }));
	await vi.advanceTimersByTimeAsync(0);
	return { screen, onZusammengefuehrt };
}

/** Die Vorschauen, die der Dialog gefragt hat. */
const vorschauen = () =>
	vi
		.mocked(apiFetch)
		.mock.calls.map(([url]) => String(url))
		.filter((u) => u.includes('/zusammenfuehren?'));

beforeEach(() => {
	vi.useFakeTimers();
	vi.mocked(apiFetch).mockReset();
	vi.mocked(apiClient.post).mockReset();
	vi.mocked(snackbars.addToast).mockReset();
});
afterEach(() => {
	vi.useRealTimers();
});

describe('TitelZusammenfuehrenDialog', () => {
	it('bietet jeden anderen Titel an, auch ein Buch der Bücherei, nicht diesen selbst', async () => {
		server();
		const screen = render(TitelZusammenfuehrenDialog, {
			props: { open: true, titel: dieser, onZusammengefuehrt: vi.fn() }
		});
		await fireEvent.input(screen.getByLabelText('Anderen Titel suchen'), {
			target: { value: 'Natur' }
		});
		await vi.advanceTimersByTimeAsync(250);

		const treffer = screen.getAllByRole('button', { name: /Natur und Technik/ });
		expect(treffer.map((t) => t.textContent?.replace(/\s+/g, ' ').trim())).toEqual([
			'Natur und Technik - Biologie 7/10 9783060158423 · 1 Exemplar',
			'Natur und Technik Biologie 7-10 ohne ISBN · 1 Exemplar'
		]);
	});

	it('lässt den Titel mit ISBN bleiben und zeigt in einer Zeile, wie er danach aussieht', async () => {
		server();
		const { screen } = await bisZurVorschau();

		expect(vorschauen()).toEqual(['/api/buecher/titel/klein/zusammenfuehren?quelle_id=gross']);
		expect(screen.getByText(/9783060158423 · 59 Exemplare/)).toBeTruthy();
		expect(screen.getByText(/\(ohne ISBN\) wird gelöscht/)).toBeTruthy();
		expect(screen.getByRole('button', { name: /9783060158423/ }).getAttribute('aria-pressed')).toBe(
			'true'
		);
		expect(screen.getByRole('button', { name: 'Zusammenführen' }).hasAttribute('disabled')).toBe(
			false
		);
	});

	it('lässt diesen Titel bleiben, wenn der andere keine ISBN hat', async () => {
		server();
		const { screen } = await bisZurVorschau();

		await fireEvent.click(screen.getByRole('button', { name: /ohne ISBN/ }));
		await vi.advanceTimersByTimeAsync(0);

		expect(vorschauen().at(-1)).toBe('/api/buecher/titel/gross/zusammenfuehren?quelle_id=liste');
	});

	// Wer am Titel ohne ISBN die ISBN eintippt, scheitert beim Speichern, weil der andere sie trägt,
	// und führt dann zusammen. Die eingetippte ISBN ist verworfen; es bleibt der Titel mit ISBN.
	it('entscheidet nach der gespeicherten ISBN, nicht nach einer eingetippten', async () => {
		server();
		const getippt = { ...dieser, isbn: '9783060158423', geladen: { ...dieser } };
		const screen = render(TitelZusammenfuehrenDialog, {
			props: { open: true, titel: getippt, onZusammengefuehrt: vi.fn() }
		});
		await fireEvent.input(screen.getByLabelText('Anderen Titel suchen'), {
			target: { value: 'Natur' }
		});
		await vi.advanceTimersByTimeAsync(250);
		await fireEvent.click(screen.getByRole('button', { name: /9783060158423/ }));
		await vi.advanceTimersByTimeAsync(0);

		expect(vorschauen()).toEqual(['/api/buecher/titel/klein/zusammenfuehren?quelle_id=gross']);
	});

	it('führt zusammen und meldet den bleibenden Titel mit seiner Zahl', async () => {
		server();
		vi.mocked(apiClient.post).mockResolvedValueOnce(antwort(200, vorschau));
		const { screen, onZusammengefuehrt } = await bisZurVorschau();

		await fireEvent.click(screen.getByRole('button', { name: 'Zusammenführen' }));
		await vi.advanceTimersByTimeAsync(0);

		expect(apiClient.post).toHaveBeenCalledWith('/api/buecher/titel/klein/zusammenfuehren', {
			quelle_id: 'gross'
		});
		expect(onZusammengefuehrt).toHaveBeenCalledWith('klein');
		expect(vi.mocked(snackbars.addToast).mock.calls[0]).toEqual([
			'Zusammengeführt: „Natur und Technik - Biologie 7/10“ hat jetzt 59 Exemplare.',
			'success'
		]);
	});

	// Zwei ISBN sind zwei Ausgaben; der Server lehnt ab, und der Dialog sagt warum.
	it('zeigt die Ablehnung des Servers und lässt „Zusammenführen" gesperrt', async () => {
		server(antwort(409, { error: 'Beide Titel tragen eine ISBN, sind also zwei Ausgaben.' }));
		const { screen } = await bisZurVorschau();

		expect(screen.getByRole('alert').textContent).toContain('Beide Titel tragen eine ISBN');
		expect(screen.getByRole('button', { name: 'Zusammenführen' }).hasAttribute('disabled')).toBe(
			true
		);
		expect(screen.getByRole('button', { name: 'Abbrechen' }).hasAttribute('disabled')).toBe(false);
	});
});
