<!-- @component NavIcon — das Symbol eines Navigationsziels.

     Die EINE Stelle, an der die Zuordnung Menüpunkt → Symbol steht. Vorher gab es sie
     dreifach: als `icons`-Export in menu.js (den niemand importierte) und zweimal inline
     in Sidebar.svelte — je einmal für die System-Gruppe und einmal für alle anderen.
     Jedes Symbol war ein handgeschriebener SVG-Pfad; wer eines tauschen wollte, musste
     zwei Stellen finden und übersah die dritte. Sidebar.svelte war dadurch 476 Zeilen lang.

     Die Symbole kommen jetzt aus @lucide/svelte statt aus abgetippten Pfaden. Das ist
     nicht Geschmack, sondern die Regel des Hauses: frontend-hygiene.test.js sagt beim
     Anschlagen wörtlich "Bitte ein Lucide-Icon verwenden". Handgesetzte Pfade altern
     nicht mit, lassen sich nicht durchsuchen und haben hier über die Jahre zwei
     Strichstärken nebeneinander erzeugt. -->
<script>
	import {
		ScanBarcode,
		Bell,
		Library,
		BookOpen,
		Printer,
		Users,
		IdCard,
		GraduationCap,
		ShoppingBag,
		ClipboardCheck,
		Clock,
		ChartColumn,
		ShieldCheck,
		KeyRound,
		Settings
	} from '@lucide/svelte';

	/** @type {{ name: string, class?: string }} */
	let { name, class: klasse = 'h-5 w-5 shrink-0' } = $props();

	// Die Schlüssel sind die `icon`-Werte aus menu.js. Ein unbekannter Name liefert
	// bewusst nichts, statt ein Ersatzsymbol zu zeigen: Ein falsches Symbol im Menü
	// fällt niemandem auf, eine Lücke schon.
	const symbole = {
		kiosk: ScanBarcode,
		bell: Bell,
		catalog: Library,
		book: BookOpen,
		printer: Printer,
		users: Users,
		identification: IdCard,
		'academic-cap': GraduationCap,
		'shopping-bag': ShoppingBag,
		clipboard: ClipboardCheck,
		clock: Clock,
		'chart-bar': ChartColumn,
		shield: ShieldCheck,
		key: KeyRound,
		cog: Settings
	};

	const Symbol = $derived(symbole[name]);
</script>

{#if Symbol}
	<Symbol class={klasse} aria-hidden="true" />
{/if}
