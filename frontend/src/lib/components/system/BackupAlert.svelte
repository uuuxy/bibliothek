<!-- @component BackupAlert — der Backup-Wächter als Inline-Alert über dem Inhalt.

     1. Der Ort. Die Sidebar ist Navigation; ein dauerhafter Konfigurationsfehler würde
        dort zwischen Menüpunkten zur Tapete. Der Alert steht über dem Inhalt.
     2. Die Form. Ein Streifen von 3 px in der Farbe des Schweregrads links auf ruhigem
        Grund plus ein Icon — auffällig durch Kante und Position.
     3. Die Handlung. Der Text sagt, was zu tun ist, und der Knopf führt auf den richtigen
        Reiter. -->
<script>
	import { onMount } from 'svelte';
	import { TriangleAlert, ArrowRight, X } from '@lucide/svelte';
	import { backupStatus } from '../../stores/backupStatus.svelte.js';
	import { uiStore } from '../../stores/uiStore.svelte.js';
	import Button from '../ui/Button.svelte';

	onMount(() => backupStatus.load());

	const critical = $derived(backupStatus.data?.status === 'critical');

	// Für die Sitzung wegklickbar: Eine Warnung, die auf jedem Bildschirm steht und nie
	// verschwindet, liest niemand mehr. Nicht gespeichert (weder localStorage noch Server):
	// Beim nächsten Laden steht sie wieder da, und es gibt keinen geteilten Zustand, der
	// zwischen den Arbeitsplätzen auseinanderlaufen könnte.
	let weggeklickt = $state(false);

	// Ziel ist die Betriebsbereitschaft: Dort steht der Backup-Befund samt Anleitung, und
	// sie verlangt dasselbe Recht wie dieser Alert (manage_settings).
	function openBetriebsbereitschaft() {
		uiStore.requestedSettingsTab = 'betrieb';
		uiStore.activeTab = 'settings';
	}
</script>

{#if backupStatus.needsAction && !weggeklickt}
	<div
		role="alert"
		class="no-print mb-5 flex items-start gap-3 rounded-md border border-outline-variant border-l-[3px] bg-surface-container-lowest py-3 pr-4 pl-3.5
			{critical ? 'border-l-error' : 'border-l-warning'}"
	>
		<TriangleAlert
			class="mt-0.5 h-4 w-4 shrink-0 {critical ? 'text-error' : 'text-warning'}"
			aria-hidden="true"
		/>
		<div class="min-w-0 flex-1">
			<p class="text-sm font-semibold text-on-surface">{backupStatus.message}</p>
			<p class="mt-0.5 text-xs leading-relaxed text-on-surface-variant">{backupStatus.hint}</p>
		</div>
		<Button
			variant="secondary"
			size="sm"
			onclick={openBetriebsbereitschaft}
			class="mt-0.5 shrink-0"
		>
			Betriebsbereitschaft öffnen
			<ArrowRight class="h-3.5 w-3.5" />
		</Button>
		<Button
			variant="symbol"
			size="sm"
			onclick={() => (weggeklickt = true)}
			class="icon-btn mt-0.5 shrink-0 px-2"
			aria-label="Hinweis für diese Sitzung ausblenden"
			data-tip="Für diese Sitzung ausblenden — beim nächsten Laden wieder da"
		>
			<X class="h-4 w-4" />
		</Button>
	</div>
{/if}
