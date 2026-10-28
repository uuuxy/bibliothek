import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

vi.mock('../apiFetch.js', async (original) => ({
	.../** @type {any} */ (await original()),
	apiClient: { post: vi.fn() },
	apiGet: vi.fn(async () => []),
	apiPost: vi.fn(async () => ({})),
	apiPut: vi.fn(async () => ({})),
	apiDelete: vi.fn(async () => ({}))
}));
vi.mock('./snackbars.svelte.js', () => ({
	snackbars: { addToast: vi.fn() }
}));

import { apiClient, apiGet, apiPost, apiPut, FRIST_MAILVERSAND_MS } from '../apiFetch.js';
import { bestellStore } from './bestellStore.svelte.js';
import { snackbars } from './snackbars.svelte.js';

const apiPostMock = vi.mocked(apiPost);

function resetStore() {
	bestellStore.cart = [];
	bestellStore.suppliers = [];
	bestellStore.selectedSupplierId = '';
	bestellStore.attachBarcodes = true;
	bestellStore.searchQuery = '';
	bestellStore.searchResults = [];
	bestellStore.showDropdown = false;
	bestellStore.dnbAusfall = false;
}

/**
 * Die Antwort der Bestellsuche: die Treffer als Rumpf, der Ausfall der DNB als Kopf.
 * @param {any[]} treffer @param {{ dnbAusfall?: boolean, status?: number }} [lage]
 */
