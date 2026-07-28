<script>
	import Button from '../../../../lib/components/ui/Button.svelte';
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
			class="w-full h-10 rounded-xl border border-slate-300 bg-white px-4 outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all text-slate-800 placeholder-slate-400 shadow-sm"
		/>
		<Button
			size="lg"
			onclick={() => {
				onSubmit(manualISBN);
				manualISBN = '';
			}}
			{disabled}
			class="px-5"
		>
			Senden
		</Button>
	</div>
</div>
