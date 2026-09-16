<!-- @component LeserArtWahl — die erste Frage beim Anlegen: Wer ist das?

     Zuerst die Art, dann die Felder (Peter, 16.09.2026). Nicht aus Ordnungsliebe: An der
     Art hängt, welche Angaben überhaupt Pflicht sind. Ein Schüler braucht Klasse und
     Geburtsdatum (der LUSD-Import erkennt ihn nur daran wieder), ein Kollege hat beides
     nicht. Stünde die Frage am Ende, hätte man vorher Felder ausgefüllt, die verschwinden.

     Die Arten sind dieselben drei wie in der Datenbank (chk_leser_art, Migration 123). -->
<script>
	import Radio from '../ui/Radio.svelte';
	import { leserArtText } from '../../leserArt.js';

	/**
	 * Alle drei Arten stehen IMMER da — die Maske hat für jeden dieselbe Form (Peter,
	 * 16.09.2026: „bitte nicht verkomplizieren"). Was nicht gewählt werden darf, steht in
	 * `gesperrt` und ist abgeschaltet statt versteckt.
	 *
	 * Beim ANLEGEN ist nichts gesperrt. Beim ÄNDERN einer bestehenden Akte ist es die
	 * Grenze zum Schüler, und zwar in beide Richtungen: Ein Schüler kommt aus der LUSD und
	 * bleibt Schüler („ein Schüler kann nie ein Lehrer werden!"), ein Kollege wird keiner.
	 * Dieselbe Grenze hält der Server (pruefeUndSetzeArt) — hier steht sie, damit niemand
	 * erst auf Speichern drücken muss, um es zu erfahren.
	 * @type {{ art: string, disabled?: boolean, arten?: string[], gesperrt?: string[] }}
	 */
	let {
		art = $bindable(),
		disabled = false,
		arten = ['schueler', 'lehrkraft', 'liv'],
		gesperrt = []
	} = $props();
</script>

<fieldset class="border-0 p-0 m-0">
	<legend class="text-xs font-medium text-on-surface-variant mb-2">Art des Lesers</legend>
	<div class="flex flex-wrap items-center gap-6">
		{#each arten as wert (wert)}
			<Radio
				bind:group={art}
				value={wert}
				label={leserArtText(wert)}
				disabled={disabled || gesperrt.includes(wert)}
			/>
		{/each}
	</div>
</fieldset>
