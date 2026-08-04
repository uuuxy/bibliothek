<!-- @component BackupStatusBadge — ruhige Bestätigungszeile im Sidebar-Fuß.
     Der Backup-Job überspringt sich still, wenn BACKUP_ENCRYPTION_KEY fehlt.
     Diese Zeile beantwortet nur die passive Frage „läuft es?" — sie zeigt
     ausschließlich den ok-Fall. Handlungsbedarf gehört nicht in die Navigation
     und wird von BackupAlert über dem Inhalt gemeldet (mit Weg zur Behebung).
     Eingeklappt bleibt ein Punkt als Hinweis, dass es dazu etwas zu sehen gibt. -->
<script>
	import { onMount } from 'svelte';
	import { backupStatus } from '../../stores/backupStatus.svelte.js';

	/** @type {{ collapsed?: boolean }} */
	let { collapsed = false } = $props();

	onMount(() => backupStatus.load());
</script>

{#if backupStatus.data}
	{#if collapsed}
		{#if backupStatus.needsAction}
			<div class="flex justify-center py-2" title={backupStatus.message}>
				<span
					class="h-2.5 w-2.5 animate-pulse rounded-full {backupStatus.data.status === 'critical'
						? 'bg-rose-500'
						: 'bg-amber-500'}"
				></span>
			</div>
		{/if}
	{:else if !backupStatus.needsAction}
		<div class="flex items-center gap-1.5 px-4 py-2 text-label-small font-semibold text-emerald-700">
			<svg class="h-3 w-3 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"
				><path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2.5"
					d="M5 13l4 4L19 7"
				/></svg
			>
			{backupStatus.message}
		</div>
	{/if}
{/if}
