<script>
	import Button from '../ui/Button.svelte';
	import Switch from '../ui/Switch.svelte';

	let { suppliers, onAddSupplier, onEditSupplier, onRemoveSupplier } = $props();

	let newName = $state('');
	let newEmail = $state('');
	let newCustNum = $state('');
	let newMitBarcode = $state(false);

	/** @type {string|null} */
	let editingId = $state(null);
	let editName = $state('');
	let editEmail = $state('');
	let editCustNum = $state('');
	let editMitBarcode = $state(false);

	/** @param {SubmitEvent} e */
	function handleSubmit(e) {
		e.preventDefault();
		onAddSupplier(newName, newEmail, newCustNum, newMitBarcode);
		newName = '';
		newEmail = '';
		newCustNum = '';
		newMitBarcode = false;
	}

	/** @param {{ id: string, name: string, email: string, customerNumber: string, liefert_mit_barcode?: boolean }} s */
	function startEdit(s) {
		editingId = s.id;
		editName = s.name;
		editEmail = s.email;
		editCustNum = s.customerNumber;
		// Ohne diese Zeile stünde beim Bearbeiten immer „aus" im Feld, und wer nur die
		// E-Mail korrigiert, schaltete die Beklebung des Händlers still ab.
		editMitBarcode = s.liefert_mit_barcode ?? false;
	}

	function cancelEdit() {
		editingId = null;
	}

	async function saveEdit() {
		if (!editingId) return;
		await onEditSupplier(editingId, editName, editEmail, editCustNum, editMitBarcode);
		editingId = null;
	}
</script>

<div class="grid grid-cols-1 md:grid-cols-3 gap-8 items-start overflow-y-auto">
	<div class="space-y-4">
		<h2 class="text-base font-bold text-slate-800 border-b border-slate-200 pb-3">
			Neuer Lieferant
		</h2>
		<form onsubmit={handleSubmit} class="space-y-4 text-base">
			<div class="space-y-1.5">
				<label for="n" class="block font-medium text-slate-600 text-sm">Name</label><input
					id="n"
					type="text"
					bind:value={newName}
					required
					class="w-full px-3 py-2.5 rounded-lg border border-slate-200 bg-white text-base"
				/>
			</div>
			<div class="space-y-1.5">
				<label for="e" class="block font-medium text-slate-600 text-sm">E-Mail</label><input
					id="e"
					type="email"
					bind:value={newEmail}
					required
					class="w-full px-3 py-2.5 rounded-lg border border-slate-200 bg-white text-base"
				/>
			</div>
			<div class="space-y-1.5">
				<label for="c" class="block font-medium text-slate-600 text-sm">Kundennummer</label><input
					id="c"
					type="text"
					bind:value={newCustNum}
					required
					class="w-full px-3 py-2.5 rounded-lg border border-slate-200 bg-white text-base"
				/>
			</div>
			<!-- Ein Zustand, kein Auswahlpunkt — deshalb ein Schalter und kein Häkchen (M3).
			     Die Folge steht im Klartext daneben: „Barcodes" allein liest jeder anders, und
			     diese Einstellung entscheidet, ob Exemplare auf der Nachdruck-Liste landen. -->
			<div class="flex items-start justify-between gap-4 pt-1">
				<label for="mit-barcode" class="cursor-pointer text-sm">
					<span class="block font-semibold text-slate-700">Händler beklebt die Bücher</span>
					<span class="mt-0.5 block text-xs text-slate-500">
						Der Barcodebogen geht wie bisher mit der Bestellung mit. Die Exemplare gelten dann
						als beklebt und erscheinen nicht auf der Liste der fehlenden Etiketten.
					</span>
				</label>
				<Switch id="mit-barcode" bind:checked={newMitBarcode} label="Händler beklebt die Bücher" />
			</div>
			<Button type="submit" size="lg" class="w-full">Lieferanten speichern</Button>
		</form>
	</div>

	<div class="md:col-span-2 space-y-4">
		<h2 class="text-base font-bold text-slate-800 border-b border-slate-200 pb-3">
			Aktive Lieferanten
		</h2>
		{#if !suppliers.length}
			<div class="py-12 text-center text-slate-400 text-base">Keine Lieferanten angelegt.</div>
		{:else}
			<table class="w-full text-left border-collapse text-base">
				<thead>
					<tr class="border-b border-slate-200 text-sm font-semibold text-slate-500">
						<th class="py-2.5">Name</th>
						<th class="py-2.5">E-Mail</th>
						<th class="py-2.5">Kundennummer</th>
						<th class="py-2.5">Etikettendruck</th>
						<th class="py-2.5 text-right">Aktionen</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-slate-100">
					{#each suppliers as s (s.id)}
						{#if editingId === s.id}
							<tr class="bg-blue-50/60">
								<td class="py-2 pr-2"
									><input
										type="text"
										bind:value={editName}
										class="w-full px-2 py-1.5 rounded border border-blue-300 text-sm"
									/></td
								>
								<td class="py-2 pr-2"
									><input
										type="email"
										bind:value={editEmail}
										class="w-full px-2 py-1.5 rounded border border-blue-300 text-sm"
									/></td
								>
								<td class="py-2 pr-2"
									><input
										type="text"
										bind:value={editCustNum}
										class="w-full px-2 py-1.5 rounded border border-blue-300 text-sm"
									/></td
								>
								<td class="py-2 pr-2">
									<Switch
										bind:checked={editMitBarcode}
										label="Händler beklebt die Bücher ({s.name})"
									/>
								</td>
								<td class="py-2 text-right whitespace-nowrap">
									<button
										onclick={saveEdit}
										class="text-blue-600 hover:text-blue-800 font-bold cursor-pointer text-sm mr-3"
										>Speichern</button
									>
									<button
										onclick={cancelEdit}
										class="text-slate-400 hover:text-slate-600 cursor-pointer text-sm"
										>Abbrechen</button
									>
								</td>
							</tr>
						{:else}
							<tr class="hover:bg-slate-50/40">
								<td class="py-3 font-bold text-slate-800">{s.name}</td>
								<td class="py-3 text-slate-600">{s.email}</td>
								<td class="py-3 text-slate-600">{s.customerNumber}</td>
								<!-- Zwei Werte, eine Sprache: Wer die Etiketten druckt. Vorher standen hier
								     ein grüner Chip gegen graue Kleinschrift — zwei Darstellungen für EINE
								     Ja/Nein-Angabe. Und „wir drucken" in jeder Zeile ließ das Auge viermal
								     lesen, um nichts zu erfahren; auffallen soll die Abweichung. -->
								<td class="py-3 text-slate-600">
									<span
										data-tip={s.liefert_mit_barcode
											? 'Die Bücher kommen beklebt an und stehen nicht auf der Nachdruck-Liste'
											: 'Die Etiketten druckt die Bibliothek selbst im Druck-Center'}
									>
										{s.liefert_mit_barcode ? 'Händler' : 'Bibliothek'}
									</span>
								</td>
								<td class="py-3 text-right whitespace-nowrap">
									<button
										onclick={() => startEdit(s)}
										class="text-slate-500 hover:text-blue-600 cursor-pointer text-sm mr-3"
										>Bearbeiten</button
									>
									<button
										onclick={() => onRemoveSupplier(s.id)}
										class="text-rose-600/80 hover:text-rose-700 cursor-pointer text-sm"
										>Löschen</button
									>
								</td>
							</tr>
						{/if}
					{/each}
				</tbody>
			</table>
		{/if}
	</div>
</div>
