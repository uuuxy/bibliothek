<!--
  @component Kalender
  Der aufgeklappte Kalender des Datumsfelds, M3 Docked date picker: Monat und Jahr je mit Pfeilen
  und Menü, darunter die Tage, unten Abbrechen und OK. Rollen und Tastatur wie in Googles Angular
  Material (datepicker): ein Dialog mit einem Raster (role="grid", aus <div>, weil Tabellen hier
  aus ui/Tabelle kommen); Pfeile, Pos1, Ende, Bild auf und ab, mit Umschalt oder Alt ein Jahr.
  Enter übernimmt den Tag, M3 (Date pickers, Accessibility): „Closes the calendar and saves the
  selected date“.
-->
<script>
	import { onMount, tick, untrack } from 'svelte';
	import Button from './Button.svelte';
	import KalenderKopf from './KalenderKopf.svelte';
	import KalenderListe from './KalenderListe.svelte';
	import { MONATE, WOCHENTAGE, kalenderBlatt, langesDatum } from '../../utils/datum.js';
	import { lastOfMonth, localISO, verschiebe } from '../../utils/datum.js';

	/** @type {{ wert: string, box: { left: number, top: number }, kennung: string,
	 *   ausloeser: HTMLElement | undefined, onuebernehmen: (iso: string) => void,
	 *   onschliessen: (zurueck: boolean) => void }} */
	let { wert, box, kennung, ausloeser, onuebernehmen, onschliessen } = $props();

	const heute = localISO(new Date());
	// Der Kalender beginnt bei dem Wert, mit dem er aufgeht; er geht für jedes Öffnen neu auf.
	const anfang = untrack(() => wert);
	let gewaehlt = $state(anfang);
	let fokus = $state(anfang || heute);
	/** @type {'tage' | 'monate' | 'jahre'} */
	let ansicht = $state('tage');
	/** @type {HTMLElement | undefined} */
	let flaeche = $state();

	const jahr = $derived(Number(fokus.slice(0, 4)));
	const monat = $derived(Number(fokus.slice(5, 7)));
	const wochen = $derived.by(() => {
		const blatt = kalenderBlatt(jahr, monat);
		return Array.from({ length: 6 }, (_, i) => blatt.slice(i * 7, i * 7 + 7));
	});

	/** Der Fokus folgt dem markierten Tag, Monat oder Jahr der Ansicht. */
	async function fokussiere() {
		await tick();
		const ziel = /** @type {HTMLElement | null | undefined} */ (
			flaeche?.querySelector('[data-aktiv]')
		);
		ziel?.focus();
		ziel?.scrollIntoView?.({ block: 'center' });
	}

	/** @param {{ tage?: number, monate?: number, jahre?: number }} schritt */
	function gehe(schritt) {
		fokus = verschiebe(fokus, schritt);
		fokussiere();
	}

	/** @param {'monate' | 'jahre'} liste */
	function umschalten(liste) {
		ansicht = ansicht === liste ? 'tage' : liste;
		fokussiere();
	}

	/** @param {KeyboardEvent} e */
	function tagesTaste(e) {
		const jahrweise = e.shiftKey || e.altKey;
		/** @type {Record<string, { tage?: number, monate?: number, jahre?: number }>} */
		const schritte = {
			ArrowLeft: { tage: -1 },
			ArrowRight: { tage: 1 },
			ArrowUp: { tage: -7 },
			ArrowDown: { tage: 7 },
			PageUp: jahrweise ? { jahre: -1 } : { monate: -1 },
			PageDown: jahrweise ? { jahre: 1 } : { monate: 1 }
		};
		if (schritte[e.key]) gehe(schritte[e.key]);
		else if (e.key === 'Home') gehe({ tage: 1 - Number(fokus.slice(8)) });
		else if (e.key === 'End')
			gehe({ tage: Number(lastOfMonth(fokus.slice(0, 7)).slice(8)) - Number(fokus.slice(8)) });
		else if (e.key === 'Enter') onuebernehmen(fokus);
		else if (e.key === ' ') gewaehlt = fokus;
		else return;
		e.preventDefault();
	}

	/** @param {KeyboardEvent} e */
	function dialogTaste(e) {
		if (e.key === 'Escape') {
			e.preventDefault();
			e.stopPropagation();
			onschliessen(true);
		} else if (e.shiftKey && (e.key === 'M' || e.key === 'Y')) {
			// M3 (Date pickers, Accessibility): „Shift + M: Moves to the month list dropdown“.
			e.preventDefault();
			/** @type {HTMLElement | null | undefined} */ (
				flaeche?.querySelector(`[data-menue="${e.key === 'M' ? 'monate' : 'jahre'}"]`)
			)?.focus();
		}
	}

	/**
	 * Googles Token des Docked date picker: Ein Tag außerhalb des Monats kennt nur den Zustand
	 * „unselected outside-month“, on-surface bei 38 %. Er ist nicht wählbar, und als inaktives
	 * Bedienelement fällt er nicht unter die Kontrastregel (WCAG 1.4.3).
	 * @param {string} tag @param {boolean} imMonat
	 */
	function farbe(tag, imMonat) {
		if (!imMonat) return 'text-on-surface/38';
		if (tag === gewaehlt) return 'bg-primary text-on-primary';
		return tag === heute ? 'border border-primary text-primary' : 'text-on-surface';
	}

	onMount(() => {
		fokussiere();
		// Ein Klick neben den Kalender schließt ihn wie ein Menü; der Fokus bleibt, wo geklickt wurde.
		/** @param {PointerEvent} e */
		const daneben = (e) => {
			const ziel = /** @type {Node} */ (e.target);
			if (!flaeche?.contains(ziel) && !ausloeser?.contains(ziel)) onschliessen(false);
		};
		window.addEventListener('pointerdown', daneben, true);
		return () => window.removeEventListener('pointerdown', daneben, true);
	});
