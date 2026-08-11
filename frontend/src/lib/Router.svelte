<script>
	import { authStore } from './stores/authStore.svelte.js';
	import { uiStore } from './stores/uiStore.svelte.js';
	import { appState } from '../inventur/lib/store.svelte.js';
	import { erlaubteTabs, tabIstGesperrt } from './menu.js';

	import Betriebsbereitschaft from './Betriebsbereitschaft.svelte';
	import Omnibox from './Omnibox.svelte';
	import BookAkte from './BookAkte.svelte';
	import BestellWorkspace from './BestellWorkspace.svelte';
	import UnifiedInventory from './UnifiedInventory.svelte';
	import MediaCatalog from './MediaCatalog.svelte';
	import SignaturenView from './SignaturenView.svelte';
	import StatsDashboard from './StatsDashboard.svelte';
	import StudentDirectory from './StudentDirectory.svelte';
	import Schulklassen from './Schulklassen.svelte';
	import KollegiumPortal from './KollegiumPortal.svelte';
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
	// neu ergänzter Tab (das Kollegiums-Portal) in beiden Kopien vergessen, seine URL nie
	// gesetzt/wiederhergestellt, und ein Refresh warf die Lehrkraft aus dem Portal.
	//
	// media_catalog liegt auf /medienkatalog und NICHT auf dem Pfad des öffentlichen
	// OPAC. Solange beide denselben beanspruchten, landete ein angemeldeter Benutzer
	// nach F5 im öffentlichen Katalog — und die UI-Gates (control-hoehen,
	// icon-trefferflaechen) vermaßen still den OPAC statt des internen Katalogs.
	// Audit-Befund vom 01.08.2026.
	/** @type {Record<string, string>} */
	const tabToPath = {
		settings: '/einstellungen',
		inventory: '/inventur',
		students_dir: '/schuelerdatei',
		schulklassen: '/schulklassen',
		orders: '/bestellungen',
		media_catalog: '/medienkatalog',
		signaturen: '/signaturen',
		graduates: '/abgaenger',
		stats: '/statistiken',
		mahnwesen: '/mahnwesen',
		kollegium_portal: '/kollegium-portal',
		'system-logs': '/system-logs',
		lmf_actions: '/lmf-aktionen',
		betriebsbereitschaft: '/betriebsbereitschaft',
		'druck-center': '/druck-center',
		kiosk: '/kiosk'
	};

	// Parametrisierte Sonderrouten (Tab braucht einen Zusatzparameter, passt nicht in tabToPath).
	const STATS_DETAIL_KINDS = ['renner', 'ladenhueter'];

	/**
	 * Setzt Tab (+ ggf. Store-Parameter) aus einem Pfad. BEWUSST die einzige Quelle für
	 * Initial-Match UND popstate — vorher lag die book_detail-Logik dupliziert in beiden,
	 * neue Routen wurden leicht in einer Kopie vergessen (siehe Portal-Bug oben).
	 * @param {string} path
	 */
	function applyPathToState(path) {
		if (path.startsWith('/medienkatalog/buch/')) {
			uiStore.activeTab = 'book_detail';
			appState.activeBookId = path.replace('/medienkatalog/buch/', '');
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
			return `/medienkatalog/buch/${appState.activeBookId}`;
		}
		if (uiStore.activeTab === 'stats_detail') {
			return `/statistiken/${uiStore.statsDetailKind}`;
		}
		return tabToPath[uiStore.activeTab];
	}

	// ── Wer darf welchen Bildschirm? EINE Regel, nicht zwei ────────────────────
	//
	// Hier stand bis zum 08.08.2026 eine handgepflegte Liste für die Helfer-Rolle
	// ('kiosk' und 'media_catalog'). Damit gab es ZWEI Definitionen davon, was eine
	// Rolle erreichen darf — die Navigation entschied nach canSeeItem, der Router nach
	// dieser Liste. Sie sind auseinandergelaufen: Als der Menüpunkt „Schulklassen" von
	// manage_users auf view_books wechselte, erschien er dem Helfer im Menü und warf
	// ihn beim Klick wortlos zurück an die Theke. Ein Menüpunkt, der nichts tut, ist
	// schlimmer als keiner — er sieht aus wie ein Defekt der Seite dahinter.
	//
	// Jetzt fragt der Router dieselbe Funktion wie das Menü. Eine Rechteänderung wirkt
	// damit an beiden Stellen gleichzeitig oder an keiner.
	// erlaubteTabs/tabIstGesperrt liegen bei canSeeItem in menu.js — dort steht die Regel,
	// wer was erreichen darf, und der Router liest sie nur ab.

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

	// Escape bringt von überall zurück an die Theke — aber nur, wenn die Taste nicht
	// schon jemandem gehört.
	//
	// Vorher galt sie bedingungslos, und das machte ausgerechnet die Berichte unbenutzbar:
	// Die Ansicht besteht aus Monats-, Jahres- und Datumsfeldern, und Escape ist dort die
	// normale Art, ein aufgeklapptes Auswahlfenster wieder zuzumachen. Der Tastendruck kam
	// beim Fenster an, blubberte weiter — und statt des Auswahlfensters schloss sich die
	// ganze Ansicht. Für den Benutzer sprang das Programm grundlos in die Ausleihe.
	//
	// Zwei Ausnahmen, beide am Ereignis selbst ablesbar und damit auch für Bauteile
	// gültig, die es noch gar nicht gibt:
	/** @param {KeyboardEvent} e */
	function escapeGehoertJemandAnderem(e) {
		// 1. Jemand hat die Taste bereits verarbeitet (Dialog, Menü, Overlay).
		if (e.defaultPrevented) return true;
		// 2. Der Fokus steht in einem Eingabefeld. Dort heisst Escape "Auswahl schließen"
		//    oder "Eingabe verwerfen" — nie "Ansicht verlassen".
		const ziel = /** @type {HTMLElement | null} */ (e.target);
		if (!ziel) return false;
		return ziel.isContentEditable || ['INPUT', 'SELECT', 'TEXTAREA'].includes(ziel.tagName);
	}

	$effect(() => {
		/** @param {KeyboardEvent} e */
		function handleGlobalKeyDown(e) {
			if (e.key !== 'Escape' || uiStore.activeTab === 'kiosk') return;
			if (escapeGehoertJemandAnderem(e)) return;
			uiStore.activeTab = 'kiosk';
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
		<!-- Der Zweig activeTab === 'books' ist entfallen: Ihn hat nie jemand gesetzt (kein
     tabToPath-Eintrag, kein Menüpunkt, keine Zuweisung im Code). Er war seit Langem
     unerreichbar und hielt BookDetails samt uiStore.selectedBook künstlich am Leben —
     die Buchansicht läuft über 'book_detail' und appState.activeBookId.
     Der Routing-Test prüft nur die Gegenrichtung (jedes Ziel wird gerendert), deshalb
     fiel es dort nicht auf. Audit-Befund vom 01.08.2026. -->
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
	{:else if uiStore.activeTab === 'signaturen'}
		<div class="w-full animate-fade-in"><SignaturenView /></div>
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
	{:else if uiStore.activeTab === 'kollegium_portal'}
		<div class="w-full animate-fade-in"><KollegiumPortal user={authStore.currentUser} /></div>
	{:else if uiStore.activeTab === 'lmf_actions'}
		<div class="w-full animate-fade-in"><GlobalLMFExtendWidget /></div>
	{:else if uiStore.activeTab === 'betriebsbereitschaft'}
		<div class="w-full animate-fade-in"><Betriebsbereitschaft /></div>
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
