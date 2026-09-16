import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render } from '@testing-library/svelte';

// Der Offline-Hinweis ist ein BAND und darf die Theke nicht anhalten.
//
// Anlass: der Stufe-1-Nachweis am echten Chrome (16.09.2026, OFFEN.md Abschnitt 2).
// Befund: „der balken ist so gross, dass ich nichts buchen kann". Der Block trug
// Riesenschrift und grosse Knoepfe und rief auch dann „bitte Sicherung speichern", wenn
// es gar nichts zu sichern gab — „0 Vorgaenge nur auf diesem Rechner".
//
// Bis dahin gab es zu dieser Datei KEINEN Test; keine Zusage war gegen ihre Form
// abgesichert. Die drei Lautstaerken stehen hier, weil sonst der naechste, der die
// Bedingung anfasst, wieder eine einzige daraus macht.

const sync = {
	pendingCount: 0,
	isOffline: false,
	isSyncing: false,
	warteschlangeFehler: false,
	exportQueueAsJSON: vi.fn(),
	importQueueFromJSON: vi.fn()
};

vi.mock('../stores/offlineSync.svelte.js', () => ({ offlineSync: sync }));
vi.mock('../stores/toastStore.svelte.js', () => ({ toastStore: { addToast: vi.fn() } }));

const { default: OfflineIndicator } = await import('./OfflineIndicator.svelte');

describe('Offline-Band', () => {
	beforeEach(() => {
		sync.pendingCount = 0;
		sync.isOffline = false;
		sync.isSyncing = false;
		sync.warteschlangeFehler = false;
	});

	it('ist kein Vollbild — es deckt die Anwendung nicht zu', () => {
		sync.isOffline = true;
		const screen = render(OfflineIndicator, {});
		const band = screen.container.querySelector('div');
		expect(band).not.toBeNull();
		// `inset-0` waere ein Vollbild; `h-screen`/`min-h-screen` ebenso. Ein Band sitzt oben.
		const klassen = band?.className ?? '';
		expect(klassen).not.toMatch(/inset-0|h-screen|min-h-screen/);
		expect(klassen).toContain('top-0');
	});

	it('ohne Netz und ohne offene Vorgaenge bleibt es leise', () => {
		sync.isOffline = true;
		const screen = render(OfflineIndicator, {});
		const text = (screen.container.textContent ?? '').replace(/\s+/g, ' ');
		expect(text).toContain('Scannen geht weiter');
		// Nichts zu sichern, also auch keine Aufforderung dazu und kein Knopf dafuer.
		expect(text).not.toContain('Sicherung speichern');
		expect(text).not.toMatch(/0 Vorgang/);
	});

	it('nennt offene Vorgaenge mit Zahl und bietet die Sicherung an', () => {
		sync.isOffline = true;
		sync.pendingCount = 3;
		const screen = render(OfflineIndicator, {});
		const text = (screen.container.textContent ?? '').replace(/\s+/g, ' ');
		expect(text).toContain('3 Vorgänge nur auf diesem Rechner');
		expect(text).toContain('Sicherung speichern');
	});

	it('wird laut, wenn die Warteschlange nicht lesbar ist — dann gehen Scans verloren', () => {
		sync.warteschlangeFehler = true;
		const screen = render(OfflineIndicator, {});
		const text = (screen.container.textContent ?? '').replace(/\s+/g, ' ');
		expect(text).toContain('NICHT gespeichert');
		expect(screen.container.querySelector('div')?.className).toContain('bg-error');
	});

	it('meldet den ausgefallenen Herzschlag im selben Band statt in einer zweiten Schicht', () => {
		// Bis zum 16.09.2026 legte App.svelte dafuer ein eigenes Vollbild darueber.
		const screen = render(OfflineIndicator, { verbindungVerloren: true });
		expect(screen.container.textContent ?? '').toContain('Keine Verbindung');
	});

	it('zeigt gar nichts, wenn alles in Ordnung ist', () => {
		const screen = render(OfflineIndicator, {});
		expect(screen.container.querySelector('div')).toBeNull();
	});
});