</script>

<div
	bind:this={flaeche}
	id={kennung}
	role="dialog"
	aria-label="Datum wählen"
	tabindex="-1"
	style="position:fixed; left:{box.left}px; top:{box.top}px; z-index:60;"
	class="w-90 rounded-2xl bg-surface-container-high shadow-level3"
	onkeydown={dialogTaste}
	onfocusout={(e) => {
		const nach = /** @type {Node | null} */ (e.relatedTarget);
		if (nach && !flaeche?.contains(nach) && nach !== ausloeser) onschliessen(false);
	}}
>
	<KalenderKopf {jahr} {monat} {ansicht} ongehe={gehe} onumschalten={umschalten} />

	{#if ansicht === 'tage'}
		<div
			role="grid"
			tabindex="-1"
			aria-label="{MONATE[monat - 1]} {jahr}"
			class="mx-3 my-1"
			onkeydown={tagesTaste}
		>
			<div role="row" class="flex">
				{#each WOCHENTAGE as w (w.kurz)}
					<div
						role="columnheader"
						aria-label={w.lang}
						data-tip={w.lang}
						class="grid size-12 place-items-center text-base text-on-surface"
					>
						{w.kurz}
					</div>
				{/each}
			</div>
			{#each wochen as woche (woche[0])}
				<div role="row" class="flex">
					{#each woche as tag (tag)}
						{@const imMonat = tag.slice(0, 7) === fokus.slice(0, 7)}
						<div
							role="gridcell"
							class="grid size-12 place-items-center"
							aria-selected={imMonat && tag === gewaehlt}
						>
							<button
								type="button"
								data-tag={tag}
								data-aktiv={tag === fokus || undefined}
								tabindex={tag === fokus ? 0 : -1}
								disabled={!imMonat}
								aria-label={langesDatum(tag)}
								aria-current={imMonat && tag === heute ? 'date' : undefined}
								class="relative size-10 rounded-full text-base {farbe(tag, imMonat)}"
								onclick={() => (gewaehlt = fokus = tag)}>{Number(tag.slice(8))}</button
							>
						</div>
					{/each}
				</div>
			{/each}
		</div>
	{:else}
		<KalenderListe
			art={ansicht}
			aktuell={ansicht === 'monate' ? monat : jahr}
			onwahl={(zahl) => {
				gehe(ansicht === 'monate' ? { monate: zahl - monat } : { jahre: zahl - jahr });
				ansicht = 'tage';
			}}
		/>
	{/if}

	<div class="flex justify-end gap-2 px-3 pt-1 pb-3">
		<Button variant="ghost" onclick={() => onschliessen(true)}>Abbrechen</Button>
		<Button variant="ghost" disabled={!gewaehlt} onclick={() => onuebernehmen(gewaehlt)}>OK</Button>
	</div>
</div>
