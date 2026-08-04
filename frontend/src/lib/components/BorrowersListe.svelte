<!-- @component BorrowersListe — wer dieses Buch gerade hat, eine Zeile je Exemplar.
     Ein Klick auf den Namen legt den Schüler-Barcode in den Scan-Kanal und springt
     zurück: derselbe Weg wie ein Scan am Pult. -->
<script>
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
	<ul class="divide-y divide-slate-50">
		{#each zeilen as b, _i (_i)}
			<li
				class="px-5 py-3.5 hover:bg-slate-50 transition-colors flex items-center justify-between group"
			>
				<div class="flex items-center gap-3 min-w-0">
					<div
						class="w-9 h-9 rounded-full bg-indigo-50 text-indigo-600 flex items-center justify-center font-bold text-xs shrink-0"
					>
						{b.schueler_name?.[0] ?? ''}{b.schueler_nachname?.[0] ?? ''}
					</div>
					<div class="min-w-0">
						<button
							onclick={() => {
								appState.triggerStudentScan = b.schueler_barcode;
								onBack();
							}}
							class="text-sm font-semibold text-slate-800 hover:text-indigo-600 text-left cursor-pointer truncate block"
						>
							{b.schueler_name}
							{b.schueler_nachname}
							<span class="text-xs font-normal text-slate-400 ml-1"
								>({b.klasse || 'Unbekannt'})</span
							>
						</button>
						<p class="text-xs text-slate-400 font-mono mt-0.5">Exemplar: {b.exemplar_barcode}</p>
					</div>
				</div>
				<div class="text-right shrink-0 ml-4 flex gap-6 items-center">
					<div class="text-right hidden sm:block">
						<p class="text-[10px] font-medium text-slate-400">Ausgeliehen</p>
						<p class="text-sm font-semibold text-slate-600">
							{fmtDate(b.ausgeliehen_am)}
						</p>
					</div>
					<div class="text-right">
						<p class="text-[10px] font-medium text-slate-400">Rückgabe bis</p>
						<p
							class="text-sm font-bold {new Date(b.rueckgabe_frist) < new Date()
								? 'text-rose-600'
								: 'text-slate-700'}"
						>
							{fmtDate(b.rueckgabe_frist)}
						</p>
					</div>
				</div>
			</li>
		{/each}
	</ul>
	{#if zeilen.length === 0}
		<div class="py-8 text-center text-sm text-slate-400">
			Keine Ausleihen entsprechen dem Filter.
		</div>
	{/if}
</div>
