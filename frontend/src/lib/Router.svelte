<script>
	import { authStore } from './stores/authStore.svelte.js';
	import { uiStore } from './stores/uiStore.svelte.js';
	import { appState } from '../inventur/lib/store.svelte.js';

	import Omnibox from './Omnibox.svelte';
	import BookDetails from './BookDetails.svelte';
	import BookAkte from './BookAkte.svelte';
	import BestellWorkspace from './BestellWorkspace.svelte';
	import UnifiedInventory from './UnifiedInventory.svelte';
	import MediaCatalog from './MediaCatalog.svelte';
	import StatsDashboard from './StatsDashboard.svelte';
	import StudentDirectory from './StudentDirectory.svelte';
	import Schulklassen from './Schulklassen.svelte';
	import LehrerPortal from './LehrerPortal.svelte';
	import Mahnwesen from './Mahnwesen.svelte';
	import StatistikDetailPage from './components/stats/StatistikDetailPage.svelte';
	import SystemSettings from './SystemSettings.svelte';
	import GlobalLMFExtendWidget from './GlobalLMFExtendWidget.svelte';
	import DruckCenter from './DruckCenter.svelte';
	import SystemLogs from './SystemLogs.svelte';
	import Graduates from './Graduates.svelte';
	import RouteFallback from './components/layout/RouteFallback.svelte';

	// Zentrale Tab→Pfad-Zuordnung. Bewusst nur EINMAL definiert: Vorher lag dieselbe
	// Map dupliziert im Routing-$effect und im popstate-Handler — dadurch wurde ein
	// neu ergänzter Tab (lehrer_portal) in beiden Kopien vergessen, seine URL nie
	// gesetzt/wiederhergestellt, und ein Refresh warf den Lehrer aus dem Portal.
	/** @type {Record<string, string>} */
	const tabToPath = {
		settings: '/einstellungen',
		inventory: '/inventur',
		students_dir: '/schuelerdatei',
		schulklassen: '/schulklassen',
		orders: '/bestellungen',
		media_catalog: '/katalog',
		graduates: '/abgaenger',
		stats: '/statistiken',
		mahnwesen: '/mahnwesen',
		lehrer_portal: '/lehrer-portal',
		'system-logs': '/system-logs',
		lmf_actions: '/lmf-aktionen',
		'druck-center': '/druck-center',
		kiosk: '/kiosk'
	};

	// Parametrisierte Sonderrouten (Tab braucht einen Zusatzparameter, passt nicht in tabToPath).
	const STATS_DETAIL_KINDS = ['renner', 'ladenhueter'];

	/**
	 * Setzt Tab (+ ggf. Store-Parameter) aus einem Pfad. BEWUSST die einzige Quelle für
	 * Initial-Match UND popstate — vorher lag die book_detail-Logik dupliziert in beiden,
	 * neue Routen wurden leicht in einer Kopie vergessen (siehe lehrer_portal-Bug).
	 * @param {string} path
	 */
	function applyPathToState(path) {
		if (path.startsWith('/katalog/buch/')) {
			uiStore.activeTab = 'book_detail';
			appState.activeBookId = path.replace('/katalog/buch/', '');
			return;
		}
		const statsKind = path.startsWith('/statistiken/') && path.replace('/statistiken/', '');
		if (statsKind && STATS_DETAIL_KINDS.includes(statsKind)) {
			uiStore.activeTab = 'stats_detail';
			uiStore.statsDetailKind = /** @type {'renner'|'ladenhueter'} */ (statsKind);
			return;
		}
		const matchedTab = Object.keys(tabToPath).find((key) => tabToPath[key] === path);
		if (matchedTab) uiStore.activeTab = matchedTab;
	}

	/** Zielpfad für den aktuellen Tab — inkl. der parametrisierten Sonderrouten. */
	function currentTargetPath() {
		if (uiStore.activeTab === 'book_detail' && appState.activeBookId) {
			return `/katalog/buch/${appState.activeBookId}`;
		}
		if (uiStore.activeTab === 'stats_detail') {
			return `/statistiken/${uiStore.statsDetailKind}`;
		}
		return tabToPath[uiStore.activeTab];
	}

	function handleSelectBook(book) {
		// Ein in der Omnibox angeklicktes Buch soll die Detail-/Akte-Ansicht dieses Buchs
		// öffnen (book_detail → BookAkte via bookId, inkl. Deep-Link /katalog/buch/{id}) —
		// NICHT den allgemeinen Medienkatalog.
		if (!book?.id) return;
		appState.activeBookId = book.id;
		uiStore.activeTab = 'book_detail';
	}

	// Routing effects
	$effect(() => {
		if (authStore.isLoggedIn && authStore.currentUser) {
			const role = authStore.currentUser.rolle ? authStore.currentUser.rolle.toLowerCase() : '';
			const path = window.location.pathname;
			const isHelfer = role === 'helfer';

			if (isHelfer) {
				if (uiStore.activeTab !== 'kiosk' && uiStore.activeTab !== 'media_catalog') {
					uiStore.activeTab = 'kiosk';
				}
				if (path !== '/' && path !== '/kiosk' && path !== '/katalog') {
					window.history.replaceState(null, '', '/kiosk');
				}
			} else {
				if (!uiStore.isInitialRouteMatched && path !== '/') {
					applyPathToState(path);
				}
				uiStore.isInitialRouteMatched = true;

				const targetPath = currentTargetPath();
				if (targetPath && path !== targetPath && uiStore.isInitialRouteMatched) {
					window.history.pushState(null, '', targetPath);
				}
			}
		}
	});

	$effect(() => {
		/** @param {KeyboardEvent} e */
		function handleGlobalKeyDown(e) {
			if (e.key === 'Escape' && uiStore.activeTab !== 'kiosk') {
				uiStore.activeTab = 'kiosk';
			}
		}
		function handlePopState() {
			applyPathToState(window.location.pathname);
		}
		window.addEventListener('keydown', handleGlobalKeyDown);
		window.addEventListener('popstate', handlePopState);
		return () => {
			window.removeEventListener('keydown', handleGlobalKeyDown);
			window.removeEventListener('popstate', handlePopState);
		};
	});
