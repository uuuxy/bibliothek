<script>
	import OpacSearch from './lib/OpacSearch.svelte';
	import Monitor from './lib/Monitor.svelte';

	import { authStore } from './lib/stores/authStore.svelte.js';
	import { uiStore } from './lib/stores/uiStore.svelte.js';
	import { offlineSync } from './lib/stores/offlineSync.svelte.js';
	import { appState } from './inventur/lib/store.svelte.js';
	import { printQueue } from './lib/stores/printQueue.svelte.js';

	import Login from './lib/components/auth/Login.svelte';
	import Sidebar from './lib/components/layout/Sidebar.svelte';
	import Router from './lib/Router.svelte';
	import OfflineIndicator from './lib/components/OfflineIndicator.svelte';
	import ToastContainer from './lib/ToastContainer.svelte';
	import * as Sentry from '@sentry/svelte';

	const _currentPath = window.location.pathname;

	// Boot-Restore: bestehende Session aus dem Cookie wiederherstellen,
	// bevor Login-Screen oder App gerendert werden (sonst: F5 = UI-Logout).
	authStore.restoreSession();

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
		offlineSync.init();
		const id = setInterval(() => uiStore.fetchPendingReservierungen(), 30_000);
		return () => clearInterval(id);
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
	class="min-h-screen bg-slate-50 text-slate-800 font-sans selection:bg-slate-200 selection:text-slate-900"
>
	{#if _currentPath === '/katalog'}
		<OpacSearch />
	{:else if _currentPath === '/monitor'}
		<Monitor />
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
		{:else}
			<div class="h-screen flex w-full overflow-hidden">
				<Sidebar />
				<div
					class="flex-1 flex flex-col min-w-0 bg-slate-50 px-4 md:px-8 py-6 w-full overflow-y-auto"
				>
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
