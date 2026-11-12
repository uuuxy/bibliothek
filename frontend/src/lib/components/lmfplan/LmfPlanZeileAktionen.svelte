<!-- @component LmfPlanZeileAktionen — das Überlaufmenü einer Zeile im Planer (M3 Menus:
     „Use menus in situations that need extra actions, like: Overflow menus"). Mit der Maus
     wird die Zeile gezogen; ohne Ziehen stehen die Wege hier, ein Platz und die weiten
     Sprünge, wie „Folie nach oben verschieben" bis „Folie ans Ende verschieben" in Google
     Präsentationen. Dazu zusammenlegen oder trennen, davor einfügen, festlegen oder lösen,
     Klasse herausnehmen, entfernen. Pfeile in jeder Zeile wären über hundert Knöpfe. -->
<script>
	import {
		ArrowDown,
		ArrowDownToLine,
		ArrowUp,
		ArrowUpToLine,
		Merge,
		Pin,
		PinOff,
		Plus,
		Split,
		Trash,
		X
	} from '@lucide/svelte';
	import Menue from '../ui/Menue.svelte';

	/** @type {{ nummer: number, anzahl: number, klassen: number, fest: boolean, platzlos?: boolean, onhoch: () => void, onrunter: () => void, onanfang: () => void, onende: () => void, onzusammen: () => void, ontrennen: () => void, oneinfuegen: () => void, onfest: () => void, onklasseraus: () => void, onentfernen: () => void }} */
	let {
		nummer,
		anzahl,
		klassen,
		fest,
		platzlos = false,
		onhoch,
		onrunter,
		onanfang,
		onende,
		onzusammen,
		ontrennen,
		oneinfuegen,
		onfest,
		onklasseraus,
		onentfernen
	} = $props();

	const eintraege = $derived([
		{ id: 'hoch', text: 'Nach oben', icon: ArrowUp, disabled: nummer === 1 },
		{ id: 'runter', text: 'Nach unten', icon: ArrowDown, disabled: nummer === anzahl },
		{ id: 'anfang', text: 'An den Anfang', icon: ArrowUpToLine, disabled: nummer === 1 },
		{ id: 'ende', text: 'Ans Ende', icon: ArrowDownToLine, disabled: nummer === anzahl },
		klassen > 1
			? { id: 'trennen', text: 'In einzelne Stunden trennen', icon: Split, trennerDavor: true }
			: {
					id: 'zusammen',
					text: 'Mit der Zeile davor zusammenlegen',
					icon: Merge,
					disabled: nummer === 1,
					trennerDavor: true
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
		hoch: () => onhoch(),
		runter: () => onrunter(),
		anfang: () => onanfang(),
		ende: () => onende(),
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
