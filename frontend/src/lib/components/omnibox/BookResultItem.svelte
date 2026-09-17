<script>
	import { bestandSatz } from '../../utils/format.js';

	let { book, index, selected, onSelect } = $props();

	// Vierte Spalte: der Bestand. Punkt 4 des Protokolls vom 16.09.2026 — „Bücher, zu
	// denen es keine Exemplare gibt, tauchen in der Trefferliste auf". Verstecken wäre
	// falsch (ein Titel ohne Exemplare ist ein legitimer Zustand: angelegt ohne
	// Bestandsangabe, Altbestand aus Littera), die Liste muss es SAGEN.
	//
	// Fehlt die Zahl, bleibt die Spalte leer: `bestand` ist ein Zeiger und nur die
	// Suchabfragen füllen ihn (repository/models.go). „Keine Exemplare" über einen
	// Titel zu schreiben, den niemand gezählt hat, wäre eine Behauptung.
	const satz = $derived(bestandSatz(book.bestand, book.verfuegbar));
	const ohneBestand = $derived(book.bestand === 0);
</script>

<!-- Gleiches Spaltenraster wie StudentResultItem, damit beide Gruppen im selben
     Dropdown auf einer Kante stehen. Und dieselbe Primärfarbe: Vorher war die
     Auswahl bei Schülern blau und bei Büchern indigo — zwei Akzente für dieselbe
     Interaktion in derselben Liste.
     Die Signatur steht vor dem Autor, weil sie beim Suchen im Regal die Frage
     beantwortet; das Cover ist raus, es hielt die Zeile auf und trug nichts bei. -->
<div
	id="dropdown-item-{index}"
	role="option"
	aria-selected={selected}
	aria-label="Buch: {book.titel} von {book.autor}, Signatur {book.signatur || 'keine'}{satz
		? `, ${satz}`
		: ''}"
	tabindex="-1"
	class="grid grid-cols-[minmax(0,1fr)_5rem_11rem_10rem] items-center gap-4 px-4 h-12 cursor-pointer {selected
		? 'bg-blue-50 text-blue-900'
		: 'text-slate-900 hover:bg-slate-50'}"
	onclick={() => onSelect(index)}
	onkeydown={(e) => {
		if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			onSelect(index);
		}
	}}
>
	<span class="truncate font-medium">{book.titel}</span>
	<span class="text-sm truncate {selected ? 'text-blue-700' : 'text-slate-600'}"
		>{book.signatur}</span
	>
	<span class="text-sm truncate {selected ? 'text-blue-700' : 'text-slate-600'}">{book.autor}</span>
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
