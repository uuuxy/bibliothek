<!-- @component SeitenleisteFuss — angemeldete Person, Dunkles Design, Abmelden und der DSGVO-Hinweis.

     Eigene Datei: Die Navigation ist eine Aufgabe, der Fuß eine zweite.

     Abmelden ist nicht rot: Rot ist in Material 3 die Fehlerrolle, und Abmelden ist weder
     Fehler noch Datenverlust. In derselben Farbe wie die echten Gefahrenbereiche wäre der
     unwichtigste Knopf der Navigation der auffälligste, und der Unterschied zwischen „du
     gehst nach Hause" und „hier gehen Daten verloren" wäre eingeebnet. -->
<script>
	import { LogOut, ShieldCheck } from '@lucide/svelte';
	import Button from '../ui/Button.svelte';
	import Switch from '../ui/Switch.svelte';
	import { darstellung } from '../../stores/darstellung.svelte.js';

	/** @type {{ collapsed: boolean, benutzer: any, onLogout: () => void }} */
	let { collapsed, benutzer, onLogout } = $props();

	const initialen = $derived(
		benutzer ? benutzer.vorname[0] + (benutzer.nachname ? benutzer.nachname[0] : '') : 'U'
	);

	// Eine Aussage, zwei Darstellungen — der Text steht deshalb nur einmal hier.
	const DSGVO_HINWEIS = 'Scans werden nach 14 Tagen anonymisiert';
</script>

{#if collapsed}
	<div class="no-print flex shrink-0 flex-col items-center gap-3 p-4">
		<div
			class="bg-surface-container-highest text-on-surface-variant flex h-8 w-8 cursor-default items-center justify-center rounded-full text-xs font-bold"
			title="{benutzer?.vorname} {benutzer?.nachname} ({benutzer?.rolle})"
		>
			{initialen}
		</div>
		<button
			onclick={onLogout}
			class="icon-btn bg-surface-container text-on-surface-variant hover:bg-surface-container-high"
			data-tip="Abmelden"
			aria-label="Abmelden"
		>
			<LogOut class="h-4 w-4" aria-hidden="true" />
		</button>
		<span class="text-on-surface-variant cursor-default" title="DSGVO: {DSGVO_HINWEIS}">
			<ShieldCheck class="h-4 w-4" aria-label="DSGVO: {DSGVO_HINWEIS}" />
		</span>
	</div>
{:else}
	<div class="animate-fade-in no-print flex shrink-0 flex-col gap-3 p-4 text-left">
		<!-- Gemini, Googles Programm mit Seitenleiste, legt „Dark theme" unten in das Menü „Settings &
		     help". Hier wäre es der einzige Eintrag, deshalb steht der Schalter ohne Menü davor. Er gilt
		     für diesen Rechner, nicht für die Person, und steht deshalb über ihr, mit Abstand, statt
		     zwischen Name und Abmelden (Material 3, Spacing: „establish boundaries between unrelated
		     items"); in den Einstellungen gälte er für alle Arbeitsplätze.
		     Eingeklappt zeigt die Leiste ihn beim Überfahren. -->
		<div class="mb-2 flex items-center justify-between gap-3">
			<label for="dunkles-design" class="text-on-surface text-sm font-medium">Dunkles Design</label>
			<Switch
				id="dunkles-design"
				checked={darstellung.dunkel}
				onchange={(an) => darstellung.schalte(an)}
			/>
		</div>
		{#if benutzer}
			<div class="flex flex-col">
				<span class="text-on-surface truncate text-xs font-bold">
					{benutzer.vorname}
					{benutzer.nachname}
				</span>
				<span class="text-on-surface-variant text-label-small mt-0.5 font-medium capitalize">
					Rolle: {benutzer.rolle}
				</span>
			</div>
		{/if}
		<Button variant="secondary" size="sm" onclick={onLogout} class="w-full gap-1.5">
			<LogOut class="h-3.5 w-3.5" aria-hidden="true" />
			<span>Abmelden</span>
		</Button>

		<!-- Der Hinweis bleibt: Dass Scans nach 14 Tagen anonymisiert werden, ist in einer
		     Schulanwendung eine Zusage, keine Zierde. Nur das Schild-Emoji ist weg — es
		     rendert je nach Betriebssystem anders, lässt sich nicht einfärben und steht
		     deshalb auf der Ratsche in frontend-hygiene.test.js. -->
		<div
			class="text-on-surface-variant flex items-center justify-center gap-1.5 text-center text-sm"
			title={DSGVO_HINWEIS}
		>
			<ShieldCheck class="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
			<span>DSGVO anonymisiert</span>
		</div>
	</div>
{/if}
