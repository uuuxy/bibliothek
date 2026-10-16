import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import {
	srcRoot,
	relPfad,
	sammleQuelldateien,
	sammleTestdateien,
	ohneKommentare
} from './hygiene-quellen.js';

// Ratsche: Datei- und Ordnernamen der Oberfläche sind deutsch (docs/STIL.md, Abschnitt 13).
//
// Dieselbe Sache heißt im Go-Code und in der Oberfläche gleich: Wer in der Leserakte einen
// Fehler sucht, findet die Tür in api/leser und soll die Komponente nicht unter Student suchen.
// Gezählt werden die Wörter eines Pfads unter src/, zerlegt an Großbuchstaben, Unterstrichen und
// Bindestrichen, gegen eine feste Liste englischer Wörter. Lehnwörter (Barcode, Cover, Scan,
// Import, Upload), die Bauteile von M3 ohne deutschen Namen (Button, Chip, Dialog, Snackbar,
// Switch, Select, Radio, Badge, Tooltip) und Wörter von Svelte und dem Web (stores, actions,
// route, fetch) stehen nicht darin.
//
// BESTAND nennt je Pfad, was noch umzubenennen ist. Ein neuer Pfad mit einem englischen Wort ist
// rot, ein Pfad aus dem Bestand mit einem Wort mehr ebenso; wer einen Pfad umbenennt, trägt ihn
// aus.
//
// Der zweite Teil unten prüft die Namen, die eine Datei erklärt, mit derselben Liste.
//
// Sieht nicht: Wörter außerhalb der Liste und die Namen unter e2e/.
const ENGLISCH = new Set([
	'active',
	'alert',
	'area',
	'assign',
	'assignment',
	'audit',
	'book',
	'books',
	'borrowed',
	'borrowers',
	'camera',
	'capture',
	'card',
	'cart',
	'catalog',
	'chart',
	'checklist',
	'class',
	'config',
	'consistency',
	'create',
	'creation',
	'damage',
	'danger',
	'dashboard',
	'data',
	'dates',
	'delete',
	'deleted',
	'detector',
	'directory',
	'edit',
	'editor',
	'extend',
	'face',
	'flash',
	'form',
	'graduates',
	'grid',
	'history',
	'idle',
	'indicator',
	'inventory',
	'item',
	'keyboard',
	'label',
	'labels',
	'list',
	'lock',
	'login',
	'management',
	'manager',
	'media',
	'menu',
	'metadata',
	'modal',
	'order',
	'overdue',
	'page',
	'panel',
	'peek',
	'permission',
	'permissions',
	'picker',
	'preview',
	'print',
	'printer',
	'profile',
	'promote',
	'properties',
	'queue',
	'receipt',
	'recommendations',
	'report',
	'result',
	'results',
	'screen',
	'search',
	'selector',
	'session',
	'settings',
	'sheet',
	'sidebar',
	'sort',
	'sorting',
	'staging',
	'stats',
	'student',
	'students',
	'summary',
	'supplier',
	'tab',
	'tabs',
	'table',
	'templates',
	'toast',
	'toolbar',
	'unified',
	'user',
	'view',
	'widget',
	'workspace'
]);

const BESTAND = {
	'src/lib/TabelleSortKopf.test.js': ['sort'],
	'src/lib/components/ui/CoverPeek.svelte': ['peek'],
	'src/lib/components/ui/TabelleSortKopf.svelte': ['sort']
};

/** @param {string} name */
const stamm = (name) => name.replace(/\.(test|spec)\.js$|\.svelte\.js$|\.svelte$|\.js$/, '');

/** Die englischen Wörter eines Pfads unter src/, jedes einmal.
 * @param {string} pfad */
function englischeWoerter(pfad) {
	const teile = pfad.split('/').slice(1);
	/** @type {string[]} */
	const gefunden = [];
	teile.forEach((teil, i) => {
		const name = i === teile.length - 1 ? stamm(teil) : teil;
		for (const wort of name.match(/[A-Z]?[a-z]+|[A-Z]+(?![a-z])/g) ?? []) {
			const klein = wort.toLowerCase();
			if (ENGLISCH.has(klein) && !gefunden.includes(klein)) gefunden.push(klein);
		}
	});
	return gefunden;
}

