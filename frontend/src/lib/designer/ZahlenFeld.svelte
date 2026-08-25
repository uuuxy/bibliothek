<!-- @component ZahlenFeld — beschriftetes Zahlenfeld des Ausweis-Designers (mm/pt).
     War ein Snippet in PropertiesPanel; als eigene Datei kann auch der
     Text-Bereich es benutzen, ohne dass beide dasselbe zweimal beschreiben.

     Rundet auf eine Nachkommastelle: Der Designer rechnet in mm mit
     Fließkommazahlen, ungerundet stünden im Feld sonst Werte wie 12.300000000001. -->
<script>
	import Feld from '../components/ui/Feld.svelte';

	/**
	 * @type {{
	 *   label: string, value: number, min: number, max?: number, step: number,
	 *   onInput: (v: number) => void
	 * }}
	 */
	let { label, value, min, max, step, onInput } = $props();
</script>

<Feld
	type="number"
	{label}
	{min}
	{max}
	{step}
	value={Math.round(value * 10) / 10}
	oninput={(e) => onInput(parseFloat(/** @type {HTMLInputElement} */ (e.currentTarget).value) || 0)}
/>
