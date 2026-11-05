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
	 *   abschnitt: { topf: string, titel: string, exemplare: number, zeilen: any[] },
	 *   spalten: {kopf: string, feld: string, klasse?: string}[],
	 *   buchname: string,
	 *   nichts: import('svelte').Snippet
	 * }}
	 */
	let { abschnitt, spalten, buchname, nichts } = $props();

	let grenze = $state(SCHRITT);
	// Die Überschrift zählt Exemplare, „Mehr laden" Zeilen: Eine Zeile trägt alle Exemplare
	// eines Titels von einem Tag. Der Knopf nennt deshalb das Wort, sonst läse sich „5 weitere"
	// als fünf Bücher.
	const exemplare = $derived(abschnitt.exemplare);
	const anzahl = $derived(abschnitt.zeilen.length);
	const sichtbar = $derived(abschnitt.zeilen.slice(0, grenze));
</script>

<Abschnitt
	titel="{abschnitt.titel} · {formatZahl(exemplare)} {exemplare === 1 ? 'Exemplar' : 'Exemplare'}"
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
			{#each sichtbar as zeile, i (i)}
				<tr>
					{#each spalten as spalte (spalte.feld)}
						<td class={spalte.klasse ?? ''}>
							{#if spalte.feld === 'barcodes'}
								<!-- Ein Barcode bricht nicht an seinem Bindestrich um, nur zwischen zweien. -->
								{#each zeile.barcodes.split(', ') as teil, k (k)}{k > 0 ? ', ' : ''}<span
										class="whitespace-nowrap">{teil}</span
									>{/each}
							{:else}
								{spalte.feld === 'datum' ? formatDatum(zeile.datum) : zeile[spalte.feld]}
							{/if}
						</td>
					{/each}
				</tr>
			{/each}
		</tbody>
	</Tabelle>
	{#if grenze < anzahl}
		<div class="flex justify-center pt-2">
			<Button variant="secondary" onclick={() => (grenze += SCHRITT)}>
				Mehr laden ({formatZahl(anzahl - grenze)} weitere {anzahl - grenze === 1
					? 'Zeile'
					: 'Zeilen'})
			</Button>
		</div>
	{/if}
{/if}
