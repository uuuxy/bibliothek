<script>
	import { toastStore } from './stores/toastStore.svelte.js';
	import { flip } from 'svelte/animate';
	import { fly, fade } from 'svelte/transition';
</script>

<!-- z-9999: Toasts müssen über allen Modals (z-50/z-60) und Omnibox-Alerts (z-100) liegen -->
<div class="fixed top-6 right-6 z-9999 flex flex-col gap-3 pointer-events-none items-end">
	{#each toastStore.toasts as toast (toast.id)}
		<div
			animate:flip={{ duration: 250 }}
			in:fly={{ y: -20, duration: 300 }}
			out:fade={{ duration: 200 }}
			class="pointer-events-auto px-5 py-3 rounded-2xl shadow-xl text-sm font-semibold flex items-center gap-2 max-w-sm w-full
        {toast.type === 'error'
				? 'bg-rose-600 text-white'
				: toast.type === 'success'
					? 'bg-emerald-600 text-white'
					: 'bg-slate-800 text-white'}"
		>
			{#if toast.type === 'error'}
				<svg
					xmlns="http://www.w3.org/2000/svg"
					class="h-5 w-5 shrink-0"
					viewBox="0 0 20 20"
					fill="currentColor"
				>
					<path
						fill-rule="evenodd"
						d="M10 18a8 8 0 100-16 8 8 0 000 16zm-1-9a1 1 0 112 0v4a1 1 0 11-2 0v-4zm1-3a1 1 0 100 2 1 1 0 000-2z"
						clip-rule="evenodd"
					/>
				</svg>
			{:else if toast.type === 'success'}
				<svg
					xmlns="http://www.w3.org/2000/svg"
					class="h-5 w-5 shrink-0"
					viewBox="0 0 20 20"
					fill="currentColor"
				>
					<path
						fill-rule="evenodd"
						d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
						clip-rule="evenodd"
					/>
				</svg>
			{:else}
				<svg
					xmlns="http://www.w3.org/2000/svg"
					class="h-5 w-5 shrink-0"
					viewBox="0 0 20 20"
					fill="currentColor"
				>
					<path
						fill-rule="evenodd"
						d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
						clip-rule="evenodd"
					/>
				</svg>
			{/if}
			<span class="wrap-break-word w-full">{toast.message}</span>
			<button
				onclick={() => toastStore.removeToast(toast.id)}
				class="ml-2 text-white/70 hover:text-white transition-colors cursor-pointer shrink-0"
				aria-label="Schließen"
			>
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true"
					><path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M6 18L18 6M6 6l12 12"
					></path></svg
				>
			</button>
		</div>
	{/each}
</div>
