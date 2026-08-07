import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { srcRoot, sammleQuelldateien, relPfad, vergleicheMitBestand } from './hygiene-quellen.js';

// Vierte Struktur-Invariante, dieselbe Bauart wie die drei in frontend-hygiene.test.js:
// Symbole kommen aus @lucide/svelte, nicht aus handgeschriebenem <svg>.
//
// Warum das zählt: Derselbe Pfad steht mehrfach im Baum — das Kiosk-Symbol allein
// viermal (menu.js, Sidebar 2×, BookTableToolbar). Jede Kopie hat ihre eigene
// Strichstärke, ihre eigene Kantenform und altert für sich. Material 3 verlangt an
// erster Stelle Konsistenz; ein Symbolsatz, der an 67 Stellen nachgezeichnet wird,
// ist genau das Gegenteil. Lucide liefert die Rolle EINMAL.
//
// Gemessen am 07.08.2026: 140 <svg> in 68 Dateien, 46 von 180 Komponenten nutzen
// bereits Lucide. Die Liste ist ein Bestand, KEINE Erlaubnis — sie friert nur ein,
// was schon da ist, damit nichts dazukommt, während der Tausch offen ist.
const SVG = /<svg[\s>]/;

// Echte Zeichnungen: Hier entsteht ein Bild, kein Symbol. Erkennbar am viewBox —
// alles andere im Baum ist 24×24 (bzw. 20×20), also eine Symbolfläche.
// Diese Liste schrumpft NICHT; sie ist die Ausnahme, nicht der Rückstand.
const ZEICHNUNGEN = [
	'src/lib/components/stats/StatsTrendChart.svelte' // Verlaufsgraph, viewBox aus den Daten
];

// ── Ratsche ─────────────────────────────────────────────────────────────────
// Wer eine Datei auf Lucide umstellt, nimmt sie hier heraus. Der Test meldet
// beides: neu hinzugekommene Dateien UND Einträge, die inzwischen sauber sind.
//
// Ein Sonderfall, damit ihn niemand zweimal untersucht: BuchCoverUpload.svelte
// zählt doppelt — ein Symbol UND ein Platzhalterbild als data:-URI im Attribut.
// Nach dem Symboltausch bleibt die Datei deshalb zu Recht in der Liste.
const SVG_BESTAND = [
	'src/inventur/lib/components/BuchKarte.svelte',
	'src/inventur/lib/components/KlassenSuchfeld.svelte',
	'src/inventur/lib/components/KlassenUebersichtStartseite.svelte',
	'src/inventur/lib/components/StartseitenFilter.svelte',
	'src/inventur/lib/components/admin/BookTableToolbar.svelte',
	'src/inventur/lib/components/admin/BookTableZeile.svelte',
	'src/inventur/lib/components/admin/BuchCoverUpload.svelte',
	'src/inventur/lib/components/admin/BuchExemplareListe.svelte',
	'src/inventur/lib/components/admin/BuchFormular.svelte',
	'src/inventur/lib/components/admin/ClassAssignmentBookGrid.svelte',
	'src/inventur/lib/components/admin/ClassAssignmentDialog.svelte',
	'src/inventur/lib/components/admin/ClassAssignmentSelector.svelte',
	'src/inventur/lib/components/admin/ClassAssignmentSummary.svelte',
	'src/inventur/lib/components/admin/IsbnFeld.svelte',
	'src/inventur/lib/components/admin/KlassenBuchKachel.svelte',
	'src/inventur/lib/components/admin/KlassenKarte.svelte',
	'src/inventur/lib/components/admin/KlassenUebersicht.svelte',
	'src/inventur/lib/components/scanner/FileUploader.svelte',
	'src/inventur/lib/components/scanner/StrichcodeScannerOverlay.svelte',
	'src/lib/BookAkte.svelte',
	'src/lib/BookBorrowersTab.svelte',
	'src/lib/BookExemplareTab.svelte',
	'src/lib/BookHistoryTab.svelte',
	'src/lib/BookVormerkungenTab.svelte',
	'src/lib/BorrowedBooksCard.svelte',
	'src/lib/BorrowedBooksList.svelte',
	'src/lib/CameraScanner.svelte',
	'src/lib/LehrerPortal.svelte',
	'src/lib/Mahnwesen.svelte',
	'src/lib/MailTemplates.svelte',
	'src/lib/OmniboxTeacherCard.svelte',
	'src/lib/OpacSearch.svelte',
	'src/lib/OverdueWidget.svelte',
	'src/lib/PermissionManager.svelte',
	'src/lib/StatsDashboard.svelte',
	'src/lib/StudentCreateModal.svelte',
	'src/lib/StudentEditSheet.svelte',
	'src/lib/StudentProfile.svelte',
	'src/lib/StudentProfileCard.svelte',
	'src/lib/StudentProfileDeleteModal.svelte',
	'src/lib/StudentProfileStammdaten.svelte',
	'src/lib/StudentVormerkungenCard.svelte',
	'src/lib/SystemSettingsRouting.svelte',
	'src/lib/UnifiedInventory.svelte',
	'src/lib/UserManagement.svelte',
	'src/lib/components/AbgaengerTabelle.svelte',
	'src/lib/components/BookExemplarCard.svelte',
	'src/lib/components/InventoryFinishModal.svelte',
	'src/lib/components/admin/DataManagement.svelte',
	'src/lib/components/bestellungen/BestellBerichte.svelte',
	'src/lib/components/bestellungen/IncomingShipments.svelte',
	'src/lib/components/bestellungen/KlassensatzReservierungen.svelte',
	'src/lib/components/bestellungen/OrderCart.svelte',
	'src/lib/components/bestellungen/OrderRecommendations.svelte',
	'src/lib/components/mahnwesen/MahnwesenDruckMenue.svelte',
	'src/lib/components/mahnwesen/MahnwesenAktionen.svelte',
	'src/lib/components/mahnwesen/MahnwesenSuchleiste.svelte',
	'src/lib/components/mahnwesen/MahnwesenTable.svelte',
	'src/lib/components/stats/StatistikDetailPage.svelte',
	'src/lib/components/students/ActiveStudentList.svelte',
	'src/lib/components/students/DeletedStudentList.svelte',
	'src/lib/components/students/LusdImportView.svelte',
	'src/lib/components/system/BackupStatusBadge.svelte',
	'src/lib/components/ui/CoverPeek.svelte',
	'src/lib/components/ui/Snackbar.svelte',
	'src/lib/designer/CanvasArea.svelte',
	'src/lib/designer/PropertiesPanel.svelte'
];

describe('Symbol-Hygiene', () => {
	it('zeichnet keine neuen Symbole von Hand (Icons kommen aus @lucide/svelte)', () => {
		const betroffen = sammleQuelldateien(srcRoot)
			.filter((f) => SVG.test(readFileSync(f, 'utf8')))
			.map(relPfad)
			.filter((f) => !ZEICHNUNGEN.includes(f))
			.sort();

		const { neu, inzwischenSauber } = vergleicheMitBestand(betroffen, [...SVG_BESTAND].sort());

		expect(
			neu,
			`Handgezeichnete Symbole. @lucide/svelte hat die Rolle bereits — importieren\n` +
				`statt nachzeichnen, sonst driften Strichstärke und Kantenform auseinander:\n  ${neu.join('\n  ')}`
		).toEqual([]);

		expect(
			inzwischenSauber,
			`Diese Dateien zeichnen nicht mehr selbst — bitte aus SVG_BESTAND entfernen,\ndamit die Ratsche greift:\n  ${inzwischenSauber.join('\n  ')}`
		).toEqual([]);
	});
});
