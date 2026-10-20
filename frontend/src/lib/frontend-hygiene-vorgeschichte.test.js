import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { srcRoot, relPfad, sammleQuelldateien, kommentarZeilen } from './hygiene-quellen.js';

// Ratsche: Ein Kommentar der Oberfläche nennt kein Datum (CLAUDE.md, Abschnitt 4).
//
// Ein Kommentar sagt, warum der Code so ist; wann er so wurde und was vorher dastand, steht in
// der Commit-Nachricht. Am Datum lässt sich diese Vorgeschichte zählen. Das Muster ist dasselbe
// wie stilDatum und stilQuelle in stil_ratsche_test.go: Ein Datum, das zu einer Quelle gehört
// („Erlass vom …", „abgerufen am …"), zählt nicht (docs/STIL.md, Abschnitt 8).
//
// BESTAND nennt je Datei, wie viele Kommentarzeilen noch ein Datum tragen; die Zahl kann nur
// sinken.
//
// Sieht nicht: ein Datum ohne Jahr („bis zum 16.09."), Vorgeschichte ohne Datum („bis dahin"),
// einen Zeilenkommentar hinter Code und die Tests. Ob ein Kommentar das Warum nennt, liest ein
// Mensch.
const DATUM = /\b\d{1,2}\.\d{1,2}\.20\d{2}\b|\b20\d{2}-\d{2}-\d{2}\b/;
const QUELLE =
	/(Erlass|Beschluss|Gesetz|Verordnung) vom \d{1,2}\.\d{1,2}\.20\d{2}|abgerufen am \d{1,2}\.\d{1,2}\.20\d{2}/g;

/** Trägt die Zeile ein Datum, das nicht zu einer Quelle gehört? @param {string} zeile */
const mitDatum = (zeile) => DATUM.test(zeile.replace(QUELLE, ''));

