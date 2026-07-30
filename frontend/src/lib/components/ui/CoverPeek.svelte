<!-- @component CoverPeek — zeigt das Coverbild eines Titels auf Anforderung in einem
     schwebenden Feld. Gedacht für dichte Arbeitslisten, in denen ein dauerhaftes Cover
     zu teuer wäre: In der Bestellliste kostete es 53 px je Zeile (71 % der Zeilenhöhe)
     und einen Proxy-Request je Titel, war aber fast immer leer.

     Drei Entwurfsentscheidungen, die nicht offensichtlich sind:

     1. LAZY. Der Request geht erst raus, wenn jemand das Cover sehen will. Bei 334
        Titeln sind das 0 statt 334 Requests beim Laden der Seite.
     2. FIXED statt absolute. Die Liste ist ein overflow-y-auto-Container; ein absolut
        positioniertes Kind würde an dessen Kante abgeschnitten. Die Position wird
        deshalb aus dem Trigger-Rechteck berechnet und an den Viewport-Rändern gespiegelt.
     3. Kein reines CSS-:hover. Das beherrscht weder Touch noch Tastatur — auf dem iPad
        beim Bestandsabgleich im Raum wäre das Feature sonst nicht erreichbar. -->
<script>
	/** children ist der Auslöser. Ohne Angabe erscheint das Bild-Symbol (Bestellliste,
	 *  wo kein Cover in der Zeile steht). Mit Angabe wird der übergebene Inhalt selbst
	 *  zum Auslöser — die Ausleihliste hängt so ihr vorhandenes Miniaturbild hinein,
	 *  statt daneben noch ein zweites Bedienelement zu setzen.
	 *  @type {{ isbn?: string, coverUrl?: string, titel?: string, children?: import('svelte').Snippet }} */
	let { isbn = '', coverUrl = '', titel = '', children } = $props();

	let offen = $state(false);
	/** ungeprueft → laedt → da | keins. Bleibt nach dem ersten Versuch erhalten,
	 *  damit erneutes Zeigen sofort das Ergebnis kennt. */
	let status = $state(/** @type {'ungeprueft' | 'laedt' | 'da' | 'keins'} */ ('ungeprueft'));
	let pos = $state({ top: 0, left: 0 });
	/** @type {HTMLButtonElement | undefined} */
	let trigger = $state();

	const BREITE = 150;
	const HOEHE = 200; // 3:4
	const LUFT = 8;

	function oeffnen() {
		if (!trigger) return;
		const r = trigger.getBoundingClientRect();
		// Standard ist LINKS neben dem Auslöser. Der sitzt am Zeilenanfang, also legt sich
		// das Feld dorthin, wo keine Listeninhalte stehen. Nach rechts würde es Titel und
		// ISBN derselben Zeile verdecken — ausgerechnet die Angaben, mit denen das Cover
		// abgeglichen werden soll. Nur wenn links kein Platz ist, kippt es nach rechts.
		let left = r.left - BREITE - LUFT;
		if (left < LUFT) left = Math.min(r.right + LUFT, window.innerWidth - BREITE - LUFT);
		// Senkrecht am Auslöser zentrieren, aber im Viewport halten.
		const top = Math.max(
			LUFT,
			Math.min(r.top + r.height / 2 - HOEHE / 2, window.innerHeight - HOEHE - LUFT)
		);
		pos = { top, left: Math.max(LUFT, left) };
		offen = true;
		if (status === 'ungeprueft') status = 'laedt';
	}

	/** Zeigertyp des letzten Kontakts — entscheidet, ob click noch etwas tun darf. */
	let letzterZeiger = 'mouse';

	/** @param {PointerEvent} e */
	function zeigen(e) {
		if (e.pointerType === 'mouse') oeffnen();
	}

	/** @param {PointerEvent} e */
	function verbergen(e) {
		if (e.pointerType === 'mouse') offen = false;
	}

	/**
	 * Umschalten NUR für Touch/Stift und Tastatur.
	 *
	 * Bei der Maus hat pointerenter bereits geöffnet; ein zusätzlich umschaltender Klick
	 * würde das Feld im selben Moment wieder zuklappen — man klickt auf das Symbol und
	 * das Bild verschwindet. Tastaturbedienung erkennt man an detail === 0 (Enter/Space
	 * erzeugen einen Klick ohne Zeigerkoordinaten).
	 * @param {MouseEvent} e
	 */
	function umschalten(e) {
		if (e.detail !== 0 && letzterZeiger === 'mouse') return;
		if (offen) offen = false;
		else oeffnen();
	}
</script>

<svelte:window
	onkeydown={(e) => {
		if (e.key === 'Escape') offen = false;
	}}
/>

<button
	bind:this={trigger}
	type="button"
	onpointerdown={(e) => (letzterZeiger = e.pointerType)}
	onpointerenter={zeigen}
	onpointerleave={verbergen}
	onclick={umschalten}
	onblur={() => (offen = false)}
	aria-expanded={offen}
	aria-label="Cover von {titel} anzeigen"
	class="shrink-0 rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 transition-colors cursor-pointer {children
		? ''
		: 'w-5 h-5 flex items-center justify-center text-slate-300 hover:text-slate-600 hover:bg-slate-100'}"
>
	{#if children}
		{@render children()}
	{:else}
		<svg
			class="w-4 h-4"
			fill="none"
			viewBox="0 0 24 24"
			stroke="currentColor"
			stroke-width="1.8"
			aria-hidden="true"
		>
			<rect x="3" y="4" width="18" height="16" rx="2" />
			<path stroke-linecap="round" stroke-linejoin="round" d="M3 15l5-4 4 3 3-2 6 4" />
		</svg>
	{/if}
</button>

{#if offen}
	<div
		class="fixed z-50 rounded-md border border-slate-200 bg-white shadow-xl overflow-hidden pointer-events-none"
		style="top: {pos.top}px; left: {pos.left}px; width: {BREITE}px; height: {HOEHE}px;"
		role="tooltip"
	>
		{#if status === 'keins' || !coverUrl}
			<div
				class="w-full h-full flex flex-col items-center justify-center gap-1.5 px-3 text-center text-slate-400"
			>
				<svg
					class="w-6 h-6"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="1.5"
					aria-hidden="true"
				>
					<rect x="3" y="4" width="18" height="16" rx="2" />
					<path stroke-linecap="round" d="M4 19L20 5" />
				</svg>
				<span class="text-[11px] leading-tight font-medium">Kein Coverbild hinterlegt</span>
			</div>
		{:else}
			{#if status === 'laedt'}
				<div class="absolute inset-0 flex items-center justify-center">
					<div
						class="w-5 h-5 border-2 border-slate-200 border-t-slate-400 rounded-full animate-spin"
					></div>
				</div>
			{/if}
			<img
				src="/api/images/cover?isbn={isbn}&url={encodeURIComponent(coverUrl)}"
				alt="Cover von {titel}"
				class="w-full h-full object-contain bg-white"
				onload={(e) => {
					// Der Cover-Proxy antwortet bei fehlendem Bild mit einem transparenten
					// 1×1-GIF (bewusst 200 statt 404, gegen Konsolen-Spam). onerror greift
					// dann NICHT — der Fall ist nur an der Winzgröße zu erkennen.
					status =
						/** @type {HTMLImageElement} */ (e.currentTarget).naturalWidth <= 1 ? 'keins' : 'da';
				}}
				onerror={() => (status = 'keins')}
			/>
		{/if}
	</div>
{/if}
