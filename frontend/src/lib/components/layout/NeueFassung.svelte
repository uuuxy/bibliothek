<!-- @component NeueFassung — nach einem Update der Hinweis, dass dieses Fenster noch mit dem
     alten Programm läuft (stores/programmstand.svelte.js).

     Ort und Form wie BackupHinweis: Ein Systemzustand, der eine Handlung braucht, steht über dem
     Inhalt. Keine Snackbar: Die verschwindet nach Sekunden, und an der Theke ersetzt sie der
     nächste Scan. Nicht wegklickbar, weil das alte Programm eine Antwort des neuen Servers
     falsch lesen kann, ohne Meldung.

     Die Fläche steht in „Primary container“, damit niemand sie übersieht (M3, Color roles:
     „Standout fill color against surface, for key components like FAB“); Rot bleibt Fehlern. -->
<script>
	import Icon from '../ui/Icon.svelte';
	import { programmstand } from '../../stores/programmstand.svelte.js';
	import Button from '../ui/Button.svelte';
</script>

{#if programmstand.veraltet}
	<div
		role="status"
		class="no-print mb-5 flex w-full items-start gap-3 rounded-md bg-primary-container px-4 py-3"
	>
		<Icon name="refresh" class="mt-0.5 text-on-primary-container" />
		<div class="min-w-0 flex-1">
			<p class="text-body-medium-emphasized text-on-primary-container">
				Eine neue Fassung des Programms ist eingespielt
			</p>
			<p class="mt-0.5 text-xs leading-relaxed text-on-primary-container">
				Bitte neu laden, sobald hier nichts mehr offen ist. Was nicht gespeichert ist, geht dabei
				verloren.
			</p>
		</div>
		<Button
			onclick={() => programmstand.ladeNeu()}
			disabled={programmstand.laedt}
			class="mt-0.5 shrink-0"
		>
			{programmstand.laedt ? 'Wird geladen …' : 'Neu laden'}
		</Button>
	</div>
{/if}
