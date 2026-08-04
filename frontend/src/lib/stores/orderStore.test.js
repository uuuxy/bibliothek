import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

vi.mock('../apiFetch.js', () => ({
	apiGet: vi.fn(async () => []),
	apiPost: vi.fn(async () => ({})),
	apiPut: vi.fn(async () => ({})),
	apiDelete: vi.fn(async () => ({}))
}));
vi.mock('./toastStore.svelte.js', () => ({
	toastStore: { addToast: vi.fn() }
}));

import { apiGet, apiPost } from '../apiFetch.js';
import { orderStore } from './orderStore.svelte.js';

const apiPostMock = vi.mocked(apiPost);

function resetStore() {
	orderStore.cart = [];
	orderStore.suppliers = [];
	orderStore.selectedSupplierId = '';
	orderStore.attachBarcodes = true;
	orderStore.searchQuery = '';
	orderStore.searchResults = [];
	orderStore.showDropdown = false;
}

// Seit dem DNB-Preisvorschlag loest addToCart im Hintergrund eine Suche ueber
// /api/bestellungen/suche aus. Die Zusicherungen sprechen deshalb den Bestell-Endpunkt
// ausdruecklich an, statt sich auf die Aufrufreihenfolge zu verlassen — sonst prueft ein
// Test auf "die Bestellung" und meint in Wahrheit die Preisabfrage.

/** @param {any} mock Wurde eine BESTELLUNG abgesetzt? */
function hatBestellungGesendet(mock) {
	return mock.mock.calls.some((/** @type {any[]} */ args) => args[0] === '/api/bestellungen');
}

/** @param {any} mock Das Payload der abgesetzten BESTELLUNG. */
function bestellPayload(mock) {
	return mock.mock.calls.find((/** @type {any[]} */ args) => args[0] === '/api/bestellungen')?.[1];
}

/**
 * Antwort NUR fuer den Bestell-Endpunkt festlegen. mockResolvedValueOnce waere von der
 * Reihenfolge abhaengig — die Preisabfrage im Hintergrund wuerde sie aufbrauchen.
 * @param {any} antwort @param {boolean} [scheitert]
 */
function bestellAntwort(antwort, scheitert = false) {
	apiPostMock.mockImplementation(async (/** @type {string} */ url) => {
		if (url !== '/api/bestellungen') return [];
		if (scheitert) throw antwort;
		return antwort;
	});
}

