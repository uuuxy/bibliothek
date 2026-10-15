<!-- @component ListenNichtGeladen — nennt die Listen der Akte, deren Abruf gescheitert ist.

     Über den Karten, nicht in ihnen: Eine leere Karte sähe sonst aus wie „nichts offen".
     Eigenes Bauteil, weil dieselbe Zeile in zwei Reitern der Akte steht. -->
<script>
	import Button from '../ui/Button.svelte';

	/** @type {{ listen?: string[], onErneut: () => void }} */
	let { listen = [], onErneut } = $props();
</script>

{#if listen.length > 0}
	<p class="text-sm font-semibold text-error" role="alert">
		Nicht geladen: {listen.join(', ')}. Diese Karten sind leer, weil der Abruf gescheitert ist —
		nicht, weil nichts vorläge.
		<Button variant="ghost" size="sm" onclick={onErneut}>Erneut laden</Button>
	</p>
{/if}
