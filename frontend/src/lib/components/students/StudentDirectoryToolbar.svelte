<script>
	import { Plus } from '@lucide/svelte';
	import Suchpille from '../ui/Suchpille.svelte';
	import Button from '../ui/Button.svelte';

	/** @type {{ searchQuery?: string, darfAnlegen?: boolean, trefferzahl?: number, suchend?: boolean, gekuerzt?: boolean, onsearch?: () => void, oncreate?: () => void }} */
	let {
		searchQuery = $bindable(''),
		darfAnlegen = false,
		trefferzahl = 0,
		suchend = false,
		gekuerzt = false,
		onsearch,
		oncreate
	} = $props();
</script>

<!-- Flach und edge-to-edge: kein Kachel-Container, nur dezenter Abstand zu den Tabs. -->
<div class="mt-4 flex flex-col gap-3">
	<Suchpille
		id="schuelerdatei-suchfeld"
		bind:wert={searchQuery}
		oninput={onsearch}
		platzhalter="Name, Klasse oder Barcode eingeben …"
		etikett="Schüler suchen"
	/>

	<div class="flex items-center gap-4">
		{#if darfAnlegen}
			<Button variant="primary" onclick={oncreate} aria-label="Neuen Schüler anlegen">
				<Plus class="w-4 h-4" />
				Neuer Schüler
			</Button>
		{/if}

		<!-- Sagt, was tatsächlich zu sehen ist. Vorher stand hier "500 / 500", während die
		     Schule 875 Schüler hatte — die Zahl bestätigte dem Benutzer eine Vollständigkeit,
		     die es nicht gab, und machte das Fehlen einzelner Namen unerklärlich. -->
		<div class="ml-auto shrink-0 text-xs font-semibold text-slate-500">
			{#if suchend}
				Treffer: {trefferzahl}
			{:else if gekuerzt}
				Erste {trefferzahl} — zum Finden bitte suchen
			{:else}
				Einträge: {trefferzahl}
			{/if}
		</div>
	</div>
</div>
