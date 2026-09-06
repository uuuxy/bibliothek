<!-- @component LmfPlanKlassenZelle — die Klassen einer Planer-Zeile. Eine Klasse steht
     als Text (M3: „Don't display a single chip by itself"), ab zwei Klassen (geteilte
     Stunde) als Input-Chips mit × je Chip; ein Status-Chip „ohne Schüler" hängt an der
     Klasse. Seit dem 06.09.2026 lässt sich jede Klasse anklicken und tauschen — die
     Zelle überschreiben wie im Excel: Der Klick zeigt an ihrer Stelle das Auswahlfeld
     mit der Klasse selbst und allen aus „Noch nicht im Plan" (offen und eingeklappt),
     schon geöffnet. Wählen tauscht (die alte Klasse geht zurück in den Vorrat, die
     Zeile behält Platz und Vermerk); Escape oder Fokus woandershin lässt alles, wie
     es war. Das Feld steht nur während des Tauschs da — sechzig Auswahlfelder auf der
     Seite wären kein Plan mehr, sondern ein Formular. -->
<script>
	import { tick } from 'svelte';
	import Select from '../ui/Select.svelte';
	import StatusChip from '../ui/StatusChip.svelte';
	import LmfKlasseChip from './LmfKlasseChip.svelte';

	/** @type {{ klassen: string[], i: number, vorrat: string[], marker: ReturnType<typeof import('../../lmfplanDienst.js').klassenMarker>, onklasseraus: (klasse: string) => void, ontausch: (alt: string, neu: string) => void }} */
	let { klassen, i, vorrat, marker, onklasseraus, ontausch } = $props();

	const OHNE_SCHUELER_TIP =
		'Noch kein Schüler in dieser Klasse — sie kommt mit dem LUSD-Import oder gehört aus dem Plan';

	/** Die Klasse, die gerade getauscht wird — oder null. @type {string | null} */
	let tauscht = $state(null);
	/** @type {{ oeffnen: () => void } | undefined} */
	let feld = $state();
	/** @type {HTMLTableCellElement | undefined} */
	let zelle = $state();

	const optionen = $derived(
		tauscht === null
			? []
			: [
					{ value: tauscht, label: tauscht },
					...vorrat
						.filter((k) => !klassen.includes(k))
						.map((k) => ({
							value: k,
							label: marker.ohneSchueler(k) ? `${k} · ohne Schüler` : k
						}))
				]
	);

	/** @param {string} k */
	async function beginnen(k) {
		tauscht = k;
		await tick();
		document.getElementById(`lmf-zeile-klasse-${i}`)?.focus();
		feld?.oeffnen();
	}

	/** @param {string} neu */
	function waehlen(neu) {
		const alt = tauscht;
		tauscht = null;
		if (alt !== null && neu !== alt) ontausch(alt, neu);
	}

	/** Fokus wandert per Tab aus der Zelle: Tausch abbrechen. Ohne Ziel (relatedTarget
	 *  null) nicht — so meldet Chrome auch den Knopf, den der Klick gerade ersetzt hat;
	 *  das brach den Tausch ab, bevor das Feld stand (E2E 06.09.2026). Klicks woandershin
	 *  fängt der pointerdown am Fenster.
	 *  @param {FocusEvent} e */
	function fokusWeg(e) {
		const ziel = /** @type {Node | null} */ (e.relatedTarget);
		if (!ziel || zelle?.contains(ziel)) return;
		tauscht = null;
	}

	/** @param {PointerEvent} e */
	function klickWoanders(e) {
		if (tauscht === null || zelle?.contains(/** @type {Node} */ (e.target))) return;
		tauscht = null;
	}
</script>

<svelte:window onpointerdown={klickWoanders} />

<td
	bind:this={zelle}
	class="px-4 py-1"
	onfocusout={tauscht === null ? undefined : fokusWeg}
	onkeydown={(e) => {
		if (e.key === 'Escape' && tauscht !== null) {
			e.stopPropagation();
			tauscht = null;
		}
	}}
>
	{#if klassen.length === 1}
		{#if tauscht === klassen[0]}
			<Select
				bind:this={feld}
				id="lmf-zeile-klasse-{i}"
				aria-label="Klasse Zeile {i + 1} tauschen"
				value={tauscht}
				options={optionen}
				onchange={waehlen}
				class="w-44"
			/>
		{:else}
			<span class="inline-flex items-center gap-2">
				<button
					type="button"
					class="-mx-2 h-8 cursor-pointer rounded-md px-2 font-medium text-on-surface hover:bg-surface-container"
					title="{klassen[0]} gegen eine andere Klasse tauschen"
					onclick={() => beginnen(klassen[0])}>{klassen[0]}</button
				>
				{#if marker.ohneSchueler(klassen[0])}
					<StatusChip ton="warten" text="ohne Schüler" tip={OHNE_SCHUELER_TIP} />
				{/if}
			</span>
		{/if}
	{:else if klassen.length > 1}
		<div class="flex flex-wrap gap-1">
			{#each klassen as k (k)}
				{#if tauscht === k}
					<Select
						bind:this={feld}
						id="lmf-zeile-klasse-{i}"
						aria-label="Klasse Zeile {i + 1} tauschen"
						value={tauscht}
						options={optionen}
						onchange={waehlen}
						class="w-44"
					/>
				{:else}
					<LmfKlasseChip
						name={k}
						hinweis={marker.ohneSchueler(k) ? 'ohne Schüler' : ''}
						onklick={() => beginnen(k)}
						onentfernen={() => onklasseraus(k)}
					/>
				{/if}
			{/each}
		</div>
	{/if}
</td>
