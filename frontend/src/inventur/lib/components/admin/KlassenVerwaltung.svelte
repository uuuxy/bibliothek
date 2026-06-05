<script>
	import { apiFetch } from '../../../../lib/apiFetch.js';
	import { onMount } from "svelte";
	import BuchAuswahlListe from "$lib/components/admin/BuchAuswahlListe.svelte";
	import KlassenNamenEditor from "$lib/components/admin/KlassenNamenEditor.svelte";

	let { initialGroup, onClose, onSaved } = $props();

	/** @type {string[]} */
	let classNames = $state([]);
	let classInput = $state("");
	/** @type {any[]} */
	let allBooks = $state([]);
	let selectedBookIds = $state(new Set());
	let loading = $state(true);
	let saving = $state(false);
	/** @type {string|null} */
	let error = $state(null);

	onMount(async () => {
		try {
			const res = await apiFetch("/api/books", {
				credentials: "include",
			});
			if (!res.ok) throw new Error("Fehler beim Laden der Bücher");
			const json = await res.json();
			allBooks = json.data || [];
		} catch (err) {
			error = err instanceof Error ? err.message : String(err);
		} finally {
			loading = false;
		}
	});

	$effect(() => {
		if (initialGroup) {
			classNames = [initialGroup.className];
			selectedBookIds = new Set(initialGroup.books.map((/** @type {{id: string}} */ b) => b.id));
		}
	});

	/** @param {KeyboardEvent} e */
	function handleKeydown(e) {
		if (e.key === "Enter" || e.key === ",") {
			e.preventDefault();
			addClassName();
		} else if (e.key === "Backspace" && classInput === "" && classNames.length > 0) {
			classNames.pop();
		}
	}

	function handleBlur() {
		addClassName();
	}

	function addClassName() {
		// Split by comma in case user pastes comma-separated values
		const values = classInput.split(",").map(v => v.trim()).filter(v => v !== "");
		for (const val of values) {
			if (!classNames.includes(val)) {
				classNames.push(val);
			}
		}
		classInput = "";
	}

	/** @param {number} index */
	function removeClassName(index) {
		classNames.splice(index, 1);
	}

	async function save() {
		addClassName(); // Ensure any pending input is captured
		if (classNames.length === 0) {
			alert("Bitte mindestens einen Klassennamen eingeben (z.B. 5G1)");
			return;
		}
		saving = true;
		
		/** @type {{classNames: string[], bookIds: string[], oldClassName?: string}} */
		const payload = {
			classNames: classNames,
			bookIds: Array.from(selectedBookIds),
		};

		if (initialGroup && !classNames.includes(initialGroup.className)) {
			payload.oldClassName = initialGroup.className;
		} else if (initialGroup) {
			// Even if they renamed it and kept the old one, we only really delete oldClassName if we actually rename.
			// But for safety:
			payload.oldClassName = initialGroup.className;
		}

		try {
			const res = await apiFetch("/api/admin/class-books", {
				method: "POST",
				credentials: "include",
				headers: {
					"Content-Type": "application/json",
				},
				body: JSON.stringify(payload),
			});
			if (!res.ok) throw new Error("Fehler beim Speichern");
			onSaved();
		} catch (err) {
			alert(err instanceof Error ? err.message : String(err));
		} finally {
			saving = false;
		}
	}
</script>

<div
	class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4"
>
	<div
		class="bg-white rounded-2xl shadow-2xl w-full max-w-2xl max-h-[90vh] flex flex-col overflow-hidden"
	>
		<div
			class="p-6 border-b border-gray-100 flex justify-between items-center bg-gray-50"
		>
			<h2 class="text-xl font-bold text-gray-800">
				Klasse & Bücher zuweisen
			</h2>
			<button
				onclick={onClose}
				class="text-gray-400 hover:text-gray-600"
				aria-label="Schließen"
			>
				<svg
					class="w-6 h-6"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					><path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M6 18L18 6M6 6l12 12"
					/></svg
				>
			</button>
		</div>

		<div class="p-6 flex-1 overflow-y-auto">
			{#if loading}
				<div class="flex justify-center py-8">
					<div
						class="animate-spin rounded-full h-8 w-8 border-b-2 border-emerald-600"
					></div>
				</div>
			{:else if error}
				<div class="text-red-500 p-4 bg-red-50 rounded-lg">{error}</div>
			{:else}
				<KlassenNamenEditor
					bind:classNames
					bind:classInput
					onKeydown={handleKeydown}
					onBlur={handleBlur}
					onRemove={removeClassName}
				/>

				<BuchAuswahlListe {allBooks} bind:selectedBookIds />
			{/if}
		</div>

		<div
			class="p-6 border-t border-gray-100 bg-gray-50 flex justify-end gap-3"
		>
			<button
				onclick={onClose}
				class="px-5 py-2 text-gray-600 hover:bg-gray-200 rounded-lg font-medium transition-colors"
				>Abbrechen</button
			>
			<button
				onclick={save}
				disabled={saving || (classNames.length === 0 && classInput.trim() === "")}
				class="px-5 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg font-medium transition-colors disabled:opacity-50 flex items-center gap-2 shadow-sm shadow-emerald-600/20"
			>
				{#if saving}
					<div
						class="animate-spin rounded-full h-4 w-4 border-b-2 border-white"
					></div>
				{/if}
				Speichern
			</button>
		</div>
	</div>
</div>
