<script>
	import { escapeSchliesst } from './escapeSchliesst.js';
	import { fokusFalle } from './fokusFalle.js';
	import { X } from '@lucide/svelte';
	/**
	 * Dialog — generic overlay container that accepts snippet render-props.
	 *
	 * Usage:
	 *   <Dialog open={showModal} onclose={() => showModal = false} size="md">
	 *     {#snippet header()}<h3>Titel</h3>{/snippet}
	 *     {#snippet children()}<p>Inhalt</p>{/snippet}
	 *   </Dialog>
	 *
	 * Props:
	 *   open     — controls visibility
	 *   onclose  — optional; if provided, a close button is rendered in the header bar
	 *   size     — "sm" | "md" | "lg" | "xl" | "2xl" | "3xl" | "4xl" | "voll" (default: "md")
	 *   header   — optional snippet: rendered inside the top bar (title area)
	 *   children — required snippet: the modal body content
	 *
	 *   ebene            — "basis" (z-50) | "darueber" (z-60) | "oberst" (z-100): die drei
	 *                      Stufen des Hauses. Ein Dialog, der aus der Schülerakte oder der
	 *                      Theke heraus öffnet, liegt über deren Overlay — sonst öffnete er
	 *                      unsichtbar dahinter.
	 *   beschriftung     — aria-label des Dialogs; beschriftetDurch — aria-labelledby
	 *                      (die id der Überschrift). Ohne Namen findet ihn weder ein
	 *                      Screenreader noch getByRole('dialog', { name }). Fehlen beide,
	 *                      benennt der header den Dialog.
	 *   size "voll"      — der M3 full-screen dialog: Vollbild auf dem Handy, 90 vh ab
	 *                      Tablet (Klasse & Bücher zuweisen).
	 */

	/** @type {{ open: boolean, onclose?: () => void, size?: 'sm' | 'md' | 'lg' | 'xl' | '2xl' | '3xl' | '4xl' | 'voll', ebene?: 'basis' | 'darueber' | 'oberst', beschriftung?: string, beschriftetDurch?: string, header?: import('svelte').Snippet, children: import('svelte').Snippet }} */
	let {
		open,
		onclose,
		size = 'md',
		ebene = 'basis',
		beschriftung,
		beschriftetDurch,
		header,
		children
	} = $props();

	// Wer den Titel nur als header übergibt, bekäme sonst einen Dialog ohne Namen: Der
	// Bildschirmleser sagte nur „Dialog", ausgerechnet bei „Endgültig löschen?".
	const kopfId = $props.id();
	const benanntDurch = $derived(beschriftetDurch ?? (header && !beschriftung ? kopfId : undefined));

	const voll = $derived(size === 'voll');
	const breitenKlasse = $derived(
		{
			sm: 'max-w-sm',
			md: 'max-w-md',
			lg: 'max-w-lg',
			xl: 'max-w-xl',
			'2xl': 'max-w-2xl',
			'3xl': 'max-w-3xl',
			'4xl': 'max-w-4xl',
			voll: 'rounded-none sm:rounded-3xl lg:w-300 max-w-[100vw] lg:max-w-[90vw] h-dvh sm:h-[90vh] lg:h-212.5 max-h-dvh lg:max-h-[95vh]'
		}[size] ?? 'max-w-md'
	);
	const ebenenKlasse = $derived(
		{ basis: 'z-50', darueber: 'z-60', oberst: 'z-100' }[ebene] ?? 'z-50'
	);
</script>

{#if open}
	<!-- Die Dialog-Semantik sitzt am Fenster darunter, nicht am Hintergrund: Der Hintergrund
	     ist Dekoration (role="presentation"), das weiße Feld ist der Dialog. Der Schleier
	     trägt die Rolle scrim mit 32 % (M3, Elevation). -->
	<div
		class="fixed inset-0 bg-scrim/32 backdrop-blur-xs {ebenenKlasse} flex items-center justify-center {voll
			? 'p-0 sm:p-4'
			: 'p-4'} animate-fade-in"
		role="presentation"
		onclick={(e) => {
			if (e.target === e.currentTarget) onclose?.();
		}}
	>
		<!-- Kein Rahmen: M3 gibt dem Dialog `container-elevation: level3` und weder
		     outline-width noch outline-color (material-web v0.192). Die Erhebung ist die
		     Abgrenzung. Die Fläche bleibt weiß wie die Arbeitsfläche. -->
		<div
			class="bg-surface-container-lowest w-full {breitenKlasse} {voll
				? ''
				: 'rounded-3xl'} shadow-2xl overflow-hidden animate-scale-up"
			role="dialog"
			aria-modal="true"
			aria-label={beschriftung}
			aria-labelledby={benanntDurch}
			tabindex="-1"
			use:escapeSchliesst={onclose}
			use:fokusFalle
		>
			{#if header}
				<!-- Ohne Linie und eigene Fläche: Der einfache Dialog von Material 3 hat keine
				     Kopfzeile, der Titel steht auf der Fläche des Dialogs. -mb-2 ergibt mit den
				     24 px Innenabstand des Inhalts die 16 px zwischen Titel und Text (Specs:
				     „Padding between title and body 16dp"). -->
				<div class="px-6 pt-6 -mb-2 flex items-center justify-between gap-4">
					<!-- display: contents: Die Hülle trägt nur die id, die Kopfzeile bleibt, wie sie war. -->
					<div id={kopfId} class="contents">{@render header()}</div>
					{#if onclose}
						<button
							onclick={onclose}
							class="icon-btn text-on-surface-variant"
							aria-label="Schließen"
							data-tip="Schließen"
						>
							<X class="h-4 w-4" aria-hidden="true" />
						</button>
					{/if}
				</div>
			{/if}
			{@render children()}
		</div>
	</div>
{/if}
