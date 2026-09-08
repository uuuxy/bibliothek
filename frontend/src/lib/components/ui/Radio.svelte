<!-- @component Radio — DER Auswahlknopf der Anwendung (Material 3).

     Schwesterbauteil von Kaestchen.svelte, gleiche Geschichte: Die fünf nativen
     <input type="radio"> trugen tote Forms-Plugin-Klassen und zeigten Chromes
     eigenes Blau. Maße nach M3: Ring 20 px mit 2 px Rahmen in `on-surface-variant`,
     ausgewählt Ring und 10-px-Punkt in `primary`, State-Layer 40 px (das Eingabeelement
     selbst, siehe Kaestchen.svelte), deaktiviert 38 % `on-surface`.

     Gruppe: `bind:group` wie am nativen Element — `group` ist der gebundene Wert der
     Gruppe, `value` der Wert dieses Knopfs. Beschriftung wie bei Kaestchen: `label`
     für sichtbaren Text, sonst `aria-label` Pflicht. -->
<script>
	/** @type {{
	 *   group?: any,
	 *   value: any,
	 *   label?: string,
	 *   disabled?: boolean,
	 *   class?: string,
	 *   [rest: string]: any
	 * }} */
	let {
		group = $bindable(),
		value,
		label = '',
		disabled = false,
		class: klasse = '',
		...rest
	} = $props();
</script>

{#snippet knopf(zusatz)}
	<span class="relative -m-[10px] inline-grid h-10 w-10 shrink-0 place-items-center {zusatz}">
		<input
			type="radio"
			bind:group
			{value}
			{disabled}
			class="peer col-start-1 row-start-1 h-10 w-10 cursor-pointer appearance-none rounded-full outline-none transition-colors hover:bg-on-surface/8 focus-visible:bg-on-surface/10 checked:hover:bg-primary/8 checked:focus-visible:bg-primary/10 disabled:cursor-not-allowed"
			{...rest}
		/>
		<span
			aria-hidden="true"
			class="pointer-events-none col-start-1 row-start-1 grid h-5 w-5 place-items-center rounded-full border-2 border-on-surface-variant transition-colors peer-checked:border-primary peer-focus-visible:ring-2 peer-focus-visible:ring-primary peer-focus-visible:ring-offset-2 peer-disabled:border-on-surface/38 peer-disabled:peer-checked:border-on-surface/38 peer-disabled:peer-checked:[&>span]:bg-on-surface/38"
		>
			<span
				class="h-2.5 w-2.5 rounded-full bg-primary opacity-0 transition-opacity {group === value
					? 'opacity-100'
					: ''}"
			></span>
		</span>
	</span>
{/snippet}

{#if label}
	<label
		class="inline-flex items-center gap-3 text-sm text-on-surface select-none {disabled
			? 'cursor-not-allowed opacity-60'
			: 'cursor-pointer'} {klasse}"
	>
		{@render knopf('')}
		<span>{label}</span>
	</label>
{:else}
	{@render knopf(klasse)}
{/if}