describe('Datei- und Ordnernamen der Oberfläche sind deutsch', () => {
	const pfade = [...sammleQuelldateien(srcRoot), ...sammleTestdateien(srcRoot)].map(relPfad);
	/** @type {Map<string, string[]>} */
	const aktuell = new Map();
	for (const pfad of pfade) {
		const woerter = englischeWoerter(pfad);
		if (woerter.length > 0) aktuell.set(pfad, woerter);
	}

	it('liest überhaupt Dateien und erkennt ein englisches Wort', () => {
		expect(pfade.length).toBeGreaterThan(600);
		expect(englischeWoerter('src/lib/components/students/StudentLockModal')).toEqual([
			'students',
			'student',
			'lock',
			'modal'
		]);
		expect(englischeWoerter('src/lib/components/ui/BestaetigungsDialog')).toEqual([]);
	});

	it('kennt keinen neuen Pfad mit einem englischen Wort', () => {
		/** @type {string[]} */
		const neu = [];
		for (const [pfad, woerter] of aktuell) {
			const erlaubt = BESTAND[/** @type {keyof typeof BESTAND} */ (pfad)] ?? [];
			const zuviel = woerter.filter((w) => !erlaubt.includes(w));
			if (zuviel.length > 0) neu.push(`${pfad}: ${zuviel.join(', ')}`);
		}
		expect(
			neu,
			'Englisches Wort im Namen. Deutsch benennen nach docs/STIL.md, Abschnitt 13 (Wörterbuch in ' +
				'Abschnitt 2), oder ein Lehnwort aus der Liste nehmen.'
		).toEqual([]);
	});

	it('führt nichts, was es nicht mehr gibt oder inzwischen deutsch heißt', () => {
		const veraltet = Object.entries(BESTAND)
			.filter(([pfad, woerter]) => {
				const jetzt = aktuell.get(pfad) ?? [];
				return woerter.some((w) => !jetzt.includes(w));
			})
			.map(([pfad]) => pfad);
		expect(veraltet, 'Diese Einträge im Bestand austragen oder kürzen.').toEqual([]);
	});
});

// Ratsche: Namen innerhalb der Dateien sind deutsch (docs/STIL.md, Abschnitt 13), Stufe 2 des
// Umbaus (OFFEN 5.66). Gezählt werden die Namen, die eine Quelldatei erklärt (let, const, var,
// function, class), ohne Kommentare, zerlegt an Großbuchstaben, gegen dieselbe Liste wie die
// Pfade und dazu Wörter, die in Namen häufig sind und in Pfaden nicht vorkamen. Wörter von Svelte
// und dem Web (value, event, submit, open) stehen nicht darin, ebenso wenig die Bauteile von M3.
//
// NAMEN_BESTAND nennt je Datei, wie viele Erklärungen noch ein englisches Wort tragen; die Zahl
// kann nur sinken.
//
// Sieht nicht: Props, Schlüssel von Objekten, Parameter, Methoden der Stores und Namen in Tests.
const ENGLISCH_IN_NAMEN = new Set([
	...ENGLISCH,
	'handle',
	'error',
	'errors',
	'selected',
	'loading',
	'loaded',
	'load',
	'save',
	'saving',
	'deleting',
	'remove',
	'add',
	'query',
	'editing',
	'update',
	'users'
]);

const ERKLAERUNG = /\b(?:let|const|var|function|class)\s+([A-Za-z_$][\w$]*)/g;

/** Trägt der Name ein englisches Wort? @param {string} name */
function englischImNamen(name) {
	return (name.match(/[A-Z]?[a-z]+|[A-Z]+(?![a-z])/g) ?? []).some((w) =>
		ENGLISCH_IN_NAMEN.has(w.toLowerCase())
	);
}

