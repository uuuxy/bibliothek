<script>
	/**
	 * @type {{
	 *   onSubmit: (isbn: string) => void,
	 *   disabled: boolean
	 * }}
	 */
	let { onSubmit, disabled } = $props();
	let manualISBN = $state('');

	/**
	 * @param {KeyboardEvent} event
	 */
	function handleKeydown(event) {
		if (event.key === 'Enter') {
			event.preventDefault();
			onSubmit(manualISBN);
			manualISBN = '';
		}
	}
</script>

<div class="mt-4">
	<label
		for="manual-isbn"
		class="block text-xs font-semibold uppercase tracking-wider text-slate-400"
		>Handscanner / ISBN-Eingabe</label
	>
	<div class="mt-2 flex gap-2">
		<input
			id="manual-isbn"
			type="text"
			bind:value={manualISBN}
			onkeydown={handleKeydown}
			placeholder="ISBN scannen oder eintippen"
			class="w-full rounded-xl border border-slate-300 bg-white px-4 py-2.5 outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all text-slate-800 placeholder-slate-400 shadow-sm"
		/>
		<button
			onclick={() => {
				onSubmit(manualISBN);
				manualISBN = '';
			}}
			{disabled}
			class="rounded-xl bg-blue-600 px-5 py-2.5 text-sm font-bold text-white hover:bg-blue-700 disabled:opacity-60 transition-colors cursor-pointer shadow-sm"
		>
			Senden
		</button>
	</div>
</div>
