<script>
	import { authStore } from './stores/authStore.svelte.js';
	import { uiStore } from './stores/uiStore.svelte.js';
	import { leerlaufSperre } from './stores/leerlaufSperre.svelte.js';
	import { appState } from '../inventur/lib/store.svelte.js';
	import { erlaubteBildschirme, bildschirmGesperrt } from './menue.js';
	import { escapeGehoertJemandAnderem } from './escapeRegel.js';

	import Berechtigungen from './Berechtigungen.svelte';
	import Omnibox from './Omnibox.svelte';
	import Buchakte from './Buchakte.svelte';
	import Bestellwesen from './Bestellwesen.svelte';
	import Inventur from './Inventur.svelte';
	import Medienkatalog from './Medienkatalog.svelte';
	import Bestandsbuecher from './components/bestand/Bestandsbuecher.svelte';
	import BestellBerichte from './components/bestellungen/BestellBerichte.svelte';
	import Signaturen from './Signaturen.svelte';
	import Statistik from './Statistik.svelte';
	import Schuelerdatei from './Schuelerdatei.svelte';
	import Schulklassen from './Schulklassen.svelte';
	import KollegiumPortal from './KollegiumPortal.svelte';
	import Mahnwesen from './Mahnwesen.svelte';
	import StatistikDetail from './components/statistik/StatistikDetail.svelte';
	import Einstellungen from './Einstellungen.svelte';
	import DruckCenter from './DruckCenter.svelte';
	import SystemLogs from './SystemLogs.svelte';
	import Abgaenger from './Abgaenger.svelte';
	import LmfPlan from './LmfPlan.svelte';
	import RouteFallback from './components/layout/RouteFallback.svelte';
	import { applyPathToState, currentTargetPath } from './routenAnwenden.js';

	// ── Wer darf welchen Bildschirm? Eine Regel, nicht zwei ────────────────────
	// Der Router liest dieselbe Regel wie das Menü nur ab (erlaubteBildschirme und
	// bildschirmGesperrt bei darfSehen in menue.js): Eine zweite Liste liefe auseinander, und
	// ein Menüpunkt, der beim Klick wortlos an die Theke wirft, sieht aus wie ein Defekt der
	// Seite dahinter.

	function oeffneTitel(book) {
		// Ein in der Omnibox angeklicktes Buch soll die Detail-/Akte-Ansicht dieses Buchs
		// öffnen (book_detail → Buchakte via bookId, inkl. Deep-Link /medienkatalog/buch/{id}) —
		// NICHT den allgemeinen Medienkatalog.
		if (!book?.id) return;
		appState.activeBookId = book.id;
		uiStore.activeTab = 'book_detail';
	}

	// Routing effects
	$effect(() => {
		if (authStore.isLoggedIn && authStore.currentUser) {
			const path = window.location.pathname;
			const erlaubt = erlaubteBildschirme(authStore.currentUser);

			if (!uiStore.isInitialRouteMatched && path !== '/') {
				applyPathToState(path);
			}
			uiStore.isInitialRouteMatched = true;

			// Gesperrten Bildschirm auf den ERSTEN erlaubten zurückstellen — für den
			// Helfer ist das der Kiosk (Gruppe „Kiosk" steht in menue.js oben), für die
			// Lehrkraft ihr Portal. Kein Rollenname im Router: Wer eine Rolle ergänzt,
			// pflegt ihre Rechte in menue.js und hier gar nichts.
			if (bildschirmGesperrt(uiStore.activeTab, erlaubt)) {
				uiStore.activeTab = [...erlaubt][0] ?? 'kiosk';
			}

			const targetPath = currentTargetPath();
			if (targetPath && path !== targetPath) {
				window.history.pushState(null, '', targetPath);
			}
		}
	});

	// Escape-Regel: escapeRegel.js (Theke, außer die Taste gehört schon jemandem).
	$effect(() => {
		/** @param {KeyboardEvent} e */
		function beiTaste(e) {
			if (e.key !== 'Escape' || uiStore.activeTab === 'kiosk') return;
			if (escapeGehoertJemandAnderem(e)) return;
			uiStore.activeTab = 'kiosk';
		}
		function beiVerlaufswechsel() {
			// Hinter der Sperre bleibt der Bildschirm, wie er ist: Die Zurück-Taste des
			// Browsers verließe sonst die offene Maske. Die Adresszeile geht zurück.
			if (leerlaufSperre.gesperrt) {
				window.history.pushState(null, '', currentTargetPath());
				return;
			}
			applyPathToState(window.location.pathname);
			// Verlassen-Schutz hat angehalten (uiStore): Adresszeile auf den offenen Tab zurück.
			if (uiStore.blockierterWechsel !== null) {
				window.history.pushState(null, '', currentTargetPath());
			}
		}
		window.addEventListener('keydown', beiTaste);
		window.addEventListener('popstate', beiVerlaufswechsel);
		return () => {
			window.removeEventListener('keydown', beiTaste);
			window.removeEventListener('popstate', beiVerlaufswechsel);
		};
	});
