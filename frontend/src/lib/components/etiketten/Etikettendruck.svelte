<script>
	import Icon from '../ui/Icon.svelte';
	import { onMount } from 'svelte';
	import { etikettenStore } from '../../stores/etiketten.svelte.js';
	import EtikettEinstellungen from './EtikettEinstellungen.svelte';
	import EtikettVorschau from './EtikettVorschau.svelte';
	import Button from '../ui/Button.svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';

	onMount(() => {
		etikettenStore.ladeKlassen();
	});

	// Ein zweiter Klick vor der Antwort schickte den Bogen noch einmal und öffnete ihn in einem
	// zweiten Tab: dieselben Etiketten zweimal auf dem Papier.
	let druckt = $state(false);
	async function drucke() {
		if (druckt) return;
		druckt = true;
		try {
			await etikettenStore.drucke();
		} finally {
			druckt = false;
		}
	}
</script>

<div class="w-full space-y-6 no-print text-on-surface animate-fade-in">
	<!-- Zwei Bereiche ab expanded (md): M3, Breakpoints: „Expanded and large breakpoints: Two
	     panes are recommended", dazwischen „The spacer between panes is 24dp". -->
	<div class="grid grid-cols-1 md:grid-cols-12 gap-6 items-start">
		<EtikettEinstellungen />
		<EtikettVorschau />
	</div>

	<!-- Der Druckknopf steht am Ende: Er wird erst scharf, wenn ein Titel gewählt ist, und
	     Material 3 setzt die bestätigende Aktion ans Ende des Flusses. -->
	<div class="flex justify-end border-t border-outline-variant pt-5">
		<Button
			size="lg"
			onclick={drucke}
			disabled={druckt || etikettenStore.bogen.filter((lbl) => !lbl.isBlank).length === 0}
			class="px-5"
		>
			{#if druckt}
				<Ladekreis size="sm" farbe="aktuell" />
			{:else}
				<Icon name="print" />
			{/if}
			A4-Bogen drucken
		</Button>
	</div>
</div>
