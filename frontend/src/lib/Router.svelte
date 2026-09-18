<script>
	import { authStore } from './stores/authStore.svelte.js';
	import { uiStore } from './stores/uiStore.svelte.js';
	import { appState } from '../inventur/lib/store.svelte.js';
	import { erlaubteTabs, tabIstGesperrt } from './menu.js';
	import { escapeGehoertJemandAnderem } from './escapeRegel.js';

	import Berechtigungen from './Berechtigungen.svelte';
	import Omnibox from './Omnibox.svelte';
	import BookAkte from './BookAkte.svelte';
	import BestellWorkspace from './BestellWorkspace.svelte';
	import UnifiedInventory from './UnifiedInventory.svelte';
	import MediaCatalog from './MediaCatalog.svelte';
	import Bestandsbuecher from './components/bestand/Bestandsbuecher.svelte';
	import BestellBerichte from './components/bestellungen/BestellBerichte.svelte';
	import SignaturenView from './SignaturenView.svelte';
	import StatsDashboard from './StatsDashboard.svelte';
	import StudentDirectory from './StudentDirectory.svelte';
	import Schulklassen from './Schulklassen.svelte';
	import KollegiumPortal from './KollegiumPortal.svelte';
	import Mahnwesen from './Mahnwesen.svelte';
	import StatistikDetailPage from './components/stats/StatistikDetailPage.svelte';
	import SystemSettings from './SystemSettings.svelte';
	import DruckCenter from './DruckCenter.svelte';
	import SystemLogs from './SystemLogs.svelte';
	import Graduates from './Graduates.svelte';
	import LmfPlan from './LmfPlan.svelte';
	import RouteFallback from './components/layout/RouteFallback.svelte';
	import { applyPathToState, currentTargetPath } from './routenAnwenden.js';

	// ── Wer darf welchen Bildschirm? EINE Regel, nicht zwei ────────────────────
	// Bis zum 08.08.2026 stand hier eine handgepflegte Helfer-Liste ('kiosk',
	// 'media_catalog') neben canSeeItem im Menü — zwei Definitionen, die auseinanderliefen:
	// Als „Schulklassen" von manage_users auf view_books wechselte, sah der Helfer den
	// Menüpunkt und wurde beim Klick wortlos an die Theke geworfen. Ein Menüpunkt, der
	// nichts tut, sieht aus wie ein Defekt der Seite dahinter. Jetzt liest der Router
	// dieselbe Regel (erlaubteTabs/tabIstGesperrt bei canSeeItem in menu.js) nur ab.

	function handleSelectBook(book) {
		// Ein in der Omnibox angeklicktes Buch soll die Detail-/Akte-Ansicht dieses Buchs
		// öffnen (book_detail → BookAkte via bookId, inkl. Deep-Link /medienkatalog/buch/{id}) —
		// NICHT den allgemeinen Medienkatalog.
		if (!book?.id) return;
		appState.activeBookId = book.id;
		uiStore.activeTab = 'book_detail';
	}

	// Routing effects
	$effect(() => {
		if (authStore.isLoggedIn && authStore.currentUser) {
			const path = window.location.pathname;
			const erlaubt = erlaubteTabs(authStore.currentUser);

			if (!uiStore.isInitialRouteMatched && path !== '/') {
				applyPathToState(path);
			}
			uiStore.isInitialRouteMatched = true;

			// Gesperrten Bildschirm auf den ERSTEN erlaubten zurückstellen — für den
			// Helfer ist das der Kiosk (Gruppe „Kiosk" steht in menu.js oben), für die
			// Lehrkraft ihr Portal. Kein Rollenname im Router: Wer eine Rolle ergänzt,
			// pflegt ihre Rechte in menu.js und hier gar nichts.
			if (tabIstGesperrt(uiStore.activeTab, erlaubt)) {
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
		function handleGlobalKeyDown(e) {
			if (e.key !== 'Escape' || uiStore.activeTab === 'kiosk') return;
			if (escapeGehoertJemandAnderem(e)) return;
			uiStore.activeTab = 'kiosk';
		}
		function handlePopState() {
			applyPathToState(window.location.pathname);
			// Verlassen-Schutz hat angehalten (uiStore): Adresszeile auf den offenen Tab zurück.
			if (uiStore.blockierterWechsel !== null) {
				window.history.pushState(null, '', currentTargetPath());
			}
		}
		window.addEventListener('keydown', handleGlobalKeyDown);
		window.addEventListener('popstate', handlePopState);
		return () => {
			window.removeEventListener('keydown', handleGlobalKeyDown);
			window.removeEventListener('popstate', handlePopState);
		};
	});
</script>

<div class="flex-1 overflow-y-auto flex flex-col w-full">
	{#if uiStore.activeTab === 'kiosk'}
		<div class="flex-1 flex flex-col w-full animate-fade-in">
			<Omnibox onSelectBook={handleSelectBook} />
		</div>
		<!-- Der Zweig 'books' ist entfallen (nie gesetzt, unerreichbar; Audit 01.08.2026) —
		     die Buchansicht läuft über 'book_detail' und appState.activeBookId. -->
	{:else if uiStore.activeTab === 'orders'}
		<div class="w-full animate-fade-in"><BestellWorkspace /></div>
	{:else if uiStore.activeTab === 'stats'}
		<!-- flex-1: die graue Statistik-Fläche (bg-slate-50) reicht bis zum unteren Rand,
		     auch wenn der Inhalt kürzer als der Viewport ist.
		     min-h-0: ohne das behält der Flex-Item seine implizite min-height:auto und kann
		     NICHT unter seine Inhaltshöhe schrumpfen — das Dashboard könnte seine Bento-Cards
		     dann nicht auf die Viewport-Höhe deckeln und liefe wieder in einen Scroll-Wurm. -->
		<div class="w-full flex-1 min-h-0 flex flex-col animate-fade-in"><StatsDashboard /></div>
	{:else if uiStore.activeTab === 'stats_detail'}
		<div class="w-full animate-fade-in"><StatistikDetailPage /></div>
	{:else if uiStore.activeTab === 'system-logs'}
		<div class="w-full animate-fade-in h-full"><SystemLogs /></div>
	{:else if uiStore.activeTab === 'druck-center'}
		<div class="w-full animate-fade-in h-full"><DruckCenter /></div>
	{:else if uiStore.activeTab === 'media_catalog'}
		<div class="w-full animate-fade-in"><MediaCatalog /></div>
	{:else if uiStore.activeTab === 'bestellberichte'}
		<BestellBerichte />
	{:else if uiStore.activeTab === 'bestandsbuecher'}
		<div class="w-full animate-fade-in"><Bestandsbuecher /></div>
	{:else if uiStore.activeTab === 'signaturen'}
		<div class="w-full animate-fade-in"><SignaturenView /></div>
	{:else if uiStore.activeTab === 'inventory'}
		<div class="w-full animate-fade-in"><UnifiedInventory /></div>
	{:else if uiStore.activeTab === 'students_dir'}
		<div class="w-full animate-fade-in"><StudentDirectory /></div>
	{:else if uiStore.activeTab === 'schulklassen'}
		<div class="w-full animate-fade-in"><Schulklassen /></div>
	{:else if uiStore.activeTab === 'graduates'}
		<div class="w-full animate-fade-in"><Graduates /></div>
	{:else if uiStore.activeTab === 'schuljahr'}
		<div class="w-full animate-fade-in"><LmfPlan /></div>
	{:else if uiStore.activeTab === 'mahnwesen'}
		<div class="w-full animate-fade-in"><Mahnwesen /></div>
	{:else if uiStore.activeTab === 'kollegium_portal'}
		<div class="w-full animate-fade-in"><KollegiumPortal user={authStore.currentUser} /></div>
	{:else if uiStore.activeTab === 'berechtigungen'}
		<div class="w-full animate-fade-in"><Berechtigungen /></div>
	{:else if uiStore.activeTab === 'settings'}
		<div class="w-full animate-fade-in"><SystemSettings /></div>
	{:else if uiStore.activeTab === 'book_detail'}
		<div class="w-full animate-fade-in">
			<BookAkte
				bookId={appState.activeBookId}
				onBack={() => {
					uiStore.activeTab = 'media_catalog';
					appState.activeBookId = null;
				}}
			/>
		</div>
	{:else}
		<!-- Unbekannter Tab: sichtbarer Fallback statt lautloser weißer Seite (+ Sentry). -->
		<RouteFallback tab={uiStore.activeTab} />
	{/if}
</div>
