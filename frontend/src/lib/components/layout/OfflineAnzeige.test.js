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
	wartendeAnzahl: 0,
	ohneNetz: false,
	gleichtAb: false,
	warteschlangeFehler: false,
	/** @type {{ status: number, grund: string } | null} */
	abgelehnt: null,
	exportiereWarteschlange: vi.fn(),
	importiereWarteschlange: vi.fn()
};

// Die Meldungen aus dem Nachbuchen haengen am selben Band (Schritt C, 16.09.2026).
const meldungen = {
	offen: 0,
	liste: [],
	laeuft: false,
	fehler: '',
	geoeffnet: false,
	auchQuittierte: false,
	oeffne: vi.fn(),
	schliesse: vi.fn(),
	lade: vi.fn(),
	quittiere: vi.fn()
};
const recht = { wert: true };

vi.mock('../../stores/offlineSync.svelte.js', () => ({ offlineSync: sync }));
vi.mock('../../stores/nachbuchMeldungen.svelte.js', () => ({ nachbuchMeldungen: meldungen }));
vi.mock('../../menue.js', () => ({ hatRecht: () => recht.wert }));
vi.mock('../../stores/authStore.svelte.js', () => ({
	authStore: { currentUser: { rolle: 'admin' } }
}));
vi.mock('../../stores/snackbars.svelte.js', () => ({ snackbars: { addToast: vi.fn() } }));

const { default: OfflineAnzeige } = await import('./OfflineAnzeige.svelte');

