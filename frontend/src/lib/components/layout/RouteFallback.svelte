<script>
	import * as Sentry from '@sentry/svelte';
	import { uiStore } from '../../stores/uiStore.svelte.js';

	let { tab } = $props();

	// Diese Komponente wird NUR im {:else} des Routers gerendert — also genau dann,
	// wenn ein activeTab gesetzt ist, den keine Branch behandelt. Früher rendert der
	// Router in diesem Fall lautlos nichts → weiße Seite (siehe White-Screen beim
	// Etikettendruck, der wochenlang unbemerkt blieb). Jetzt: sichtbar für den Nutzer
	// UND an Sentry gemeldet, damit ein künftiger Tab-Namens-Desync sofort auffällt.
	$effect(() => {
		Sentry.captureMessage(`Router: unbehandelter activeTab '${tab}'`, 'error');
	});
</script>

<div class="w-full flex flex-col items-center justify-center py-24 text-center animate-fade-in">
	<div class="text-4xl mb-3">🧭</div>
	<h2 class="text-lg font-bold text-slate-800">Ansicht nicht gefunden</h2>
	<p class="mt-1 text-sm text-slate-500">Dieser Bereich ist unbekannt oder nicht verfügbar.</p>
	<button
		onclick={() => (uiStore.activeTab = 'kiosk')}
		class="mt-5 px-4 py-2 text-sm font-bold text-white bg-blue-600 hover:bg-blue-700 rounded-lg cursor-pointer transition-colors"
	>
		Zur Startseite
	</button>
</div>
