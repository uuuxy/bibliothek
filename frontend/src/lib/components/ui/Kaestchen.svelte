<!-- @component Kaestchen — DAS Kontrollkästchen der Anwendung (Material 3).

     Bis zum 08.09.2026 gab es 22 native <input type="checkbox"> in 17 Dateien. Sie
     trugen Klassen wie `rounded border-slate-300 text-blue-600 focus:ring-blue-500`
     — das Rezept des Tailwind-Forms-Plugins, das in diesem Repo nie installiert war.
     Auf einem nativen Kästchen bewirken diese Klassen NICHTS: Chrome zeichnete sein
     eigenes Blau, eckig, in 13, 16 oder 18 px, ohne Fokusring. Gemessen am 08.09.
     (appearance: auto an allen 22 Stellen). Nur die 8 Stellen mit `accent-*`
     färbten das Häkchen überhaupt in unser Primärblau.

     Ein natives Kästchen lässt sich nicht nach M3 einkleiden, also zeichnet dieses
     Bauteil selbst (appearance-none):
       - Kasten 18 px, Rahmen 2 px in `on-surface-variant`, Ecken 2 px (rounded-xs).
       - Ausgewählt: Fläche `primary`, Häkchen in `on-primary` (Lucide Check, nicht
         handgezeichnet — Ratsche frontend-hygiene-icons.test.js).
       - Unbestimmt (`indeterminate`): dieselbe Fläche, ein Strich statt Häkchen.
       - State-Layer 40 px rund für Hover 8 % und Fokus 10 % — das ist das Eingabeelement
         selbst. Es ist zugleich der Berührbereich, größer als der Kasten. Nach außen
         beansprucht das Bauteil aber nur 18 px (negativer Rand), damit es in eine
         36-px-Tabellenzeile passt; der Kreis ragt beim Hover über die Zeile hinaus,
         genau wie in M3-Listen. Entschieden am 08.09.2026.
       - Fehler (`ungueltig`): Rahmen und Fläche in `error`.
       - Deaktiviert: 38 % `on-surface`, wie M3 es vorschreibt.

     Bedeutung (M3): Ein Kästchen wählt aus einer Liste aus oder schaltet eine Option
     eines Formulars. Ein ZUSTAND („Konto aktiv") ist ein Switch.svelte.

     Beschriftung: Mit `label` ist das Bauteil ein <label> mit sichtbarem Text. Ohne
     `label` (Tabellenzelle) ist `aria-label` PFLICHT — sonst liest der Screenreader
     nur „Kontrollkästchen". Alles Unbenannte (name, value, aria-*, title, onclick …)
     landet unverändert auf dem Eingabeelement. -->
<script>
	import { Check, Minus } from '@lucide/svelte';

	/** @type {{
	 *   checked?: boolean,
	 *   indeterminate?: boolean,
	 *   label?: string,
	 *   disabled?: boolean,
	 *   ungueltig?: boolean,
	 *   class?: string,
	 *   onchange?: (event: Event & { currentTarget: HTMLInputElement }) => void,
	 *   [rest: string]: any
	 * }} */
	let {
		checked = $bindable(false),
		indeterminate = false,
		label = '',
		disabled = false,
		ungueltig = false,
		class: klasse = '',
		...rest
	} = $props();

	const kastenFarbe = $derived(
		ungueltig
			? 'border-error peer-checked:bg-error peer-indeterminate:bg-error'
			: 'border-on-surface-variant peer-checked:border-primary peer-checked:bg-primary peer-indeterminate:border-primary peer-indeterminate:bg-primary'
	);
	const schichtFarbe = $derived(
		ungueltig
			? 'hover:bg-error/8 focus-visible:bg-error/10'
			: 'hover:bg-on-surface/8 focus-visible:bg-on-surface/10 checked:hover:bg-primary/8 checked:focus-visible:bg-primary/10'
	);
</script>

{#snippet kasten(zusatz)}
	<span class="relative -m-[11px] inline-grid h-10 w-10 shrink-0 place-items-center {zusatz}">
		<input
			type="checkbox"
			bind:checked
			{indeterminate}
			{disabled}
			class="peer col-start-1 row-start-1 h-10 w-10 cursor-pointer appearance-none rounded-full outline-none transition-colors disabled:cursor-not-allowed {schichtFarbe}"
			{...rest}
		/>
		<span
			aria-hidden="true"
			class="pointer-events-none col-start-1 row-start-1 grid h-[18px] w-[18px] place-items-center rounded-xs border-2 text-on-primary transition-colors peer-focus-visible:ring-2 peer-focus-visible:ring-primary peer-focus-visible:ring-offset-2 peer-disabled:border-on-surface/38 peer-disabled:peer-checked:border-transparent peer-disabled:peer-checked:bg-on-surface/38 peer-disabled:peer-indeterminate:border-transparent peer-disabled:peer-indeterminate:bg-on-surface/38 {kastenFarbe}"
		>
			{#if indeterminate}
				<Minus class="h-3.5 w-3.5" strokeWidth={3} />
			{:else if checked}
				<Check class="h-3.5 w-3.5" strokeWidth={3} />
			{/if}
		</span>
	</span>
{/snippet}

{#if label}
	<label
		class="inline-flex items-center gap-3 text-sm text-on-surface select-none {disabled
			? 'cursor-not-allowed opacity-60'
			: 'cursor-pointer'} {klasse}"
	>
		{@render kasten('')}
		<span>{label}</span>
	</label>
{:else}
	{@render kasten(klasse)}
{/if}
