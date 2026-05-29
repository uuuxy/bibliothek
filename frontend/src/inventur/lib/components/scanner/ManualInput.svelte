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
	<label for="manual-isbn" class="block text-xs font-semibold uppercase tracking-wider text-zinc-400">Handscanner / ISBN-Eingabe</label>
	<div class="mt-2 flex gap-2">
		<input 
			id="manual-isbn" 
			type="text" 
			bind:value={manualISBN} 
			onkeydown={handleKeydown} 
			placeholder="ISBN scannen oder eintippen" 
			class="w-full rounded-xl border border-zinc-800 bg-zinc-950 px-3 py-2 outline-none focus:ring-2 focus:ring-emerald-500/50 focus:border-emerald-500 transition-all text-zinc-100 placeholder-zinc-500" 
		/>
		<button 
			onclick={() => { onSubmit(manualISBN); manualISBN = ''; }} 
			disabled={disabled} 
			class="rounded-full bg-emerald-500 px-5 py-2.5 text-sm font-bold text-zinc-950 hover:bg-emerald-400 disabled:opacity-60 transition-colors cursor-pointer"
		>
			Senden
		</button>
	</div>
</div>