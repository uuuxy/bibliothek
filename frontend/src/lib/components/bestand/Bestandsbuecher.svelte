<!-- @component Bestandsbücher — Zugangsbuch und Abgangsbuch unter einem Menüpunkt.

     Beide standen bis zum 17.09.2026 als Reiter im Medienkatalog. Sie gehören dorthin
     fachlich durchaus — es ist der Bestand —, nur sind sie etwas anderes als die tägliche
     Arbeit am Katalog: ein Nachweis, den jemand ein- bis zweimal im Jahr zum Stichtag
     (15.3./15.9.) ausdruckt und abheftet. Dieselbe Begründung, mit der der
     Schuljahreswechsel unter „System" liegt und nicht im Bibliotheks-Menü: Was selten
     gebraucht wird, lenkt im täglichen Weg nur ab (Betreiber-Entscheidung 17.09.2026).

     EIN Menüpunkt und nicht zwei: Die beiden Bücher sind derselbe Nachweis in zwei
     Richtungen — was kam, was ging. Zwei Punkte nebeneinander im Menü behaupteten zwei
     Aufgaben, wo es eine ist.

     Die Spaltenbeschreibungen stehen hier, weil sie zum BUCH gehören und nicht zum
     Bauteil: Bestandsbuch.svelte zeichnet, was es bekommt. -->
<script>
	import PageShell from '../layout/PageShell.svelte';
	import Reiter from '../ui/Reiter.svelte';
	import Bestandsbuch from './Bestandsbuch.svelte';

	// Zugang zuerst: Die Leserichtung der Arbeitshilfe ist „was kam, dann was ging", und
	// das Zugangsbuch ist das, was die Schule häufiger braucht.
	let aktiv = $state('zugangsbuch');
</script>

<PageShell>
	<Reiter
		etikett="Bestandsbücher"
		reiter={[
			{ id: 'zugangsbuch', label: 'Zugangsbuch' },
			{ id: 'abgangsbuch', label: 'Abgangsbuch' }
		]}
		{aktiv}
		onwahl={(id) => (aktiv = id)}
	/>

	{#if aktiv === 'zugangsbuch'}
		<Bestandsbuch
			pfad="zugangsbuch"
			buchname="Zugangsbuch"
			wortSingular="Zugang"
			spalten={[
				{ kopf: 'Zugang', feld: 'datum', klasse: 'whitespace-nowrap' },
				{ kopf: 'Nummer', feld: 'barcode', klasse: 'whitespace-nowrap font-mono' },
				{ kopf: 'Titel', feld: 'titel' },
				{ kopf: 'Lieferant', feld: 'lieferant', klasse: 'whitespace-nowrap' }
			]}
		/>
	{:else}
		<Bestandsbuch
			pfad="abgangsbuch"
			buchname="Abgangsbuch"
			wortSingular="Abgang"
			spalten={[
				{ kopf: 'Abgang', feld: 'datum', klasse: 'whitespace-nowrap' },
				{ kopf: 'Nummer', feld: 'barcode', klasse: 'whitespace-nowrap font-mono' },
				{ kopf: 'Titel', feld: 'titel' },
				{ kopf: 'Signatur', feld: 'signatur', klasse: 'whitespace-nowrap' },
				{ kopf: 'Grund', feld: 'grund_text', klasse: 'whitespace-nowrap' }
			]}
		/>
	{/if}
</PageShell>
