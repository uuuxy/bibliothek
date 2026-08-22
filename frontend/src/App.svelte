<script>
	import OpacSearch from './lib/OpacSearch.svelte';
	import Monitor from './lib/Monitor.svelte';
	import BestellBestaetigung from './lib/BestellBestaetigung.svelte';

	import { authStore } from './lib/stores/authStore.svelte.js';
	import { uiStore } from './lib/stores/uiStore.svelte.js';
	import { offlineSync } from './lib/stores/offlineSync.svelte.js';
	import { appState } from './inventur/lib/store.svelte.js';
	import { printQueue } from './lib/stores/printQueue.svelte.js';
	import { idleLock } from './lib/stores/idleLock.svelte.js';

	import Login from './lib/components/auth/Login.svelte';
	import Sperrbildschirm from './lib/components/auth/Sperrbildschirm.svelte';
	import Sidebar from './lib/components/layout/Sidebar.svelte';
	import BackupAlert from './lib/components/system/BackupAlert.svelte';
	import Router from './lib/Router.svelte';
	import OfflineIndicator from './lib/components/OfflineIndicator.svelte';
	import ToastContainer from './lib/ToastContainer.svelte';
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

	$effect(() => {
		if (!authStore.isLoggedIn || !authStore.currentUser) {
			uiStore.pendingReservierungen = 0;
			return;
		}
		if (authStore.currentUser.rolle !== 'admin' && authStore.currentUser.rolle !== 'mitarbeiter')
			return;
		uiStore.fetchPendingReservierungen();
		// Speist das Badge an „Druck-Center". Seltener als die Reservierungen: Offene
		// Etiketten aendern sich nur beim Einbuchen und beim Drucken, nicht im Minutentakt.
		uiStore.fetchOffeneEtiketten();
		offlineSync.init();
		const id = setInterval(() => uiStore.fetchPendingReservierungen(), 30_000);
		const idEtiketten = setInterval(() => uiStore.fetchOffeneEtiketten(), 120_000);
		return () => {
			clearInterval(id);
			clearInterval(idEtiketten);
		};
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

<main
	class="min-h-screen bg-surface text-on-surface font-sans selection:bg-slate-200 selection:text-slate-900"
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
		{#if authStore.isLoggedIn && !authStore.heartbeatOk}
			<div
				class="fixed inset-0 bg-white/45 backdrop-blur-lg z-50 flex flex-col items-center justify-center space-y-4"
			>
				<div
					class="w-12 h-12 border-4 border-t-slate-800 border-slate-200/50 rounded-full animate-spin"
				></div>
				<h2 class="text-lg font-bold text-slate-800 tracking-wide">VERBINDUNG VERLOREN</h2>
				<p class="text-slate-500 text-xs font-medium">Reconnecting...</p>
			</div>
		{/if}

		{#if !authStore.sessionChecked}
			<!-- Boot-Restore läuft — kurzer neutraler Zustand statt Login-Flackern -->
			<div class="fixed inset-0 flex items-center justify-center">
				<div
					class="w-10 h-10 border-4 border-t-slate-800 border-slate-200/60 rounded-full animate-spin"
				></div>
			</div>
		{:else if !authStore.isLoggedIn}
			<Login />
		{:else if idleLock.gesperrt}
			<!-- Gesperrt: die App wird NICHT gerendert, nicht nur verdeckt. Sonst zeigte die
			     Druckvorschau (Strg+P) die Seite dahinter, Tab verließ die Sperre, und
			     Screenreader lasen Schülerdaten vor (Prüfung 22.08.2026, A6). -->
			<Sperrbildschirm />
		{:else}
			<div class="h-screen flex w-full overflow-hidden">
				<Sidebar />
				<!-- Arbeitsflaeche WEISS, nicht getoent. Am 07.08. hatte ich sie auf `surface`
				     gestellt, damit die weissen Karten sich abheben — und genau das war der
				     Fehler: Die Karten gab es laengst, sie lagen nur unsichtbar auf weissem
				     Grund. Die Toenung hat sie erst hervorgeholt und die Anwendung wirkte
				     "in Kacheln gezwaengt". Drei Commits (f2320e1, e81ce75, 95d5d33) hatten
				     das Floating-Card-Muster vorher ausdruecklich abgeschafft: edge-to-edge,
				     volle Breite, getrennt nur durch divide-y. -->
				<div
					class="bg-surface-container-lowest flex w-full min-w-0 flex-1 flex-col overflow-y-auto px-4 py-6 md:px-8"
				>
					<!-- Systemzustand, der eine Handlung braucht, steht über dem Inhalt —
					     nicht in der Navigation. Nur Admins können das Problem beheben. -->
					{#if authStore.currentUser?.rolle === 'admin'}
						<BackupAlert />
					{/if}
					<Router />
				</div>
			</div>
		{/if}
	{/if}
	<OfflineIndicator />
	<ToastContainer />
</main>

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
		main {
			background: white !important;
		}
		:global(.no-print) {
			display: none !important;
		}
	}
</style>
