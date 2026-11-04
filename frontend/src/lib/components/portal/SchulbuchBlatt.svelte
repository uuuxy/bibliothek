<!-- @component SchulbuchBlatt — die Angaben zu einem Buch aus Mein Portal im Seitenblatt: Fach und
     Jahrgang, der Bestand und die ISBN zum Kopieren. Für den Fachsprecher, der nachbestellt und
     dafür die genaue Nummer braucht; die Kachel nennt sie nur beim Zeigen und hat keinen Knopf.
     Zahlen und Nummer stehen wie im Kopf der Buchakte (BuchakteKopf, NummerKopieren). -->
<script>
	import Seitenblatt from '../ui/Seitenblatt.svelte';
	import NummerKopieren from '../ui/NummerKopieren.svelte';
	import { jahrgangSpanne } from '../../utils/format.js';
	import { auflagenAufschluesselung } from '../../utils/auflagenText.js';

	/** @type {{ buch: any, onschliessen: () => void }} */
	let { buch, onschliessen } = $props();

	const jahrgang = $derived(jahrgangSpanne(buch.jahrgangVon, buch.jahrgangBis));
	const einordnung = $derived(
		[buch.subject, jahrgang && `Jahrgang ${jahrgang}`, buch.auflage].filter(Boolean).join(' · ')
	);
	const bestellt = $derived(buch.imZulauf ?? 0);
</script>

<Seitenblatt titel={buch.title} {onschliessen}>
	{#if einordnung}
		<p class="text-sm text-on-surface-variant">{einordnung}</p>
	{/if}
	<dl class="flex divide-x divide-outline-variant">
		<div class="pr-6">
			<dd class="text-lg font-semibold tabular-nums text-on-surface">
				{buch.verfuegbar ?? 0} von {buch.gesamt ?? 0}
			</dd>
			<dt class="text-xs text-on-surface-variant">verfügbar</dt>
		</div>
		{#if bestellt > 0}
			<div class="pl-6">
				<dd class="text-lg font-semibold tabular-nums text-on-surface">{bestellt}</dd>
				<dt class="text-xs text-on-surface-variant">bestellt</dt>
			</div>
		{/if}
	</dl>
	{#if buch.auflagenBestand?.length}
		<p class="text-xs text-on-surface-variant">{auflagenAufschluesselung(buch.auflagenBestand)}</p>
	{/if}
	<div class="text-sm text-on-surface-variant">
		<NummerKopieren nummer={buch.isbn} />
	</div>
</Seitenblatt>
