<!-- @component NavIcon — das Icon eines Navigationsziels.

     Die eine Stelle, an der die Zuordnung Menüpunkt → Icon steht; menue.js nennt nur den
     Schlüssel. Größe 24 px, M3 Navigation drawer: Icon 24 dp. Das gewählte Ziel zeigt sein Icon
     gefüllt, M3 (Icons, Fill): „Bottom navigation with filled symbols in selected and unselected
     states“. -->
<script>
	import Icon from '../ui/Icon.svelte';

	/** @type {{ name: string, aktiv?: boolean }} */
	let { name, aktiv = false } = $props();

	// Die Schlüssel sind die `icon`-Werte aus menue.js.
	/** @type {Record<string, import('../ui/iconNamen.js').IconName>} */
	const symbole = {
		kiosk: 'barcode_scanner',
		bell: 'notifications',
		catalog: 'local_library',
		book: 'menu_book',
		printer: 'print',
		users: 'group',
		identification: 'id_card',
		'academic-cap': 'school',
		calendar: 'calendar_month',
		'shopping-bag': 'shopping_bag',
		clipboard: 'fact_check',
		clock: 'schedule',
		'chart-bar': 'bar_chart',
		// Bestandsbücher (Zugang/Abgang) — ein Nachweis zum Abheften, kein Katalog.
		'book-marked': 'book',
		// Bestellberichte — das Blatt selbst. NICHT 'clipboard' (ClipboardCheck): Das trägt
		// die Inventur, und zwei Ziele mit demselben Symbol unterscheidet niemand.
		'file-text': 'description',
		shield: 'verified_user',
		key: 'key',
		cog: 'settings'
	};

	const symbol = $derived(symbole[name]);
</script>

{#if symbol}
	<Icon name={symbol} groesse={24} gefuellt={aktiv} />
{/if}
