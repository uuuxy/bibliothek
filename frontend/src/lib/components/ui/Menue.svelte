<!-- @component Menue — das Material-3-Menü hinter einem „Mehr"-Knopf.

     M3 (Menus, Guidelines): „Use a menu to show a temporary set of actions. To show
     actions on screen at all times, use a toolbar instead" — und Überlaufmenüs sind
     der genannte Fall. Maße nach M3 Specs: Fläche surface-container mit 4-px-Ecken,
     Einträge 48 px hoch mit 12 px Innenabstand, führendes Symbol 24 dp (hier die
     Symbolgröße des Hauses), Breite zwischen 112 und 280 dp. Dieselbe Fläche wie die
     Liste des Auswahlfelds (SelectListe.svelte).

     Bis zum 06.09.2026 gab es zwei Menüs im Haus, jedes selbst gebaut
     (MahnwesenDruckMenue, StudentProfileActions) — beide in Paletten-Farben und ohne
     Tastaturbedienung der Einträge. Dieses Bauteil ist die eine Stelle; die beiden
     alten stehen im Befund-Register zum Umstellen.

     POSITION FIXED wie SelectListe: Das Menü öffnet sich in Tabellen und Spalten mit
     overflow, absolut positioniert würde es dort abgeschnitten.

     Tastatur: Pfeile wandern, Pos1/Ende springen, Enter und Leertaste wählen (der
     Eintrag ist ein <button>), Escape schließt und gibt den Fokus an den Knopf zurück
     (escapeSchliesst — nur das oberste Overlay reagiert), Tab verlässt das Menü.

     Seit 07.09.2026 tragen die beiden alten Menüs dieses Bauteil, und dafür kann es
     zweierlei mehr: einen eigenen Auslöser (`ausloeser`-Snippet — der Split-Button
     „Mahnbriefe ▾" bzw. „Ausweis drucken ▾"; der Chevron trägt aria-haspopup, dorthin
     kehrt der Fokus zurück) und einen Kopf über den Einträgen (`kopf`-Snippet — das
     Auswahlfeld „Ganze Klasse" im Mahnwesen). Mit Kopf steht die Höhe erst nach dem
     Rendern fest, deshalb wird gemessen statt gerechnet (menueGeometrie.js), und Tab
     wandert IN den Kopf statt das Menü zu schließen — geschlossen wird, wenn der Fokus
     das Menü verlässt. Gruppen-Überschriften: `ueberschriftDavor` am Eintrag. -->
<script>
	import { tick } from 'svelte';
	import { EllipsisVertical } from '@lucide/svelte';
	import Button from './Button.svelte';
	import { escapeSchliesst } from './escapeSchliesst.js';
	import { berechneMenueBox } from './menueGeometrie.js';

	/** @typedef {import('./menueGeometrie.js').Eintrag} Eintrag */
	/** @type {{ etikett: string, eintraege: Eintrag[], onwahl: (id: string) => void,
	 *   ausloeser?: import('svelte').Snippet<[{ offen: boolean, umschalten: () => void }]>,
	 *   kopf?: import('svelte').Snippet, ausrichtung?: 'rechts' | 'links', breite?: number }} */
	let {
		etikett,
		eintraege,
		onwahl,
		ausloeser,
		kopf,
		ausrichtung = 'rechts',
		breite = 256
	} = $props();

	/** Breite in px — innerhalb der M3-Spanne 112–280 dp. */
	const BREITE = $derived(Math.max(112, Math.min(280, breite)));

	let offen = $state(false);
	let aktiv = $state(0);
	/** Lage erst nach dem Messen — bis dahin unsichtbar gerendert. @type {{ left: number, top: number } | null} */
	let box = $state(null);
	/** @type {HTMLDivElement | undefined} */
	let anker = $state();
	/** @type {HTMLDivElement | undefined} */
	let flaeche = $state();

	async function oeffnen() {
		if (!anker) return;
		aktiv = Math.max(
			0,
			eintraege.findIndex((e) => !e.disabled)
		);
		box = null;
		offen = true;
		await tick();
		// Gemessen, nicht gerechnet: Der Kopf hat keine feste Zeilenhöhe.
		const hoehe = flaeche?.getBoundingClientRect().height ?? 0;
		box = berechneMenueBox(anker.getBoundingClientRect(), hoehe, BREITE, ausrichtung, {
			breite: window.innerWidth,
			hoehe: window.innerHeight
		});
		fokussiere();
	}

	/** @param {boolean} [fokusZurueck] */
	function schliessen(fokusZurueck = true) {
		offen = false;
		if (fokusZurueck)
			/** @type {HTMLElement | null} */ (anker?.querySelector('[aria-haspopup="menu"]'))?.focus();
	}

	/** Fokus verlässt das Menü (Tab aus dem letzten Element, Klick in ein fremdes Feld). @param {FocusEvent} e */
	function fokusWeg(e) {
		const ziel = /** @type {Node | null} */ (e.relatedTarget);
		if (ziel && !flaeche?.contains(ziel) && !anker?.contains(ziel)) schliessen(false);
	}

	/** @param {Eintrag} e */
	function waehlen(e) {
		if (e.disabled) return;
		schliessen();
		onwahl(e.id);
	}

	function fokussiere() {
		const knoepfe = flaeche?.querySelectorAll('[role="menuitem"]');
		/** @type {HTMLElement | undefined} */ (knoepfe?.[aktiv])?.focus();
	}

	/** @param {KeyboardEvent} e */
	function taste(e) {
		// Nur die Einträge wandern mit den Pfeilen — im Kopf gehören die Tasten dem
		// Auswahlfeld, das dort steht.
		const ziel = /** @type {HTMLElement} */ (e.target);
		if (ziel !== flaeche && !ziel.matches('[role="menuitem"]')) return;
		const n = eintraege.length;
		if (e.key === 'ArrowDown') aktiv = (aktiv + 1) % n;
		else if (e.key === 'ArrowUp') aktiv = (aktiv - 1 + n) % n;
		else if (e.key === 'Home') aktiv = 0;
		else if (e.key === 'End') aktiv = n - 1;
		else return;
		e.preventDefault();
		fokussiere();
	}

	// Klick außerhalb schließt — Zeiger statt Klick, damit auch ein Druck auf ein
	// anderes Bedienelement das Menü sofort zumacht.
	$effect(() => {
		if (!offen) return;
		/** @param {PointerEvent} ev */
		const beiZeiger = (ev) => {
			const ziel = /** @type {Node} */ (ev.target);
			if (!anker?.contains(ziel) && !flaeche?.contains(ziel)) schliessen(false);
		};
		document.addEventListener('pointerdown', beiZeiger);
		return () => document.removeEventListener('pointerdown', beiZeiger);
	});
</script>

<div class="inline-block" bind:this={anker}>
	{#if ausloeser}
		{@render ausloeser({ offen, umschalten: () => (offen ? schliessen() : oeffnen()) })}
	{:else}
		<Button
			variant="ghost"
			size="sm"
			aria-haspopup="menu"
			aria-expanded={offen}
			aria-label={etikett}
			title={etikett}
			onclick={() => (offen ? schliessen() : oeffnen())}
		>
			<EllipsisVertical class="h-4 w-4" aria-hidden="true" />
		</Button>
	{/if}
</div>

{#if offen}
	<div
		bind:this={flaeche}
		role="menu"
		aria-label={etikett}
		tabindex="-1"
		use:escapeSchliesst={() => schliessen()}
		onkeydown={taste}
		onfocusout={fokusWeg}
		style="position:fixed; left:{box?.left ?? 0}px; top:{box?.top ??
			0}px; width:{BREITE}px; z-index:60; visibility:{box ? 'visible' : 'hidden'};"
		class="rounded-sm bg-surface-container py-2 shadow-xl"
	>
		{#if kopf}
			<div class="px-3 pb-2">{@render kopf()}</div>
		{/if}
		{#each eintraege as e, i (e.id)}
			{#if e.trennerDavor}
				<div class="my-2 border-t border-outline-variant" role="separator"></div>
			{/if}
			{#if e.ueberschriftDavor}
				<div class="px-3 pt-1 pb-1 text-label-small font-medium text-on-surface-variant">
					{e.ueberschriftDavor}
				</div>
			{/if}
			<button
				type="button"
				role="menuitem"
				tabindex={i === aktiv ? 0 : -1}
				disabled={e.disabled}
				onclick={() => waehlen(e)}
				onpointerenter={() => (aktiv = i)}
				class="m3-state flex h-12 w-full cursor-pointer items-center gap-3 px-3 text-left text-sm text-on-surface focus:outline-none focus-visible:bg-on-surface/8 disabled:cursor-not-allowed disabled:opacity-40"
			>
				{#if e.icon}
					{@const Icon = e.icon}
					<Icon class="h-5 w-5 shrink-0 text-on-surface-variant" aria-hidden="true" />
				{/if}
				<span class="min-w-0 flex-1 truncate">{e.text}</span>
			</button>
		{/each}
	</div>
{/if}
