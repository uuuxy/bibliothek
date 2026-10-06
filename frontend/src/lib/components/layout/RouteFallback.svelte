<script>
	import * as Sentry from '@sentry/svelte';
	import { uiStore } from '../../stores/uiStore.svelte.js';
	import Button from '../ui/Button.svelte';

	let { tab } = $props();

	// Der Router rendert diese Komponente nur in seinem {:else}: wenn ein activeTab gesetzt
	// ist, den kein Zweig behandelt. Ohne sie stünde dort eine weiße Seite; so sieht man es,
	// und Sentry erfährt den Namen des Tabs.
	$effect(() => {
		Sentry.captureMessage(`Router: unbehandelter activeTab '${tab}'`, 'error');
	});
</script>

<div class="w-full flex flex-col items-center justify-center py-24 text-center animate-fade-in">
	<div class="text-4xl mb-3">🧭</div>
	<h2 class="text-lg font-bold text-on-surface">Ansicht nicht gefunden</h2>
	<p class="mt-1 text-sm text-on-surface-variant">
		Dieser Bereich ist unbekannt oder nicht verfügbar.
	</p>
	<Button onclick={() => (uiStore.activeTab = 'kiosk')} class="mt-5">Zur Startseite</Button>
</div>
