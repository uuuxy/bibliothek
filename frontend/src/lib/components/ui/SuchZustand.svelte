<script>
	/**
	 * Der Zustand unter einer Suchpille: „noch nichts gesucht" oder „nichts gefunden".
	 *
	 * Eine Form für alle Suchseiten: Symbol, Überschrift, erklärender Satz darunter.
	 *
	 * @type {{
	 *   symbol: any,
	 *   titel: string,
	 *   hinweis?: string
	 * }}
	 */
	let { symbol, titel, hinweis = '' } = $props();

	const Symbol = $derived(symbol);
</script>

<div class="text-center py-20 text-on-surface-variant select-none">
	<!-- mx-auto ist nötig: Tailwinds Preflight setzt svg auf display:block, damit greift
	     das text-center des Containers nicht — das Symbol klebte sonst links. -->
	<Symbol class="h-10 w-10 mx-auto mb-3" aria-hidden="true" />
	<p class="text-xl font-semibold">{titel}</p>
	{#if hinweis}
		<p class="text-sm mt-1">{hinweis}</p>
	{/if}
</div>
