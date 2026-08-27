<!-- @component MahnwesenTabs — Register nach Dringlichkeit, mit den Zahlen dahinter.

     Die vier Register waren viermal derselbe Block mit vertauschten Wörtern; hier
     stehen sie als Liste, damit ein fünftes Register eine Zeile kostet und nicht
     zwanzig. Die Zählungen wohnen bei den Registern, weil sie sonst nirgends
     gebraucht werden. -->
<script>
	import { mahnwesenStore } from '../../stores/mahnwesen.svelte.js';

	/** Höchste Überfälligkeit eines Schülers in Tagen. @param {any} s */
	const maxTage = (s) =>
		s.medien.reduce(
			(/** @type {number} */ max, /** @type {any} */ m) =>
				m.tage_ueberfaellig > max ? m.tage_ueberfaellig : max,
			0
		);
	/** @param {(s: any) => boolean} passt */
	const zaehle = (passt) =>
		mahnwesenStore.klassen.reduce(
			(/** @type {number} */ sum, /** @type {any} */ k) => sum + k.schueler.filter(passt).length,
			0
		);

	// „Akut fällig" = überfällig bis 14 Tage (inkl. der <24h-Fälle mit maxTage 0),
	// passend zur Mahnstufe '1. Erinnerung'. So stimmt die Register-Zahl mit der Liste.
	const register = $derived([
		{ filter: 'Alle', titel: 'Alle', anzahl: zaehle(() => true) },
		{
			filter: '1. Erinnerung',
			titel: 'Akut fällig',
			anzahl: zaehle((s) => maxTage(s) <= 14)
		},
		{
			filter: 'Mahnung',
			titel: 'Eskaliert',
			anzahl: zaehle((s) => maxTage(s) > 14)
		}
	]);
	// Das Register „Kollegium" (klasse='lehrer') ist mit Migration 072 gefallen:
	// Lehrkräfte sind Personal-Konten, ihre Handapparat-Ausleihen laufen bewusst
	// ohne Mahn-Eskalation (1 Jahr Frist) — siehe Befund F4, bewertung/.
</script>

<div
	role="tablist"
	aria-label="Filter für Mahnstufen"
	class="flex space-x-1 border-b border-slate-200 mt-6 print:hidden"
>
	{#each register as tab (tab.filter)}
		{@const aktiv = mahnwesenStore.activeFilter === tab.filter}
		<button
			role="tab"
			aria-selected={aktiv}
			class="flex items-center px-4 py-2 text-sm font-medium transition-colors {aktiv
				? 'border-b-2 border-blue-600 text-blue-600'
				: 'text-slate-600 hover:text-slate-900 hover:bg-slate-50'}"
			onclick={() => (mahnwesenStore.activeFilter = tab.filter)}
		>
			{tab.titel}
			<span
				class="ml-2 py-0.5 px-2 rounded-full text-xs font-bold {aktiv && tab.anzahl > 0
					? 'bg-blue-100 text-blue-600'
					: 'bg-slate-100 text-slate-600'}"
			>
				{tab.anzahl}
			</span>
		</button>
	{/each}
</div>
