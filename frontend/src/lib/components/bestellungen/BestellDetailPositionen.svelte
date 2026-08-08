<!-- @component Die bestellten Titel einer Bestellung — mit Cover, Autor und Verlag.

     In der Historie stand hier eine Tabellenzeile mit Titel, ISBN und Menge. Das reicht,
     um eine Bestellung zu FINDEN, aber nicht, um eine Lieferung wiederzuerkennen: Beim
     Auspacken hat man ein Buch in der Hand, keinen Datensatz. Deshalb steht das Cover
     vorn — es ist das Merkmal, das man ohne Lesen abgleicht. -->
<script>
	import { coverSrc } from '../../utils/coverSrc.js';
	import { orderStore } from '../../stores/orderStore.svelte.js';
	import { BookOpen, Printer } from '@lucide/svelte';

	/**
	 * @type {{
	 *   positionen: any[],
	 *   euro: (n: number) => string,
	 *   onNachdruck: (pos: any) => void,
	 *   onTitel: (pos: any) => void
	 * }}
	 */
	let { positionen, euro, onNachdruck, onTitel } = $props();
</script>

{#if positionen.length === 0}
	<p class="text-sm text-slate-400 italic">Keine Positionen gespeichert.</p>
{:else}
	<ul class="divide-y divide-slate-100">
		{#each positionen as p (p.titel_name + p.isbn)}
			{@const quelle = coverSrc(p.cover_url, p.isbn)}
			<li class="flex items-start gap-4 py-4">
				<!-- Feste Größe mit Platzhalter darunter: Ein Cover, das erst nachlädt oder gar
				     nicht existiert, darf die Liste nicht springen lassen. -->
				<div
					class="h-24 w-16 shrink-0 overflow-hidden rounded-lg border border-slate-200 bg-slate-100"
				>
					{#if quelle}
						<img src={quelle} alt="" class="h-full w-full object-cover" loading="lazy" />
					{:else}
						<div class="flex h-full w-full items-center justify-center text-slate-400">
							<BookOpen class="h-6 w-6" aria-hidden="true" />
						</div>
					{/if}
				</div>

				<div class="min-w-0 flex-1">
					<p class="font-semibold text-slate-800">{p.titel_name}</p>
					<p class="text-sm text-slate-500">
						{[p.autor, p.verlag].filter(Boolean).join(' · ') || 'Autor und Verlag nicht hinterlegt'}
					</p>
					<p class="mt-0.5 font-mono text-xs text-slate-400">{p.isbn || 'ohne ISBN'}</p>

					<div class="mt-2 flex flex-wrap items-center gap-1">
						<!-- Beide Verweise nur, wenn sie auch irgendwohin führen: der Titelsatz nur bei
						     vorhandener titel_id (die Bestellung überlebt den Titel, ON DELETE SET NULL),
						     der Nachdruck nur bei offenen Etiketten. Ein Verweis ins Leere entwertet alle
						     anderen gleich mit. -->
						{#if p.etiketten_offen > 0}
							<button
								type="button"
								onclick={() => onNachdruck(p)}
								data-tip="{p.etiketten_offen} Exemplare dieses Titels haben kein Etikett — im Druck-Center nachdrucken"
								aria-label="Etiketten für {p.titel_name} nachdrucken"
								class="icon-btn gap-1 px-1.5 text-xs font-semibold text-blue-700 hover:bg-blue-50"
							>
								<Printer class="h-4 w-4" aria-hidden="true" />
								{p.etiketten_offen}
							</button>
						{/if}
						{#if p.titel_id}
							<button
								type="button"
								onclick={() => onTitel(p)}
								data-tip="Titelsatz öffnen"
								aria-label="Titelsatz von {p.titel_name} öffnen"
								class="icon-btn text-slate-400 hover:bg-slate-100 hover:text-slate-700"
							>
								<BookOpen class="h-4 w-4" aria-hidden="true" />
							</button>
						{/if}
					</div>
				</div>

				<div class="shrink-0 text-right">
					<p class="font-semibold text-slate-800 tabular-nums">{p.menge}×</p>
					{#if orderStore.preiseErfassen}
						<p class="text-xs text-slate-400 tabular-nums">{euro(p.einzelpreis)}</p>
						<p class="mt-1 font-bold text-slate-900 tabular-nums">{euro(p.gesamtpreis)}</p>
					{/if}
				</div>
			</li>
		{/each}
	</ul>
{/if}
