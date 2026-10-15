<!-- @component LeserArtWahl — die erste Frage beim Anlegen: Wer ist das?

     Zuerst die Art, dann die Felder (16.09.2026). Nicht aus Ordnungsliebe: An der
     Art hängt, welche Angaben überhaupt Pflicht sind. Ein Schüler braucht Klasse und
     Geburtsdatum (der LUSD-Import erkennt ihn nur daran wieder), ein Kollege hat beides
     nicht. Stünde die Frage am Ende, hätte man vorher Felder ausgefüllt, die verschwinden.

     Die Arten sind dieselben sieben wie in der Datenbank (chk_leser_art, Migration 153).
     Bis zum 30.09.2026 waren es drei, als Auswahlknöpfe nebeneinander. Mit den Sonderkonten
     (Praktikum, Sekretariat, U-plus, Fachbereich) ist es eine Auswahlliste — Material 3,
     Radio button, Guidelines: „Use radio buttons when there are five or fewer options."
     und „Consider using a drop-down menu instead of radio buttons when space is
     constrained". Das Bauteil ist ui/Select, wie die Klasse eine Zeile tiefer. -->
<script>
	import Select from '../ui/Select.svelte';
	import { LESER_ARTEN, leserArtText } from '../../leserArt.js';

	/**
	 * Alle Arten stehen IMMER in der Liste — die Maske hat für jeden dieselbe Form (Absprache
	 * vom 16.09.2026: „bitte nicht verkomplizieren"). Was nicht gewählt werden darf, steht in
	 * `gesperrt` und ist abgeschaltet statt versteckt.
	 *
	 * Beim ANLEGEN ist nichts gesperrt. Beim ÄNDERN einer bestehenden Akte ist es die
	 * Grenze zum Schüler, und zwar in beide Richtungen: Ein Schüler kommt aus der LUSD und
	 * bleibt Schüler („ein Schüler kann nie ein Lehrer werden!"), ein Kollege wird keiner.
	 * Dieselbe Grenze hält der Server (pruefeUndSetzeArt) — hier steht sie, damit niemand
	 * erst auf Speichern drücken muss, um es zu erfahren.
	 * `id` wie bei ui/Feld: Die Akte gibt eine feste (ihre Felder heißen alle fest), der
	 * Anlegen-Dialog nicht — dann vergibt Svelte eine.
	 * @type {{ art: string, disabled?: boolean, gesperrt?: string[], hint?: string, class?: string, id?: string }}
	 */
	let {
		art = $bindable(),
		disabled = false,
		gesperrt = [],
		hint = '',
		class: className = '',
		id = undefined
	} = $props();

	// Eine eigene Kennung je Einbau: Akte und Anlegen-Dialog dürfen sich keine teilen.
	const eigen = $props.id();
	const kennung = $derived(id ?? `${eigen}-art`);

	const optionen = $derived(
		LESER_ARTEN.map((wert) => ({
			value: wert,
			label: leserArtText(wert),
			disabled: gesperrt.includes(wert)
		}))
	);
</script>

<!-- Beschriftung, Feld und Hinweis wie ui/Feld: drei Zeilen im Subgrid, damit die Nachbarn
     in einem Raster fluchten. -->
<div class="row-span-3 grid grid-rows-subgrid gap-y-1.5 {className}">
	<label for={kennung} class="text-sm font-medium text-on-surface-variant">Art des Lesers</label>
	<Select id={kennung} bind:value={art} options={optionen} {disabled} />
	{#if hint}
		<span class="text-xs text-on-surface-variant">{hint}</span>
	{/if}
</div>
