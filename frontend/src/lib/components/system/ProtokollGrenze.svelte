<!-- @component ProtokollGrenze — der Satz über einer Protokoll-Liste, die der Server bei den
     jüngsten Einträgen kappt (Logbuch und Verwaltungsprotokoll). Die Zahl nennt der Server
     (utils/listenGrenze.js). Ohne den Satz sieht eine gekappte Liste aus wie eine vollständige,
     und wer einen älteren Vorgang sucht und nicht findet, schlösse daraus, er sei nie
     protokolliert worden. `mr-auto` schiebt den Knopf daneben nach rechts; ohne Satz steht er
     dort allein. -->
<script>
	import { istGekappt } from '../../utils/listenGrenze.js';

	/** @type {{ anzahl: number, grenze: number }} */
	let { anzahl, grenze } = $props();
</script>

{#if istGekappt(anzahl, grenze)}
	<p class="mr-auto text-sm text-on-surface-variant">
		Die <strong class="font-semibold">{grenze}</strong> jüngsten Einträge. Ältere Vorgänge sind protokolliert,
		aber hier nicht sichtbar.
	</p>
{/if}
