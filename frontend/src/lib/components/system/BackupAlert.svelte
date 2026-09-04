<!-- @component BackupAlert — der Backup-Wächter als Inline-Alert über dem Inhalt.
     Vorher saß diese Meldung als flächig gefüllte Warnkarte im Sidebar-Fuß, direkt
     über „Abmelden". Drei Dinge sind daran anders:

     1. Der Ort. Die Sidebar ist Navigation; ein dauerhafter Konfigurationsfehler wird
        dort zwischen Menüpunkten nach drei Tagen zur Tapete. Der Alert steht jetzt über
        dem Inhalt, wo er zum Bildschirm gehört und nicht zur Wegweisung.
     2. Die Form. Kein Farbblock mit Emoji, sondern ein 3-px-Schweregrad-Streifen links
        auf ruhigem Grund plus ein Icon — auffällig durch Kante und Position.
     3. Die Handlung. Ein Alert ohne Weg zur Behebung ist eine Sackgasse. Der Text sagt,
        was zu tun ist, und der Knopf führt direkt auf den richtigen Reiter. -->
<script>
	import { onMount } from 'svelte';
	import { AlertTriangle, ArrowRight, X } from '@lucide/svelte';
	import { backupStatus } from '../../stores/backupStatus.svelte.js';
	import { uiStore } from '../../stores/uiStore.svelte.js';
	import Button from '../ui/Button.svelte';

	onMount(() => backupStatus.load());

	const critical = $derived(backupStatus.data?.status === 'critical');

	// Für die Sitzung wegklickbar. Eine Warnung, die auf JEDEM Bildschirm steht und
	// nie verschwindet, ist keine Warnung mehr, sondern Möblierung — und sie kostet
	// die oberste, wertvollste Bildschirmzeile.
	//
	// Bewusst NICHT gespeichert (weder localStorage noch Server): Beim nächsten Laden
	// steht sie wieder da. Wer den Schlüssel nicht hinterlegt, soll morgen erneut
	// erinnert werden — nur nicht den ganzen Tag lang. Ohne Persistenz gibt es
	// ausserdem keinen geteilten Zustand, der zwischen den Arbeitsplätzen ausdriften
	// könnte.
	let weggeklickt = $state(false);

	// Ziel ist die Betriebsbereitschaft: Dort steht der Backup-Befund samt Anleitung, und
	// sie verlangt dasselbe Recht wie dieser Alert (manage_settings). „Datenverwaltung"
	// (bis 24.08.2026) zeigte gar keinen Backup-Stand und braucht ein anderes Recht —
	// der Sprung wurde dann still verworfen.
	function openBetriebsbereitschaft() {
		uiStore.requestedSettingsTab = 'betrieb';
		uiStore.activeTab = 'settings';
	}
</script>

{#if backupStatus.needsAction && !weggeklickt}
	<div
		role="alert"
		class="no-print mb-5 flex items-start gap-3 rounded-md border border-slate-200 border-l-[3px] bg-white py-3 pr-4 pl-3.5
			{critical ? 'border-l-rose-600' : 'border-l-amber-500'}"
	>
		<AlertTriangle
			class="mt-0.5 h-4 w-4 shrink-0 {critical ? 'text-rose-600' : 'text-amber-600'}"
			aria-hidden="true"
		/>
		<div class="min-w-0 flex-1">
			<p class="text-sm font-semibold text-slate-800">{backupStatus.message}</p>
			<p class="mt-0.5 text-xs leading-relaxed text-slate-500">{backupStatus.hint}</p>
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
			variant="ghost"
			size="sm"
			onclick={() => (weggeklickt = true)}
			class="icon-btn mt-0.5 shrink-0 px-2 text-slate-400 hover:text-slate-600"
			aria-label="Hinweis für diese Sitzung ausblenden"
			data-tip="Für diese Sitzung ausblenden — beim nächsten Laden wieder da"
		>
			<X class="h-4 w-4" />
		</Button>
	</div>
{/if}
