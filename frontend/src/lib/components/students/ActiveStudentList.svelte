<!--
  @component
  ActiveStudentList

  Diese Komponente rendert die Liste der aktiven Schüler mit Filter- und Sortierfunktionen.
  Sie zeigt ein Profilbild, den Namen, die Klasse, die Anzahl der ausgeliehenen Bücher und den Status an.
-->
<script>
	import Sheet from '../layout/Sheet.svelte';
	import { BookOpen, ChevronRight } from '@lucide/svelte';

	/**
	 * @typedef {Object} Props
	 * @property {any[]} filteredStudents
	 * @property {boolean} loading
	 * @property {(s: any) => void} onSelectStudent
	 * @property {Set<string>} [auswahl]    markierte Schüler-IDs (Ausweis-Stapeldruck)
	 * @property {(id: string) => void} [onToggle]
	 * @property {() => void} [onToggleAlle]
	 */
	/** @type {Props} */
	let {
		filteredStudents = [],
		loading = false,
		onSelectStudent = () => {},
		auswahl = new Set(),
		onToggle,
		onToggleAlle
	} = $props();

	// Die Auswahlspalte erscheint nur, wenn der Aufrufer sie auch verarbeitet. So bleibt
	// die Liste woanders (Kiosk, Abgänger) unverändert schmal.
	const auswaehlbar = $derived(typeof onToggle === 'function');
	const alleGewaehlt = $derived(
		filteredStudents.length > 0 &&
			filteredStudents.every((/** @type {any} */ s) => auswahl.has(s.id))
	);
	const teilweise = $derived(auswahl.size > 0 && !alleGewaehlt);
</script>

{#snippet avatar(s)}
	<div
		class="relative w-8 h-8 rounded-full overflow-hidden border border-slate-100/80 bg-slate-50 flex items-center justify-center shrink-0"
	>
		{#if s.foto_url}
			<img
				src={s.foto_url}
				alt="Passbild von {s.vorname} {s.nachname}"
				class="w-full h-full object-cover"
			/>
		{:else}
			<div
				class="w-full h-full flex items-center justify-center bg-slate-100 text-slate-500 font-medium text-xs"
				aria-hidden="true"
			>
				{s.vorname.charAt(0)}{s.nachname.charAt(0)}
			</div>
		{/if}
	</div>
{/snippet}

{#snippet statusBadge(s)}
	<div class="inline-flex items-center justify-end gap-1.5 py-1">
		{#if s.ueberfaellig_count > 0}
			<span class="w-1.5 h-1.5 rounded-full bg-rose-500 animate-pulse" aria-hidden="true"></span>
			<span class="text-xs font-semibold text-rose-600">Überfällig</span>
		{:else if s.ist_gesperrt}
			<span class="w-1.5 h-1.5 rounded-full bg-amber-500" aria-hidden="true"></span>
			<span class="text-xs font-semibold text-amber-600">Gesperrt</span>
		{:else}
			<span class="w-1.5 h-1.5 rounded-full bg-emerald-500" aria-hidden="true"></span>
			<span class="text-xs font-semibold text-emerald-600">Alles ok</span>
		{/if}
	</div>
{/snippet}

<Sheet>
	{#if loading}
		<div class="py-16 flex justify-center items-center">
			<div
				class="w-8 h-8 border-4 border-t-blue-600 border-slate-200 rounded-full animate-spin"
				aria-hidden="true"
			></div>
		</div>
	{:else if filteredStudents.length === 0}
		<div class="py-16 flex flex-col items-center justify-center text-slate-400 space-y-2">
			<BookOpen class="h-10 w-10 text-slate-300" aria-hidden="true" />
			<span class="text-xs font-semibold">Keine Schüler im Verzeichnis gefunden.</span>
		</div>
	{:else}
		<div class="overflow-x-auto w-full text-left">
			<table class="w-full text-base text-slate-700">
				<thead class="border-b border-slate-200 text-sm font-semibold text-slate-500 font-sans">
					<tr>
						{#if auswaehlbar}
							<th class="px-4 py-2 w-10">
								<input
									type="checkbox"
									checked={alleGewaehlt}
									indeterminate={teilweise}
									onchange={onToggleAlle}
									aria-label="Alle angezeigten Schüler für den Ausweisdruck markieren"
									class="h-4 w-4 cursor-pointer rounded border-slate-300 accent-blue-600"
								/>
							</th>
						{/if}
						<th class="px-4 py-2 w-16">Foto</th>
						<th class="px-4 py-2">Name</th>
						<th class="px-4 py-2 w-24">Klasse</th>
						<th class="px-4 py-2 w-44 text-right">Geliehene Bücher</th>
						<th class="px-4 py-2 w-36 text-right">Status</th>
						<th class="px-4 py-2 w-10"></th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-100">
					{#each filteredStudents as s, _i (_i)}
						<tr
							onclick={() => onSelectStudent(s)}
							onkeydown={(e) => {
								if (e.key === 'Enter' || e.key === ' ') {
									e.preventDefault();
									onSelectStudent(s);
								}
							}}
							tabindex="0"
							role="button"
							aria-label="Profil von {s.vorname} {s.nachname} (Klasse {s.klasse || 'N/A'}) anzeigen"
							class="hover:bg-slate-50/50 cursor-pointer transition-colors group focus-visible:outline-2 focus-visible:outline-blue-600 focus-visible:-outline-offset-2"
						>
							{#if auswaehlbar}
								<!-- stopPropagation: Die gesamte Zeile oeffnet das Profil. Ohne das
								     wuerde jedes Ankreuzen den Bildschirm wechseln — und die
								     Markierung waere weg, bevor man die zweite setzen kann. -->
								<td class="px-4 py-2" onclick={(e) => e.stopPropagation()}>
									<input
										type="checkbox"
										checked={auswahl.has(s.id)}
										onchange={() => onToggle?.(s.id)}
										aria-label="{s.vorname} {s.nachname} für den Ausweisdruck markieren"
										class="h-4 w-4 cursor-pointer rounded border-slate-300 accent-blue-600"
									/>
								</td>
							{/if}
							<td class="px-4 py-2">
								{@render avatar(s)}
							</td>
							<td class="px-4 py-2 font-semibold text-slate-800">
								{s.vorname}
								{s.nachname}
								<div class="text-label-small text-slate-400 font-normal mt-0.5">{s.barcode_id}</div>
							</td>
							<td class="px-4 py-2 font-medium text-slate-600">
								Kl. {s.klasse || 'N/A'}
							</td>
							<td class="px-4 py-2 text-right">
								<span
									class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-bold {s.ausgeliehen_count >
									0
										? 'bg-blue-50 text-blue-700'
										: 'bg-slate-100 text-slate-500'}"
								>
									{s.ausgeliehen_count || 0}
								</span>
							</td>
							<td class="px-4 py-2 text-right">
								{@render statusBadge(s)}
							</td>
							<td class="px-4 py-2 text-right">
								<ChevronRight
									class="w-4 h-4 text-slate-300 opacity-0 group-hover:opacity-100 transition-opacity ml-auto"
									aria-hidden="true"
								/>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</Sheet>
