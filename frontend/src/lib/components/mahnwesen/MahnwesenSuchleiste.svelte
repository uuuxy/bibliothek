<!-- @component MahnwesenSuchleiste — die eine Suche dieser Seite, darunter Filter und Knöpfe.

     Jede Seite trägt genau eine Suche, in derselben Gestalt wie Medienkatalog, Portal und
     Theke: die 48-px-Suchpille über die volle Breite. Filter und Knöpfe stehen darunter in
     einer eigenen Zeile; nebeneinander säße die Pille 12 px höher als jedes Auswahlfeld und
     risse die Control-Grundlinie aus styles/basis.css auf. Die Knöpfe stehen neben dem
     Klassenfilter, so wie „Neuer Schüler" in SchuelerdateiLeiste, und nicht über den
     Reitern: In M3 ist die Suchleiste „a persistent and prominent search field at the top
     of the screen" und nimmt den Platz der App-Leiste ein; Reiter sitzen „at the top of the
     content pane under an app bar". Eine Knopfzeile darüber gibt es in dieser Reihenfolge
     nicht.

     Warum die Zeile nicht an `daten` hängt (die Reiter schon): Schlägt das Laden fehl,
     bleibt `daten` null; die Reiter würden dann „Alle 0" behaupten, was der Fehlermeldung
     darunter widerspricht. Der Knopf „Neu laden" ist in genau diesem Moment aber der
     einzige Rückweg und muss stehen bleiben. -->
<script>
	import { mahnwesenStore } from '../../stores/mahnwesen.svelte.js';
	import Select from '../ui/Select.svelte';
	import Suchpille from '../ui/Suchpille.svelte';
	import MahnwesenAktionen from './MahnwesenAktionen.svelte';

	/** @type {{ onMahnlauf: () => void, onBescheid: (schuelerId: string) => void, darfBescheid: boolean, darfMahnen: boolean }} */
	let { onMahnlauf, onBescheid, darfBescheid, darfMahnen } = $props();
</script>

<div class="mt-4 flex flex-col gap-3 print:hidden">
	<Suchpille
		id="mahnwesen-suchfeld"
		bind:wert={mahnwesenStore.suche}
		platzhalter="Schüler oder Klasse suchen …"
		etikett="Schüler oder Klasse suchen"
	/>
	<div class="flex flex-wrap items-center gap-3">
		<Select
			bind:value={mahnwesenStore.gewaehlteKlasse}
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
			<MahnwesenAktionen {onMahnlauf} {onBescheid} {darfBescheid} {darfMahnen} />
		</div>
	</div>
</div>
