<!--
  @component
  ActiveStudentList

  Diese Komponente rendert die Liste der aktiven Leser mit Filter- und Sortierfunktionen.
  Sie zeigt ein Profilbild, den Namen, die Art, die Klasse, die Anzahl der ausgeliehenen
  Bücher und den Status an.

  Die Spalte „Art" kam am 16.09.2026 dazu: Seit Migration 125 steht das Kollegium in
  derselben Liste, und ein Kollege hat keine Klasse. Ohne die Art stünde er da wie ein
  Schüler, bei dem die Klasse fehlt.
-->
<script>
	import { BookOpen, ChevronRight } from '@lucide/svelte';
	import Tabelle from '../ui/Tabelle.svelte';
	import Ladekreis from '../ui/Ladekreis.svelte';
	import Kaestchen from '../ui/Kaestchen.svelte';
	import LadeFehler from '../ui/LadeFehler.svelte';
	import { ausleiheGesperrt } from '../../sperrStatus.js';
	import LeserAvatar from './LeserAvatar.svelte';
	import { leserArtText, istKollegium } from '../../leserArt.js';

	/**
	 * @typedef {Object} Props
	 * @property {any[]} filteredStudents
	 * @property {boolean} loading
	 * @property {string} [ladefehler]  gescheiterter Abruf — dann steht hier kein leeres
	 *   Verzeichnis, sondern der Grund (die Suche der Leserdatei setzt ihn)
	 * @property {() => void} [onErneut]
	 * @property {(s: any) => void} onSelectStudent
	 * @property {Set<string>} [auswahl]    markierte Leser-IDs (Ausweis-Stapeldruck)
	 * @property {(id: string) => void} [onToggle]
	 * @property {() => void} [onToggleAlle]
	 */
	/** @type {Props} */
	let {
		filteredStudents = [],
		loading = false,
		ladefehler = '',
		onErneut,
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

{#snippet statusBadge(s)}
	<div class="inline-flex items-center justify-end gap-1.5 py-1">
		{#if s.ueberfaellig_count > 0}
			<span class="w-1.5 h-1.5 rounded-full bg-rose-500 animate-pulse" aria-hidden="true"></span>
			<span class="text-sm font-semibold text-rose-600">Überfällig</span>
		{:else if ausleiheGesperrt(s)}
			<span class="w-1.5 h-1.5 rounded-full bg-amber-500" aria-hidden="true"></span>
			<span class="text-sm font-semibold text-amber-600">Gesperrt</span>
		{:else}
			<span class="w-1.5 h-1.5 rounded-full bg-emerald-500" aria-hidden="true"></span>
			<span class="text-sm font-semibold text-emerald-600">Alles ok</span>
		{/if}
	</div>
{/snippet}

<div class="w-full">
	{#if loading}
		<div class="py-16 flex justify-center items-center">
			<Ladekreis size="lg" />
		</div>
	{:else if ladefehler}
		<LadeFehler onerneut={onErneut} titel="Verzeichnis nicht geladen" text={ladefehler} />
	{:else if filteredStudents.length === 0}
		<div class="py-16 flex flex-col items-center justify-center text-slate-400 space-y-2">
			<BookOpen class="h-10 w-10 text-slate-300" aria-hidden="true" />
			<span class="text-xs font-semibold">Keine Leser im Verzeichnis gefunden.</span>
		</div>
	{:else}
		<div class="overflow-x-auto w-full text-left">
			<Tabelle beschriftung="Leserinnen und Leser">
				<thead class="font-semibold">
					<tr>
						{#if auswaehlbar}
							<th class="w-10">
								<Kaestchen
									checked={alleGewaehlt}
									indeterminate={teilweise}
									onchange={onToggleAlle}
									aria-label="Alle angezeigten Leser für den Ausweisdruck markieren"
								/>
							</th>
						{/if}
						<th class="w-16">Foto</th>
						<th>Name</th>
						<th class="w-28">Art</th>
						<th class="w-24">Klasse</th>
						<th class="w-44 text-right">Geliehene Bücher</th>
						<th class="w-36 text-right">Status</th>
						<th class="w-10"></th>
					</tr>
				</thead>
				<tbody>
					{#each filteredStudents as s, _i (_i)}
						<!-- Die ZEILE ist kein Knopf mehr, der NAME ist es (wie im Mahnwesen). Eine Zeile
						     mit role="button", in der ein Kästchen steckt, ist ein Bedienelement im
						     Bedienelement — für Screenreader unlesbar, für axe 273 Verstöße
						     (nested-interactive, 09.09.2026). Das Kästchen braucht damit auch kein
						     stopPropagation mehr: Ankreuzen öffnet nichts. -->
						<tr class="group">
							{#if auswaehlbar}
								<td>
									<Kaestchen
										checked={auswahl.has(s.id)}
										onchange={() => onToggle?.(s.id)}
										aria-label="{s.vorname} {s.nachname} für den Ausweisdruck markieren"
									/>
								</td>
							{/if}
							<td><LeserAvatar leser={s} /></td>
							<td class="font-semibold">
								<button
									type="button"
									onclick={() => onSelectStudent(s)}
									aria-label="Profil von {s.vorname} {s.nachname} ({istKollegium(s)
										? leserArtText(s.art)
										: `Klasse ${s.klasse || 'N/A'}`}) anzeigen"
									class="text-left font-semibold text-on-surface hover:text-primary hover:underline cursor-pointer rounded focus-visible:outline-2 focus-visible:outline-primary"
								>
									{s.vorname}
									{s.nachname}
								</button>
								<!-- Ein Kollege aus der Selbstanmeldung hat noch keine Ausweisnummer. Das
								     gehört hingeschrieben: Eine leere Zeile sähe nach einem Anzeigefehler
								     aus, und ohne Nummer lässt sich kein Ausweis drucken. -->
								<div
									class="text-sm text-slate-400 font-normal mt-0.5 {s.barcode_id
										? 'font-mono'
										: 'italic'}"
								>
									{s.barcode_id || 'ohne Ausweis'}
								</div>
							</td>
							<td>{leserArtText(s.art)}</td>
							<td class="font-medium">
								{#if istKollegium(s)}
									<span class="text-on-surface-variant" aria-label="keine Klasse">—</span>
								{:else}
									Kl. {s.klasse || 'N/A'}
								{/if}
							</td>
							<td class="text-right">
								<span
									class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-bold {s.ausgeliehen_count >
									0
										? 'bg-blue-50 text-blue-700'
										: 'bg-slate-100 text-slate-500'}"
								>
									{s.ausgeliehen_count || 0}
								</span>
							</td>
							<td class="text-right">
								{@render statusBadge(s)}
							</td>
							<td class="text-right">
								<ChevronRight
									class="w-4 h-4 text-slate-300 opacity-0 group-hover:opacity-100 transition-opacity ml-auto"
									aria-hidden="true"
								/>
							</td>
						</tr>
					{/each}
				</tbody>
			</Tabelle>
		</div>
	{/if}
</div>
