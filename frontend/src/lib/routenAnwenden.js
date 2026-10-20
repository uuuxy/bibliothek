/**
 * @file routenAnwenden.js
 * Adresse → Bildschirm und zurück, die beiden Richtungen derselben Zuordnung. Das ist keine
 * Darstellung, sondern Zuordnung: Es liest und setzt Stores und zeichnet nichts.
 *
 * Beide Funktionen sind die einzige Quelle für Initial-Match und popstate, damit eine neue
 * Route nicht in einer von zwei Kopien fehlt.
 */
import { uiStore } from './stores/uiStore.svelte.js';
import { appState } from '../inventur/lib/store.svelte.js';
import { pfadJeBildschirm } from './routenTabelle.js';

/**
 * Setzt Tab (+ ggf. Store-Parameter) aus einem Pfad.
 * @param {string} path
 */
export function applyPathToState(path) {
	if (path === '/lmf-plan') {
		// Ältere Adresse, damit Lesezeichen weiter führen: System → Schuljahreswechsel.
		uiStore.activeTab = 'schuljahr';
		return;
	}
	if (path === '/lmf-aktionen') {
		// Ältere Adresse, damit Lesezeichen weiter führen: die Kategorie der Einstellungen.
		uiStore.activeTab = 'settings';
		uiStore.requestedSettingsTab = 'lmf';
		return;
	}
	if (path.startsWith('/medienkatalog/buch/')) {
		uiStore.activeTab = 'book_detail';
		appState.activeBookId = path.replace('/medienkatalog/buch/', '');
		return;
	}
	// Parametrisierte Sonderroute: der Tab braucht einen Zusatzparameter, passt nicht in pfadJeBildschirm.
	const statistikArt = path.startsWith('/statistiken/') && path.replace('/statistiken/', '');
	if (statistikArt && ['renner', 'ladenhueter'].includes(statistikArt)) {
		uiStore.activeTab = 'stats_detail';
		uiStore.statsDetailKind = /** @type {'renner'|'ladenhueter'} */ (statistikArt);
		return;
	}
	const passenderBildschirm = Object.keys(pfadJeBildschirm).find(
		(key) => pfadJeBildschirm[key] === path
	);
	if (passenderBildschirm) uiStore.activeTab = passenderBildschirm;
}

/** Zielpfad für den aktuellen Tab — inkl. der parametrisierten Sonderrouten. */
export function currentTargetPath() {
	if (uiStore.activeTab === 'book_detail' && appState.activeBookId) {
		return `/medienkatalog/buch/${appState.activeBookId}`;
	}
	if (uiStore.activeTab === 'stats_detail') {
		return `/statistiken/${uiStore.statsDetailKind}`;
	}
	return pfadJeBildschirm[uiStore.activeTab];
}
