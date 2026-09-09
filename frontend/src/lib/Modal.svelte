<script>
	import { escapeSchliesst } from './components/ui/escapeSchliesst.js';
	import { fokusFalle } from './components/ui/fokusFalle.js';
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
	 *   size     — "sm" | "md" | "lg" | "xl" | "2xl" | "3xl" | "4xl" | "voll" (default: "md")
	 *   header   — optional snippet: rendered inside the top bar (title area)
	 *   children — required snippet: the modal body content
	 *
	 * Seit 07.09.2026, als die elf selbstgebauten Overlays hierher zogen (Register 05.09.):
	 *   ebene            — "basis" (z-50) | "darueber" (z-60) | "oberst" (z-100): die drei
	 *                      Stufen des Hauses. Ein Dialog, der aus der Schülerakte oder der
	 *                      Theke heraus öffnet, liegt ÜBER deren Overlay — sonst öffnete er
	 *                      unsichtbar dahinter.
	 *   beschriftung     — aria-label des Dialogs; beschriftetDurch — aria-labelledby
	 *                      (die id der Überschrift). Ohne Namen findet ihn weder ein
	 *                      Screenreader noch getByRole('dialog', { name }).
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

	const voll = $derived(size === 'voll');
	const sizeClass = $derived(
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
	<!-- Die Dialog-Semantik sitzt am Fenster darunter, nicht am Hintergrund: Der
	     Hintergrund ist Dekoration (role="presentation"), das weisse Feld IST der Dialog.
	     Vorher trug der Hintergrund role="dialog" — das machte den abgedunkelten Bereich
	     fuer Screenreader zum Dialog samt tabindex, obwohl darin nur Unschaerfe liegt. -->
	<div
		class="fixed inset-0 bg-slate-900/40 backdrop-blur-xs {ebenenKlasse} flex items-center justify-center {voll
			? 'p-0 sm:p-4'
			: 'p-4'} animate-fade-in"
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
			class="bg-white w-full {sizeClass} {voll
				? ''
				: 'rounded-3xl'} shadow-2xl overflow-hidden animate-scale-up"
			role="dialog"
			aria-modal="true"
			aria-label={beschriftung}
			aria-labelledby={beschriftetDurch}
			tabindex="-1"
			use:escapeSchliesst={onclose}
			use:fokusFalle
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
