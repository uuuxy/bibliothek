<!-- @component Menue — das Material-3-Menü hinter einem „Mehr"-Knopf.

     M3 (Menus, Guidelines): „Use a menu to show a temporary set of actions. To show
     actions on screen at all times, use a toolbar instead" — und Überlaufmenüs sind
     der genannte Fall. Maße nach M3 Specs: Fläche surface-container mit 4-px-Ecken,
     Einträge mindestens 48 px hoch mit 12 px Innenabstand, führendes Symbol 24 dp (hier
     die Symbolgröße des Hauses), Breite zwischen 112 und 280 dp. Dieselbe Fläche wie die
     Liste des Auswahlfelds (SelectListe.svelte).

     Die Breite folgt dem längsten Eintrag; was über 280 dp hinausgeht, bricht um, statt
     abgeschnitten zu werden (M3, Text truncation: „Don't cut off text without providing a
     way for users to view it"). So baut es Google: Compose misst das Menü nach dem breitesten
     Eintrag (112–280 dp, Text ohne Zeilengrenze), md-menu-item gibt 12 px oben und unten.

     Dieses Bauteil ist die eine Stelle für Menüs im Haus.

     POSITION FIXED wie SelectListe: Das Menü öffnet sich in Tabellen und Spalten mit
     overflow, absolut positioniert würde es dort abgeschnitten.

     Tastatur: Pfeile wandern, Pos1/Ende springen, Enter und Leertaste wählen (der
     Eintrag ist ein <button>), Escape schließt und gibt den Fokus an den Knopf zurück
     (escapeSchliesst — nur das oberste Overlay reagiert), Tab verlässt das Menü.
     Gesperrte Einträge nehmen den Fokus und wählen nichts (M3 Menus, Accessibility:
     „Disabled menu items can receive focus but aren't selectable") — deshalb
     aria-disabled wie in SelectListe; ein <button disabled> nimmt im Browser keinen Fokus.

     Über die Einträge hinaus kann der Aufrufer den Auslöser selbst stellen (`ausloeser`-Snippet —
     der Split-Button „Ausweis drucken ▾"; der Knopf mit aria-haspopup bekommt den Fokus
     zurück). Breite und Höhe werden nach dem Rendern gemessen (menueGeometrie.js); geschlossen wird,
     wenn der Fokus das Menü verlässt. -->
<script>
	import { tick } from 'svelte';
	import { EllipsisVertical } from '@lucide/svelte';
	import Button from './Button.svelte';
	import { escapeSchliesst } from './escapeSchliesst.js';
	import { berechneMenueBox } from './menueGeometrie.js';

	/** @typedef {import('./menueGeometrie.js').Eintrag} Eintrag */
	/** @type {{ etikett: string, eintraege: Eintrag[], onwahl: (id: string) => void,
	 *   ausloeser?: import('svelte').Snippet<[{ offen: boolean, umschalten: () => void }]>,
	 *   ausrichtung?: 'rechts' | 'links' }} */
	let { etikett, eintraege, onwahl, ausloeser, ausrichtung = 'rechts' } = $props();

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
		// Gemessen, nicht gerechnet: Die Breite folgt dem längsten Eintrag, die Höhe den Trennern
		// und den umbrochenen Einträgen.
		const masse = flaeche?.getBoundingClientRect();
		box = berechneMenueBox(
			anker.getBoundingClientRect(),
			masse?.height ?? 0,
			masse?.width ?? 0,
			ausrichtung,
			{ breite: window.innerWidth, hoehe: window.innerHeight }
		);
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
			variant="symbol"
			size="sm"
			aria-haspopup="menu"
			aria-expanded={offen}
			aria-label={etikett}
			data-tip={etikett}
			onclick={() => (offen ? schliessen() : oeffnen())}
		>
			<EllipsisVertical class="h-4 w-4" aria-hidden="true" />
		</Button>
	{/if}
</div>

{#if offen}
	<!-- Die Fläche hängt im DOM am Auslöser, oft in einer Zelle mit whitespace-nowrap; ohne
	     whitespace-normal erbte sie das, und ein langer Eintrag liefe über den Rand. -->
	<div
		bind:this={flaeche}
		role="menu"
		aria-label={etikett}
		tabindex="-1"
		use:escapeSchliesst={() => schliessen()}
		onkeydown={taste}
		onfocusout={fokusWeg}
		style="position:fixed; left:{box?.left ?? 0}px; top:{box?.top ??
			0}px; z-index:60; visibility:{box ? 'visible' : 'hidden'};"
		class="w-max max-w-70 min-w-28 rounded-sm bg-surface-container py-2 whitespace-normal shadow-xl"
	>
		{#each eintraege as e, i (e.id)}
			{#if e.trennerDavor}
				<div class="my-2 border-t border-outline-variant" role="separator"></div>
			{/if}
			<button
				type="button"
				role="menuitem"
				tabindex={i === aktiv ? 0 : -1}
				aria-disabled={e.disabled || undefined}
				onclick={() => waehlen(e)}
				onpointerenter={() => (aktiv = i)}
				class="m3-state flex min-h-12 w-full cursor-pointer items-center gap-3 px-3 py-3 text-left text-sm text-on-surface focus:outline-none focus-visible:bg-on-surface/8 aria-disabled:cursor-not-allowed aria-disabled:opacity-40"
			>
				{#if e.icon}
					{@const Icon = e.icon}
					<Icon class="h-5 w-5 shrink-0 text-on-surface-variant" aria-hidden="true" />
				{/if}
				<span class="min-w-0 flex-1 wrap-break-word">{e.text}</span>
			</button>
		{/each}
	</div>
{/if}
