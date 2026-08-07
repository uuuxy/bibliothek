<script>
	import { Check, CircleAlert } from '@lucide/svelte';
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
			<CircleAlert class="w-4 h-4 shrink-0 text-rose-300" aria-hidden="true" />
		{:else}
			<Check class="w-4 h-4 shrink-0 text-emerald-400" aria-hidden="true" />
		{/if}
		{snackbar.msg}
	</div>
{/if}
