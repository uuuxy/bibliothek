<!-- @component BestelllinkHinweis — der Wächter über dem Bestellwesen.
     Es gibt einen Hauptlieferanten, aber keine öffentliche Adresse: Seine Bestellmails
     gehen dann ohne Bestätigungs-Link raus, und die Bestellhistorie wartet auf eine
     Bestätigung, die niemand geben kann.

     Der Hinweis steht VOR dem Bestellen, nicht als Warnung danach (die gibt es zusätzlich,
     siehe bestellVersandMeldung im Backend): Eine schon versendete Mail lässt sich nicht
     zurückholen — der Link muss dann von Hand aus der Bestellhistorie nachgereicht werden.

     Form wie BackupAlert: 3-px-Schweregradstreifen auf ruhigem Grund, ein Satz zur Lage,
     ein Satz zur Folge, ein Knopf, der genau dorthin führt, wo es zu beheben ist. -->
<script>
	import { AlertTriangle, ArrowRight } from '@lucide/svelte';
	import { orderStore } from '../../stores/orderStore.svelte.js';
	import { authStore } from '../../stores/authStore.svelte.js';
	import { hatRecht } from '../../menu.js';
	import { uiStore } from '../../stores/uiStore.svelte.js';
	import Button from '../ui/Button.svelte';

	// Nicht wegklickbar, anders als der Backup-Wächter: Der steht auf JEDEM Bildschirm und
	// wird zur Möblierung. Dieser hier steht nur im Bestellwesen — also genau dort, wo die
	// fehlende Einstellung gleich Schaden anrichtet, und nur so lange, bis sie da ist.
	const sichtbar = $derived(orderStore.bestelllinkOhneAdresse);
	// Kann der Leser die Einstellungen selbst öffnen? Dasselbe Recht wie der Menüpunkt.
	const istAdmin = $derived(hatRecht(authStore.currentUser, 'manage_settings'));

	// Dritter Zustand neben „alles gut" und „Adresse fehlt": Die Frage liess sich gar
	// nicht beantworten. Bis zum 08.08.2026 gab es ihn nicht — loadKonfiguration fing den
	// Fehler still ab, das Feld blieb auf seinem Anfangswert false, und dieser Waechter
	// verschwand. Ausgerechnet bei einer Stoerung behauptete das Bestellwesen also
	// „alles in Ordnung", und wer dann bestellte, verschickte Mails ohne
	// Bestaetigungs-Link.
	//
	// Aufgefallen ist es, weil der Server unter der Last der E2E-Suite mit 429 antwortete
	// (Ratenbegrenzung, korrektes Verhalten) — im Testlauf sah das aus wie ein
	// sprunghafter Test. Der Toast aus apiFetch meldet die Stoerung zwar, aber er
	// verschwindet; wer danach auf den Bildschirm sieht, braucht die Auskunft dort.
	const ungeprueft = $derived(!orderStore.konfigurationGeladen);

	function einstellungenOeffnen() {
		uiStore.requestedSettingsTab = 'erreichbarkeit';
		uiStore.activeTab = 'settings';
	}
</script>

{#if sichtbar}
	<div
		role="alert"
		class="no-print flex items-start gap-3 rounded-md border border-slate-200 border-l-[3px] border-l-amber-500 bg-white py-3 pr-4 pl-3.5 shadow-xs"
	>
		<AlertTriangle class="mt-0.5 h-4 w-4 shrink-0 text-amber-600" aria-hidden="true" />
		<div class="min-w-0 flex-1">
			<p class="text-sm font-semibold text-slate-800">
				Bestellungen gehen ohne Bestätigungs-Link raus
			</p>
			<p class="mt-0.5 text-xs leading-relaxed text-slate-500">
				Der Hauptlieferant soll die Etikettengröße selbst wählen und damit bestätigen — dafür
				braucht das System die öffentliche Adresse, unter der er es von außen erreicht
				(Einstellungen → Allgemein → Schule).
				{#if !istAdmin}
					Ein Administrator kann sie hinterlegen.
				{/if}
			</p>
		</div>
		{#if istAdmin}
			<Button variant="secondary" size="sm" onclick={einstellungenOeffnen} class="mt-0.5 shrink-0">
				Einstellungen öffnen
				<ArrowRight class="h-3.5 w-3.5" />
			</Button>
		{/if}
	</div>
{:else if ungeprueft}
	<!-- Grauer Streifen statt gelbem: Das ist keine Fehlmeldung über die Anlage, sondern
	     eine über diese Ansicht. Sie sagt, was sie nicht weiß, statt Ruhe vorzutäuschen. -->
	<div
		role="status"
		class="no-print flex items-start gap-3 rounded-md border border-slate-200 border-l-[3px] border-l-slate-400 bg-white py-3 pr-4 pl-3.5 shadow-xs"
	>
		<AlertTriangle class="mt-0.5 h-4 w-4 shrink-0 text-slate-400" aria-hidden="true" />
		<div class="min-w-0 flex-1">
			<p class="text-sm font-semibold text-slate-800">Bestell-Einstellungen nicht geladen</p>
			<p class="mt-0.5 text-xs leading-relaxed text-slate-500">
				Ob die Bestellmails einen Bestätigungs-Link tragen, konnte gerade nicht geprüft werden.
				Seite neu laden — bleibt der Hinweis, vor dem Bestellen nachsehen.
			</p>
		</div>
	</div>
{/if}