/** @type {Record<string, number>} */
const BESTAND = {
	'src/lib/actions/tastaturNavigation.js': 1,
	'src/lib/components/AbgaengerKopfzeile.svelte': 1,
	'src/lib/components/AbgaengerTabelle.svelte': 1,
	'src/lib/components/ExemplarEigentumDialog.svelte': 1,
	'src/lib/components/GeraeteVerwaltung.svelte': 1,
	'src/lib/components/NachbuchMeldungen.svelte': 1,
	'src/lib/components/OfflineAnzeige.svelte': 4,
	'src/lib/components/OmniboxAktiverLeser.svelte': 1,
	'src/lib/components/OmniboxPrueflisteDialog.svelte': 1,
	'src/lib/components/OmniboxThekeHinweise.svelte': 3,
	'src/lib/components/ThekenDaten.svelte': 1,
	'src/lib/components/admin/BestandImport.svelte': 1,
	'src/lib/components/admin/ListenImport.svelte': 2,
	'src/lib/components/auth/Sperrbildschirm.svelte': 1,
	'src/lib/components/bestand/Bestandsbuecher.svelte': 2,
	'src/lib/components/bestellungen/ArbeitsZeile.svelte': 1,
	'src/lib/components/bestellungen/BedarfZeile.svelte': 1,
	'src/lib/components/bestellungen/BestaetigungEtiketten.svelte': 1,
	'src/lib/components/bestellungen/BestellBerichte.svelte': 1,
	'src/lib/components/bestellungen/BestellHistorieKopf.svelte': 1,
	'src/lib/components/bestellungen/BestellHistorieTabelle.svelte': 1,
	'src/lib/components/bestellungen/BestellMittelDialog.svelte': 1,
	'src/lib/components/bestellungen/BestelllinkHinweis.svelte': 1,
	'src/lib/components/bestellungen/KlassensatzErledigte.svelte': 1,
	'src/lib/components/bestellungen/KlassensatzReservierungen.svelte': 2,
	'src/lib/components/bestellungen/LieferantZeile.svelte': 1,
	'src/lib/components/bestellungen/Lieferanten.svelte': 1,
	'src/lib/components/bestellungen/NeueAuflageDialog.svelte': 1,
	'src/lib/components/bestellungen/TrefferPruefung.svelte': 3,
	'src/lib/components/bestellungen/TrefferPruefungSchlagworte.svelte': 2,
	'src/lib/components/bestellungen/ausIsbn.js': 1,
	'src/lib/components/bestellungen/klassensatzListe.svelte.js': 2,
	'src/lib/components/einstellungen/KategorieDetail.svelte': 1,
	'src/lib/components/einstellungen/KategorieRahmen.svelte': 1,
	'src/lib/components/einstellungen/SchlagwortPflegeDialog.svelte': 2,
	'src/lib/components/einstellungen/kategorien.js': 5,
	'src/lib/components/einstellungen/kategorien/DatenschutzKategorie.svelte': 2,
	'src/lib/components/einstellungen/kategorien/ErreichbarkeitKategorie.svelte': 3,
	'src/lib/components/einstellungen/kategorien/LieferantenKategorie.svelte': 1,
	'src/lib/components/einstellungen/kategorien/MailKategorie.svelte': 1,
	'src/lib/components/einstellungen/kategorien/SchlagworteKategorie.svelte': 1,
	'src/lib/components/einstellungen/kategorien/SchuleKategorie.svelte': 3,
	'src/lib/components/einstellungen/kategorien/SommerferienKategorie.svelte': 1,
	'src/lib/components/einstellungen/schlagwortPflege.js': 1,
	'src/lib/components/einstellungen/schlagwortPflegeListe.svelte.js': 1,
	'src/lib/components/etiketten/AltbestandDialog.svelte': 1,
	'src/lib/components/etiketten/EtikettLayoutOptionen.svelte': 1,
	'src/lib/components/etiketten/EtikettenNachdruck.svelte': 2,
	'src/lib/components/exemplarErsatzwert.js': 2,
	'src/lib/components/geraeteListe.svelte.js': 2,
	'src/lib/components/inventur/ScanRueckmeldung.svelte': 1,
	'src/lib/components/layout/Hauptbereich.svelte': 1,
	'src/lib/components/layout/Seitenleiste.svelte': 1,
	'src/lib/components/layout/SeitenleisteFuss.svelte': 1,
	'src/lib/components/leser/AbgangsjahrFeld.svelte': 1,
	'src/lib/components/leser/AktiveLeserListe.svelte': 2,
	'src/lib/components/leser/EhemaligeListe.svelte': 2,
	'src/lib/components/leser/KlassenDruckEinstieg.svelte': 1,
	'src/lib/components/leser/KollegiumEingabeFelder.svelte': 2,
	'src/lib/components/leser/LeserArtWahl.svelte': 3,
	'src/lib/components/leser/LeserKontaktFelder.svelte': 2,
	'src/lib/components/leser/LeserKontoStatus.svelte': 4,
	'src/lib/components/leser/LeserakteFelder.svelte': 6,
	'src/lib/components/leser/LusdImport.svelte': 1,
	'src/lib/components/leser/SchuelerDokumente.svelte': 1,
	'src/lib/components/leser/SchuelerEingabeFelder.svelte': 1,
	'src/lib/components/leser/SchuelerZusammenfuehren.svelte': 1,
	'src/lib/components/leser/SchuelerZusammenfuehrenDialog.svelte': 1,
	'src/lib/components/leser/SchuelerdateiLeiste.svelte': 3,
	'src/lib/components/leser/klassenVorschlaege.svelte.js': 3,
	'src/lib/components/leser/papierkorbListe.svelte.js': 2,
	'src/lib/components/leser/schuelerSuche.svelte.js': 6,
	'src/lib/components/leser/zusammenfuehrenSuche.svelte.js': 1,
	'src/lib/components/lmfplan/LmfKlasseChip.svelte': 1,
	'src/lib/components/lmfplan/LmfPlanFreieTage.svelte': 2,
	'src/lib/components/lmfplan/LmfPlanKlassenZelle.svelte': 2,
	'src/lib/components/lmfplan/LmfPlanKopf.svelte': 3,
	'src/lib/components/lmfplan/LmfPlanPlatzZellen.svelte': 1,
	'src/lib/components/lmfplan/LmfPlanRahmen.svelte': 5,
	'src/lib/components/lmfplan/LmfPlanReihenfolge.svelte': 4,
	'src/lib/components/lmfplan/LmfPlanVorrat.svelte': 2,
	'src/lib/components/lmfplan/LmfPlanZeileAktionen.svelte': 1,
	'src/lib/components/mahnwesen/BescheidDialog.svelte': 1,
	'src/lib/components/mahnwesen/BescheideTabelle.svelte': 1,
	'src/lib/components/mahnwesen/MahnwesenAktionen.svelte': 2,
	'src/lib/components/mahnwesen/MahnwesenReiter.svelte': 1,
	'src/lib/components/mahnwesen/MahnwesenSuchleiste.svelte': 2,
	'src/lib/components/mahnwesen/MahnwesenTabelle.svelte': 1,
	'src/lib/components/mahnwesen/bescheidFormular.svelte.js': 1,
	'src/lib/components/omnibox/LeserTreffer.svelte': 1,
	'src/lib/components/omnibox/TitelTreffer.svelte': 1,
	'src/lib/components/portal/FachKarte.svelte': 2,
	'src/lib/components/portal/PortalLernmittel.svelte': 4,
	'src/lib/components/portal/PortalLmfPlan.svelte': 1,
	'src/lib/components/portal/PortalSchulbuecher.svelte': 2,
	'src/lib/components/portal/klassensatzReservierung.svelte.js': 1,
	'src/lib/components/portal/portalSuche.svelte.js': 1,
	'src/lib/components/statistik/statistikAbruf.svelte.js': 1,
	'src/lib/components/ui/Abschnitt.svelte': 1,
	'src/lib/components/ui/AuswahlLeiste.svelte': 1,
	'src/lib/components/ui/EingabeDialog.svelte': 2,
	'src/lib/components/ui/LadeFehler.svelte': 3,
	'src/lib/components/ui/Ladekreis.svelte': 1,
	'src/lib/components/ui/LogoRelief.svelte': 1,
	'src/lib/components/ui/Reiter.svelte': 4,
	'src/lib/components/ui/Segmente.svelte': 1,
	'src/lib/components/ui/Select.svelte': 1,
	'src/lib/components/ui/SelectListe.svelte': 1,
	'src/lib/components/ui/Tabelle.svelte': 2,
	'src/lib/components/ui/TabelleSortierKopf.svelte': 1,
	'src/lib/components/ui/Zaehlerpille.svelte': 1,
	'src/lib/components/ui/escapeSchliesst.js': 3,
	'src/lib/components/ui/fokusFalle.js': 1,
	'src/lib/components/ui/selectTastatur.js': 2,
	'src/lib/designer/CanvasElement.svelte': 1,
	'src/lib/designer/Druckvorschau.svelte': 1,
	'src/lib/designer/Kartenseite.svelte': 2,
	'src/lib/designer/VorlagenGalerie.svelte': 2,
	'src/lib/designer/WerkzeugleisteAuswahl.svelte': 1,
	'src/lib/designer/WerkzeugleisteDruck.svelte': 1,
	'src/lib/designer/ausweisVorlagen.js': 2,
	'src/lib/designer/idDesignAltbestand.js': 2,
	'src/lib/designer/idDesignPersistenz.svelte.js': 2,
	'src/lib/designer/idDesignerStore.svelte.js': 1,
	'src/lib/monitor/monitorTakt.svelte.js': 1,
	'src/lib/stores/authStore.svelte.js': 7,
	'src/lib/stores/bescheide.svelte.js': 1,
	'src/lib/stores/bestaetigung.svelte.js': 1,
	'src/lib/stores/bestellStore.svelte.js': 2,
	'src/lib/stores/buchBarcodes.svelte.js': 1,
	'src/lib/stores/etiketten.svelte.js': 5,
	'src/lib/stores/hintergrundAbrufe.svelte.js': 1,
	'src/lib/stores/mahnwesen.svelte.js': 1,
	'src/lib/stores/mahnwesenMail.svelte.js': 3,
	'src/lib/stores/nachbuchMeldungen.svelte.js': 1,
	'src/lib/stores/netzLage.svelte.js': 1,
	'src/lib/stores/offlineSync.svelte.js': 5,
	'src/lib/stores/thekeLeeren.js': 2,
	'src/lib/stores/uiStore.svelte.js': 3,
	'src/lib/utils/auflagenText.js': 1,
	'src/lib/utils/dnbSchlagwortVorschlag.svelte.js': 1,
	'src/lib/utils/escapeHtml.js': 1,
	'src/lib/utils/format.js': 9,
	'src/lib/utils/schlagwortVorschlaege.svelte.js': 2,
	'src/lib/utils/schlagworte.js': 1,
	'src/lib/utils/signaturen.js': 1
};

