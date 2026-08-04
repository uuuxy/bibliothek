<!-- @component ZahlenFeld — beschriftetes Zahlenfeld des Ausweis-Designers (mm/pt).
     War ein Snippet in PropertiesPanel; als eigene Datei kann auch der
     Text-Bereich es benutzen, ohne dass beide dasselbe zweimal beschreiben.

     Rundet auf eine Nachkommastelle: Der Designer rechnet in mm mit
     Fließkommazahlen, ungerundet stünden im Feld sonst Werte wie 12.300000000001. -->
<script>
	/**
	 * @type {{
	 *   label: string, value: number, min: number, max?: number, step: number,
	 *   onInput: (v: number) => void
	 * }}
	 */
	let { label, value, min, max, step, onInput } = $props();
</script>

<div class="space-y-1">
	<span class="text-xs text-slate-400 font-medium block">{label}</span>
	<input
		type="number"
		{min}
		{max}
		{step}
		value={Math.round(value * 10) / 10}
		oninput={(e) =>
			onInput(parseFloat(/** @type {HTMLInputElement} */ (e.currentTarget).value) || 0)}
		class="w-full bg-white border border-slate-200 rounded-xl px-2 py-1.5 text-xs focus:outline-none focus:ring-1 focus:ring-blue-500"
	/>
</div>