</script>

<main class="flex-1 overflow-y-auto flex flex-col w-full">
	{#if uiStore.activeTab === 'kiosk'}
		<div class="flex-1 flex flex-col w-full animate-fade-in">
			<Omnibox onSelectBook={handleSelectBook} role={authStore.currentUser?.rolle} />
		</div>
	{:else if uiStore.activeTab === 'books'}
		<div class="w-full animate-fade-in">
			<BookDetails title={uiStore.selectedBook || undefined} />
		</div>
	{:else if uiStore.activeTab === 'orders'}
		<div class="w-full animate-fade-in"><BestellWorkspace /></div>
	{:else if uiStore.activeTab === 'stats'}
		<div class="w-full animate-fade-in"><StatsDashboard /></div>
	{:else if uiStore.activeTab === 'stats_detail'}
		<div class="w-full animate-fade-in"><StatistikDetailPage /></div>
	{:else if uiStore.activeTab === 'system-logs'}
		<div class="w-full animate-fade-in h-full"><SystemLogs /></div>
	{:else if uiStore.activeTab === 'druck-center'}
		<div class="w-full animate-fade-in h-full"><DruckCenter /></div>
	{:else if uiStore.activeTab === 'media_catalog'}
		<div class="w-full animate-fade-in"><MediaCatalog /></div>
	{:else if uiStore.activeTab === 'inventory'}
		<div class="w-full animate-fade-in"><UnifiedInventory /></div>
	{:else if uiStore.activeTab === 'students_dir'}
		<div class="w-full animate-fade-in">
			<StudentDirectory role={authStore.currentUser?.rolle} />
		</div>
	{:else if uiStore.activeTab === 'graduates'}
		<div class="w-full animate-fade-in"><Graduates /></div>
	{:else if uiStore.activeTab === 'schulklassen'}
		<div class="w-full animate-fade-in"><Schulklassen /></div>
	{:else if uiStore.activeTab === 'mahnwesen'}
		<div class="w-full animate-fade-in"><Mahnwesen /></div>
	{:else if uiStore.activeTab === 'lehrer_portal'}
		<div class="w-full animate-fade-in"><LehrerPortal user={authStore.currentUser} /></div>
	{:else if uiStore.activeTab === 'lmf_actions'}
		<div class="w-full animate-fade-in p-6 max-w-6xl mx-auto">
			<GlobalLMFExtendWidget />
		</div>
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
</main>
