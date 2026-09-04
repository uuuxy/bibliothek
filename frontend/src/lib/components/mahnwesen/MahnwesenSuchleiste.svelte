<!-- @component MahnwesenSuchleiste — die EINE Suche dieser Seite, darunter der Klassenfilter.

     Peter am 04.09.2026: „eine Leiste! aber nicht 2 … es soll gleich aussehen." Bis dahin
     stand über jeder Verwaltungsseite zusätzlich eine globale Suchleiste, und die eigene
     Suche saß als 36-px-Feld daneben im Werkzeugbalken. Jetzt trägt jede Seite genau eine
     Suche, und zwar in derselben Gestalt wie Medienkatalog, Portal und Theke: die
     48-px-Suchpille über die volle Breite. Filter und Knöpfe stehen DARUNTER in einer
     eigenen Zeile — nebeneinander säße die Pille 12 px höher als jedes Auswahlfeld und
     risse die Control-Grundlinie aus styles/basis.css auf. -->
<script>
	import { mahnwesenStore } from '../../stores/mahnwesen.svelte.js';
	import Select from '../ui/Select.svelte';
	import Suchpille from '../ui/Suchpille.svelte';
</script>

<div class="mt-4 flex flex-col gap-3 print:hidden">
	<Suchpille
		id="mahnwesen-suchfeld"
		bind:wert={mahnwesenStore.searchQuery}
		platzhalter="Schüler oder Klasse suchen …"
		etikett="Schüler oder Klasse suchen"
	/>
	<div class="flex items-center gap-3">
		<Select
			bind:value={mahnwesenStore.selectedKlasse}
			options={[
				{ value: '', label: 'Alle Klassen' },
				...mahnwesenStore.klassen.map((/** @type {any} */ k) => ({
					value: k.klasse,
					label: k.klasse
				}))
			]}
			class="w-40"
			aria-label="Nach Klasse filtern"
		/>
	</div>
</div>