</script>

<div class="print-flow flex-1 overflow-y-auto flex flex-col w-full">
	{#if uiStore.activeTab === 'kiosk'}
		<div class="flex-1 flex flex-col w-full animate-fade-in">
			<Omnibox onSelectBook={oeffneTitel} />
		</div>
	{:else if uiStore.activeTab === 'orders'}
		<div class="w-full animate-fade-in"><Bestellwesen /></div>
	{:else if uiStore.activeTab === 'stats'}
		<!-- flex-1: die Statistik-Fläche reicht bis zum unteren Rand, auch wenn der Inhalt
		     kürzer als der Viewport ist.
		     min-h-0: ohne das behält der Flex-Item seine implizite min-height:auto und kann
		     nicht unter seine Inhaltshöhe schrumpfen — das Dashboard könnte seine Karten dann
		     nicht auf die Viewport-Höhe deckeln. -->
		<div class="w-full flex-1 min-h-0 flex flex-col animate-fade-in"><Statistik /></div>
	{:else if uiStore.activeTab === 'stats_detail'}
		<div class="w-full animate-fade-in"><StatistikDetail /></div>
	{:else if uiStore.activeTab === 'system-logs'}
		<div class="w-full animate-fade-in h-full"><SystemLogs /></div>
	{:else if uiStore.activeTab === 'druck-center'}
		<div class="w-full animate-fade-in h-full"><DruckCenter /></div>
	{:else if uiStore.activeTab === 'media_catalog'}
		<div class="w-full animate-fade-in"><Medienkatalog /></div>
	{:else if uiStore.activeTab === 'bestellberichte'}
		<BestellBerichte />
	{:else if uiStore.activeTab === 'bestandsbuecher'}
		<div class="w-full animate-fade-in"><Bestandsbuecher /></div>
	{:else if uiStore.activeTab === 'signaturen'}
		<div class="w-full animate-fade-in"><Signaturen /></div>
	{:else if uiStore.activeTab === 'inventory'}
		<div class="w-full animate-fade-in"><Inventur /></div>
	{:else if uiStore.activeTab === 'students_dir'}
		<div class="w-full animate-fade-in"><Schuelerdatei /></div>
	{:else if uiStore.activeTab === 'schulklassen'}
		<div class="w-full animate-fade-in"><Schulklassen /></div>
	{:else if uiStore.activeTab === 'graduates'}
		<div class="w-full animate-fade-in"><Abgaenger /></div>
	{:else if uiStore.activeTab === 'schuljahr'}
		<div class="w-full animate-fade-in"><LmfPlan /></div>
	{:else if uiStore.activeTab === 'mahnwesen'}
		<div class="w-full animate-fade-in"><Mahnwesen /></div>
	{:else if uiStore.activeTab === 'kollegium_portal'}
		<div class="w-full animate-fade-in"><KollegiumPortal konto={authStore.currentUser} /></div>
	{:else if uiStore.activeTab === 'berechtigungen'}
		<div class="w-full animate-fade-in"><Berechtigungen /></div>
	{:else if uiStore.activeTab === 'settings'}
		<div class="w-full animate-fade-in"><Einstellungen /></div>
	{:else if uiStore.activeTab === 'book_detail'}
		<div class="w-full animate-fade-in">
			<Buchakte
				titelId={appState.activeBookId}
				onBack={() => {
					uiStore.activeTab = 'media_catalog';
					appState.activeBookId = null;
				}}
			/>
		</div>
	{:else}
		<!-- Unbekannter Tab: sichtbarer Fallback statt lautloser weißer Seite (+ Sentry). -->
		<RouteFallback bildschirm={uiStore.activeTab} />
	{/if}
</div>
