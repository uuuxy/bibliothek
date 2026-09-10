<script>
	import Button from '../ui/Button.svelte';
	import Tabelle from '../ui/Tabelle.svelte';
	import Switch from '../ui/Switch.svelte';
	import Feld from '../ui/Feld.svelte';
	import LieferantZeile from './LieferantZeile.svelte';

	let { suppliers, onAddSupplier, onEditSupplier, onRemoveSupplier } = $props();

	let newName = $state('');
	let newEmail = $state('');
	let newCustNum = $state('');
	let newCustNumSchultraeger = $state('');
	let newIstHaupt = $state(false);

	/** @type {string|null} */
	let editingId = $state(null);

	/** @param {SubmitEvent} e */
	function handleSubmit(e) {
		e.preventDefault();
		onAddSupplier(newName, newEmail, newCustNum, newIstHaupt, newCustNumSchultraeger);
		newName = '';
		newEmail = '';
		newCustNum = '';
		newCustNumSchultraeger = '';
		newIstHaupt = false;
	}

	/** @param {{ name: string, email: string, customerNumber: string, istHauptlieferant: boolean, kundennummerSchultraeger: string }} w */
	async function saveEdit(w) {
		if (!editingId) return;
		await onEditSupplier(
			editingId,
			w.name,
			w.email,
			w.customerNumber,
			w.istHauptlieferant,
			w.kundennummerSchultraeger
		);
		editingId = null;
	}
</script>

<!-- Formular und Tabelle UNTEREINANDER: Seit dem Umzug in die Einstellungen (25.08.2026)
     steht die Maske in der Detail-Spalte neben der Kategorienliste, max-w-4xl. Drei
     Spalten nebeneinander quetschten dort die Tabelle: Spaltenköpfe stießen zusammen,
     „Hauptlieferant" wurde abgeschnitten, ein Scrollbalken erschien. -->
<div class="flex flex-col gap-10">
	<div class="space-y-4 max-w-md">
		<h2 class="text-base font-bold text-slate-800 border-b border-slate-200 pb-3">
			Neuer Lieferant
		</h2>
		<form onsubmit={handleSubmit} class="space-y-4 text-sm">
			<Feld id="n" label="Name" bind:value={newName} required />
			<Feld id="e" label="E-Mail" type="email" bind:value={newEmail} required />
			<Feld id="c" label="Kundennummer" bind:value={newCustNum} required />
			<!-- Händler führen Lernmittel und Bibliothek oft als getrennte Kundenkonten
			     (anderer Nachlass, andere Rechnungsstelle). Bestellungen der Schülerbücherei
			     tragen dann diese Nummer; leer heißt: dieselbe wie oben (Migration 109). -->
			<Feld
				id="cs"
				label="Kundennummer Schülerbücherei (falls abweichend)"
				bind:value={newCustNumSchultraeger}
			/>
			<!-- EIN Schalter statt drei. Vorher standen hier „beklebt die Bücher",
			     „voreingestellt beim Bestellen" und „bekommt den Bestelllink" einzeln — drei
			     Haken für eine einzige Tatsache aus dem Schulalltag, und eine Kombination davon
			     war eine stille Falle: Bestelllink ohne „beklebt" hiess, der Händler klebt und
			     die Bibliothek druckt trotzdem noch einmal. Siehe Migration 066. -->
			<div class="flex items-start justify-between gap-4 border-t border-slate-100 pt-4">
				<label for="ist-hauptlieferant" class="cursor-pointer text-sm">
					<span class="block font-semibold text-slate-700">Hauptlieferant der Schule</span>
					<span class="mt-0.5 block text-xs text-slate-500">
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
					label="Hauptlieferant der Schule"
				/>
			</div>
			<p class="text-xs text-slate-500">
				Alle anderen Lieferanten bekommen einfach nur die Bestellmail.
			</p>
			<Button type="submit" size="lg" class="w-full">Lieferanten speichern</Button>
		</form>
	</div>

	<!-- min-w-0 + eigener Scrollbereich: Ohne beides schiebt ein langer Lieferantenname die
	     Tabelle über ihre Rasterzelle hinaus (Raster-Kinder haben min-width:auto und
	     schrumpfen nicht unter ihren Inhalt). Gemessen bei 1700 px Fensterbreite: Tabelle
	     1072 px in einer 909-px-Zelle, "Bearbeiten" landete bei 1760 px — ausserhalb des
	     Fensters und damit unerreichbar. Die Spalte war da, nur nicht anklickbar. -->
	<div class="space-y-4 min-w-0">
		<h2 class="text-base font-bold text-slate-800 border-b border-slate-200 pb-3">
			Aktive Lieferanten
		</h2>
		{#if !suppliers.length}
			<div class="py-12 text-center text-slate-400 text-base">Keine Lieferanten angelegt.</div>
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
							{#key editingId === s.id}
								<LieferantZeile
									{s}
									bearbeiten={editingId === s.id}
									onEdit={(z) => (editingId = z.id)}
									onRemove={onRemoveSupplier}
									onSave={saveEdit}
									onCancel={() => (editingId = null)}
								/>
							{/key}
						{/each}
					</tbody>
				</Tabelle>
			</div>
		{/if}
	</div>
</div>
