<!-- @component Die bestellten Titel einer Bestellung — mit Cover, Autor und Verlag.

     In der Historie stand hier eine Tabellenzeile mit Titel, ISBN und Menge. Das reicht,
     um eine Bestellung zu FINDEN, aber nicht, um eine Lieferung wiederzuerkennen: Beim
     Auspacken hat man ein Buch in der Hand, keinen Datensatz. Deshalb steht das Cover
     vorn — es ist das Merkmal, das man ohne Lesen abgleicht. -->
<script>
	import BuchCover from '../ui/BuchCover.svelte';
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
	<p class="text-sm text-on-surface-variant italic">Keine Positionen gespeichert.</p>
{:else}
	<ul class="divide-y divide-outline-variant">
		{#each positionen as p (p.titel_name + p.isbn)}
			<li class="flex items-start gap-4 py-4">
				<!-- Nur das gespeicherte Cover: Die Liste zeigt Bücher aus dem Bestand und fragt
				     dafür keine fremde Quelle. -->
				<BuchCover
					coverUrl={p.cover_url || ''}
					isbn={p.isbn || ''}
					titel={p.titel_name}
					groesse="gross"
					dekorativ
					nurGespeichert
				/>

				<div class="min-w-0 flex-1">
					<p class="font-semibold text-on-surface">{p.titel_name}</p>
					<p class="text-sm text-on-surface-variant">
						{[p.autor, p.verlag].filter(Boolean).join(' · ') || 'Autor und Verlag nicht hinterlegt'}
					</p>
					<p class="mt-0.5 font-mono text-xs text-on-surface-variant">{p.isbn || 'ohne ISBN'}</p>

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
								class="icon-btn gap-1 px-1.5 text-xs font-semibold text-primary"
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
								class="icon-btn text-on-surface-variant"
							>
								<BookOpen class="h-4 w-4" aria-hidden="true" />
							</button>
						{/if}
					</div>
				</div>

				<div class="shrink-0 text-right">
					<p class="font-semibold text-on-surface tabular-nums">{p.menge}×</p>
					{#if orderStore.preiseErfassen}
						<p class="text-xs text-on-surface-variant tabular-nums">{euro(p.einzelpreis)}</p>
						<p class="mt-1 font-bold text-on-surface tabular-nums">{euro(p.gesamtpreis)}</p>
					{/if}
				</div>
			</li>
		{/each}
	</ul>
{/if}