/** @type {Record<string, number>} */
const NAMEN_BESTAND = {
	'src/inventur/lib/admin_api.js': 1,
	'src/inventur/lib/components/BuchKarte.svelte': 2,
	'src/inventur/lib/components/admin/AdminBuchAktionen.svelte': 2,
	'src/inventur/lib/components/admin/KlassenBuchKachel.svelte': 2,
	'src/inventur/lib/components/admin/KlassenKarte.svelte': 1,
	'src/inventur/lib/components/admin/KlassenUebersicht.svelte': 6,
	'src/inventur/lib/components/admin/KlassensatzDialog.svelte': 5,
	'src/inventur/lib/components/admin/KlassensatzTitelRaster.svelte': 2,
	'src/inventur/lib/components/admin/KlassensatzZuweisenDialog.svelte': 2,
	'src/inventur/lib/components/scanner/KameraScanner.svelte': 2,
	'src/inventur/lib/components/scanner/barcodeErkennung.js': 3,
	'src/inventur/lib/store.svelte.js': 1,
	'src/inventur/lib/titelSortierung.js': 2,
	'src/inventur/routes/KatalogSuche.svelte': 5,
	'src/lib/Abgaenger.svelte': 7,
	'src/lib/AusleiheRueckgabe.svelte': 1,
	'src/lib/AusweisDesigner.svelte': 4,
	'src/lib/BenutzerUndRechte.svelte': 7,
	'src/lib/Bestellwesen.svelte': 1,
	'src/lib/Buchakte.svelte': 1,
	'src/lib/BuchakteAusleiher.svelte': 2,
	'src/lib/BuchakteExemplare.svelte': 4,
	'src/lib/BuchakteVormerkungen.svelte': 4,
	'src/lib/Dialog.svelte': 1,
	'src/lib/DruckCenter.svelte': 1,
	'src/lib/Einstellungen.svelte': 1,
	'src/lib/Inventur.svelte': 1,
	'src/lib/Klassenleitungen.svelte': 2,
	'src/lib/Konten.svelte': 17,
	'src/lib/LMFVerlaengerung.svelte': 1,
	'src/lib/LeserAnlegenDialog.svelte': 1,
	'src/lib/LeserAusleihenListe.svelte': 1,
	'src/lib/LeserBearbeiten.svelte': 1,
	'src/lib/LeserGebuehrenKarte.svelte': 1,
	'src/lib/LeserLoeschenDialog.svelte': 2,
	'src/lib/LeserSperreDialog.svelte': 1,
	'src/lib/Leserakte.svelte': 1,
	'src/lib/LeserakteAktionen.svelte': 1,
	'src/lib/LitteraImport.svelte': 3,
	'src/lib/MailVorlagen.svelte': 6,
	'src/lib/Medienkatalog.svelte': 1,
	'src/lib/Omnibox.svelte': 3,
	'src/lib/OpacSuche.svelte': 3,
	'src/lib/SchadenMeldenDialog.svelte': 2,
	'src/lib/Schuelerdatei.svelte': 3,
	'src/lib/Statistik.svelte': 4,
	'src/lib/SystemLogs.svelte': 1,
	'src/lib/UeberfaelligQuote.svelte': 1,
	'src/lib/WebcamAufnahme.svelte': 6,
	'src/lib/abgaengerDienst.js': 1,
	'src/lib/actions/tastaturNavigation.js': 2,
	'src/lib/apiFetch.js': 4,
	'src/lib/audio.js': 2,
	'src/lib/components/ExemplarKarte.svelte': 7,
	'src/lib/components/ExemplarStatusBearbeiten.svelte': 6,
	'src/lib/components/GeraeteVerwaltung.svelte': 3,
	'src/lib/components/admin/BestandImport.svelte': 2,
	'src/lib/components/admin/Datenverwaltung.svelte': 3,
	'src/lib/components/admin/MailEinstellungen.svelte': 5,
	'src/lib/components/bestellungen/AnliegenListe.svelte': 2,
	'src/lib/components/bestellungen/BestaetigungEtiketten.svelte': 1,
	'src/lib/components/bestellungen/BestellHistorie.svelte': 1,
	'src/lib/components/bestellungen/BestellSuche.svelte': 4,
	'src/lib/components/bestellungen/KlassensatzReservierungen.svelte': 1,
	'src/lib/components/bestellungen/LieferantZeile.svelte': 5,
	'src/lib/components/bestellungen/Lieferanten.svelte': 2,
	'src/lib/components/bestellungen/Wareneingang.svelte': 2,
	'src/lib/components/bestellungen/WareneingangTabelle.svelte': 2,
	'src/lib/components/bestellungen/mittel.js': 1,
	'src/lib/components/einstellungen/kategorien/DatenschutzKategorie.svelte': 1,
	'src/lib/components/etiketten/EtikettEinstellungen.svelte': 1,
	'src/lib/components/etiketten/EtikettenNachdruck.svelte': 1,
	'src/lib/components/leser/KlassenDruckEinstieg.svelte': 1,
	'src/lib/components/leser/LusdImport.svelte': 11,
	'src/lib/components/leser/Papierkorb.svelte': 2,
	'src/lib/components/leser/Schuljahreswechsel.svelte': 4,
	'src/lib/components/leser/schuelerSuche.svelte.js': 4,
	'src/lib/components/mahnwesen/MahnwesenTabelle.svelte': 1,
	'src/lib/components/portal/PortalSchulbuecher.svelte': 1,
	'src/lib/components/portal/problemMeldung.svelte.js': 1,
	'src/lib/components/statistik/StatistikVerlauf.svelte': 1,
	'src/lib/components/ui/EingabeDialog.svelte': 1,
	'src/lib/components/ui/Feld.svelte': 1,
	'src/lib/components/ui/TabelleSortKopf.svelte': 1,
	'src/lib/designer/CanvasFlaeche.svelte': 2,
	'src/lib/designer/Werkzeugleiste.svelte': 1,
	'src/lib/designer/idDesignAltbestand.js': 1,
	'src/lib/designer/idDesignPersistenz.svelte.js': 1,
	'src/lib/designer/idDesignerStore.svelte.js': 4,
	'src/lib/inventurApi.js': 1,
	'src/lib/lmfplanDienst.js': 1,
	'src/lib/menue.js': 8,
	'src/lib/offlineWarteschlange.js': 1,
	'src/lib/passbildAusschnitt.js': 1,
	'src/lib/rechteKategorien.js': 1,
	'src/lib/routenAnwenden.js': 2,
	'src/lib/routenTabelle.js': 1,
	'src/lib/stores/authStore.svelte.js': 2,
	'src/lib/stores/bestellStore.svelte.js': 5,
	'src/lib/stores/druckWarteschlange.svelte.js': 1,
	'src/lib/stores/etiketten.svelte.js': 23,
	'src/lib/stores/leerlaufSperre.svelte.js': 1,
	'src/lib/stores/mahnwesen.svelte.js': 11,
	'src/lib/stores/mahnwesenMail.svelte.js': 1,
	'src/lib/stores/mahnwesenPdf.svelte.js': 3,
	'src/lib/stores/netzLage.svelte.js': 1,
	'src/lib/stores/offlineSync.svelte.js': 7,
	'src/lib/stores/omnibox.svelte.js': 29,
	'src/lib/stores/snackbars.svelte.js': 1,
	'src/lib/stores/uiStore.svelte.js': 3,
	'src/lib/useBuchakte.svelte.js': 9,
	'src/lib/useFehlbestand.svelte.js': 2,
	'src/lib/useInventur.svelte.js': 4,
	'src/lib/useInventurlauf.svelte.js': 14,
	'src/lib/useLeserakte.svelte.js': 18,
	'src/lib/useLeserakteFormular.svelte.js': 6
};

