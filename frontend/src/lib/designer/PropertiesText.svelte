<!-- @component PropertiesText — Textformatierung des ausgewählten Ausweis-Elements.

     Getrennt von PropertiesPanel, weil dort ALLES für ALLE Elementarten steht:
     Lage, Größe, Ebene, Sichtbarkeit. Schrift betrifft nur Textelemente — und bei
     dynamischen Feldern (Name, Klasse …) kommt der Inhalt aus den Schülerdaten,
     deshalb fehlt dort das Inhaltsfeld. -->
<script>
	import Select from '../components/ui/Select.svelte';
	import Feld from '../components/ui/Feld.svelte';
	import ZahlenFeld from './ZahlenFeld.svelte';
	import Kaestchen from '../components/ui/Kaestchen.svelte';

	/**
	 * @type {{
	 *   el: any,
	 *   istDynamisch: boolean,
	 *   schriften: Array<{ value: string, label: string }>
	 * }}
	 */
	let { el, istDynamisch, schriften } = $props();

	const AUSRICHTUNG = [
		{ wert: 'left', zeichen: '⬅', name: 'Linksbündig' },
		{ wert: 'center', zeichen: '↔', name: 'Zentriert' },
		{ wert: 'right', zeichen: '➡', name: 'Rechtsbündig' }
	];
</script>

<div class="space-y-3 pt-2 border-t border-outline-variant">
	<span class="text-xs font-medium text-on-surface-variant block">Textformatierung</span>

	{#if !istDynamisch}
		<Feld label="Inhalt" bind:value={el.content} />
	{/if}

	<div class="space-y-1">
		<span class="text-xs text-on-surface-variant font-medium block">Schriftart</span>
		<Select bind:value={el.style.fontFamily} options={schriften} aria-label="Schriftart" />
	</div>

	<div class="grid grid-cols-2 gap-2">
		<ZahlenFeld
			label="Größe (pt)"
			value={el.style.fontSize}
			min={4}
			max={20}
			step={0.5}
			onInput={(v) => (el.style.fontSize = v)}
		/>
		<div class="space-y-1">
			<span class="text-xs text-on-surface-variant font-medium block">Farbe</span>
			<input
				type="color"
				bind:value={el.style.color}
				class="w-full h-8 rounded-xl border border-outline-variant cursor-pointer bg-surface-container-lowest px-1"
			/>
		</div>
	</div>

	<div class="grid grid-cols-3 gap-1">
		{#each AUSRICHTUNG as a (a.wert)}
			<button
				onclick={() => (el.style.textAlign = a.wert)}
				aria-pressed={el.style?.textAlign === a.wert}
				class="py-1 rounded-lg text-sm transition-colors {el.style?.textAlign === a.wert
					? 'bg-secondary-container text-on-secondary-container'
					: 'bg-surface-container text-on-surface-variant'}"
				title={a.name}
				aria-label={a.name}>{a.zeichen}</button
			>
		{/each}
	</div>

	<Kaestchen
		checked={el.style.fontWeight === 'bold'}
		onchange={(e) => {
			el.style.fontWeight = e.currentTarget.checked ? 'bold' : 'normal';
		}}
		label="Fett"
	/>
</div>
