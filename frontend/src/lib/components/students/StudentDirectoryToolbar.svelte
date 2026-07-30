<script>
	import { Search, Plus } from '@lucide/svelte';
	import Button from '../ui/Button.svelte';

	/** @type {{ searchQuery?: string, role?: string, trefferzahl?: number, suchend?: boolean, gekuerzt?: boolean, onsearch?: () => void, oncreate?: () => void }} */
	let {
		searchQuery = $bindable(''),
		role = '',
		trefferzahl = 0,
		suchend = false,
		gekuerzt = false,
		onsearch,
		oncreate
	} = $props();
</script>

<!-- Flach und edge-to-edge: kein Kachel-Container, nur dezenter Abstand zu den Tabs -->
<div class="flex items-center gap-4 mt-4">
	<div class="relative flex-1 max-w-2xl">
		<Search class="w-4 h-4 absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-400" />
		<input
			type="text"
			aria-label="Schüler suchen"
			placeholder="Nach Name, Klasse oder Barcode suchen..."
			bind:value={searchQuery}
			oninput={onsearch}
			class="w-full h-9 pl-10 pr-4 bg-white border border-slate-200 rounded-xl text-sm text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all"
		/>
	</div>

	{#if role === 'admin' || role === 'mitarbeiter'}
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
