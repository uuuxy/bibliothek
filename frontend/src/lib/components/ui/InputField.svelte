<script>
	/**
	 * @typedef {Object} Props
	 * @property {string} id - HTML ID for the input and label.
	 * @property {string} label - Display label text.
	 * @property {string} [type] - HTML input type (e.g. text, date, number, email).
	 * @property {string|number} value - The bound value.
	 * @property {string} [placeholder] - Placeholder text.
	 * @property {number|null} [maxlength] - Maximum string length (HTML-Attribut, Zahl).
	 * @property {string} [extraClasses] - Additional utility classes for the input field.
	 * @property {boolean} [disabled] - Deaktiviert die Eingabe (schreibgeschützt).
	 * @property {string} [hint] - Kleiner Hinweistext unter dem Feld.
	 */

	/** @type {Props} */
	let {
		id,
		label,
		type = 'text',
		value = $bindable(),
		placeholder = '',
		maxlength = null,
		extraClasses = '',
		disabled = false,
		hint = ''
	} = $props();
</script>

<div>
	<label for={id} class="block text-xs font-medium text-slate-500 mb-1.5">{label}</label>
	<input
		{id}
		{type}
		bind:value
		{placeholder}
		{maxlength}
		{disabled}
		class="w-full h-9 px-3.5 bg-slate-50 border border-slate-200 rounded-xl text-sm text-slate-800
           focus:bg-white focus:outline-none focus:border-blue-400 focus:ring-2 focus:ring-blue-100 transition-all
           disabled:cursor-not-allowed disabled:opacity-60 {extraClasses}"
	/>
	{#if hint}
		<p class="text-xs text-slate-400 mt-1">{hint}</p>
	{/if}
</div>
