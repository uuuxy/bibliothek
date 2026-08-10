<script>
	/**
	 * Der Zustand unter einer Suchpille: „noch nichts gesucht" oder „nichts gefunden".
	 *
	 * Beide Seiten hatten dafür eine eigene Fassung. Im OPAC eine Überschrift mit
	 * Unterzeile, im Kollegiums-Portal eine einzelne kleine graue Zeile — dieselbe
	 * Aussage in zwei Tonlagen, direkt unter zwei Suchfeldern, die sich schon
	 * unterschieden. Material 3 kennt dafür eine Form: Symbol, Überschrift, erklärender
	 * Satz darunter.
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

<div class="text-center py-20 text-slate-400 select-none">
	<!-- mx-auto ist nötig: Tailwinds Preflight setzt svg auf display:block, damit greift
	     das text-center des Containers nicht — das Symbol klebte sonst links. -->
	<Symbol class="h-10 w-10 mx-auto mb-3" aria-hidden="true" />
	<p class="text-xl font-semibold">{titel}</p>
	{#if hinweis}
		<p class="text-sm mt-1">{hinweis}</p>
	{/if}
</div>
