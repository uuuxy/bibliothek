<!-- @component Die Bücher, die AUS DIESER Bestellung entstanden sind — mit ihren Barcodes.

     Das ist der Teil, den die Historie bisher nicht zeigen konnte. Bis Migration 063 war
     es auch nicht möglich: Ein Exemplar wusste nicht, aus welcher Lieferung es stammt.
     Seitdem trägt es seine bestellung_id, und damit lässt sich die Frage beantworten,
     die beim Auspacken zuerst kommt — welche Nummern gehören zu diesem Karton.

     Leer heisst hier NICHT „nichts geliefert": Altbestand aus dem Littera-Import und
     alles, was vor Migration 063 bestellt wurde, hat keine bestellung_id. Die Ansicht
     sagt das ausdrücklich, statt eine leere Liste für eine Aussage zu halten. -->
<script>
	import Icon from '../ui/Icon.svelte';
	import { anzahlText } from '../../utils/format.js';

	/** @type {{ exemplare: any[] }} */
	let { exemplare } = $props();

	// Nach Titel gebündelt: Ein Karton mit 30 Büchern ergäbe sonst 30 gleiche Zeilen, in
	// denen nur die Nummer wechselt. Gebündelt wird nach der Kennung des Titels, denn
	// Schülerbuch und Arbeitsheft heißen oft gleich. Die Reihenfolge kommt aus der Abfrage
	// (Titel, Kennung, dann Barcode aufsteigend) — hier wird nur gruppiert, nicht umsortiert.
	const gruppen = $derived.by(() => {
		/** @type {{titelId: string, titel: string, stuecke: any[]}[]} */
		const out = [];
		for (const e of exemplare) {
			const letzte = out.at(-1);
			if (letzte && letzte.titelId === e.titel_id) letzte.stuecke.push(e);
			else out.push({ titelId: e.titel_id, titel: e.titel_name, stuecke: [e] });
		}
		return out;
	});
</script>

{#if exemplare.length === 0}
	<p class="text-sm text-on-surface-variant italic">
		Zu dieser Bestellung sind keine Exemplare hinterlegt. Bei Bestellungen aus der Zeit vor der
		Exemplar-Zuordnung ist das normal. Die bestellten Titel stehen oben.
	</p>
{:else}
	<div class="space-y-4">
		{#each gruppen as g (g.titelId)}
			<div>
				<p class="mb-1.5 text-body-medium-emphasized text-on-surface">
					{g.titel}
					<span class="ml-1 font-normal text-on-surface-variant"
						>· {anzahlText(g.stuecke.length, 'Exemplar', 'Exemplare')}</span
					>
				</p>
				<ul class="flex flex-wrap gap-1.5">
					{#each g.stuecke as e (e.barcode_id)}
						<li>
							<!-- Drei Zustände, drei Töne. Der Barcode bleibt in jedem Fall lesbar — er ist
							     die Nummer, die auf dem Buch klebt, und der Grund, warum man hier ist. -->
							<span
								class="inline-flex items-center gap-1 rounded-lg px-2 py-1 font-mono text-xs
								{e.ist_ausgesondert
									? 'bg-surface-container text-on-surface-variant line-through'
									: e.etikett_gedruckt
										? 'bg-success-container text-on-success-container'
										: 'bg-warning-container text-on-warning-container'}"
								data-tip={e.ist_ausgesondert
									? 'Ausgesondert'
									: e.etikett_gedruckt
										? 'Etikett gedruckt'
										: 'Etikett steht noch aus'}
							>
								{#if e.ist_ausgesondert}
									<Icon name="block" groesse={18} />
								{:else if e.etikett_gedruckt}
									<Icon name="check_circle" groesse={18} />
								{:else}
									<Icon name="schedule" groesse={18} />
								{/if}
								{e.barcode_id}
							</span>
						</li>
					{/each}
				</ul>
			</div>
		{/each}
	</div>
{/if}
