/**
 * @file routenAnwenden.js
 * Adresse → Bildschirm und zurück. Die beiden Richtungen derselben Zuordnung.
 *
 * Herausgelöst aus Router.svelte am 17.09.2026, aus demselben Grund wie die Tabelle
 * selbst (routenTabelle.js): Router.svelte ist eine geduldete Datei an ihrer Obergrenze,
 * und eine geduldete Datei darf nicht weiter wachsen. Was hier steht, ist ohnehin keine
 * Darstellung, sondern Zuordnung — es liest und setzt Stores, es zeichnet nichts.
 *
 * Beide Funktionen sind die EINZIGE Quelle für Initial-Match UND popstate. Vorher lag
 * die book_detail-Logik in beiden Zweigen doppelt, und neue Routen wurden leicht in
 * einer der Kopien vergessen.
 */
import { uiStore } from './stores/uiStore.svelte.js';
import { appState } from '../inventur/lib/store.svelte.js';
import { tabToPath } from './routenTabelle.js';

/**
 * Setzt Tab (+ ggf. Store-Parameter) aus einem Pfad.
 * @param {string} path
 */
export function applyPathToState(path) {
	if (path === '/lmf-plan') {
		// Alte Adresse (Menüpunkt am 05.09.2026): jetzt System → Schuljahreswechsel.
		uiStore.activeTab = 'schuljahr';
		return;
	}
	if (path === '/lmf-aktionen') {
		// Alte Adresse (Menüpunkt bis 24.08.2026): jetzt Einstellungs-Kategorie.
		uiStore.activeTab = 'settings';
		uiStore.requestedSettingsTab = 'lmf';
		return;
	}
	if (path.startsWith('/medienkatalog/buch/')) {
		uiStore.activeTab = 'book_detail';
		appState.activeBookId = path.replace('/medienkatalog/buch/', '');
		return;
	}
	// Parametrisierte Sonderroute: der Tab braucht einen Zusatzparameter, passt nicht in tabToPath.
	const statsKind = path.startsWith('/statistiken/') && path.replace('/statistiken/', '');
	if (statsKind && ['renner', 'ladenhueter'].includes(statsKind)) {
		uiStore.activeTab = 'stats_detail';
		uiStore.statsDetailKind = /** @type {'renner'|'ladenhueter'} */ (statsKind);
		return;
	}
	const matchedTab = Object.keys(tabToPath).find((key) => tabToPath[key] === path);
	if (matchedTab) uiStore.activeTab = matchedTab;
}

/** Zielpfad für den aktuellen Tab — inkl. der parametrisierten Sonderrouten. */
export function currentTargetPath() {
	if (uiStore.activeTab === 'book_detail' && appState.activeBookId) {
		return `/medienkatalog/buch/${appState.activeBookId}`;
	}
	if (uiStore.activeTab === 'stats_detail') {
		return `/statistiken/${uiStore.statsDetailKind}`;
	}
	return tabToPath[uiStore.activeTab];
}
