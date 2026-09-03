import { uiStore } from './uiStore.svelte.js';
import { appState } from '../../inventur/lib/store.svelte.js';

// Springen: die eine Stelle, die aus einer Kennung eine Ansicht macht. Die globale
// Suchleiste (GlobalSuche) und künftig jeder Verweis nutzen sie — kein Bauteil setzt
// activeTab + Store-Parameter selbst zusammen (03.09.2026).

/** Buchakte öffnen. @param {string} titelId */
export function springeZuBuch(titelId) {
	appState.activeBookId = titelId;
	uiStore.activeTab = 'book_detail';
}

/** Schülerakte öffnen — die Schülerdatei greift requestedStudentId auf. @param {string} schuelerId */
export function springeZuSchueler(schuelerId) {
	uiStore.requestedStudentId = schuelerId;
	uiStore.activeTab = 'students_dir';
}
