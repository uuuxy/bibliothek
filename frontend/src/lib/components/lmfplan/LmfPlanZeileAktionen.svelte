<!-- @component LmfPlanZeileAktionen — das Überlaufmenü einer Zeile im Planer (M3 Menus:
     „Use menus in situations that need extra actions, like: Overflow menus"). Verschoben wird
     nur am Griff der Zeile: Die Reihenfolge schlägt das Programm vor, von Hand bleibt die
     Korrektur „diese Klasse neben jene". Hier stehen zusammenlegen oder trennen, davor
     einfügen, festlegen oder lösen, Klasse herausnehmen, entfernen. -->
<script>
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
			? { id: 'trennen', text: 'In einzelne Stunden trennen', icon: 'call_split' }
			: {
					id: 'zusammen',
					text: 'Mit der Zeile davor zusammenlegen',
					icon: 'merge',
					disabled: nummer === 1
				},
		{ id: 'einfuegen', text: 'Zeile davor einfügen', icon: 'add' },
		fest
			? { id: 'fest', text: 'Festen Platz lösen', icon: 'keep_off' }
			: // Ohne gerechneten Platz gäbe es nichts vorzubelegen, und der Server nähme die
				// Zeile nicht an (400): Der Eintrag bleibt sichtbar, aber gesperrt.
				{ id: 'fest', text: 'Datum und Stunde festlegen', icon: 'keep', disabled: platzlos },
		...(klassen === 1
			? [{ id: 'klasseraus', text: 'Klasse aus dem Plan nehmen', icon: 'close' }]
			: []),
		{ id: 'entfernen', text: 'Zeile entfernen', icon: 'delete', trennerDavor: true }
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
