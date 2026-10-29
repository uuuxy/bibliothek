<script>
	import Button from '../ui/Button.svelte';
	import Tabelle from '../ui/Tabelle.svelte';
	import Switch from '../ui/Switch.svelte';
	import Feld from '../ui/Feld.svelte';
	import LieferantZeile from './LieferantZeile.svelte';
	import LadeFehler from '../ui/LadeFehler.svelte';
	import { meldeFormular } from '../../stores/verlassenSchutz.svelte.js';
	import { loeschenBestaetigen } from '../../stores/bestaetigung.svelte.js';

	// ladefehler: der Grund, wenn die Lieferanten nicht kamen. „Keine Lieferanten angelegt" hieße
	// sonst, es gebe keine, und lüde zum doppelten Anlegen ein. Eine Liste von vorhin bleibt stehen.
	// Ohne onErneut zeigt LadeFehler keinen Knopf.
	/**
	 * @type {{
	 *   suppliers: import('../../stores/bestellStore.svelte.js').Supplier[],
	 *   ladefehler?: string,
	 *   onErneut?: () => void,
	 *   onAddSupplier: (name: string, email: string, kundennummer: string, haupt: boolean, kundennummerSchultraeger: string) => Promise<any>,
	 *   onEditSupplier: (id: string, aenderung: Record<string, any>) => Promise<any>,
	 *   onRemoveSupplier: (id: string) => Promise<any>
	 * }}
	 */
	let {
		suppliers,
		ladefehler = '',
		onErneut,
		onAddSupplier,
		onEditSupplier,
		onRemoveSupplier
	} = $props();

	let newName = $state('');
	let newEmail = $state('');
	let newCustNum = $state('');
	let newCustNumSchultraeger = $state('');
	let newIstHaupt = $state(false);
	meldeFormular(() =>
		[newName, newEmail, newCustNum, newCustNumSchultraeger].some((f) => f !== '')
	);

	/** @type {string|null} */
	let bearbeiteteKennung = $state(null);
	let legtAn = $state(false);

	// Geleert wird erst nach dem Anlegen: Weist der Server ab, bleibt die Eingabe zum Berichtigen
	// stehen (die Meldung zeigt apiFetch). Ein zweiter Klick in der Zeit legte denselben
	// Lieferanten noch einmal an.
	/** @param {SubmitEvent} e */
	async function speichere(e) {
		e.preventDefault();
		if (legtAn) return;
		legtAn = true;
		try {
			await onAddSupplier(newName, newEmail, newCustNum, newIstHaupt, newCustNumSchultraeger);
		} catch {
			return;
		} finally {
			legtAn = false;
		}
		newName = '';
		newEmail = '';
		newCustNum = '';
		newCustNumSchultraeger = '';
		newIstHaupt = false;
	}

	/** @type {string|null} */
	let loeschtKennung = $state(null);

	// Fragt vorher, weil „Löschen" neben „Bearbeiten" steht und nichts es zurücknimmt. Gesperrt, bis
	// die Liste nachgeladen ist: Ein zweiter Klick schickte ein zweites DELETE.
	/** @param {{ id: string, name: string }} s */
	async function loesche(s) {
		if (loeschtKennung) return;
		const folge =
			'Frühere Bestellungen behalten ihre Angaben. Der Lieferant lässt sich nicht wiederherstellen.';
		if (!(await loeschenBestaetigen(`Lieferant „${s.name}“ löschen?`, folge))) return;
		loeschtKennung = s.id;
		try {
			await onRemoveSupplier(s.id);
		} finally {
			loeschtKennung = null;
		}
	}

	// Die Zeile schließt erst, wenn gespeichert ist; sonst gingen die Änderungen mit ihr.
	/** @param {Record<string, any>} aenderung die seit dem Öffnen geänderten Felder */
	async function speichereAenderung(aenderung) {
		if (!bearbeiteteKennung) return;
		try {
			await onEditSupplier(bearbeiteteKennung, aenderung);
		} catch {
			return;
		}
		bearbeiteteKennung = null;
	}
</script>

<!-- Formular und Tabelle untereinander: Die Maske steht in den Einstellungen in der
     Detail-Spalte neben der Kategorienliste, max-w-4xl. Drei Spalten nebeneinander
     quetschten dort die Tabelle: Spaltenköpfe stießen zusammen, „Hauptlieferant" wurde
     abgeschnitten, ein Scrollbalken erschien. -->
