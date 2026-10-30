<!-- @component BestandsbuchAbschnitt — ein Topf eines Bestandsbuchs: Überschrift mit Zahl,
     Tabelle und „Mehr laden".

     Der Bildschirm zeichnet zuerst 50 Zeilen, wie die Katalogsuche: Ein Zeitraum kann
     Zehntausende Zugänge haben, etwa am Tag eines Listenimports, und sie alle zu zeichnen hält
     die Seite Sekunden an. Vollständig bleiben die Zahl in der Überschrift und der Ausdruck. -->
<script>
	import Abschnitt from '../ui/Abschnitt.svelte';
	import Button from '../ui/Button.svelte';
	import Tabelle from '../ui/Tabelle.svelte';
	import { formatDatum, formatZahl } from '../../utils/format.js';

	const SCHRITT = 50;

	/**
	 * @type {{
	 *   abschnitt: { topf: string, titel: string, zeilen: any[] },
	 *   spalten: {kopf: string, feld: string, klasse?: string}[],
	 *   buchname: string,
	 *   nichts: import('svelte').Snippet
	 * }}
	 */
	let { abschnitt, spalten, buchname, nichts } = $props();

	let grenze = $state(SCHRITT);
	const anzahl = $derived(abschnitt.zeilen.length);
	const sichtbar = $derived(abschnitt.zeilen.slice(0, grenze));
</script>

<Abschnitt
	titel="{abschnitt.titel} · {formatZahl(anzahl)} {anzahl === 1 ? 'Exemplar' : 'Exemplare'}"
/>
{#if anzahl === 0}
	{@render nichts()}
{:else}
	<Tabelle beschriftung="{buchname} — {abschnitt.titel}">
		<thead>
			<tr>
				{#each spalten as spalte (spalte.feld)}
					<th>{spalte.kopf}</th>
				{/each}
			</tr>
		</thead>
		<tbody>
			{#each sichtbar as zeile (zeile.barcode)}
				<tr>
					{#each spalten as spalte (spalte.feld)}
						<td class={spalte.klasse ?? ''}>
							{spalte.feld === 'datum' ? formatDatum(zeile.datum) : zeile[spalte.feld]}
						</td>
					{/each}
				</tr>
			{/each}
		</tbody>
	</Tabelle>
	{#if grenze < anzahl}
		<div class="flex justify-center pt-2">
			<Button variant="secondary" onclick={() => (grenze += SCHRITT)}>
				Mehr laden ({formatZahl(anzahl - grenze)} weitere)
			</Button>
		</div>
	{/if}
{/if}
