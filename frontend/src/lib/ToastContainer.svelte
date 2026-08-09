<script>
	import { toastStore } from './stores/toastStore.svelte.js';
	import { CircleAlert, CircleCheck, Info, TriangleAlert, X } from '@lucide/svelte';

	const symbole = {
		error: CircleAlert,
		success: CircleCheck,
		warning: TriangleAlert,
		info: Info
	};

	const flaechen = {
		error: 'bg-rose-600 text-white',
		success: 'bg-emerald-600 text-white',
		warning: 'bg-amber-500 text-white',
		info: 'bg-slate-800 text-white'
	};
</script>

<!-- z-9999: Toasts müssen über allen Modals (z-50/z-60) und Omnibox-Alerts (z-100) liegen -->
<div class="fixed top-6 right-6 z-9999 flex flex-col gap-2 pointer-events-none items-end">
	{#each toastStore.toasts as toast (toast.id)}
		{@const Symbol = symbole[toast.type] ?? Info}
		<div
			class="pointer-events-auto flex items-start gap-2 px-4 py-3 rounded-sm text-sm max-w-sm w-full {flaechen[
				toast.type
			] ?? flaechen.info}"
		>
			<Symbol class="h-5 w-5 shrink-0" aria-hidden="true" />
			<span class="wrap-break-word w-full">{toast.message}</span>
			{#if toast.aktion}
				<!-- M3-Snackbar: genau EINE Folgehandlung, textbetont und rechts neben der
				     Meldung. Sie schliesst den Toast selbst — die Meldung hat ihren Zweck
				     erfuellt, sobald man ihr gefolgt ist. -->
				<button
					onclick={() => {
						toast.aktion?.onClick();
						toastStore.removeToast(toast.id);
					}}
					class="ml-2 shrink-0 rounded-sm px-1 font-semibold whitespace-nowrap underline underline-offset-2 hover:opacity-80"
				>
					{toast.aktion.label}
				</button>
			{/if}
			<button
				onclick={() => toastStore.removeToast(toast.id)}
				class="ml-2 shrink-0 text-white/70 hover:text-white transition-colors cursor-pointer"
				aria-label="Schließen"
			>
				<X class="h-4 w-4" aria-hidden="true" />
			</button>
		</div>
	{/each}
</div>
