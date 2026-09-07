import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('../apiFetch.js', () => ({ apiFetch: vi.fn() }));

// Verlassen-Schutz (Register B, 07.09.2026): Der Setter von activeTab ist die EINE
// Stelle, an der ein Bildschirm mit ungespeicherter Arbeit den Wechsel anhalten kann —
// egal ob Seitenleiste, Router, Deep-Link oder Escape-Regel ihn auslösen.
describe('uiStore.activeTab mit Verlassen-Schutz', () => {
	/** @type {import('./uiStore.svelte.js').uiStore} */
	let store;
	beforeEach(async () => {
		vi.resetModules();
		({ uiStore: store } = await import('./uiStore.svelte.js'));
	});

	it('wechselt ohne Wächter sofort', () => {
		store.activeTab = 'settings';
		expect(store.activeTab).toBe('settings');
		expect(store.blockierterWechsel).toBeNull();
	});

	it('hält den Wechsel an, solange der Wächter Arbeit meldet', () => {
		store.verlassenSperre = () => true;
		store.activeTab = 'settings';
		expect(store.activeTab, 'bleibt auf dem alten Tab').toBe('kiosk');
		expect(store.blockierterWechsel, 'merkt sich das Ziel').toBe('settings');
	});

	it('bleibe() verwirft nur das Ziel, erzwingeWechsel() führt es aus', () => {
		store.verlassenSperre = () => true;
		store.activeTab = 'settings';
		store.bleibe();
		expect(store.activeTab).toBe('kiosk');
		expect(store.blockierterWechsel).toBeNull();

		store.activeTab = 'stats';
		store.erzwingeWechsel();
		expect(store.activeTab).toBe('stats');
		expect(store.blockierterWechsel).toBeNull();
		expect(store.verlassenSperre, 'der Wächter des verlassenen Bildschirms ist weg').toBeNull();
	});

	it('lässt durch, sobald der Wächter nichts mehr meldet — und beim selben Tab', () => {
		store.verlassenSperre = () => false;
		store.activeTab = 'settings';
		expect(store.activeTab).toBe('settings');
		store.verlassenSperre = () => true;
		store.activeTab = 'settings';
		expect(store.blockierterWechsel, 'kein Wechsel, kein Dialog').toBeNull();
	});
});
