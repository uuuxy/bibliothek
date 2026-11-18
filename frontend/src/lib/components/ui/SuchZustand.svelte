<script>
	import Icon from './Icon.svelte';

	/**
	 * Der leere Zustand einer Liste: noch nichts gesucht, nichts gefunden oder nichts da.
	 *
	 * Eine Form für alle Listen und Suchseiten: Symbol, Überschrift, erklärender Satz darunter,
	 * wo es einen gibt, der Knopf, der die Liste füllt.
	 *
	 * @type {{
	 *   symbol: import('./iconNamen.js').IconName,
	 *   titel: string,
	 *   hinweis?: string,
	 *   aktion?: import('svelte').Snippet
	 * }}
	 */
	let { symbol, titel, hinweis = '', aktion } = $props();
</script>

<div class="text-center py-20 text-on-surface-variant select-none">
	<!-- mx-auto ist nötig: Tailwinds Preflight setzt svg auf display:block, damit greift
	     das text-center des Containers nicht — das Symbol klebte sonst links. -->
	<Icon name={symbol} groesse={40} class="mx-auto mb-3 block" />
	<p class="text-xl">{titel}</p>
	{#if hinweis}
		<p class="text-sm mt-1 max-w-md mx-auto">{hinweis}</p>
	{/if}
	{#if aktion}
		<div class="mt-6 flex justify-center">{@render aktion()}</div>
	{/if}
</div>
