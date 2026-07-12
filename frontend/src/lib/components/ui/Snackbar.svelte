<script>
	/**
	 * @typedef {Object} Props
	 * @property {{msg: string, type: 'success' | 'error'} | null} snackbar - The snackbar state object
	 */

	/** @type {Props} */
	let { snackbar } = $props();
</script>

{#if snackbar}
	<div
		class="fixed bottom-8 left-1/2 -translate-x-1/2 z-200
           flex items-center gap-3 px-5 py-3.5
           rounded-2xl shadow-2xl
           text-sm font-semibold
           animate-fade-in
           {snackbar.type === 'error' ? 'bg-rose-700 text-white' : 'bg-slate-900 text-white'}"
	>
		{#if snackbar.type === 'error'}
			<svg
				class="w-4 h-4 shrink-0 text-rose-300"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2.5"
					d="M12 9v2m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
				/>
			</svg>
		{:else}
			<svg
				class="w-4 h-4 shrink-0 text-emerald-400"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2.5"
					d="M5 13l4 4L19 7"
				/>
			</svg>
		{/if}
		{snackbar.msg}
	</div>
{/if}
