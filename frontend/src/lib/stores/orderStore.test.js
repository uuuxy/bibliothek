import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

vi.mock('../apiFetch.js', () => ({
    apiGet: vi.fn(async () => []),
    apiPost: vi.fn(async () => ({})),
    apiPut: vi.fn(async () => ({})),
    apiDelete: vi.fn(async () => ({})),
}));
vi.mock('./toastStore.svelte.js', () => ({
    toastStore: { addToast: vi.fn() },
}));

import { apiGet, apiPost } from '../apiFetch.js';
import { orderStore } from './orderStore.svelte.js';

function resetStore() {
    orderStore.cart = [];
    orderStore.suppliers = [];
    orderStore.selectedSupplierId = '';
    orderStore.attachBarcodes = true;
    orderStore.searchQuery = '';
    orderStore.searchResults = [];
    orderStore.showDropdown = false;
}

describe('orderStore.addToCart', () => {
    beforeEach(() => {
        resetStore();
        vi.clearAllMocks();
    });

    it('legt neue Positionen mit titel_id als Schlüssel an', () => {
        orderStore.addToCart({ titel_id: 't1', id: 'buch-1', titel: 'Faust', autor: 'Goethe', isbn: '978-1' });
        expect(orderStore.cart).toHaveLength(1);
        expect(orderStore.cart[0].id).toBe('t1');
    });

    it('dedupliziert, wenn dasselbe Buch einmal mit titel_id und einmal ohne kommt', () => {
        // Suchergebnis liefert titel_id, Empfehlung liefert nur id — früherer Duplikat-Problem
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
        orderStore.cart[1].preis = '4.50'; // Eingabefeld liefert Strings
        expect(orderStore.totalQty).toBe(5);
        expect(orderStore.total).toBeCloseTo(2 * 10.5 + 3 * 4.5);
    });

    it('wertet ungültige Preise als 0', () => {
        orderStore.addToCart({ id: 'a', titel: 'A', autor: '', isbn: '1' }, 2);
        orderStore.cart[0].preis = 'abc';
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
        expect(apiPost).not.toHaveBeenCalled();

        orderStore.selectedSupplierId = 's1';
        orderStore.cart = [];
        await orderStore.submitOrder();
        expect(apiPost).not.toHaveBeenCalled();
    });

    it('baut das Payload korrekt und leert den Warenkorb', async () => {
        apiPost.mockResolvedValueOnce({ status: 'success', message: 'ok', ordered_qty: 2 });
        orderStore.addToCart({ id: 't1', titel: 'A', autor: '', isbn: '1' }, 2, true);
        orderStore.cart[0].preis = '9.90';

        await orderStore.submitOrder();

        expect(apiPost).toHaveBeenCalledWith('/api/bestellungen', {
            supplier_id: 's1',
            items: [{ titel_id: 't1', menge: 2, preis: 9.9, generate_barcodes: true }],
        });
        expect(orderStore.cart).toEqual([]);
        expect(apiGet).toHaveBeenCalledWith('/api/bestellungen/zulauf');
    });

    it('unterdrückt generate_barcodes, wenn der globale Schalter aus ist', async () => {
        apiPost.mockResolvedValueOnce({ status: 'success' });
        orderStore.attachBarcodes = false;
        orderStore.addToCart({ id: 't1', titel: 'A', autor: '', isbn: '1' }, 1, true);

        await orderStore.submitOrder();

        const payload = apiPost.mock.calls[0][1];
        expect(payload.items[0].generate_barcodes).toBe(false);
    });

    it('behält den Warenkorb bei einem API-Fehler', async () => {
        apiPost.mockRejectedValueOnce(new Error('boom'));
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
        let resolveFirst;
        apiPost
            .mockImplementationOnce(() => new Promise((res) => { resolveFirst = res; }))
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
