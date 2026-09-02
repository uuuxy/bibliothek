<!-- @component LusdUmbenennungen — die Paare „Abgänger ↔ Neuzugang", die nach Geburtsdatum,
     Schuleintritt, Klasse und Anschrift dieselbe Person sind (Backend: api/lusd_paarung.go).

     Der Export der Schule hat keine Schüler-ID; eine Namensänderung oder Datumskorrektur
     in der LUSD ergäbe sonst Abgänger + Neuanlage. Der Admin kreuzt an, welche Paare
     stimmen: bestätigt heißt, der bestehende Datensatz (Ausweis, Bücher, Historie) bekommt
     den neuen Namen. „sicher" (Schuleintritt trifft) ist vorangekreuzt, „vermutlich" nicht. -->
<script>
	import { ArrowRight, CircleCheck, UserRoundPen } from '@lucide/svelte';
	import { SvelteSet } from 'svelte/reactivity';

	/**
	 * @type {{
	 *   paare: import('./lusdVorschauRubriken.js').UmbenennungDiff[],
	 *   gewaehlt: SvelteSet<number>,
	 *   abgeschlossen?: boolean
	 * }}
	 */
	let { paare, gewaehlt = new SvelteSet(), abgeschlossen = false } = $props();

	/** @param {number} zeile */
	function umschalten(zeile) {
		if (gewaehlt.has(zeile)) gewaehlt.delete(zeile);
		else gewaehlt.add(zeile);
	}

	/** @param {string | undefined} iso */
	function datum(iso) {
		if (!iso) return '';
		const [j, m, t] = iso.split('-');
		return `${t}.${m}.${j}`;
	}

	const bestaetigte = $derived(paare.filter((p) => p.bestaetigt).length);
</script>

{#if paare.length > 0}
	<details class="group py-1" open>
		<summary
			class="flex items-center justify-between py-3 cursor-pointer select-none marker:content-none [&::-webkit-details-marker]:hidden"
		>
			<div class="min-w-0 flex items-center gap-2">
				<UserRoundPen class="w-4 h-4 text-primary shrink-0" aria-hidden="true" />
				<div class="min-w-0">
					<p class="text-sm font-bold text-on-surface">Vermutlich dieselbe Person</p>
					<p class="text-xs text-on-surface-variant mt-0.5">
						{#if abgeschlossen}
							{bestaetigte} von {paare.length} Paaren bestätigt — bestätigte Schüler behielten ihren Datensatz,
							die übrigen wurden als Abgänger + Neuzugang behandelt
						{:else}
							Ein Abgänger und ein Neuzugang, die nach Geburtsdatum, Schuleintritt, Klasse oder
							Anschrift zusammengehören (Namensänderung oder Datumskorrektur in der LUSD).
							Angekreuzt heißt: derselbe Datensatz bekommt den neuen Namen, Ausweis und Bücher
							bleiben. Nicht angekreuzt: Abgänger + Neuanlage wie bisher.
						{/if}
					</p>
				</div>
			</div>
			<span class="text-lg font-black tabular-nums shrink-0 ml-4 text-primary">{paare.length}</span>
		</summary>
		<ul class="divide-y divide-outline-variant/40 pb-2">
			{#each paare as p (p.zeile)}
				<li class="py-2 pl-5 text-xs">
					<label class="flex items-start gap-3 cursor-pointer">
						{#if abgeschlossen}
							<span class="mt-0.5 w-4 h-4 shrink-0 text-primary" aria-hidden="true">
								{#if p.bestaetigt}<CircleCheck class="w-4 h-4" />{/if}
							</span>
						{:else}
							<input
								type="checkbox"
								class="mt-0.5 h-4 w-4 accent-primary shrink-0"
								checked={gewaehlt.has(p.zeile)}
								onchange={() => umschalten(p.zeile)}
								aria-label="Paar bestätigen: {p.alt_vorname} {p.alt_nachname} ist {p.neu_vorname} {p.neu_nachname}"
							/>
						{/if}
						<span class="min-w-0 flex-1 space-y-0.5">
							<span class="flex flex-wrap items-center gap-x-2 gap-y-0.5">
								<span class="font-semibold text-on-surface"
									>{p.alt_vorname}
									{p.alt_nachname}
									<span class="font-mono text-on-surface-variant">{p.alt_klasse}</span></span
								>
								<ArrowRight class="w-3 h-3 text-on-surface-variant shrink-0" aria-hidden="true" />
								<span class="font-semibold text-on-surface"
									>{p.neu_vorname}
									{p.neu_nachname}
									<span class="font-mono text-on-surface-variant">{p.neu_klasse}</span></span
								>
								{#if p.sicher}
									<span
										class="rounded-full bg-primary-container text-on-primary-container px-2 py-0.5 text-label-small font-semibold"
										>sicher</span
									>
								{:else}
									<span
										class="rounded-full bg-surface-container-high text-on-surface-variant px-2 py-0.5 text-label-small font-semibold"
										>vermutlich</span
									>
								{/if}
								{#if p.war_abgaenger}
									<span
										class="rounded-full bg-surface-container-high text-on-surface-variant px-2 py-0.5 text-label-small font-semibold"
										>früherer Abgänger</span
									>
								{/if}
								{#if abgeschlossen && p.bestaetigt}
									<span class="text-primary font-semibold">bestätigt</span>
								{/if}
							</span>
							<span class="block text-on-surface-variant">
								{p.grund}{#if p.alt_geburtsdatum && p.neu_geburtsdatum && p.alt_geburtsdatum !== p.neu_geburtsdatum}
									· Geburtsdatum {datum(p.alt_geburtsdatum)} → {datum(p.neu_geburtsdatum)}{/if}
							</span>
						</span>
					</label>
				</li>
			{/each}
		</ul>
	</details>
{/if}
