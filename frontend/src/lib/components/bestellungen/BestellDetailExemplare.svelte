<!-- @component Die Bücher, die AUS DIESER Bestellung entstanden sind — mit ihren Barcodes.

     Das ist der Teil, den die Historie bisher nicht zeigen konnte. Bis Migration 063 war
     es auch nicht möglich: Ein Exemplar wusste nicht, aus welcher Lieferung es stammt.
     Seitdem trägt es seine bestellung_id, und damit lässt sich die Frage beantworten,
     die beim Auspacken zuerst kommt — welche Nummern gehören zu diesem Karton.

     Leer heisst hier NICHT „nichts geliefert": Altbestand aus dem Littera-Import und
     alles, was vor Migration 063 bestellt wurde, hat keine bestellung_id. Die Ansicht
     sagt das ausdrücklich, statt eine leere Liste für eine Aussage zu halten. -->
<script>
	import { CheckCircle2, Clock, Ban } from '@lucide/svelte';

	/** @type {{ exemplare: any[] }} */
	let { exemplare } = $props();

	// Nach Titel gebündelt: Ein Karton mit 30 Büchern ergäbe sonst 30 gleiche Zeilen, in
	// denen nur die Nummer wechselt. Die Reihenfolge kommt aus der Abfrage (Titel, dann
	// Barcode aufsteigend) — hier wird nur gruppiert, nicht umsortiert.
	const gruppen = $derived.by(() => {
		/** @type {{titel: string, stuecke: any[]}[]} */
		const out = [];
		for (const e of exemplare) {
			const letzte = out.at(-1);
			if (letzte && letzte.titel === e.titel_name) letzte.stuecke.push(e);
			else out.push({ titel: e.titel_name, stuecke: [e] });
		}
		return out;
	});
</script>

{#if exemplare.length === 0}
	<p class="text-sm text-slate-400 italic">
		Zu dieser Bestellung sind keine Exemplare hinterlegt. Bei Bestellungen aus der Zeit vor der
		Exemplar-Zuordnung ist das normal — die bestellten Titel stehen oben.
	</p>
{:else}
	<div class="space-y-4">
		{#each gruppen as g (g.titel)}
			<div>
				<p class="mb-1.5 text-sm font-semibold text-slate-700">
					{g.titel}
					<span class="ml-1 font-normal text-slate-400">({g.stuecke.length})</span>
				</p>
				<ul class="flex flex-wrap gap-1.5">
					{#each g.stuecke as e (e.barcode_id)}
						<li>
							<!-- Drei Zustände, drei Töne. Der Barcode bleibt in jedem Fall lesbar — er ist
							     die Nummer, die auf dem Buch klebt, und der Grund, warum man hier ist. -->
							<span
								class="inline-flex items-center gap-1 rounded-lg border px-2 py-1 font-mono text-xs
								{e.ist_ausgesondert
									? 'border-slate-200 bg-slate-50 text-slate-400 line-through'
									: e.etikett_gedruckt
										? 'border-emerald-200 bg-emerald-50 text-emerald-800'
										: 'border-amber-200 bg-amber-50 text-amber-800'}"
								data-tip={e.ist_ausgesondert
									? 'Ausgesondert'
									: e.etikett_gedruckt
										? 'Etikett gedruckt'
										: 'Etikett steht noch aus'}
							>
								{#if e.ist_ausgesondert}
									<Ban class="h-3.5 w-3.5" aria-hidden="true" />
								{:else if e.etikett_gedruckt}
									<CheckCircle2 class="h-3.5 w-3.5" aria-hidden="true" />
								{:else}
									<Clock class="h-3.5 w-3.5" aria-hidden="true" />
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