describe('Offline-Band', () => {
	beforeEach(() => {
		sync.wartendeAnzahl = 0;
		sync.ohneNetz = false;
		sync.gleichtAb = false;
		sync.warteschlangeFehler = false;
		sync.abgelehnt = null;
		meldungen.offen = 0;
		meldungen.geoeffnet = false;
		recht.wert = true;
		vi.clearAllMocks();
	});

	it('ist kein Vollbild — es deckt die Anwendung nicht zu', () => {
		sync.ohneNetz = true;
		const screen = render(OfflineAnzeige, {});
		const band = screen.container.querySelector('div');
		expect(band).not.toBeNull();
		// `inset-0` waere ein Vollbild; `h-screen`/`min-h-screen` ebenso. Ein Band sitzt oben.
		const klassen = band?.className ?? '';
		expect(klassen).not.toMatch(/inset-0|h-screen|min-h-screen/);
		expect(klassen).toContain('top-0');
	});

	it('ohne Netz und ohne offene Vorgaenge bleibt es leise', () => {
		sync.ohneNetz = true;
		const screen = render(OfflineAnzeige, {});
		const text = (screen.container.textContent ?? '').replace(/\s+/g, ' ');
		expect(text).toContain('Scannen geht weiter');
		// Nichts zu sichern, also auch keine Aufforderung dazu und kein Knopf dafuer.
		expect(text).not.toContain('Sicherung speichern');
		expect(text).not.toMatch(/0 Vorgang/);
	});

	it('nennt offene Vorgaenge mit Zahl und bietet die Sicherung an', () => {
		sync.ohneNetz = true;
		sync.wartendeAnzahl = 3;
		const screen = render(OfflineAnzeige, {});
		const text = (screen.container.textContent ?? '').replace(/\s+/g, ' ');
		expect(text).toContain('3 Vorgänge nur auf diesem Rechner');
		expect(text).toContain('Sicherung speichern');
	});

	it('wird laut, wenn die Warteschlange nicht lesbar ist — dann gehen Scans verloren', () => {
		sync.warteschlangeFehler = true;
		const screen = render(OfflineAnzeige, {});
		const text = (screen.container.textContent ?? '').replace(/\s+/g, ' ');
		expect(text).toContain('NICHT gespeichert');
		expect(screen.container.querySelector('div')?.className).toContain('bg-error');
	});

	// Warten hilft nicht — und das Band muss es sagen.
	//
	// Fund vom 16.09.2026 (OFFEN.md 5.19): Der Sync endete bei jeder Antwort ab 400 mit
	// einem schlichten Abbruch. Gedacht war das für 502/503; bei einer Antwort, die sich
	// von selbst nicht ändert (403, weil der gerade angemeldete Mensch nicht buchen darf),
	// lief der Versuch jede Minute ins Leere. Sichtbar war nur der Zähler, und der sagt
	// „noch nicht im System", nicht „geht so nicht mehr".
	it('sagt es, wenn die Vorgänge mit dieser Anmeldung nicht zu buchen sind', () => {
		sync.wartendeAnzahl = 4;
		sync.abgelehnt = { status: 403, grund: 'Keine Berechtigung für diese Aktion' };
		const screen = render(OfflineAnzeige, {});
		const text = (screen.container.textContent ?? '').replace(/\s+/g, ' ');
		expect(text).toContain('darf an der Theke nicht buchen');
		expect(text).toContain('gespeichert');
		// Nicht die leise Fassung: Hier muss jemand etwas tun.
		expect(text).not.toContain('noch nicht im System');
		expect(screen.container.querySelector('div')?.className).toContain('bg-error');
	});

	it('nennt eine andere Ablehnung, ohne den Menschen mit einer Zahl allein zu lassen', () => {
		sync.wartendeAnzahl = 1;
		sync.abgelehnt = { status: 400, grund: 'Höchstens 50 Einträge je Aufruf' };
		const screen = render(OfflineAnzeige, {});
		const text = (screen.container.textContent ?? '').replace(/\s+/g, ' ');
		expect(text).toContain('1 Vorgang lässt sich nicht buchen: Höchstens 50 Einträge je Aufruf.');
		expect(text).toContain('Bibliothek verständigen');
	});

	// Ein Grund, der schon mit einem Punkt endet, bekommt keinen zweiten.
	it('setzt hinter einen Grund mit Punkt keinen zweiten', () => {
		sync.wartendeAnzahl = 2;
		sync.abgelehnt = { status: 400, grund: 'Die Anfrage ist zu groß. Bitte in Teilen senden.' };
		const screen = render(OfflineAnzeige, {});
		const text = (screen.container.textContent ?? '').replace(/\s+/g, ' ');
		expect(text).toContain(
			'2 Vorgänge lassen sich nicht buchen: Die Anfrage ist zu groß. Bitte in Teilen senden. Gespeichert'
		);
	});

	// Die Gegenprobe: Ein Server, der gleich wiederkommt, darf die Theke NICHT anschreien.
	it('bleibt bei 502 leise — dort hilft Warten', () => {
		sync.wartendeAnzahl = 2;
		sync.abgelehnt = null; // sendeBatch merkt sich 5xx bewusst nicht
		const screen = render(OfflineAnzeige, {});
		const text = (screen.container.textContent ?? '').replace(/\s+/g, ' ');
		expect(text).toContain('noch nicht im System');
		expect(screen.container.querySelector('div')?.className).not.toContain('bg-error ');
	});

	it('meldet den ausgefallenen Herzschlag im selben Band statt in einer zweiten Schicht', () => {
		// Bis zum 16.09.2026 legte App.svelte dafuer ein eigenes Vollbild darueber.
		const screen = render(OfflineAnzeige, { verbindungVerloren: true });
		expect(screen.container.textContent ?? '').toContain('Keine Verbindung');
	});

	it('zeigt gar nichts, wenn alles in Ordnung ist', () => {
		const screen = render(OfflineAnzeige, {});
		expect(screen.container.querySelector('div')).toBeNull();
	});

	// Schritt C: Was beim Nachbuchen nicht durchging, muss jemand sehen. Die Zeilen nennen
	// Ausleiher und Klasse — deshalb die Liste nur mit `view_students`, die ZAHL fuer jede
	// Theken-Rolle. Ein Helfer sieht also, dass etwas offen ist, und holt jemanden.
	it('zeigt offene Meldungen aus dem Nachbuchen, auch wenn sonst nichts anliegt', () => {
		meldungen.offen = 2;
		const screen = render(OfflineAnzeige, {});
		// Der Quelltext bricht den Satz um; gelesen wird er als eine Zeile.
		const text = (screen.container.textContent ?? '').replace(/\s+/g, ' ');

		expect(text).toContain('2 Buchungen aus dem Nachbuchen brauchen einen Blick');
		expect(screen.getByText(/Meldungen \(2\)/)).not.toBeNull();
	});

	it('bietet einem Helfer keinen Knopf, sondern den Hinweis', () => {
		meldungen.offen = 1;
		recht.wert = false;
		const screen = render(OfflineAnzeige, {});
		const text = (screen.container.textContent ?? '').replace(/\s+/g, ' ');

		expect(text).toContain('1 Buchung aus dem Nachbuchen braucht einen Blick');
		expect(text).toContain('Bitte die Bibliothek ansprechen');
		expect(screen.queryByText(/Meldungen \(/)).toBeNull();
	});

	it('bleibt still, wenn nichts offen ist und die Verbindung steht', () => {
		const screen = render(OfflineAnzeige, {});
		expect(screen.container.querySelector('div')).toBeNull();
	});
});
