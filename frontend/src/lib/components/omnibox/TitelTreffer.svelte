<script>
	import { bestandSatz } from '../../utils/format.js';

	let { treffer, index, gewaehlt, onSelect } = $props();

	// Vierte Spalte: der Bestand, denn auch Bücher ohne Exemplare stehen in der Trefferliste.
	// Verstecken wäre falsch (ein Titel ohne Exemplare ist ein legitimer Zustand: angelegt
	// ohne Bestandsangabe, Altbestand aus Littera); die Liste muss es sagen.
	//
	// Fehlt die Zahl, bleibt die Spalte leer: `bestand` ist ein Zeiger und nur die
	// Suchabfragen füllen ihn (repository/titel.go). „Keine Exemplare" über einen
	// Titel zu schreiben, den niemand gezählt hat, wäre eine Behauptung.
	const satz = $derived(bestandSatz(treffer.bestand, treffer.verfuegbar, treffer.im_zulauf));
	// „2 bestellt" ist kein Titel ohne Bestand: Die Bücher kommen.
	const ohneBestand = $derived(treffer.bestand === 0 && !treffer.im_zulauf);
</script>

<!-- Gleiches Spaltenraster wie LeserTreffer, damit beide Gruppen im selben
     Dropdown auf einer Kante stehen. Die markierte Zeile trägt in beiden Gruppen
     dieselbe Schicht wie in den Menüs des Hauses (ui/SelectListe).
     Die Signatur steht vor dem Autor, weil sie beim Suchen im Regal die Frage
     beantwortet; das Cover ist raus, es hielt die Zeile auf und trug nichts bei. -->
<div
	id="dropdown-item-{index}"
	role="option"
	aria-selected={gewaehlt}
	aria-label="Buch: {treffer.titel} von {treffer.autor}, Signatur {treffer.signatur || 'keine'}{satz
		? `, ${satz}`
		: ''}"
	tabindex="-1"
	class="m3-state grid grid-cols-[minmax(0,1fr)_5rem_11rem_10rem] items-center gap-4 px-4 h-12 cursor-pointer text-on-surface {gewaehlt
		? 'bg-on-surface/8'
		: ''}"
	onclick={() => onSelect(index)}
	onkeydown={(e) => {
		if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			onSelect(index);
		}
	}}
>
	<span class="truncate font-medium">{treffer.titel}</span>
	<span class="text-sm truncate text-on-surface-variant">{treffer.signatur}</span>
	<span class="text-sm truncate text-on-surface-variant">{treffer.autor}</span>
	<!-- „Keine Exemplare" wird abgesetzt, „0 von 5 verfügbar" nicht: Das eine ist ein
	     Titel ohne Bestand — dort ist nichts zu holen, auch nicht morgen —, das andere
	     der Normalfall im Schuljahr, in dem fast jedes Lernmittel verliehen ist. Ein
	     Hinweis, der bei beidem gleich aussieht, sagt nichts (derselbe Grund wie auf
	     der Katalog-Kachel). -->
	{#if satz}
		<span
			class="text-sm truncate justify-self-start {ohneBestand
				? 'rounded-sm bg-error-container px-2 py-0.5 text-on-error-container'
				: 'text-on-surface-variant'}">{satz}</span
		>
	{:else}
		<span></span>
	{/if}
</div>
