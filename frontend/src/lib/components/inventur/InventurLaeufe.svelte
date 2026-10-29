<!-- @component InventurLaeufe — die laufenden und die früheren Inventuren unter „Keine Inventur
     aktiv". Kam eine Liste nicht, steht ihr Grund an ihrer Stelle: Ohne die laufenden lüde die
     Seite zu einem zweiten Lauf ein, und der Server verwiese „unten" auf eine Liste, die fehlt. -->
<script>
	import { Trash } from '@lucide/svelte';
	import Button from '../ui/Button.svelte';
	import LadeFehler from '../ui/LadeFehler.svelte';
	import Menue from '../ui/Menue.svelte';
	import { bestaetigen, loeschenBestaetigen } from '../../stores/bestaetigung.svelte.js';
	import { formatDatum } from '../../utils/format.js';

	/**
	 * Die Felder der Inventur-Fassade (useInventur), die diese Abschnitte lesen.
	 * @type {{ ablauf: {
	 *   offeneLaeufe: any[],
	 *   frueher: { laeufe: any[], weitere: number, laedtMehr: boolean, lade: () => void,
	 *     ladeMehr: () => void, loesche: (inventur: any) => void },
	 *   nichtGeladen: { laeufe: string, frueher: string },
	 *   ladeFruehereLaeuft: boolean,
	 *   ladeOffeneLaeufe: () => void,
	 *   setzeLaufFort: (session: any) => void,
	 *   verwerfeLauf: (session: any) => void,
	 *   zeigeFrueherenFehlbestand: (inventur: any) => void
	 * } }}
	 */
	let { ablauf } = $props();

	// Verwerfen lässt sich nicht zurücknehmen, und der Knopf steht neben „Fortsetzen": Ein
	// Fehlklick beendete sonst die Inventur, und alle Scans bis dahin zählten nicht mehr.
	/** @param {any} session */
	async function verwerfen(session) {
		const ja = await bestaetigen({
			titel: `Inventur „${session.label}“ verwerfen?`,
			text: 'Sie lässt sich danach nicht fortsetzen, eine neue Inventur beginnt wieder bei null. Kein Buch wird als verloren markiert.',
			aktion: 'Verwerfen',
			gefaehrlich: true
		});
		if (ja) ablauf.verwerfeLauf(session);
	}

	// Gelöscht werden Lauf und Bericht; was er an den Büchern gebucht hat, bleibt (Abgangsbuch).
	/** @param {any} inventur */
	async function loeschen(inventur) {
		const folge = inventur.verworfen
			? 'Sie wurde verworfen und hat nichts gebucht.'
			: inventur.verluste > 0
				? `Ihr Fehlbestandsbericht wird mit gelöscht. Die ${inventur.verluste} fehlenden Bücher bleiben als verloren gebucht; ein Buch, das sich findet, bucht ein Scan an der Theke zurück.`
				: 'Sie war vollständig und hat keinen Fehlbestand.';
		const ja = await loeschenBestaetigen(
			`Inventur „${inventur.label}“ löschen?`,
			`${folge} Das lässt sich nicht rückgängig machen.`
		);
		if (ja) ablauf.frueher.loesche(inventur);
	}
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
					<Button variant="secondary" size="sm" onclick={() => verwerfen(session)}>
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
		onerneut={ablauf.frueher.lade}
	/>
{:else if ablauf.frueher.laeufe.length > 0}
	<div class="w-full max-w-4xl mx-auto text-left space-y-2 pt-4">
		<h4 class="text-sm font-semibold text-on-surface-variant">Frühere Inventuren</h4>
		{#each ablauf.frueher.laeufe as inventur (inventur.session_id)}
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
				<!-- M3 (Lists): „Place supplementary actions, like a bookmark or menu, in the
				     trailing position." Löschen steht im Menü, nicht als Knopf neben „Fehlbestand". -->
				<div class="flex shrink-0 items-center gap-2">
					{#if !inventur.verworfen}
						<Button
							variant="secondary"
							size="sm"
							disabled={ablauf.ladeFruehereLaeuft}
							onclick={() => ablauf.zeigeFrueherenFehlbestand(inventur)}
						>
							Fehlbestand
						</Button>
					{/if}
					<Menue
						etikett="Aktionen für „{inventur.label}“"
						eintraege={[{ id: 'loeschen', text: 'Löschen', icon: Trash }]}
						onwahl={() => loeschen(inventur)}
					/>
				</div>
			</div>
		{/each}
		<!-- Wie „Mehr laden" in der Katalogsuche: Die Liste bleibt kurz, kein Lauf ist unerreichbar. -->
		{#if ablauf.frueher.weitere > 0}
			<div class="flex justify-center pt-2">
				<Button
					variant="secondary"
					disabled={ablauf.frueher.laedtMehr}
					onclick={ablauf.frueher.ladeMehr}
				>
					Mehr laden ({ablauf.frueher.weitere} weitere)
				</Button>
			</div>
		{/if}
	</div>
{/if}
