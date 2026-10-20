<script>
	import OpacSuche from './lib/OpacSuche.svelte';
	import Ladekreis from './lib/components/ui/Ladekreis.svelte';
	import Monitor from './lib/Monitor.svelte';
	import BestellBestaetigung from './lib/BestellBestaetigung.svelte';

	import { authStore } from './lib/stores/authStore.svelte.js';
	import { uiStore } from './lib/stores/uiStore.svelte.js';
	import { starteHintergrundAbrufe } from './lib/stores/hintergrundAbrufe.svelte.js';
	import { appState } from './inventur/lib/store.svelte.js';
	import { druckWarteschlange } from './lib/stores/druckWarteschlange.svelte.js';
	import { leerlaufSperre } from './lib/stores/leerlaufSperre.svelte.js';

	import Anmeldung from './lib/components/auth/Anmeldung.svelte';
	import Sperrbildschirm from './lib/components/auth/Sperrbildschirm.svelte';
	import Anwendungsrahmen from './lib/components/layout/Anwendungsrahmen.svelte';
	import OfflineAnzeige from './lib/components/OfflineAnzeige.svelte';
	import SnackbarAnzeige from './lib/SnackbarAnzeige.svelte';
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
		const beiFehler = (event) => Sentry.captureException(event.error || event);
		const beiAbweisung = (event) => Sentry.captureException(event.reason);

		window.addEventListener('error', beiFehler);
		window.addEventListener('unhandledrejection', beiAbweisung);

		return () => {
			window.removeEventListener('error', beiFehler);
			window.removeEventListener('unhandledrejection', beiAbweisung);
		};
	});

	// Hinter der Sperre nach Inaktivität beantwortet der Server nichts: Die Abrufe ruhen.
	$effect(() => {
		if (!authStore.isLoggedIn || !authStore.currentUser || leerlaufSperre.gesperrt) {
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
			leerlaufSperre.stop();
			return;
		}
		leerlaufSperre.start();
		leerlaufSperre.ladeFristen();
		return () => leerlaufSperre.stop();
	});

	$effect(() => {
		if (druckWarteschlange.copies) {
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
	class="app min-h-screen bg-surface text-on-surface font-sans selection:bg-surface-container-highest selection:text-on-surface"
>
	{#if _currentPath === '/katalog'}
		<OpacSuche />
	{:else if _currentPath === '/monitor'}
		<Monitor />
	{:else if _currentPath.startsWith('/bestellung/')}
		<!-- Bestätigungs-Link an den Lieferanten: Der Token steht IM Pfad, deshalb ein
		     Präfix-Vergleich statt Gleichheit. Muss vor dem Login-Zweig stehen — der
		     Lieferant hat kein Konto und darf keinen Anmeldebildschirm sehen. -->
		<BestellBestaetigung />
	{:else}
		<!-- Ein Verbindungsverlust hält die Theke nicht mit einem Vollbild an, er steht im
		     Offline-Band (unten durchgereicht): eine Lage, eine Zeile. -->
		{#if !authStore.sessionChecked}
			<!-- Boot-Restore läuft — kurzer neutraler Zustand statt Login-Flackern -->
			<div class="fixed inset-0 flex items-center justify-center">
				<Ladekreis size="lg" />
			</div>
		{:else if !authStore.isLoggedIn}
			<Anmeldung />
		{:else}
			{#if leerlaufSperre.gesperrt}
				<Sperrbildschirm />
			{/if}
			<!-- Gesperrt bleibt die Anwendung stehen, aber ausgeblendet und träge: Was getippt
			     und nicht gespeichert ist, überlebt die Sperre. Die Druckvorschau (Strg+P)
			     zeigt sie nicht, Tab erreicht sie nicht, Screenreader lesen sie nicht. Nach einem
			     Start in die Sperre steht nichts dahinter. -->
			{#if !leerlaufSperre.gesperrt || leerlaufSperre.anwendungSteht}
				<Anwendungsrahmen verdeckt={leerlaufSperre.gesperrt} />
			{/if}
		{/if}
	{/if}
	<OfflineAnzeige
		verbindungVerloren={authStore.isLoggedIn && !authStore.heartbeatOk && !leerlaufSperre.gesperrt}
	/>
	<!-- Meldungen und eine offene Rückfrage nennen Namen und Titel: Hinter der Sperre sind
	     sie nicht zu sehen. Die Rückfrage bleibt gestellt und steht nach dem Aufschließen
	     wieder da. -->
	{#if !leerlaufSperre.gesperrt}
		<SnackbarAnzeige />
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
