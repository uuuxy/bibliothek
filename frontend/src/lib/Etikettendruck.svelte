<script>
	import { Printer } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import { etikettenStore } from './stores/etiketten.svelte.js';
	import EtikettEinstellungen from './components/etiketten/EtikettEinstellungen.svelte';
	import EtikettVorschau from './components/etiketten/EtikettVorschau.svelte';
	import Button from './components/ui/Button.svelte';

	onMount(() => {
		etikettenStore.ladeKlassen();
	});
</script>

<div class="w-full space-y-6 no-print text-on-surface animate-fade-in">
	<div class="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start">
		<EtikettEinstellungen />
		<EtikettVorschau />
	</div>

	<!-- Der Druckknopf steht am Ende: Er wird erst scharf, wenn ein Titel gewählt ist, und
	     Material 3 setzt die bestätigende Aktion ans Ende des Flusses. -->
	<div class="flex justify-end border-t border-outline-variant pt-5">
		<Button
			size="lg"
			onclick={etikettenStore.drucke}
			disabled={etikettenStore.bogen.filter((lbl) => !lbl.isBlank).length === 0}
			class="px-5"
		>
			<Printer class="h-4 w-4" aria-hidden="true" />
			A4-Bogen drucken
		</Button>
	</div>
</div>
