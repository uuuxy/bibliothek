<!-- @component InventurLaeufe — die laufenden und die früheren Inventuren unter „Keine Inventur
     aktiv". Kam eine Liste nicht, steht ihr Grund an ihrer Stelle: Ohne die laufenden lüde die
     Seite zu einem zweiten Lauf ein, und der Server verwiese „unten" auf eine Liste, die fehlt. -->
<script>
	import Button from '../ui/Button.svelte';
	import LadeFehler from '../ui/LadeFehler.svelte';
	import { formatDatum } from '../../utils/format.js';

	/**
	 * Die Felder der Inventur-Fassade (useInventur), die diese Abschnitte lesen.
	 * @type {{ ablauf: {
	 *   offeneLaeufe: any[],
	 *   abgeschlosseneInventuren: any[],
	 *   nichtGeladen: { laeufe: string, frueher: string },
	 *   ladeFruehereLaeuft: boolean,
	 *   ladeOffeneLaeufe: () => void,
	 *   ladeFruehereInventuren: () => void,
	 *   setzeLaufFort: (session: any) => void,
	 *   verwerfeLauf: (session: any) => void,
	 *   zeigeFrueherenFehlbestand: (inventur: any) => void
	 * } }}
	 */
	let { ablauf } = $props();
</script>

{#if ablauf.nichtGeladen.laeufe}
	<LadeFehler
		titel="Laufende Inventuren nicht geladen"
		text={ablauf.nichtGeladen.laeufe}
		onerneut={ablauf.ladeOffeneLaeufe}
	/>
{:else if ablauf.offeneLaeufe.length > 0}
	<div class="w-full max-w-4xl mx-auto text-left space-y-2 pt-4">
		<h4 class="text-sm font-semibold text-on-surface-variant">Laufende Inventuren</h4>
		{#each ablauf.offeneLaeufe as session (session.session_id)}
			<div
				class="flex items-center justify-between gap-3 p-3 bg-warning-container text-on-warning-container rounded-lg"
			>
				<div class="min-w-0">
					<div class="font-semibold truncate">{session.label}</div>
					<div class="text-xs">
						{session.erwartet - session.fehlend} / {session.erwartet} erfasst · seit {session.gestartet_am?.slice(
							0,
							16
						)}
					</div>
				</div>
				<div class="flex items-center gap-2 shrink-0">
					<Button size="sm" onclick={() => ablauf.setzeLaufFort(session)}>Fortsetzen</Button>
					<Button variant="secondary" size="sm" onclick={() => ablauf.verwerfeLauf(session)}>
						Verwerfen
					</Button>
				</div>
			</div>
		{/each}
	</div>
{/if}

<!-- Frühere Inventuren: der Weg zurück zum Fehlbestandsbericht, auch nach dem Neuladen und am
     zweiten Arbeitsplatz, der mit der Liste ins Regal geht; die Daten liegen auf dem Server.
     Eine verworfene Inventur hat nichts gebucht und keinen Bericht; ihr Bereich ist nicht
     geprüft. -->
{#if ablauf.nichtGeladen.frueher}
	<LadeFehler
		titel="Frühere Inventuren nicht geladen"
		text={ablauf.nichtGeladen.frueher}
		onerneut={ablauf.ladeFruehereInventuren}
	/>
{:else if ablauf.abgeschlosseneInventuren.length > 0}
	<div class="w-full max-w-4xl mx-auto text-left space-y-2 pt-4">
		<h4 class="text-sm font-semibold text-on-surface-variant">Frühere Inventuren</h4>
		{#each ablauf.abgeschlosseneInventuren as inventur (inventur.session_id)}
			<div
				class="flex items-center justify-between gap-3 px-3 py-2.5 border-b border-outline-variant"
			>
				<div class="min-w-0">
					<div class="font-semibold text-on-surface truncate">{inventur.label}</div>
					<div class="text-xs text-on-surface-variant">
						{formatDatum(inventur.abgeschlossen_am)} · {inventur.erfasst} erfasst ·
						{#if inventur.verworfen}
							verworfen
						{:else if inventur.verluste > 0}
							<span class="text-error font-semibold">{inventur.verluste} fehlend</span>
						{:else}
							vollständig
						{/if}
					</div>
				</div>
				{#if !inventur.verworfen}
					<Button
						variant="secondary"
						size="sm"
						class="shrink-0"
						disabled={ablauf.ladeFruehereLaeuft}
						onclick={() => ablauf.zeigeFrueherenFehlbestand(inventur)}
					>
						Fehlbestand
					</Button>
				{/if}
			</div>
		{/each}
	</div>
{/if}
