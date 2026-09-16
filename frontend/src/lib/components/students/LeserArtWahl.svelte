<!-- @component LeserArtWahl — die erste Frage beim Anlegen: Wer ist das?

     Zuerst die Art, dann die Felder (Peter, 16.09.2026). Nicht aus Ordnungsliebe: An der
     Art hängt, welche Angaben überhaupt Pflicht sind. Ein Schüler braucht Klasse und
     Geburtsdatum (der LUSD-Import erkennt ihn nur daran wieder), ein Kollege hat beides
     nicht. Stünde die Frage am Ende, hätte man vorher Felder ausgefüllt, die verschwinden.

     Die Arten sind dieselben drei wie in der Datenbank (chk_leser_art, Migration 123). -->
<script>
	import Radio from '../ui/Radio.svelte';
	import { leserArtText } from '../../leserArt.js';

	/** @type {{ art: string, disabled?: boolean }} */
	let { art = $bindable(), disabled = false } = $props();

	const arten = ['schueler', 'lehrkraft', 'liv'];
</script>

<fieldset class="border-0 p-0 m-0">
	<legend class="text-xs font-medium text-on-surface-variant mb-2">Art des Lesers</legend>
	<div class="flex flex-wrap items-center gap-6">
		{#each arten as wert (wert)}
			<Radio bind:group={art} value={wert} label={leserArtText(wert)} {disabled} />
		{/each}
	</div>
</fieldset>
