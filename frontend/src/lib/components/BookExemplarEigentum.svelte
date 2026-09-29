<!-- @component BookExemplarEigentum — die Zeile „Eigentum: …" auf der Exemplarkarte
     (docs/OFFEN.md 4.24, Stufe 3). Dieselbe Form wie „Zustand:" und „Ersatzwert heute:"
     darüber (BookExemplarZustand): body-small, Beschriftung fett.

     Der Wert kommt fertig vom Server — dieselbe Regel wie am Etikett und beim Schadensersatz.
     Geändert wird er nicht hier, sondern über die Markierung („Eigentum ändern"). -->
<script>
	import { eigentumZeile } from './exemplarEigentum.js';

	/** @type {{ ex: import('./exemplarEigentum.js').MitEigentum }} */
	let { ex } = $props();

	const zeile = $derived(eigentumZeile(ex));
</script>

{#if zeile}
	<p class="text-xs text-on-surface-variant">
		<span class="font-semibold">Eigentum:</span>
		<span class="font-semibold text-on-surface">{zeile.wer}</span>
		· {zeile.herkunft}
	</p>
{/if}
