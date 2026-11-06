<!-- @component SignaturCoverVorschau — zwei vorhandene Cover der Titel unter einer Signatur, als
     Vorschau am Vorschlagsmenü der Signatur (ui/MenueVorschau). Ohne Titel mit Cover bleibt sie
     leer: Ein Kasten „kein Cover“ an jedem dritten Eintrag sagte nichts über das Regal.

     Die Fläche trägt Farbe, Ecken und Schatten der Menüfläche (SelectListe), denn sie steht wie
     ein Untermenü daneben. -->
<script>
	import BuchCover from '../ui/BuchCover.svelte';
	import { ladeCoverZurSignatur } from '../../utils/signaturen.js';

	/** @type {{ signatur: string }} */
	let { signatur } = $props();

	let titel = $state(
		/** @type {{ titel_id: string, titel: string, isbn: string, cover_url: string }[]} */ ([])
	);
	$effect(() => {
		let abgebrochen = false;
		titel = [];
		ladeCoverZurSignatur(signatur).then((liste) => {
			if (!abgebrochen) titel = liste;
		});
		return () => {
			abgebrochen = true;
		};
	});
</script>

{#if titel.length}
	<div class="flex gap-2 rounded-sm bg-surface-container p-2 shadow-xl">
		{#each titel as t (t.titel_id)}
			<div class="w-28">
				<BuchCover
					coverUrl={t.cover_url}
					isbn={t.isbn}
					titel={t.titel}
					groesse="kachel"
					nurGespeichert
					dekorativ
				/>
			</div>
		{/each}
	</div>
{/if}
