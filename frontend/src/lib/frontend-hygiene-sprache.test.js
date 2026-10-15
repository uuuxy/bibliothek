import { describe, it, expect } from 'vitest';
import { srcRoot, relPfad, sammleQuelldateien, sammleTestdateien } from './hygiene-quellen.js';

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
// Sieht nicht: Namen innerhalb der Dateien, Wörter außerhalb der Liste und die Namen unter e2e/.
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
	'src/inventur/lib/components/admin/ClassAssignPicker.svelte': ['class', 'assign', 'picker'],
	'src/inventur/lib/components/admin/ClassAssignmentBookGrid.svelte': [
		'class',
		'assignment',
		'book',
		'grid'
	],
	'src/inventur/lib/components/admin/ClassAssignmentBookGrid.test.js': [
		'class',
		'assignment',
		'book',
		'grid'
	],
	'src/inventur/lib/components/admin/ClassAssignmentDialog.svelte': ['class', 'assignment'],
	'src/inventur/lib/components/admin/ClassAssignmentSelector.svelte': [
		'class',
		'assignment',
		'selector'
	],
	'src/inventur/lib/components/admin/ClassAssignmentSummary.svelte': [
		'class',
		'assignment',
		'summary'
	],
	'src/inventur/lib/components/admin/buch_form_optionen.js': ['form'],
	'src/inventur/lib/components/admin/buch_form_optionen.test.js': ['form'],
	'src/inventur/lib/components/scanner/barcode_detector.js': ['detector'],
	'src/inventur/lib/components/scanner/barcode_detector.test.js': ['detector'],
	'src/inventur/routes/+page.svelte': ['page'],
	'src/inventur/routes/admin/+page.svelte': ['page'],
	'src/lib/AdminAuditLog.svelte': ['audit'],
	'src/lib/AuditLog.svelte': ['audit'],
	'src/lib/CameraScanner.svelte': ['camera'],
	'src/lib/DamageReportModal.svelte': ['damage', 'report', 'modal'],
	'src/lib/GlobalLMFExtendWidget.svelte': ['extend', 'widget'],
	'src/lib/Graduates.svelte': ['graduates'],
	'src/lib/LabelPrinter.svelte': ['label', 'printer'],
	'src/lib/LitteraImportWidget.svelte': ['widget'],
	'src/lib/MailTemplates.svelte': ['templates'],
	'src/lib/MailTemplates.test.js': ['templates'],
	'src/lib/MediaCatalog.svelte': ['media', 'catalog'],
	'src/lib/Modal.svelte': ['modal'],
	'src/lib/OpacSearch.svelte': ['search'],
	'src/lib/OpacSearch.test.js': ['search'],
	'src/lib/OverdueWidget.svelte': ['overdue', 'widget'],
	'src/lib/SignaturenView.svelte': ['view'],
	'src/lib/StatsDashboard.svelte': ['stats', 'dashboard'],
	'src/lib/SystemSettings.svelte': ['settings'],
	'src/lib/SystemSettingsRouting.svelte': ['settings'],
	'src/lib/TabelleSortKopf.test.js': ['sort'],
	'src/lib/ToastContainer.svelte': ['toast'],
	'src/lib/UnifiedInventory.svelte': ['unified', 'inventory'],
	'src/lib/WebcamCapture.svelte': ['capture'],
	'src/lib/actions/keyboardNav.js': ['keyboard'],
	'src/lib/actions/keyboardNav.test.js': ['keyboard'],
	'src/lib/components/InventoryStartModal.svelte': ['inventory', 'modal'],
	'src/lib/components/OfflineIndicator.svelte': ['indicator'],
	'src/lib/components/OfflineIndicator.test.js': ['indicator'],
	'src/lib/components/OmniboxBlockAlert.svelte': ['alert'],
	'src/lib/components/OmniboxBlockAlert.test.js': ['alert'],
	'src/lib/components/OmniboxChecklistDialog.svelte': ['checklist'],
	'src/lib/components/OmniboxChecklistDialog.test.js': ['checklist'],
	'src/lib/components/OmniboxResults.svelte': ['results'],
	'src/lib/components/OmniboxScreenFlash.svelte': ['screen', 'flash'],
	'src/lib/components/OmniboxVormerkungAlert.svelte': ['alert'],
	'src/lib/components/admin/BestandImportWidget.svelte': ['widget'],
	'src/lib/components/admin/DataManagement.svelte': ['data', 'management'],
	'src/lib/components/admin/ListenImportWidget.svelte': ['widget'],
	'src/lib/components/admin/MailConfig.svelte': ['config'],
	'src/lib/components/admin/MailConfig.test.js': ['config'],
	'src/lib/components/auth/Login.svelte': ['login'],
	'src/lib/components/labels/AltbestandDialog.svelte': ['labels'],
	'src/lib/components/labels/EtikettenListe.svelte': ['labels'],
	'src/lib/components/labels/EtikettenListe.test.js': ['labels'],
	'src/lib/components/labels/EtikettenNachdruck.svelte': ['labels'],
	'src/lib/components/labels/LabelBarcodeSchritt.svelte': ['labels', 'label'],
	'src/lib/components/labels/LabelBarcodeSchritt.test.js': ['labels', 'label'],
	'src/lib/components/labels/LabelLayoutOptionen.svelte': ['labels', 'label'],
	'src/lib/components/labels/LabelPreview.svelte': ['labels', 'label', 'preview'],
	'src/lib/components/labels/LabelPreview.test.js': ['labels', 'label', 'preview'],
	'src/lib/components/labels/LabelSettings.svelte': ['labels', 'label', 'settings'],
	'src/lib/components/layout/PageShell.svelte': ['page'],
	'src/lib/components/layout/Sidebar.svelte': ['sidebar'],
	'src/lib/components/layout/Sidebar.test.js': ['sidebar'],
	'src/lib/components/layout/SidebarFooter.svelte': ['sidebar'],
	'src/lib/components/mahnwesen/MahnwesenTable.svelte': ['table'],
	'src/lib/components/mahnwesen/MahnwesenTable.test.js': ['table'],
	'src/lib/components/mahnwesen/MahnwesenTabs.svelte': ['tabs'],
	'src/lib/components/settings/KategorieDetail.svelte': ['settings'],
	'src/lib/components/settings/KategorieListe.svelte': ['settings'],
	'src/lib/components/settings/KategorieRahmen.svelte': ['settings'],
	'src/lib/components/settings/SchlagwortPflegeDialog.svelte': ['settings'],
	'src/lib/components/settings/SchlagwortTabelle.svelte': ['settings'],
	'src/lib/components/settings/kategorien.js': ['settings'],
	'src/lib/components/settings/kategorien/AusleiheKategorie.svelte': ['settings'],
	'src/lib/components/settings/kategorien/BestellwesenKategorie.svelte': ['settings'],
	'src/lib/components/settings/kategorien/DatenschutzKategorie.svelte': ['settings'],
	'src/lib/components/settings/kategorien/ErreichbarkeitKategorie.svelte': ['settings'],
	'src/lib/components/settings/kategorien/LieferantenKategorie.svelte': ['settings'],
	'src/lib/components/settings/kategorien/MahnwesenKategorie.svelte': ['settings'],
	'src/lib/components/settings/kategorien/MailKategorie.svelte': ['settings'],
	'src/lib/components/settings/kategorien/SchadensersatzKategorie.svelte': ['settings'],
	'src/lib/components/settings/kategorien/SchlagworteKategorie.svelte': ['settings'],
	'src/lib/components/settings/kategorien/SchlagworteKategorie.test.js': ['settings'],
	'src/lib/components/settings/kategorien/SchuleKategorie.svelte': ['settings'],
	'src/lib/components/settings/kategorien/SommerferienKategorie.svelte': ['settings'],
	'src/lib/components/settings/kategorien/SommerferienKategorie.test.js': ['settings'],
	'src/lib/components/settings/schlagwortPflege.js': ['settings'],
	'src/lib/components/settings/schlagwortPflege.test.js': ['settings'],
	'src/lib/components/settings/schlagwortPflegeListe.svelte.js': ['settings'],
	'src/lib/components/settings/schlagwortPflegeListe.svelte.test.js': ['settings'],
	'src/lib/components/stats/StatistikDetailPage.svelte': ['stats', 'page'],
	'src/lib/components/stats/StatistikDetailPage.test.js': ['stats', 'page'],
	'src/lib/components/stats/StatsTrendChart.svelte': ['stats', 'chart'],
	'src/lib/components/stats/statistikAbruf.svelte.js': ['stats'],
	'src/lib/components/stats/statistikAbruf.test.js': ['stats'],
	'src/lib/components/system/BackupAlert.svelte': ['alert'],
	'src/lib/components/ui/CoverPeek.svelte': ['peek'],
	'src/lib/components/ui/TabelleSortKopf.svelte': ['sort'],
	'src/lib/designer/CanvasArea.svelte': ['area'],
	'src/lib/designer/CardFace.svelte': ['card', 'face'],
	'src/lib/designer/CardFace.test.js': ['card', 'face'],
	'src/lib/designer/PrintPreview.svelte': ['print', 'preview'],
	'src/lib/designer/PropertiesPanel.svelte': ['properties', 'panel'],
	'src/lib/designer/PropertiesText.svelte': ['properties'],
	'src/lib/designer/Toolbar.svelte': ['toolbar'],
	'src/lib/designer/ToolbarAuswahl.svelte': ['toolbar'],
	'src/lib/designer/ToolbarDruck.svelte': ['toolbar'],
	'src/lib/leserMaskeEineForm.test.js': ['form'],
	'src/lib/menu.js': ['menu'],
	'src/lib/menu.test.js': ['menu'],
	'src/lib/offlineQueue.js': ['queue'],
	'src/lib/offlineQueue.test.js': ['queue'],
	'src/lib/routing-consistency.test.js': ['consistency'],
	'src/lib/settingsWerte.js': ['settings'],
	'src/lib/settingsWerte.test.js': ['settings'],
	'src/lib/stores/idleLock.effekt.svelte.test.js': ['idle', 'lock'],
	'src/lib/stores/idleLock.sperre.test.js': ['idle', 'lock'],
	'src/lib/stores/idleLock.svelte.js': ['idle', 'lock'],
	'src/lib/stores/idleLock.test.js': ['idle', 'lock'],
	'src/lib/stores/labels.svelte.js': ['labels'],
	'src/lib/stores/labels.test.js': ['labels'],
	'src/lib/stores/printQueue.svelte.js': ['print', 'queue'],
	'src/lib/stores/toastStore.svelte.js': ['toast'],
	'src/lib/stores/toastStore.test.js': ['toast'],
	'src/lib/useInventurSession.svelte.js': ['session'],
	'src/lib/useUnifiedInventory.svelte.js': ['unified', 'inventory'],
	'src/lib/useUnifiedInventory.test.js': ['unified', 'inventory'],
	'src/lib/utils/dates.js': ['dates'],
	'src/lib/utils/dates.test.js': ['dates']
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
