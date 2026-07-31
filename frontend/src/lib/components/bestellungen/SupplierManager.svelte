<script>
	import Button from '../ui/Button.svelte';

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
			<!-- Diese Einstellung entscheidet, ob die gelieferten Exemplare auf der
			     Nachdruck-Liste landen. Deshalb steht die Folge im Klartext daneben und nicht
			     nur ein Häkchen: „Barcodes" allein liest jeder anders. -->
			<label class="flex cursor-pointer gap-2.5 rounded-lg border border-slate-200 bg-slate-50/60 p-3">
				<input type="checkbox" bind:checked={newMitBarcode} class="mt-0.5 h-4 w-4 shrink-0" />
				<span class="text-sm">
					<span class="block font-semibold text-slate-700">Händler beklebt die Bücher</span>
					<span class="block text-xs text-slate-500">
						Der Barcodebogen geht wie bisher mit der Bestellung mit. Die Exemplare gelten dann
						als beklebt und erscheinen nicht auf der Liste der fehlenden Etiketten.
					</span>
				</span>
			</label>
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
						<th class="py-2.5">Etiketten</th>
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
									<label class="flex cursor-pointer items-center gap-2 text-xs text-slate-600">
										<input type="checkbox" bind:checked={editMitBarcode} class="h-4 w-4 shrink-0" />
										Händler beklebt
									</label>
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
								<td class="py-3">
									{#if s.liefert_mit_barcode}
										<span
											class="inline-flex items-center rounded-full bg-emerald-50 px-2 py-0.5 text-xs font-semibold text-emerald-700"
											data-tip="Die Bücher kommen beklebt an und stehen nicht auf der Nachdruck-Liste"
										>
											Händler beklebt
										</span>
									{:else}
										<span class="text-xs text-slate-400">wir drucken</span>
									{/if}
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
