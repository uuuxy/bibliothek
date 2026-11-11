<!-- @component ProtokollGrenze — der Satz über einer Protokoll-Liste, die der Server bei den
     jüngsten 1000 Einträgen kappt (Logbuch und Verwaltungsprotokoll, repository.LadeProtokoll
     und LadeAdminProtokoll). Ohne ihn sieht eine gekappte Liste aus wie eine vollständige, und
     wer einen älteren Vorgang sucht und nicht findet, schlösse daraus, er sei nie protokolliert
     worden. `mr-auto` schiebt den Knopf daneben nach rechts; ohne Satz steht er dort allein. -->
<script>
	const GRENZE = 1000;

	/** @type {{ anzahl: number }} */
	let { anzahl } = $props();
</script>

{#if anzahl >= GRENZE}
	<p class="mr-auto text-sm text-on-surface-variant">
		Die <strong class="font-semibold">{GRENZE}</strong> jüngsten Einträge. Ältere Vorgänge sind protokolliert,
		aber hier nicht sichtbar.
	</p>
{/if}
