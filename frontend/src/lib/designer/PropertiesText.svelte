<!-- @component PropertiesText — Textformatierung des ausgewählten Ausweis-Elements.

     Getrennt von PropertiesPanel, weil dort ALLES für ALLE Elementarten steht:
     Lage, Größe, Ebene, Sichtbarkeit. Schrift betrifft nur Textelemente — und bei
     dynamischen Feldern (Name, Klasse …) kommt der Inhalt aus den Schülerdaten,
     deshalb fehlt dort das Inhaltsfeld. -->
<script>
	import Select from '../components/ui/Select.svelte';
	import ZahlenFeld from './ZahlenFeld.svelte';

	/**
	 * @type {{
	 *   el: any,
	 *   istDynamisch: boolean,
	 *   schriften: Array<{ value: string, label: string }>
	 * }}
	 */
	let { el, istDynamisch, schriften } = $props();

	const AUSRICHTUNG = [
		{ wert: 'left', zeichen: '⬅' },
		{ wert: 'center', zeichen: '↔' },
		{ wert: 'right', zeichen: '➡' }
	];
</script>

<div class="space-y-3 pt-2 border-t border-slate-100">
	<span class="text-xs font-medium text-slate-500 block">Textformatierung</span>

	{#if !istDynamisch}
		<div class="space-y-1">
			<span class="text-xs text-slate-400 font-medium block">Inhalt</span>
			<input
				type="text"
				bind:value={el.content}
				class="w-full bg-white border border-slate-200 rounded-xl px-2 py-1.5 text-xs focus:outline-none focus:ring-1 focus:ring-blue-500"
			/>
		</div>
	{/if}

	<div class="space-y-1">
		<span class="text-xs text-slate-400 font-medium block">Schriftart</span>
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
			<span class="text-xs text-slate-400 font-medium block">Farbe</span>
			<input
				type="color"
				bind:value={el.style.color}
				class="w-full h-8 rounded-xl border border-slate-200 cursor-pointer bg-white px-1"
			/>
		</div>
	</div>

	<div class="grid grid-cols-3 gap-1">
		{#each AUSRICHTUNG as a (a.wert)}
			<button
				onclick={() => (el.style.textAlign = a.wert)}
				class="py-1 rounded-lg text-sm transition-colors {el.style?.textAlign === a.wert
					? 'bg-blue-600 text-white'
					: 'bg-slate-100 text-slate-500 hover:bg-slate-200'}"
				title={a.wert}>{a.zeichen}</button
			>
		{/each}
	</div>

	<label class="flex items-center gap-2 cursor-pointer">
		<input
			type="checkbox"
			checked={el.style.fontWeight === 'bold'}
			onchange={(e) => {
				el.style.fontWeight = /** @type {HTMLInputElement} */ (e.currentTarget).checked
					? 'bold'
					: 'normal';
			}}
			class="rounded border-slate-300 text-blue-600"
		/>
		<span class="text-xs text-slate-600 font-medium">Fett</span>
	</label>
</div>
