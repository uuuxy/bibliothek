<!-- @component BackupHinweis — der Backup-Wächter als Inline-Alert über dem Inhalt.

     1. Der Ort. Die Seitenleiste ist Navigation; ein dauerhafter Konfigurationsfehler würde
        dort zwischen Menüpunkten zur Tapete. Der Alert steht über dem Inhalt.
     2. Die Form. Wie NeueFassung eine Fläche in „Primary container“ mit Text und Symbol in
        „On primary container“; den Schweregrad nennt der Text, Rot bleibt Fehlern.
     3. Die Handlung. Der Text sagt, was zu tun ist, und der Knopf führt auf den richtigen
        Reiter. -->
<script>
	import { onMount } from 'svelte';
	import Icon from '../ui/Icon.svelte';
	import { backupStatus } from '../../stores/backupStatus.svelte.js';
	import { uiStore } from '../../stores/uiStore.svelte.js';
	import Button from '../ui/Button.svelte';

	onMount(() => backupStatus.load());

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
		class="no-print mb-5 flex items-start gap-3 rounded-md bg-primary-container px-4 py-3"
	>
		<Icon name="warning" class="mt-0.5 text-on-primary-container" />
		<div class="min-w-0 flex-1">
			<p class="text-body-medium-emphasized text-on-primary-container">{backupStatus.message}</p>
			<p class="mt-0.5 text-xs leading-relaxed text-on-primary-container">{backupStatus.hint}</p>
		</div>
		<Button onclick={openBetriebsbereitschaft} class="mt-0.5 shrink-0">
			Betriebsbereitschaft öffnen
			<Icon name="arrow_forward" groesse={18} />
		</Button>
		<Button
			variant="symbol"
			size="sm"
			onclick={() => (weggeklickt = true)}
			class="icon-btn mt-0.5 shrink-0 px-2 text-on-primary-container"
			aria-label="Hinweis für diese Sitzung ausblenden"
			data-tip="Für diese Sitzung ausblenden, beim nächsten Laden wieder da"
		>
			<Icon name="close" />
		</Button>
	</div>
{/if}
