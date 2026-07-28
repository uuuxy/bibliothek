<script>
	import * as Sentry from '@sentry/svelte';
	import { uiStore } from '../../stores/uiStore.svelte.js';
	import Button from '../ui/Button.svelte';

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
	<Button onclick={() => (uiStore.activeTab = 'kiosk')} class="mt-5">Zur Startseite</Button>
</div>