describe('orderStore.addToCart', () => {
	beforeEach(() => {
		resetStore();
		vi.clearAllMocks();
	});

	it('legt neue Positionen mit titel_id als Schlüssel an', () => {
		orderStore.addToCart({
			titel_id: 't1',
			id: 'buch-1',
			titel: 'Faust',
			autor: 'Goethe',
			isbn: '978-1'
		});
		expect(orderStore.cart).toHaveLength(1);
		expect(orderStore.cart[0].id).toBe('t1');
	});

	it('dedupliziert, wenn dasselbe Buch einmal mit titel_id und einmal ohne kommt', () => {
		// Suchergebnis liefert titel_id, Empfehlung liefert nur id — früherer Duplikat-Bug
		orderStore.addToCart({ titel_id: 't1', titel: 'Faust', autor: 'Goethe', isbn: '978-1' }, 2);
		orderStore.addToCart({ id: 't1', titel: 'Faust', autor: 'Goethe', isbn: '978-1' }, 3);
		expect(orderStore.cart).toHaveLength(1);
		expect(orderStore.cart[0].menge).toBe(5);
	});

	it('dedupliziert über die ISBN, auch bei abweichender Feld-Schreibung', () => {
		orderStore.addToCart({ id: 'a', titel: 'Faust', autor: 'Goethe', isbn: '978-1' });
		orderStore.addToCart({ id: 'b', titel: 'Faust', autor: 'Goethe', ISBN: '978-1' });
		expect(orderStore.cart).toHaveLength(1);
		expect(orderStore.cart[0].menge).toBe(2);
	});

	it('eskaliert generate_barcodes beim Merge, nimmt es aber nie zurück', () => {
		orderStore.addToCart({ id: 't1', titel: 'Faust', autor: 'G', isbn: '978-1' }, 1, false);
		expect(orderStore.cart[0].generate_barcodes).toBe(false);
		orderStore.addToCart({ id: 't1', titel: 'Faust', autor: 'G', isbn: '978-1' }, 1, true);
		expect(orderStore.cart[0].generate_barcodes).toBe(true);
		orderStore.addToCart({ id: 't1', titel: 'Faust', autor: 'G', isbn: '978-1' }, 1, false);
		expect(orderStore.cart[0].generate_barcodes).toBe(true);
	});

	// Der Fehler, den dieser Test festhaelt: Eine Bestellung ging OHNE Barcodebogen
	// hinaus, obwohl der Schalter "Barcodes mitschicken" an war — und das Anschreiben
	// verwies den Lieferanten trotzdem auf den "beigefuegten Bogen".
	//
	// Grund war der Vorgabewert `false`. Der Bestellbedarf, die taegliche Arbeitsflaeche,
	// ruft addToCart mit EINEM Argument auf; die Position landete also immer ohne
	// Barcodes im Warenkorb. Beim Absenden wirkt der Schalter nur als Sperre
	// (`schalter ? wert : false`) — er konnte abschalten, aber niemals einschalten.
	it('uebernimmt ohne dritten Parameter den Schalter aus dem Warenkorb', () => {
		orderStore.attachBarcodes = true;
		orderStore.addToCart({ id: 'bedarf-1', titel: 'Faust', autor: 'G', isbn: '978-1' });
		expect(orderStore.cart[0].generate_barcodes).toBe(true);
	});

	it('folgt dem Schalter auch, wenn er aus ist', () => {
		orderStore.attachBarcodes = false;
		orderStore.addToCart({ id: 'bedarf-2', titel: 'Faust', autor: 'G', isbn: '978-2' });
		expect(orderStore.cart[0].generate_barcodes).toBe(false);
		orderStore.attachBarcodes = true;
	});

	// Der DNB-Preisvorschlag (MARC21 020 $c). Er fuellt das Preisfeld vor, damit aus
	// "jeden Preis tippen" ein "pruefen und korrigieren" wird — uebernimmt aber nie
	// stillschweigend einen bereits erfassten Preis.
	it('uebernimmt den DNB-Preisvorschlag in den Warenkorb', () => {
		orderStore.addToCart({
			id: 'p1',
			titel: 'Atlas',
			autor: 'K',
			isbn: '978-9',
			preis_vorschlag: 27
		});
		expect(orderStore.cart[0].preis).toBe(27);
		expect(orderStore.cart[0].preis_vorschlag).toBe(27);
	});

	it('ohne Vorschlag bleibt der Preis bei 0', () => {
		orderStore.addToCart({ id: 'p2', titel: 'Atlas', autor: 'K', isbn: '978-8' });
		expect(orderStore.cart[0].preis).toBe(0);
	});

	it('ueberschreibt einen bereits erfassten Preis nicht', () => {
		orderStore.addToCart({ id: 'p3', titel: 'Atlas', autor: 'K', isbn: '978-7' });
		orderStore.cart[0].preis = 19.5; // von Hand erfasster Schulpreis
		orderStore.addToCart({
			id: 'p3',
			titel: 'Atlas',
			autor: 'K',
			isbn: '978-7',
			preis_vorschlag: 27
		});
		expect(orderStore.cart[0].preis).toBe(19.5);
	});

	it('setzt den Such-Zustand nach dem Hinzufügen zurück', () => {
		orderStore.searchQuery = 'faust';
		orderStore.searchResults = [{ titel: 'x' }];
		orderStore.showDropdown = true;
		orderStore.addToCart({ id: 't1', titel: 'Faust', autor: 'G', isbn: '978-1' });
		expect(orderStore.searchQuery).toBe('');
		expect(orderStore.searchResults).toEqual([]);
		expect(orderStore.showDropdown).toBe(false);
	});
});