<div class="flex flex-col gap-10">
	<div class="space-y-4 max-w-md">
		<h2 class="text-base font-bold text-on-surface border-b border-outline-variant pb-3">
			Neuer Lieferant
		</h2>
		<form onsubmit={speichere} class="space-y-4 text-sm">
			<Feld id="n" etikett="Name" bind:value={newName} required />
			<Feld id="e" etikett="E-Mail" type="email" bind:value={newEmail} required />
			<Feld id="c" etikett="Kundennummer" bind:value={newCustNum} required />
			<!-- Händler führen Lernmittel und Bibliothek oft als getrennte Kundenkonten
			     (anderer Nachlass, andere Rechnungsstelle). Bestellungen der Schülerbücherei
			     tragen dann diese Nummer; leer heißt: dieselbe wie oben (Migration 109). -->
			<Feld
				id="cs"
				etikett="Kundennummer Schülerbücherei (falls abweichend)"
				bind:value={newCustNumSchultraeger}
			/>
			<!-- EIN Schalter statt drei. Vorher standen hier „beklebt die Bücher",
			     „voreingestellt beim Bestellen" und „bekommt den Bestelllink" einzeln — drei
			     Haken für eine einzige Tatsache aus dem Schulalltag, und eine Kombination davon
			     war eine stille Falle: Bestelllink ohne „beklebt" hiess, der Händler klebt und
			     die Bibliothek druckt trotzdem noch einmal. Siehe Migration 066. -->
			<div class="flex items-start justify-between gap-4 border-t border-outline-variant pt-4">
				<label for="ist-hauptlieferant" class="cursor-pointer text-sm">
					<span class="block font-semibold text-on-surface">Hauptlieferant der Schule</span>
					<span class="mt-0.5 block text-xs text-on-surface-variant">
						Beim Bestellen vorausgewählt. Bekommt statt der reinen Bestellmail einen Link: wählt
						darüber große oder kleine Etiketten, beklebt die Bücher selbst und bestätigt damit die
						Bestellung — die Bestätigung erscheint automatisch in der Bestellhistorie. Seine Bücher
						stehen deshalb nicht auf der Nachdruck-Liste. Es kann immer nur einer sein; der
						bisherige wird zum normalen Händler.
					</span>
				</label>
				<Switch
					id="ist-hauptlieferant"
					bind:checked={newIstHaupt}
					etikett="Hauptlieferant der Schule"
				/>
			</div>
			<p class="text-xs text-on-surface-variant">
				Alle anderen Lieferanten bekommen einfach nur die Bestellmail.
			</p>
			<Button type="submit" size="lg" class="w-full" disabled={legtAn}>
				{legtAn ? 'Wird gespeichert …' : 'Lieferanten speichern'}
			</Button>
		</form>
	</div>

	<!-- min-w-0 + eigener Scrollbereich: Ohne beides schiebt ein langer Lieferantenname die
	     Tabelle über ihre Rasterzelle hinaus (Raster-Kinder haben min-width:auto und
	     schrumpfen nicht unter ihren Inhalt). Gemessen bei 1700 px Fensterbreite: Tabelle
	     1072 px in einer 909-px-Zelle, "Bearbeiten" landete bei 1760 px — ausserhalb des
	     Fensters und damit unerreichbar. Die Spalte war da, nur nicht anklickbar. -->
	<div class="space-y-4 min-w-0">
		<h2 class="text-base font-bold text-on-surface border-b border-outline-variant pb-3">
			Aktive Lieferanten
		</h2>
		{#if ladefehler && !suppliers.length}
			<LadeFehler titel="Lieferanten nicht geladen" text={ladefehler} onerneut={onErneut} />
		{:else if !suppliers.length}
			<div class="py-12 text-center text-on-surface-variant text-base">
				Keine Lieferanten angelegt.
			</div>
		{:else}
			<div class="overflow-x-auto">
				<Tabelle beschriftung="Lieferanten">
					<thead>
						<tr>
							<th>Lieferant</th>
							<th>Kontakt</th>
							<th class="text-right">Aktionen</th>
						</tr>
					</thead>
					<tbody>
						{#each suppliers as s (s.id)}
							<!-- {#key}: Der Bearbeiten-Zustand beginnt mit den Werten der Zeile —
							     nicht mit denen eines früheren Bearbeitens. -->
							{#key bearbeiteteKennung === s.id}
								<LieferantZeile
									{s}
									bearbeiten={bearbeiteteKennung === s.id}
									loescht={loeschtKennung === s.id}
									onEdit={(z) => (bearbeiteteKennung = z.id)}
									onRemove={loesche}
									onSave={speichereAenderung}
									onCancel={() => (bearbeiteteKennung = null)}
								/>
							{/key}
						{/each}
					</tbody>
				</Tabelle>
			</div>
		{/if}
	</div>
</div>