function suchAntwort(treffer, { dnbAusfall = false, status = 200 } = {}) {
	return /** @type {any} */ ({
		ok: status < 400,
		status,
		json: async () => treffer,
		text: async () => JSON.stringify({ error: 'Suche gescheitert' }),
		headers: new Headers(dnbAusfall ? { 'X-DNB-Ausfall': '1' } : {})
	});
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

describe('bestellStore.addToCart', () => {
	beforeEach(() => {
		resetStore();
		vi.clearAllMocks();
	});

	it('legt neue Positionen mit titel_id als Schlüssel an', () => {
		bestellStore.addToCart({
			titel_id: 't1',
			id: 'buch-1',
			titel: 'Faust',
			autor: 'Goethe',
			isbn: '978-1'
		});
		expect(bestellStore.cart).toHaveLength(1);
		expect(bestellStore.cart[0].id).toBe('t1');
	});

	it('dedupliziert, wenn dasselbe Buch einmal mit titel_id und einmal ohne kommt', () => {
		// Suchergebnis liefert titel_id, Empfehlung liefert nur id — früherer Duplikat-Bug
		bestellStore.addToCart({ titel_id: 't1', titel: 'Faust', autor: 'Goethe', isbn: '978-1' }, 2);
		bestellStore.addToCart({ id: 't1', titel: 'Faust', autor: 'Goethe', isbn: '978-1' }, 3);
		expect(bestellStore.cart).toHaveLength(1);
		expect(bestellStore.cart[0].menge).toBe(5);
	});

	it('dedupliziert über die ISBN, auch bei abweichender Feld-Schreibung', () => {
		bestellStore.addToCart({ id: 'a', titel: 'Faust', autor: 'Goethe', isbn: '978-1' });
		bestellStore.addToCart({ id: 'b', titel: 'Faust', autor: 'Goethe', ISBN: '978-1' });
		expect(bestellStore.cart).toHaveLength(1);
		expect(bestellStore.cart[0].menge).toBe(2);
	});

	it('eskaliert generate_barcodes beim Merge, nimmt es aber nie zurück', () => {
		bestellStore.addToCart({ id: 't1', titel: 'Faust', autor: 'G', isbn: '978-1' }, 1, false);
		expect(bestellStore.cart[0].generate_barcodes).toBe(false);
		bestellStore.addToCart({ id: 't1', titel: 'Faust', autor: 'G', isbn: '978-1' }, 1, true);
		expect(bestellStore.cart[0].generate_barcodes).toBe(true);
		bestellStore.addToCart({ id: 't1', titel: 'Faust', autor: 'G', isbn: '978-1' }, 1, false);
		expect(bestellStore.cart[0].generate_barcodes).toBe(true);
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
		bestellStore.attachBarcodes = true;
		bestellStore.addToCart({ id: 'bedarf-1', titel: 'Faust', autor: 'G', isbn: '978-1' });
		expect(bestellStore.cart[0].generate_barcodes).toBe(true);
	});

	it('folgt dem Schalter auch, wenn er aus ist', () => {
		bestellStore.attachBarcodes = false;
		bestellStore.addToCart({ id: 'bedarf-2', titel: 'Faust', autor: 'G', isbn: '978-2' });
		expect(bestellStore.cart[0].generate_barcodes).toBe(false);
		bestellStore.attachBarcodes = true;
	});

	// Der DNB-Preisvorschlag (MARC21 020 $c). Er fuellt das Preisfeld vor, damit aus
	// "jeden Preis tippen" ein "pruefen und korrigieren" wird — uebernimmt aber nie
	// stillschweigend einen bereits erfassten Preis.
	it('uebernimmt den DNB-Preisvorschlag in den Warenkorb', () => {
		bestellStore.addToCart({
			id: 'p1',
			titel: 'Atlas',
			autor: 'K',
			isbn: '978-9',
			preis_vorschlag: 27
		});
		expect(bestellStore.cart[0].preis).toBe(27);
		expect(bestellStore.cart[0].preis_vorschlag).toBe(27);
	});

	it('ohne Vorschlag bleibt der Preis bei 0', () => {
		bestellStore.addToCart({ id: 'p2', titel: 'Atlas', autor: 'K', isbn: '978-8' });
		expect(bestellStore.cart[0].preis).toBe(0);
	});

	it('ueberschreibt einen bereits erfassten Preis nicht', () => {
		bestellStore.addToCart({ id: 'p3', titel: 'Atlas', autor: 'K', isbn: '978-7' });
		bestellStore.cart[0].preis = 19.5; // von Hand erfasster Schulpreis
		bestellStore.addToCart({
			id: 'p3',
			titel: 'Atlas',
			autor: 'K',
			isbn: '978-7',
			preis_vorschlag: 27
		});
		expect(bestellStore.cart[0].preis).toBe(19.5);
	});

	it('setzt den Such-Zustand nach dem Hinzufügen zurück', () => {
		bestellStore.searchQuery = 'faust';
		bestellStore.searchResults = [{ titel: 'x' }];
		bestellStore.showDropdown = true;
		bestellStore.addToCart({ id: 't1', titel: 'Faust', autor: 'G', isbn: '978-1' });
		expect(bestellStore.searchQuery).toBe('');
		expect(bestellStore.searchResults).toEqual([]);
		expect(bestellStore.showDropdown).toBe(false);
	});
});

describe('bestellStore Summen', () => {
	beforeEach(resetStore);

	it('berechnet total und totalQty über alle Positionen', () => {
		bestellStore.addToCart({ id: 'a', titel: 'A', autor: '', isbn: '1' }, 2);
		bestellStore.addToCart({ id: 'b', titel: 'B', autor: '', isbn: '2' }, 3);
		bestellStore.cart[0].preis = 10.5;
		bestellStore.cart[1].preis = /** @type {any} */ ('4.50'); // Eingabefeld liefert Strings
		expect(bestellStore.totalQty).toBe(5);
		expect(bestellStore.total).toBeCloseTo(2 * 10.5 + 3 * 4.5);
	});

	it('wertet ungültige Preise als 0', () => {
		bestellStore.addToCart({ id: 'a', titel: 'A', autor: '', isbn: '1' }, 2);
		bestellStore.cart[0].preis = /** @type {any} */ ('abc');
		expect(bestellStore.total).toBe(0);
	});
});

describe('bestellStore.submitOrder', () => {
	beforeEach(() => {
		resetStore();
		vi.clearAllMocks();
		bestellStore.suppliers = [{ id: 's1', name: 'Naacher', email: 'x@y.z', customerNumber: 'K1' }];
		bestellStore.selectedSupplierId = 's1';
	});

	it('sendet nichts ohne Lieferant oder mit leerem Warenkorb', async () => {
		bestellStore.selectedSupplierId = '';
		bestellStore.addToCart({ id: 'a', titel: 'A', autor: '', isbn: '1' });
		await bestellStore.submitOrder();
		expect(hatBestellungGesendet(apiPost)).toBe(false);

		bestellStore.selectedSupplierId = 's1';
		bestellStore.cart = [];
		await bestellStore.submitOrder();
		expect(hatBestellungGesendet(apiPost)).toBe(false);
	});

	it('baut das Payload korrekt und leert den Warenkorb', async () => {
		bestellAntwort({ status: 'success', message: 'ok', ordered_qty: 2 });
		bestellStore.addToCart({ id: 't1', titel: 'A', autor: '', isbn: '1' }, 2, true);
		bestellStore.cart[0].preis = /** @type {any} */ ('9.90');

		await bestellStore.submitOrder();

		expect(apiPost).toHaveBeenCalledWith(
			'/api/bestellungen',
			expect.objectContaining({
				supplier_id: 's1',
				// Ohne Lernmittel-Kennzeichen ist der Topf die Schülerbücherei.
				mittel: 'schultraeger',
				items: [{ titel_id: 't1', menge: 2, preis: 9.9, generate_barcodes: true }]
			}),
			// Der Server verschickt nach dem Speichern die Mail: Die Oberfläche wartet länger
			// als die Vorgabe, sonst käme die Antwort „gespeichert, Mail gescheitert" nie an.
			{ timeoutMs: FRIST_MAILVERSAND_MS }
		);
		// Doppelklick-Schutz: ein Idempotenz-Schlüssel geht mit.
		const payload = bestellPayload(apiPostMock);
		expect(typeof payload.idempotency_key).toBe('string');
		expect(payload.idempotency_key.length).toBeGreaterThan(0);
		expect(bestellStore.cart).toEqual([]);
		expect(apiGet).toHaveBeenCalledWith('/api/bestellungen/zulauf');
	});

	it('unterdrückt generate_barcodes, wenn der globale Schalter aus ist', async () => {
		bestellAntwort({ status: 'success' });
		bestellStore.attachBarcodes = false;
		bestellStore.addToCart({ id: 't1', titel: 'A', autor: '', isbn: '1' }, 1, true);

		await bestellStore.submitOrder();

		const payload = bestellPayload(apiPostMock);
		expect(payload.items[0].generate_barcodes).toBe(false);
	});

	it('behält den Warenkorb bei einem API-Fehler', async () => {
		bestellAntwort(new Error('boom'), true);
		bestellStore.addToCart({ id: 't1', titel: 'A', autor: '', isbn: '1' });

		await bestellStore.submitOrder();

		expect(bestellStore.cart).toHaveLength(1);
		expect(bestellStore.submitting).toBe(false);
	});
});

// Ging die Antwort auf das Anlegen verloren, stand der Lieferant schon am Server, die Liste zeigte
// ihn nicht, und ein zweiter Versuch legte ihn noch einmal an (OFFEN.md 5.106).
describe('bestellStore.addSupplier', () => {
	beforeEach(() => {
		resetStore();
		apiPostMock.mockReset();
		vi.mocked(apiGet).mockReset();
	});

	it('lädt die Liste auch nach einem Fehler neu und gibt den Fehler weiter', async () => {
		apiPostMock.mockRejectedValueOnce(new Error('Keine Verbindung zum Server.'));
		vi.mocked(apiGet).mockResolvedValueOnce([
			{ id: 'l1', name: 'Neu', email: 'n@example.invalid', customerNumber: 'K1' }
		]);

		await expect(bestellStore.addSupplier('Neu', 'n@example.invalid', 'K1')).rejects.toThrow(
			'Keine Verbindung zum Server.'
		);

		expect(bestellStore.suppliers.map((s) => s.name)).toEqual(['Neu']);
	});
});

describe('bestellStore Suche', () => {
	beforeEach(() => {
		resetStore();
		vi.clearAllMocks();
		vi.useFakeTimers();
	});
	afterEach(() => {
		vi.useRealTimers();
	});

	it('sucht erst ab 2 Zeichen und debounced 300ms', async () => {
		vi.mocked(apiClient.post).mockResolvedValue(suchAntwort([]));
		bestellStore.searchQuery = 'f';
		bestellStore.handleSearchInput();
		await vi.advanceTimersByTimeAsync(400);
		expect(apiClient.post).not.toHaveBeenCalled();

		bestellStore.searchQuery = 'faust';
		bestellStore.handleSearchInput();
		await vi.advanceTimersByTimeAsync(299);
		expect(apiClient.post).not.toHaveBeenCalled();
		await vi.advanceTimersByTimeAsync(1);
		expect(apiClient.post).toHaveBeenCalledWith('/api/bestellungen/suche', { query: 'faust' });
	});

	it('verwirft veraltete Antworten (Out-of-Order-Race)', async () => {
		/** @type {(value: any) => void} */
		let resolveFirst = () => {};
		vi.mocked(apiClient.post)
			.mockImplementationOnce(
				() =>
					new Promise((res) => {
						resolveFirst = res;
					})
			)
			.mockImplementationOnce(async () => suchAntwort([{ titel: 'Neu', source: 'local' }]));

		bestellStore.searchQuery = 'alte suche';
		bestellStore.handleSearchInput();
		await vi.advanceTimersByTimeAsync(300); // erste Anfrage läuft, hängt

		bestellStore.searchQuery = 'neue suche';
		bestellStore.handleSearchInput();
		await vi.advanceTimersByTimeAsync(300); // zweite Anfrage kommt sofort zurück

		expect(bestellStore.searchResults).toEqual([{ titel: 'Neu', source: 'local' }]);

		// Jetzt trudelt die ALTE Antwort ein — sie darf nichts überschreiben
		resolveFirst(suchAntwort([{ titel: 'Alt', source: 'local' }], { dnbAusfall: true }));
		await vi.advanceTimersByTimeAsync(1);

		expect(bestellStore.searchResults).toEqual([{ titel: 'Neu', source: 'local' }]);
		expect(bestellStore.showDropdown).toBe(true);
		expect(bestellStore.dnbAusfall).toBe(false);
	});

	// Ohne das Merkmal sähe ein Buch, das die DNB kennt, aus wie eines, das sie nicht kennt:
	// Bei null Treffern öffnete die Trefferliste gar nicht.
	it('öffnet die Trefferliste bei einem Ausfall der DNB auch ohne Treffer', async () => {
		vi.mocked(apiClient.post).mockResolvedValue(suchAntwort([], { dnbAusfall: true }));

		await bestellStore.sucheSofort('9783060130764');

		expect(bestellStore.searchResults).toEqual([]);
		expect(bestellStore.dnbAusfall).toBe(true);
		expect(bestellStore.showDropdown).toBe(true);
	});

	it('nennt keinen Ausfall, wenn die DNB ohne Treffer geantwortet hat', async () => {
		vi.mocked(apiClient.post).mockResolvedValue(suchAntwort([]));

		await bestellStore.sucheSofort('gibt es nicht');

		expect(bestellStore.dnbAusfall).toBe(false);
		expect(bestellStore.showDropdown).toBe(false);
	});

	it('vergisst den Ausfall mit der nächsten Suche und beim Leeren', async () => {
		vi.mocked(apiClient.post).mockResolvedValueOnce(suchAntwort([], { dnbAusfall: true }));
		await bestellStore.sucheSofort('faust');
		expect(bestellStore.dnbAusfall).toBe(true);

		vi.mocked(apiClient.post).mockResolvedValueOnce(
			suchAntwort([{ titel: 'Faust', source: 'dnb' }])
		);
		await bestellStore.sucheSofort('faust');
		expect(bestellStore.dnbAusfall).toBe(false);

		vi.mocked(apiClient.post).mockResolvedValueOnce(suchAntwort([], { dnbAusfall: true }));
		await bestellStore.sucheSofort('faust');
		bestellStore.resetSearch();
		expect(bestellStore.dnbAusfall).toBe(false);
		expect(bestellStore.showDropdown).toBe(false);
	});

	it('meldet eine abgewiesene Suche und zeigt keine Treffer', async () => {
		vi.mocked(apiClient.post).mockResolvedValue(suchAntwort([], { status: 500 }));

		await bestellStore.sucheSofort('faust');

		expect(snackbars.addToast).toHaveBeenCalledWith('Suche gescheitert', 'error');
		expect(bestellStore.searchResults).toEqual([]);
		expect(bestellStore.showDropdown).toBe(false);
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
describe('bestellStore.loadSuppliers — Standard-Lieferant', () => {
	const LISTE = [
		{ id: 's1', name: 'Cornelsen', customerNumber: 'C-1', ist_hauptlieferant: false },
		{ id: 's2', name: 'Westermann', customerNumber: 'W-2', ist_hauptlieferant: true }
	];

	beforeEach(() => {
		resetStore();
		vi.mocked(apiGet).mockReset();
		vi.mocked(apiGet).mockImplementation(async () => LISTE);
	});

	it('übernimmt den Standard, obwohl die bisherige Auswahl gültig ist', async () => {
		bestellStore.selectedSupplierId = 's1'; // gültig, aber nicht der Standard
		await bestellStore.loadSuppliers();
		expect(bestellStore.selectedSupplierId).toBe('s2');
	});

	it('lässt eine angefangene Bestellung unangetastet', async () => {
		bestellStore.selectedSupplierId = 's1';
		bestellStore.cart = [/** @type {any} */ ({ id: 'b1', titel: 'Buch', menge: 1, preis: 0 })];
		await bestellStore.loadSuppliers();
		expect(
			bestellStore.selectedSupplierId,
			'Mitten in einer Bestellung darf der Lieferant nicht unter der Hand wechseln'
		).toBe('s1');
	});

	it('greift auch bei leerer Vorauswahl', async () => {
		bestellStore.selectedSupplierId = '';
		await bestellStore.loadSuppliers();
		expect(bestellStore.selectedSupplierId).toBe('s2');
	});

	it('nimmt den ersten, wenn kein Standard hinterlegt ist', async () => {
		vi.mocked(apiGet).mockImplementation(async () =>
			LISTE.map((s) => ({ ...s, ist_hauptlieferant: false }))
		);
		bestellStore.selectedSupplierId = '';
		await bestellStore.loadSuppliers();
		expect(bestellStore.selectedSupplierId).toBe('s1');
	});
});

// Eine Bestellung = ein Topf (Migration 109). Der Titel schlägt den Topf vor, die
// Bestellung darf ihn überstimmen, und ein gemischter Warenkorb wird beim Auslösen zu
// ZWEI Bestellungen an denselben Händler — der Händler richtet Nachlass und Rechnungsweg
// nach dem Topf und kann eine gemischte Bestellung weder rabattieren noch abrechnen.
describe('bestellStore Töpfe (Lernmittelfreiheit / Schülerbücherei)', () => {
	beforeEach(() => {
		resetStore();
		vi.clearAllMocks();
		bestellStore.suppliers = [{ id: 's1', name: 'Naacher', email: 'x@y.z', customerNumber: 'K1' }];
		bestellStore.selectedSupplierId = 's1';
	});

	/** Alle Bestell-Payloads in Aufrufreihenfolge. */
	function bestellPayloads() {
		return apiPostMock.mock.calls
			.filter((/** @type {any[]} */ args) => args[0] === '/api/bestellungen')
			.map((/** @type {any[]} */ args) => args[1]);
	}

	it('schlägt den Topf aus dem Lernmittel-Kennzeichen vor', () => {
		bestellStore.addToCart({
			id: 'lmf',
			titel: 'Mathe 7',
			autor: '',
			isbn: '1',
			ist_lernmittel: true
		});
		bestellStore.addToCart({ id: 'bib', titel: 'Panem', autor: '', isbn: '2' });
		expect(bestellStore.cart.map((i) => i.mittel)).toEqual(['land', 'schultraeger']);
		expect(bestellStore.gruppen.map((g) => g.mittel)).toEqual(['land', 'schultraeger']);
	});

	it('zeigt nur nicht-leere Gruppen, Lernmittel zuerst', () => {
		bestellStore.addToCart({ id: 'bib', titel: 'Panem', autor: '', isbn: '2' });
		expect(bestellStore.gruppen).toHaveLength(1);
		expect(bestellStore.gruppen[0].mittel).toBe('schultraeger');
		bestellStore.addToCart(
			{ id: 'lmf', titel: 'Mathe 7', autor: '', isbn: '1', ist_lernmittel: true },
			30
		);
		expect(bestellStore.gruppen.map((g) => [g.mittel, g.menge])).toEqual([
			['land', 30],
			['schultraeger', 1]
		]);
	});

	it('verschiebt eine Position in den anderen Topf und behält das beim zweiten „+"', () => {
		bestellStore.addToCart({ id: 'bib', titel: 'Falsch gekennzeichnet', autor: '', isbn: '2' });
		bestellStore.verschiebe(bestellStore.cart[0]);
		expect(bestellStore.cart[0].mittel).toBe('land');
		// Ein zweites „+" auf denselben Titel schiebt ihn NICHT zurück.
		bestellStore.addToCart({ id: 'bib', titel: 'Falsch gekennzeichnet', autor: '', isbn: '2' });
		expect(bestellStore.cart[0].mittel).toBe('land');
		expect(bestellStore.cart[0].menge).toBe(2);
	});

	it('löst für einen gemischten Warenkorb zwei Bestellungen mit eigenem Schlüssel aus', async () => {
		bestellAntwort({ status: 'success', message: 'ok', ordered_qty: 1 });
		bestellStore.addToCart(
			{ id: 'lmf', titel: 'Mathe 7', autor: '', isbn: '1', ist_lernmittel: true },
			30
		);
		bestellStore.addToCart({ id: 'bib', titel: 'Panem', autor: '', isbn: '2' }, 2);

		await bestellStore.submitOrder();

		const payloads = bestellPayloads();
		expect(payloads.map((p) => p.mittel)).toEqual(['land', 'schultraeger']);
		expect(payloads[0].items).toEqual([
			{ titel_id: 'lmf', menge: 30, preis: 0, generate_barcodes: true }
		]);
		expect(payloads[1].items).toEqual([
			{ titel_id: 'bib', menge: 2, preis: 0, generate_barcodes: true }
		]);
		expect(payloads[0].supplier_id).toBe('s1');
		expect(payloads[1].supplier_id).toBe('s1');
		// Zwei Bestellungen, zwei Idempotenz-Schlüssel — sonst hielte der Server die
		// zweite für den Doppelklick der ersten.
		expect(payloads[0].idempotency_key).not.toBe(payloads[1].idempotency_key);
		expect(bestellStore.cart).toEqual([]);
	});

	it('behält bei einem Fehler nur die gescheiterte Gruppe im Warenkorb', async () => {
		apiPostMock.mockImplementation(async (/** @type {string} */ url, /** @type {any} */ body) => {
			if (url !== '/api/bestellungen') return [];
			if (body.mittel === 'schultraeger') throw new Error('boom');
			return { status: 'success', message: 'ok' };
		});
		bestellStore.addToCart({
			id: 'lmf',
			titel: 'Mathe 7',
			autor: '',
			isbn: '1',
			ist_lernmittel: true
		});
		bestellStore.addToCart({ id: 'bib', titel: 'Panem', autor: '', isbn: '2' });

		await bestellStore.submitOrder();

		// Die Lernmittel-Bestellung ist durch und weg; die Bücherei-Position wartet.
		expect(bestellStore.cart.map((i) => i.id)).toEqual(['bib']);
		expect(bestellStore.submitting).toBe(false);
		const ersterSchluessel = bestellPayloads()[1].idempotency_key;

		// Der zweite Versuch schickt DENSELBEN Schlüssel für die gescheiterte Gruppe.
		await bestellStore.submitOrder();
		const payloads = bestellPayloads();
		expect(payloads).toHaveLength(3);
		expect(payloads[2].mittel).toBe('schultraeger');
		expect(payloads[2].idempotency_key).toBe(ersterSchluessel);
	});

	// Der Schlüssel gehört zum INHALT, nicht zum Topf.
	//
	// Fund vom 16.09.2026 (OFFEN.md 5.1): Ging die Antwort verloren und änderte jemand
	// danach den Warenkorb, ging derselbe Schlüssel mit anderem Inhalt raus. Der Server
	// erkennt die Wiederholung und antwortet mit der ALTEN Bestellung — die Änderung war
	// still verschwunden, und die Oberfläche meldete Erfolg. Genau das ist der Grund,
	// warum es überhaupt einen Schlüssel gibt: Er soll eine Verdopplung verhindern, nicht
	// eine Korrektur.
	it('vergibt nach einer Änderung am Warenkorb einen neuen Schlüssel', async () => {
		let antworte = false;
		apiPostMock.mockImplementation(async (/** @type {string} */ url) => {
			if (url !== '/api/bestellungen') return [];
			if (!antworte) throw new Error('Antwort ging verloren');
			return { status: 'success', message: 'ok' };
		});
		bestellStore.addToCart({ id: 'bib', titel: 'Panem', autor: '', isbn: '2' });

		await bestellStore.submitOrder();
		const ersterSchluessel = bestellPayloads()[0].idempotency_key;
		expect(bestellStore.cart).toHaveLength(1); // gescheitert, bleibt liegen

		// Gleicher Warenkorb, zweiter Versuch: derselbe Schlüssel (Wiederholung).
		await bestellStore.submitOrder();
		expect(bestellPayloads()[1].idempotency_key).toBe(ersterSchluessel);

		// Jetzt ändert jemand die Menge — das ist eine ANDERE Bestellung.
		bestellStore.cart[0].menge = 5;
		antworte = true;
		await bestellStore.submitOrder();

		const payloads = bestellPayloads();
		expect(payloads).toHaveLength(3);
		expect(payloads[2].items[0].menge).toBe(5);
		expect(payloads[2].idempotency_key).not.toBe(ersterSchluessel);
	});
});

// Lehnt der Server einen Lieferanten ab, behält das Formular die Eingabe und die bearbeitete
// Zeile bleibt offen (Lieferanten.svelte). Dafür muss der Fehler beim Aufrufer ankommen; die
// Meldung zeigt apiFetch.
describe('bestellStore: Lieferant anlegen und ändern', () => {
	beforeEach(() => {
		resetStore();
		vi.mocked(snackbars.addToast).mockClear();
		vi.mocked(apiGet).mockReset();
		vi.mocked(apiGet).mockImplementation(async () => []);
	});

	it('gibt einen abgelehnten Lieferanten an den Aufrufer weiter', async () => {
		apiPostMock.mockRejectedValueOnce(new Error('abgelehnt'));
		await expect(bestellStore.addSupplier('Neu', 'neu@example.org', 'K-1')).rejects.toThrow(
			'abgelehnt'
		);
		vi.mocked(apiPut).mockRejectedValueOnce(new Error('abgelehnt'));
		await expect(bestellStore.editSupplier('s1', { name: 'Neu' })).rejects.toThrow('abgelehnt');
		expect(snackbars.addToast).not.toHaveBeenCalledWith('Lieferant aktualisiert.', 'success');
	});

	it('lädt nach dem Anlegen die Liste neu', async () => {
		apiPostMock.mockResolvedValueOnce({});
		await bestellStore.addSupplier('Neu', 'neu@example.org', 'K-1');
		expect(apiGet).toHaveBeenCalledWith('/api/lieferanten');
	});
});