describe('Kommentare der Oberfläche nennen kein Datum', () => {
	/** @type {Map<string, number>} */
	const zaehlung = new Map();
	// Gelesen je Form des Kommentars: Ein Scanner, der eine Form verliert, fällt so auf, auch
	// wenn der Bestand schon leer ist.
	/** @type {Record<string, number>} */
	const formen = { '//': 0, '/*': 0, '<!--': 0 };
	for (const datei of sammleQuelldateien(srcRoot)) {
		const zeilen = kommentarZeilen(readFileSync(datei, 'utf8'));
		for (const form of Object.keys(formen)) {
			formen[form] += zeilen.filter((z) => z.trimStart().startsWith(form)).length;
		}
		const n = zeilen.filter(mitDatum).length;
		if (n > 0) zaehlung.set(relPfad(datei), n);
	}

	it('liest jede Form des Kommentars und erkennt ein Datum', () => {
		expect(formen['//']).toBeGreaterThan(1500);
		expect(formen['/*']).toBeGreaterThan(1000);
		expect(formen['<!--']).toBeGreaterThan(300);
		expect(mitDatum('// Bis zum 16.09.2026 stand hier ein Vollbild.')).toBe(true);
		expect(mitDatum(' * gemessen am 2026-09-04')).toBe(true);
		expect(mitDatum('// Nach dem Erlass vom 17.12.2014 gilt …')).toBe(false);
		expect(mitDatum('// Erlass vom 17.12.2014, umgesetzt am 02.09.2026')).toBe(true);
		expect(mitDatum('// Gültig bis 31.07.')).toBe(false);
	});

	it('kennt keine Datei mit mehr Kommentarzeilen mit Datum als im Bestand', () => {
		const neu = [...zaehlung]
			.filter(([pfad, n]) => n > (BESTAND[pfad] ?? 0))
			.map(([pfad, n]) => `${pfad}: ${n} (Bestand ${BESTAND[pfad] ?? 0})`);
		expect(
			neu,
			'Datum in einem Kommentar. Der Kommentar nennt das Warum, die Vorgeschichte steht in der ' +
				'Commit-Nachricht (CLAUDE.md, Abschnitt 4).'
		).toEqual([]);
	});

	it('führt keinen Bestand, der inzwischen kleiner ist', () => {
		const veraltet = Object.entries(BESTAND)
			.filter(([pfad, n]) => (zaehlung.get(pfad) ?? 0) < n)
			.map(([pfad, n]) => `${pfad}: ${n} → ${zaehlung.get(pfad) ?? 0}`);
		expect(veraltet, 'Diese Einträge im Bestand senken oder austragen.').toEqual([]);
	});
});
