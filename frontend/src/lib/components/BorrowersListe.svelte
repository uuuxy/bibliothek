<!-- @component BorrowersListe — wer dieses Buch gerade hat, eine Zeile je Exemplar.
     Ein Klick auf den Namen legt den Schüler-Barcode in den Scan-Kanal und springt
     zurück: derselbe Weg wie ein Scan am Pult.

     Überfällig trägt Farbe, Zeichen und für Vorleseprogramme das Wort, wie in der Leserakte
     (AusleiheRueckgabe.svelte). -->
<script>
	import { CircleAlert } from '@lucide/svelte';
	import { appState } from '../../inventur/lib/store.svelte.js';

	/**
	 * @type {{
	 *   zeilen: any[],
	 *   onBack: () => void,
	 *   fmtDate: (d: string) => string
	 * }}
	 */
	let { zeilen, onBack, fmtDate } = $props();
</script>

<div class="w-full">
	<ul class="divide-y divide-outline-variant">
		{#each zeilen as b, _i (_i)}
			<!-- Dauerleihe (Kollegium): keine Frist, nie überfällig — wie in der Akte. -->
			{@const dauerleihe = !!b.ist_dauerleihe}
			{@const ueberfaellig = !dauerleihe && new Date(b.rueckgabe_frist) < new Date()}
			<li
				class="px-5 py-3.5 hover:bg-surface transition-colors flex items-center justify-between group"
			>
				<div class="flex items-center gap-3 min-w-0">
					<div
						class="w-9 h-9 rounded-full bg-surface-container-highest text-on-surface-variant flex items-center justify-center font-bold text-xs shrink-0"
					>
						{b.schueler_name?.[0] ?? ''}{b.schueler_nachname?.[0] ?? ''}
					</div>
					<div class="min-w-0">
						<button
							onclick={() => {
								appState.triggerStudentScan = b.schueler_barcode;
								onBack();
							}}
							class="text-sm font-semibold text-on-surface hover:text-primary text-left cursor-pointer truncate block"
						>
							{b.schueler_name}
							{b.schueler_nachname}
							<span class="text-xs font-normal text-on-surface-variant ml-1"
								>({b.klasse || 'Unbekannt'})</span
							>
						</button>
						<p class="text-xs text-on-surface-variant font-mono mt-0.5">
							Exemplar: {b.exemplar_barcode}
						</p>
					</div>
				</div>
				<div class="text-right shrink-0 ml-4 flex gap-6 items-center">
					<div class="text-right hidden sm:block">
						<p class="text-label-small font-medium text-on-surface-variant">Ausgeliehen</p>
						<p class="text-sm font-semibold text-on-surface-variant">
							{fmtDate(b.ausgeliehen_am)}
						</p>
					</div>
					<div class="text-right">
						<p class="text-label-small font-medium text-on-surface-variant">Rückgabe bis</p>
						<p
							class="flex items-center justify-end gap-1 text-sm font-bold {ueberfaellig
								? 'text-error'
								: 'text-on-surface'}"
						>
							{dauerleihe ? 'ohne Frist' : fmtDate(b.rueckgabe_frist)}
							{#if ueberfaellig}
								<CircleAlert class="h-4 w-4 shrink-0" aria-hidden="true" />
								<span class="sr-only">Überfällig</span>
							{/if}
						</p>
					</div>
				</div>
			</li>
		{/each}
	</ul>
	{#if zeilen.length === 0}
		<div class="py-8 text-center text-sm text-on-surface-variant">
			Keine Ausleihen entsprechen dem Filter.
		</div>
	{/if}
</div>
