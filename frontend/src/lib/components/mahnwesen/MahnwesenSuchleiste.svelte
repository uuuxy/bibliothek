<!-- @component MahnwesenSuchleiste — die EINE Suche dieser Seite, darunter Filter und Knöpfe.

     Peter am 04.09.2026: „eine Leiste! aber nicht 2 … es soll gleich aussehen." Bis dahin
     stand über jeder Verwaltungsseite zusätzlich eine globale Suchleiste, und die eigene
     Suche saß als 36-px-Feld daneben im Werkzeugbalken. Jetzt trägt jede Seite genau eine
     Suche, und zwar in derselben Gestalt wie Medienkatalog, Portal und Theke: die
     48-px-Suchpille über die volle Breite. Filter und Knöpfe stehen DARUNTER in einer
     eigenen Zeile — nebeneinander säße die Pille 12 px höher als jedes Auswahlfeld und
     risse die Control-Grundlinie aus styles/basis.css auf.

     Am selben Tag, abends: Die Knopfzeile des Mahnwesens stand bis dahin ÜBER den Reitern
     im `aktionen`-Slot von PageShell — als einzige Seite von sechzehn. Die Pille begann
     dadurch 84 px tiefer als auf Schülerdatei, Katalog, Abgängern und allen anderen; Peter
     an zwei Bildschirmfotos: „hier ist die Suchleiste immer an anderen Positionen … ich
     empfinde es als Stilbruch". Der Slot war ein Überbleibsel: Er hielt die Knöpfe „neben
     dem Seitentitel", und den Seitentitel hat 68c4810 am 08.08.2026 abgeschafft. Die
     Knöpfe stehen jetzt hier unten neben dem Klassenfilter, so wie „Neuer Schüler" in
     StudentDirectoryToolbar. In M3 ist die Suchleiste „a persistent and prominent search
     field at the top of the screen" und nimmt den Platz der App-Leiste ein; Reiter sitzen
     „at the top of the content pane under an app bar". Eine Knopfzeile darüber gibt es in
     dieser Reihenfolge nicht.

     Warum die Zeile NICHT an `data` hängt (die Reiter schon): Schlägt das Laden fehl,
     bleibt `data` null — die Reiter würden dann „Alle 0" behaupten, was der Fehlermeldung
     darunter widerspricht. Der Knopf „Neu laden" ist in genau diesem Moment aber der
     einzige Rückweg und muss stehen bleiben. Vorher trug ihn PageShell und er überlebte
     jeden Fehler; das darf der Umzug nicht kosten. -->
<script>
	import { mahnwesenStore } from '../../stores/mahnwesen.svelte.js';
	import Select from '../ui/Select.svelte';
	import Suchpille from '../ui/Suchpille.svelte';
	import MahnwesenAktionen from './MahnwesenAktionen.svelte';

	/** @type {{ onMahnlauf: () => void }} */
	let { onMahnlauf } = $props();
</script>

<div class="mt-4 flex flex-col gap-3 print:hidden">
	<Suchpille
		id="mahnwesen-suchfeld"
		bind:wert={mahnwesenStore.searchQuery}
		platzhalter="Schüler oder Klasse suchen …"
		etikett="Schüler oder Klasse suchen"
	/>
	<div class="flex flex-wrap items-center gap-3">
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

		<div class="ml-auto flex flex-wrap items-center gap-2">
			<MahnwesenAktionen {onMahnlauf} />
		</div>
	</div>
</div>