describe('orderStore Summen', () => {
	beforeEach(resetStore);

	it('berechnet total und totalQty über alle Positionen', () => {
		orderStore.addToCart({ id: 'a', titel: 'A', autor: '', isbn: '1' }, 2);
		orderStore.addToCart({ id: 'b', titel: 'B', autor: '', isbn: '2' }, 3);
		orderStore.cart[0].preis = 10.5;
		orderStore.cart[1].preis = /** @type {any} */ ('4.50'); // Eingabefeld liefert Strings
		expect(orderStore.totalQty).toBe(5);
		expect(orderStore.total).toBeCloseTo(2 * 10.5 + 3 * 4.5);
	});

	it('wertet ungültige Preise als 0', () => {
		orderStore.addToCart({ id: 'a', titel: 'A', autor: '', isbn: '1' }, 2);
		orderStore.cart[0].preis = /** @type {any} */ ('abc');
		expect(orderStore.total).toBe(0);
	});
});

describe('orderStore.submitOrder', () => {
	beforeEach(() => {
		resetStore();
		vi.clearAllMocks();
		orderStore.suppliers = [{ id: 's1', name: 'Naacher', email: 'x@y.z', customerNumber: 'K1' }];
		orderStore.selectedSupplierId = 's1';
	});

	it('sendet nichts ohne Lieferant oder mit leerem Warenkorb', async () => {
		orderStore.selectedSupplierId = '';
		orderStore.addToCart({ id: 'a', titel: 'A', autor: '', isbn: '1' });
		await orderStore.submitOrder();
		expect(hatBestellungGesendet(apiPost)).toBe(false);

		orderStore.selectedSupplierId = 's1';
		orderStore.cart = [];
		await orderStore.submitOrder();
		expect(hatBestellungGesendet(apiPost)).toBe(false);
	});

	it('baut das Payload korrekt und leert den Warenkorb', async () => {
		bestellAntwort({ status: 'success', message: 'ok', ordered_qty: 2 });
		orderStore.addToCart({ id: 't1', titel: 'A', autor: '', isbn: '1' }, 2, true);
		orderStore.cart[0].preis = /** @type {any} */ ('9.90');

		await orderStore.submitOrder();

		expect(apiPost).toHaveBeenCalledWith('/api/bestellungen', {
			supplier_id: 's1',
			items: [{ titel_id: 't1', menge: 2, preis: 9.9, generate_barcodes: true }]
		});
		expect(orderStore.cart).toEqual([]);
		expect(apiGet).toHaveBeenCalledWith('/api/bestellungen/zulauf');
	});

	it('unterdrückt generate_barcodes, wenn der globale Schalter aus ist', async () => {
		bestellAntwort({ status: 'success' });
		orderStore.attachBarcodes = false;
		orderStore.addToCart({ id: 't1', titel: 'A', autor: '', isbn: '1' }, 1, true);

		await orderStore.submitOrder();

		const payload = bestellPayload(apiPostMock);
		expect(payload.items[0].generate_barcodes).toBe(false);
	});

	it('behält den Warenkorb bei einem API-Fehler', async () => {
		bestellAntwort(new Error('boom'), true);
		orderStore.addToCart({ id: 't1', titel: 'A', autor: '', isbn: '1' });

		await orderStore.submitOrder();

		expect(orderStore.cart).toHaveLength(1);
		expect(orderStore.submitting).toBe(false);
	});
});

