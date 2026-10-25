<!-- @component NeueFassung — nach einem Update der Hinweis, dass dieses Fenster noch mit dem
     alten Programm läuft (stores/programmstand.svelte.js).

     Ort und Form wie BackupHinweis: Ein Systemzustand, der eine Handlung braucht, steht über dem
     Inhalt. Keine Snackbar: Die verschwindet nach Sekunden, und an der Theke ersetzt sie der
     nächste Scan. Nicht wegklickbar, weil das alte Programm eine Antwort des neuen Servers
     falsch lesen kann, ohne Meldung. -->
<script>
	import { RefreshCw } from '@lucide/svelte';
	import { programmstand } from '../../stores/programmstand.svelte.js';
	import Button from '../ui/Button.svelte';
</script>

{#if programmstand.veraltet}
	<div
		role="status"
		class="no-print mb-5 flex w-full items-start gap-3 rounded-md border border-outline-variant border-l-[3px] border-l-primary bg-surface-container-lowest py-3 pr-4 pl-3.5"
	>
		<RefreshCw class="mt-0.5 h-4 w-4 shrink-0 text-primary" aria-hidden="true" />
		<div class="min-w-0 flex-1">
			<p class="text-sm font-semibold text-on-surface">
				Eine neue Fassung des Programms ist eingespielt.
			</p>
			<p class="mt-0.5 text-xs leading-relaxed text-on-surface-variant">
				Bitte neu laden, sobald hier nichts mehr offen ist. Was nicht gespeichert ist, geht dabei
				verloren.
			</p>
		</div>
		<Button
			variant="secondary"
			size="sm"
			onclick={() => programmstand.ladeNeu()}
			class="mt-0.5 shrink-0"
		>
			Neu laden
		</Button>
	</div>
{/if}
