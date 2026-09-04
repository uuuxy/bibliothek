<script>
	/**
	 * Modal — generic overlay container that accepts snippet render-props.
	 *
	 * Usage:
	 *   <Modal open={showModal} onclose={() => showModal = false} size="md">
	 *     {#snippet header()}<h3>Titel</h3>{/snippet}
	 *     {#snippet children()}<p>Inhalt</p>{/snippet}
	 *   </Modal>
	 *
	 * Props:
	 *   open     — controls visibility
	 *   onclose  — optional; if provided, an × button is rendered in the header bar
	 *   size     — "sm" | "md" | "lg" | "xl" | "2xl" | "3xl" | "4xl" (default: "md")
	 *   header   — optional snippet: rendered inside the top bar (title area)
	 *   children — required snippet: the modal body content
	 */

	/** @type {{ open: boolean, onclose?: () => void, size?: 'sm' | 'md' | 'lg' | 'xl' | '2xl' | '3xl' | '4xl', header?: import('svelte').Snippet, children: import('svelte').Snippet }} */
	let { open, onclose, size = 'md', header, children } = $props();

	const sizeClass = $derived(
		{
			sm: 'max-w-sm',
			md: 'max-w-md',
			lg: 'max-w-lg',
			xl: 'max-w-xl',
			'2xl': 'max-w-2xl',
			'3xl': 'max-w-3xl',
			'4xl': 'max-w-4xl'
		}[size] ?? 'max-w-md'
	);
</script>

<!-- Escape schliesst den Dialog. Am Fenster und nicht am Element darunter, weil das
     tabindex="-1" traegt: Es ist nicht per Tastatur fokussierbar, ein onkeydown dort
     feuerte also nur, solange der Fokus zufaellig darin liegt. Bis hierher gab es gar
     keinen Escape-Weg — nur den Klick auf den Hintergrund, und der ist mit der Tastatur
     nicht erreichbar.
     `open` wird im Handler geprueft, nicht per {#if} darum herum: <svelte:window> muss
     auf der obersten Ebene der Komponente stehen. -->
<svelte:window
	onkeydown={(e) => {
		if (open && e.key === 'Escape') onclose?.();
	}}
/>

{#if open}
	<!-- Die Dialog-Semantik sitzt am Fenster darunter, nicht am Hintergrund: Der
	     Hintergrund ist Dekoration (role="presentation"), das weisse Feld IST der Dialog.
	     Vorher trug der Hintergrund role="dialog" — das machte den abgedunkelten Bereich
	     fuer Screenreader zum Dialog samt tabindex, obwohl darin nur Unschaerfe liegt. -->
	<div
		class="fixed inset-0 bg-slate-900/40 backdrop-blur-xs z-50 flex items-center justify-center p-4 animate-fade-in"
		role="presentation"
		onclick={(e) => {
			if (e.target === e.currentTarget) onclose?.();
		}}
	>
		<!-- Kein Rahmen. M3 gibt dem Dialog `container-elevation: level3` (= 6dp) und
		     definiert fuer ihn WEDER outline-width NOCH outline-color (material-web
		     v0.192). Die Erhebung IST hier die Abgrenzung; ein zusaetzlicher Rahmen
		     ist die Bauform, die in der Spezifikation bei keinem der 84 Bauteile
		     vorkommt. Der Schatten bleibt — er ist der richtige Teil des Paares.
		     Wirkt auf die 11 Dialoge, die dieses Bauteil benutzen. -->
		<div
			class="bg-white w-full {sizeClass} rounded-3xl shadow-2xl overflow-hidden animate-scale-up"
			role="dialog"
			aria-modal="true"
			tabindex="-1"
		>
			{#if header}
				<div class="p-6 border-b border-slate-100 bg-slate-50/50 flex items-center justify-between">
					{@render header()}
					{#if onclose}
						<button
							onclick={onclose}
							class="text-slate-400 hover:text-slate-600 font-bold text-lg leading-none cursor-pointer"
							aria-label="Schließen">×</button
						>
					{/if}
				</div>
			{/if}
			{@render children()}
		</div>
	</div>
{/if}