describe('orderStore Suche', () => {
	beforeEach(() => {
		resetStore();
		vi.clearAllMocks();
		vi.useFakeTimers();
	});
	afterEach(() => {
		vi.useRealTimers();
	});

	it('sucht erst ab 2 Zeichen und debounced 300ms', async () => {
		orderStore.searchQuery = 'f';
		orderStore.handleSearchInput();
		await vi.advanceTimersByTimeAsync(400);
		expect(apiPost).not.toHaveBeenCalled();

		orderStore.searchQuery = 'faust';
		orderStore.handleSearchInput();
		await vi.advanceTimersByTimeAsync(299);
		expect(apiPost).not.toHaveBeenCalled();
		await vi.advanceTimersByTimeAsync(1);
		expect(apiPost).toHaveBeenCalledWith('/api/bestellungen/suche', { query: 'faust' });
	});

	it('verwirft veraltete Antworten (Out-of-Order-Race)', async () => {
		/** @type {(value: any) => void} */
		let resolveFirst = () => {};
		apiPostMock
			.mockImplementationOnce(
				() =>
					new Promise((res) => {
						resolveFirst = res;
					})
			)
			.mockImplementationOnce(async () => [{ titel: 'Neu', source: 'local' }]);

		orderStore.searchQuery = 'alte suche';
		orderStore.handleSearchInput();
		await vi.advanceTimersByTimeAsync(300); // erste Anfrage läuft, hängt

		orderStore.searchQuery = 'neue suche';
		orderStore.handleSearchInput();
		await vi.advanceTimersByTimeAsync(300); // zweite Anfrage kommt sofort zurück

		expect(orderStore.searchResults).toEqual([{ titel: 'Neu', source: 'local' }]);

		// Jetzt trudelt die ALTE Antwort ein — sie darf nichts überschreiben
		resolveFirst([{ titel: 'Alt', source: 'local' }]);
		await vi.advanceTimersByTimeAsync(1);

		expect(orderStore.searchResults).toEqual([{ titel: 'Neu', source: 'local' }]);
		expect(orderStore.showDropdown).toBe(true);
	});
});

// Der als Standard hinterlegte Lieferant muss auch dann greifen, wenn schon eine
// GÜLTIGE Auswahl steht.
//
// Aus dem Betrieb gemeldet (zweimal): „Ich habe X als Standard angegeben, aber nichts
// ändert sich." Die Ursache lag nicht im Backend — dort stand der Haken richtig — sondern
// hier: loadSuppliers übernahm den Standard nur, wenn die bisherige Auswahl UNGÜLTIG war.
// Nach dem Markieren wurde die Liste neu geladen, die alte Auswahl war weiterhin gültig,
// und der frische Standard blieb wirkungslos. Ein Haken, der nichts tut, ist schlimmer als
// keiner, weil man sich auf ihn verlässt und die Bestellung an den falschen Händler geht.
describe('orderStore.loadSuppliers — Standard-Lieferant', () => {
	const LISTE = [
		{ id: 's1', name: 'Cornelsen', customerNumber: 'C-1', ist_standard: false },
		{ id: 's2', name: 'Westermann', customerNumber: 'W-2', ist_standard: true }
	];

	beforeEach(() => {
		resetStore();
		vi.mocked(apiGet).mockReset();
		vi.mocked(apiGet).mockImplementation(async () => LISTE);
	});

	it('übernimmt den Standard, obwohl die bisherige Auswahl gültig ist', async () => {
		orderStore.selectedSupplierId = 's1'; // gültig, aber nicht der Standard
		await orderStore.loadSuppliers();
		expect(orderStore.selectedSupplierId).toBe('s2');
	});

	it('lässt eine angefangene Bestellung unangetastet', async () => {
		orderStore.selectedSupplierId = 's1';
		orderStore.cart = [{ id: 'b1', titel: 'Buch', menge: 1, preis: 0 }];
		await orderStore.loadSuppliers();
		expect(
			orderStore.selectedSupplierId,
			'Mitten in einer Bestellung darf der Lieferant nicht unter der Hand wechseln'
		).toBe('s1');
	});

	it('greift auch bei leerer Vorauswahl', async () => {
		orderStore.selectedSupplierId = '';
		await orderStore.loadSuppliers();
		expect(orderStore.selectedSupplierId).toBe('s2');
	});

	it('nimmt den ersten, wenn kein Standard hinterlegt ist', async () => {
		vi.mocked(apiGet).mockImplementation(async () =>
			LISTE.map((s) => ({ ...s, ist_standard: false }))
		);
		orderStore.selectedSupplierId = '';
		await orderStore.loadSuppliers();
		expect(orderStore.selectedSupplierId).toBe('s1');
	});
});
