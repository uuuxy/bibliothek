<!-- @component LmfPlanZeileAktionen — das Überlaufmenü einer Zeile im Planer (M3 Menus:
     „Use menus in situations that need extra actions, like: Overflow menus"). Verschoben wird
     nur am Griff der Zeile: Die Reihenfolge schlägt das Programm vor, von Hand bleibt die
     Korrektur „diese Klasse neben jene". Hier stehen zusammenlegen oder trennen, davor
     einfügen, festlegen oder lösen, Klasse herausnehmen, entfernen. -->
<script>
	import { Merge, Pin, PinOff, Plus, Split, Trash, X } from '@lucide/svelte';
	import Menue from '../ui/Menue.svelte';

	/** @type {{ nummer: number, klassen: number, fest: boolean, platzlos?: boolean, onzusammen: () => void, ontrennen: () => void, oneinfuegen: () => void, onfest: () => void, onklasseraus: () => void, onentfernen: () => void }} */
	let {
		nummer,
		klassen,
		fest,
		platzlos = false,
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
			: // Ohne gerechneten Platz gäbe es nichts vorzubelegen, und der Server nähme die
				// Zeile nicht an (400): Der Eintrag bleibt sichtbar, aber gesperrt.
				{ id: 'fest', text: 'Datum und Stunde festlegen', icon: Pin, disabled: platzlos },
		...(klassen === 1 ? [{ id: 'klasseraus', text: 'Klasse aus dem Plan nehmen', icon: X }] : []),
		{ id: 'entfernen', text: 'Zeile entfernen', icon: Trash, trennerDavor: true }
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

<Menue
	etikett="Aktionen Zeile {nummer}"
	{eintraege}
	onwahl={(id) => aktionen[/** @type {keyof typeof aktionen} */ (id)]?.()}
/>