describe('Namen innerhalb der Dateien der Oberfläche sind deutsch', () => {
	/** @type {Map<string, number>} */
	const zaehlung = new Map();
	for (const datei of sammleQuelldateien(srcRoot)) {
		const text = ohneKommentare(readFileSync(datei, 'utf8'));
		let n = 0;
		for (const m of text.matchAll(ERKLAERUNG)) if (englischImNamen(m[1])) n++;
		if (n > 0) zaehlung.set(relPfad(datei), n);
	}

	it('liest überhaupt Dateien und erkennt einen englischen Namen', () => {
		expect(zaehlung.size).toBeGreaterThan(50);
		expect(englischImNamen('handleDeleteClick')).toBe(true);
		expect(englischImNamen('selectedTitel')).toBe(true);
		expect(englischImNamen('ladeLeser')).toBe(false);
		expect(englischImNamen('eventQuelle')).toBe(false);
	});

	it('kennt keine Datei mit mehr englischen Namen als im Bestand', () => {
		const neu = [...zaehlung]
			.filter(([pfad, n]) => n > (NAMEN_BESTAND[pfad] ?? 0))
			.map(([pfad, n]) => `${pfad}: ${n} (Bestand ${NAMEN_BESTAND[pfad] ?? 0})`);
		expect(
			neu,
			'Englisches Wort in einem erklärten Namen. Deutsch benennen nach docs/STIL.md, Abschnitte 2 ' +
				'und 13.'
		).toEqual([]);
	});

	it('führt keinen Bestand, der inzwischen kleiner ist', () => {
		const veraltet = Object.entries(NAMEN_BESTAND)
			.filter(([pfad, n]) => (zaehlung.get(pfad) ?? 0) < n)
			.map(([pfad, n]) => `${pfad}: ${n} → ${zaehlung.get(pfad) ?? 0}`);
		expect(veraltet, 'Diese Einträge im Bestand senken oder austragen.').toEqual([]);
	});
});
