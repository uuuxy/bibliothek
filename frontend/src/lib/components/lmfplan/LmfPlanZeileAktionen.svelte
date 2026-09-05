<!-- @component LmfPlanZeileAktionen — die Aktionen einer Zeile im Planer. Sichtbar
     bleibt, was jede Zeile ständig braucht: hoch und runter (M3: Icon-Buttons „to
     display actions in a compact layout"). Alles andere — zusammenlegen oder trennen,
     davor einfügen, festlegen oder lösen, Klasse herausnehmen, entfernen — liegt im
     Überlaufmenü (M3 Menus: „Use menus in situations that need extra actions, like:
     Overflow menus"). Bis 06.09.2026 standen hier sechs Icon-Buttons je Zeile, rund
     dreihundert auf der Seite. -->
<script>
	import { ArrowDown, ArrowUp, Merge, Pin, PinOff, Plus, Split, Trash2, X } from '@lucide/svelte';
	import Button from '../ui/Button.svelte';
	import Menue from '../ui/Menue.svelte';

	/** @type {{ nummer: number, anzahl: number, klassen: number, fest: boolean, onhoch: () => void, onrunter: () => void, onzusammen: () => void, ontrennen: () => void, oneinfuegen: () => void, onfest: () => void, onklasseraus: () => void, onentfernen: () => void }} */
	let {
		nummer,
		anzahl,
		klassen,
		fest,
		onhoch,
		onrunter,
		onzusammen,
		ontrennen,
		oneinfuegen,
		onfest,
		onklasseraus,
		onentfernen
	} = $props();

	const eintraege = $derived([
		klassen > 1
			? { id: 'trennen', text: 'In einzelne Stunden trennen', icon: Split }
			: {
					id: 'zusammen',
					text: 'Mit der Zeile davor zusammenlegen',
					icon: Merge,
					disabled: nummer === 1
				},
		{ id: 'einfuegen', text: 'Zeile davor einfügen', icon: Plus },
		fest
			? { id: 'fest', text: 'Festen Platz lösen', icon: PinOff }
			: { id: 'fest', text: 'Datum und Stunde festlegen', icon: Pin },
		...(klassen === 1 ? [{ id: 'klasseraus', text: 'Klasse aus dem Plan nehmen', icon: X }] : []),
		{ id: 'entfernen', text: 'Zeile entfernen', icon: Trash2, trennerDavor: true }
	]);

	const aktionen = {
		trennen: () => ontrennen(),
		zusammen: () => onzusammen(),
		einfuegen: () => oneinfuegen(),
		fest: () => onfest(),
		klasseraus: () => onklasseraus(),
		entfernen: () => onentfernen()
	};
</script>

<Button
	variant="ghost"
	size="sm"
	onclick={onhoch}
	disabled={nummer === 1}
	title="Nach oben"
	aria-label="Zeile {nummer} nach oben"
>
	<ArrowUp class="h-4 w-4" aria-hidden="true" />
</Button>
<Button
	variant="ghost"
	size="sm"
	onclick={onrunter}
	disabled={nummer === anzahl}
	title="Nach unten"
	aria-label="Zeile {nummer} nach unten"
>
	<ArrowDown class="h-4 w-4" aria-hidden="true" />
</Button>
<Menue
	etikett="Aktionen Zeile {nummer}"
	{eintraege}
	onwahl={(id) => aktionen[/** @type {keyof typeof aktionen} */ (id)]?.()}
/>
