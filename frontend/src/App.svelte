<script>
	import OpacSearch from './lib/OpacSearch.svelte';
	import Ladekreis from './lib/components/ui/Ladekreis.svelte';
	import Monitor from './lib/Monitor.svelte';
	import BestellBestaetigung from './lib/BestellBestaetigung.svelte';

	import { authStore } from './lib/stores/authStore.svelte.js';
	import { uiStore } from './lib/stores/uiStore.svelte.js';
	import { starteHintergrundAbrufe } from './lib/stores/hintergrundAbrufe.svelte.js';
	import { appState } from './inventur/lib/store.svelte.js';
	import { printQueue } from './lib/stores/printQueue.svelte.js';
	import { idleLock } from './lib/stores/idleLock.svelte.js';

	import Login from './lib/components/auth/Login.svelte';
	import Sperrbildschirm from './lib/components/auth/Sperrbildschirm.svelte';
	import Anwendungsrahmen from './lib/components/layout/Anwendungsrahmen.svelte';
	import OfflineIndicator from './lib/components/OfflineIndicator.svelte';
	import ToastContainer from './lib/ToastContainer.svelte';
	import BestaetigungsDialog from './lib/components/ui/BestaetigungsDialog.svelte';
	import ThekenDaten from './lib/components/ThekenDaten.svelte';
	import { initTooltips } from './lib/actions/tooltip.js';
	import * as Sentry from '@sentry/svelte';

	const _currentPath = window.location.pathname;

	// Boot-Restore: bestehende Session aus dem Cookie wiederherstellen,
	// bevor Login-Screen oder App gerendert werden (sonst: F5 = UI-Logout).
	authStore.restoreSession();

	// Ein Zuhörer für alle Sprechblasen (data-tip). Muss vor jedem Bildschirm stehen,
	// weil er delegiert arbeitet und die Elemente erst später entstehen.
	$effect(() => initTooltips());

	$effect(() => {
		const handleError = (event) => Sentry.captureException(event.error || event);
		const handleRejection = (event) => Sentry.captureException(event.reason);

		window.addEventListener('error', handleError);
		window.addEventListener('unhandledrejection', handleRejection);

		return () => {
			window.removeEventListener('error', handleError);
			window.removeEventListener('unhandledrejection', handleRejection);
		};
	});

	// Hinter der Sperre nach Inaktivität beantwortet der Server nichts: Die Abrufe ruhen.
	$effect(() => {
		if (!authStore.isLoggedIn || !authStore.currentUser || idleLock.gesperrt) {
			uiStore.pendingReservierungen = 0;
			return;
		}
		return starteHintergrundAbrufe(authStore.currentUser);
	});

	// Inaktivitäts-Wächter (Theke leeren, Sperrbildschirm): gilt für JEDE angemeldete
	// Sitzung, nicht nur den Kiosk — auch Schülerverwaltung und Mahnwesen zeigen PII.
	// Fristen kommen aus den Einstellungen (/api/einstellungen/sitzung), 0 = aus.
	$effect(() => {
		if (!authStore.isLoggedIn) {
			idleLock.stop();
			return;
		}
		idleLock.start();
		idleLock.ladeFristen();
		return () => idleLock.stop();
	});

	$effect(() => {
		if (printQueue.copies) {
			// 'druck-center' ist der App-Route-Name (Router.svelte); 'labels' ist nur der
			// INTERNE Unter-Tab in DruckCenter. Vorher stand hier 'labels' — den kennt der
			// Router nicht, also rendert <main> nichts → weiße Seite beim Etikettendruck.
			uiStore.activeTab = 'druck-center';
		}
	});

	$effect(() => {
		if (appState.triggerStudentScan && uiStore.activeTab !== 'kiosk') {
			uiStore.activeTab = 'kiosk';
		}
	});

	$effect(() => {
		if (!authStore.isLoggedIn) return;
		const checker = setInterval(() => {
			// Timeout auf 25 Sekunden erhöht (Backend pingt alle 15s, plus Puffer für window.print)
			if (Date.now() - authStore.lastHeartbeatTime > 25000) authStore.heartbeatOk = false;
		}, 1000);
		return () => clearInterval(checker);
	});
</script>

<ThekenDaten />

<div
	class="app min-h-screen bg-surface text-on-surface font-sans selection:bg-slate-200 selection:text-slate-900"
>
	{#if _currentPath === '/katalog'}
		<OpacSearch />
	{:else if _currentPath === '/monitor'}
		<Monitor />
	{:else if _currentPath.startsWith('/bestellung/')}
		<!-- Bestätigungs-Link an den Lieferanten: Der Token steht IM Pfad, deshalb ein
		     Präfix-Vergleich statt Gleichheit. Muss vor dem Login-Zweig stehen — der
		     Lieferant hat kein Konto und darf keinen Anmeldebildschirm sehen. -->
		<BestellBestaetigung />
	{:else}
		<!-- Hier lag bis zum 16.09.2026 ein Vollbild, das 25 s nach dem letzten Herzschlag
		     kam und die Theke anhielt. Der Verbindungsverlust steht jetzt im Offline-Band
		     (unten durchgereicht): eine Lage, eine Zeile. -->
		{#if !authStore.sessionChecked}
			<!-- Boot-Restore läuft — kurzer neutraler Zustand statt Login-Flackern -->
			<div class="fixed inset-0 flex items-center justify-center">
				<Ladekreis size="lg" />
			</div>
		{:else if !authStore.isLoggedIn}
			<Login />
		{:else}
			{#if idleLock.gesperrt}
				<Sperrbildschirm />
			{/if}
			<!-- Gesperrt bleibt die Anwendung stehen, aber ausgeblendet und träge: Was getippt
			     und nicht gespeichert ist, überlebt die Sperre. Die Druckvorschau (Strg+P)
			     zeigt sie nicht, Tab erreicht sie nicht, Screenreader lesen sie nicht (Prüfung
			     22.08.2026, A6). Nach einem Start in die Sperre steht nichts dahinter. -->
			{#if !idleLock.gesperrt || idleLock.anwendungSteht}
				<Anwendungsrahmen verdeckt={idleLock.gesperrt} />
			{/if}
		{/if}
	{/if}
	<OfflineIndicator
		verbindungVerloren={authStore.isLoggedIn && !authStore.heartbeatOk && !idleLock.gesperrt}
	/>
	<!-- Meldungen und eine offene Rückfrage nennen Namen und Titel: Hinter der Sperre sind
	     sie nicht zu sehen. Die Rückfrage bleibt gestellt und steht nach dem Aufschließen
	     wieder da. -->
	{#if !idleLock.gesperrt}
		<ToastContainer />
		<BestaetigungsDialog />
	{/if}
</div>

<style>
	@keyframes fadeIn {
		from {
			opacity: 0;
		}
		to {
			opacity: 1;
		}
	}
	@keyframes slideUp {
		from {
			opacity: 0;
			transform: translateY(8px);
		}
		to {
			opacity: 1;
			transform: none;
		}
	}
	:global(.animate-fade-in) {
		animation: fadeIn 0.4s cubic-bezier(0.16, 1, 0.3, 1) forwards;
	}
	:global(.animate-slide-up) {
		animation: slideUp 0.3s cubic-bezier(0.16, 1, 0.3, 1) forwards;
	}

	@media print {
		:global(body) {
			background: white !important;
			color: black !important;
		}
		.app {
			background: white !important;
		}
		:global(.no-print) {
			display: none !important;
		}
	}
</style>
