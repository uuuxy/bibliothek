<script>
	import { apiFetch, apiClient } from '../../../../lib/apiFetch.js';
	import { onMount } from 'svelte';

	import ClassAssignmentSelector from './ClassAssignmentSelector.svelte';
	import ClassAssignmentBookGrid from './ClassAssignmentBookGrid.svelte';
	import ClassAssignmentSummary from './ClassAssignmentSummary.svelte';

	/**
	 * @type {{
	 *   isOpen?: boolean,
	 *   onClose?: (event?: MouseEvent) => void,
	 *   onSaved?: (res: { classes: string[], count: number }) => void,
	 *   initialGroup?: any
	 * }}
	 */
	let { isOpen = true, onClose = () => {}, onSaved = () => {}, initialGroup = null } = $props();

	let selectedClasses = $state(/** @type {string[]} */ ([]));
	let selectedBookIds = $state(/** @type {Set<number>} */ (new Set()));
	let books = $state(/** @type {any[]} */ ([]));
	let isSaving = $state(false);

	$effect(() => {
		if (selectedClasses.length === 0 && selectedBookIds.size > 0) {
			selectedBookIds = new Set();
		}
	});

	onMount(async () => {
		if (initialGroup) {
			selectedClasses = [initialGroup.className];
			selectedBookIds = new Set(initialGroup.books.map((/** @type {any} */ b) => b.id));
		}

		try {
			const res = await apiFetch('/api/books');
			if (res.ok) {
				const json = await res.json();
				if (json.data) books = json.data;
			}
		} catch (e) {
			console.error('Fehler beim Laden der Bücher:', e);
		}
	});

	const selectedBooksList = $derived(
		books.filter((/** @type {any} */ b) => selectedBookIds.has(b.id))
	);

	/**
	 * @param {number} id
	 */
	function toggleBook(id) {
		if (selectedBookIds.has(id)) {
			selectedBookIds = new Set([...selectedBookIds].filter((bId) => bId !== id));
		} else {
			selectedBookIds = new Set([...selectedBookIds, id]);
		}
	}

	async function saveAssignments() {
		if (selectedClasses.length === 0) return;
		if (!initialGroup && selectedBookIds.size === 0) return;

		isSaving = true;
		try {
			const endpoint = initialGroup ? '/api/admin/class-books' : '/api/admin/class-books/add';
			const payload = {
				classNames: selectedClasses,
				bookIds: Array.from(selectedBookIds),
				oldClassName: initialGroup ? initialGroup.className : undefined
			};

			const headers = /** @type {Record<string, string>} */ ({
				'Content-Type': 'application/json'
			});

			const res = await apiFetch(endpoint, {
				method: 'POST',
				headers,
				body: JSON.stringify(payload)
			});

			if (res.ok) {
				onSaved({
					classes: selectedClasses,
					count: selectedBookIds.size
				});
				onClose();
			} else {
				console.error('Server-Fehler beim Speichern');
				alert('Ein Fehler ist aufgetreten. Bitte erneut versuchen.');
			}
		} catch (e) {
			console.error('Netzwerkfehler', e);
			alert('Fehler beim Speichern der Zuweisung.');
		} finally {
			isSaving = false;
		}
	}
</script>

{#if isOpen}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center p-0 sm:p-4 bg-black/30 backdrop-blur-sm animate-in fade-in duration-200"
		onclick={(e) => {
			if (e.target === e.currentTarget) onClose();
		}}
	>
		<div
			class="bg-white rounded-none sm:rounded-[32px] shadow-2xl w-full lg:w-[1200px] max-w-[100vw] lg:max-w-[90vw] h-dvh sm:h-[90vh] lg:h-[850px] max-h-dvh lg:max-h-[95vh] p-4 sm:p-6 lg:p-8 flex flex-col lg:flex-row gap-6 lg:gap-8 relative overflow-hidden animate-in zoom-in-95 duration-200"
		>
			<!-- Background Particles -->
			<div class="absolute inset-0 opacity-40 pointer-events-none">
				<div class="particle p1"></div>
				<div class="particle p2"></div>
				<div class="particle p3"></div>
			</div>

			<!-- Left Content Area -->
			<div class="grow flex flex-col gap-4 sm:gap-6 relative z-10 w-full overflow-hidden">
				<div class="shrink-0">
					<h2 class="text-2xl sm:text-3xl font-bold tracking-tight text-gray-900 leading-none">
						Klasse & Bücher zuweisen
					</h2>
					<p class="mt-1 sm:mt-2 text-gray-500 font-medium text-sm sm:text-lg">
						Wähle Zielklassen und die entsprechenden Schulbücher aus.
					</p>
				</div>

				<div
					class="flex-1 overflow-y-auto [&::-webkit-scrollbar]:w-1.5 [&::-webkit-scrollbar-track]:bg-transparent [&::-webkit-scrollbar-thumb]:bg-emerald-200 [&::-webkit-scrollbar-thumb]:rounded-full pr-4 pb-4"
				>
					<ClassAssignmentSelector bind:selectedClasses />
					<ClassAssignmentBookGrid {books} bind:selectedBookIds />
				</div>
			</div>

			<!-- Right Sidebar Area -->
			<aside
				class="w-full lg:w-[340px] flex-none lg:shrink-0 flex flex-col gap-4 relative z-10 border-t lg:border-t-0 lg:border-l border-gray-100 pt-4 lg:pt-0 lg:pl-8 h-[40dvh] lg:h-auto"
			>
				<ClassAssignmentSummary
					{selectedClasses}
					{selectedBookIds}
					{selectedBooksList}
					{isSaving}
					isUpdate={!!initialGroup}
					onToggleBook={toggleBook}
					onsave={saveAssignments}
				/>

				<button
					onclick={onClose}
					class="mt-auto w-full text-center py-3 text-emerald-800 font-bold text-lg hover:text-emerald-900 transition-colors uppercase tracking-widest bg-transparent border-none cursor-pointer"
				>
					Abbrechen
				</button>
			</aside>

			<!-- Close Button (Absolute Top Right) -->
			<button
				aria-label="Schließen"
				onclick={onClose}
				class="absolute top-4 sm:top-6 right-4 sm:right-6 p-2 hover:bg-gray-100 hover:text-gray-900 rounded-full transition-all duration-200 text-gray-400 z-20 cursor-pointer border-none bg-transparent"
			>
				<svg
					xmlns="http://www.w3.org/2000/svg"
					width="24"
					height="24"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2.5"
					stroke-linecap="round"
					stroke-linejoin="round"
					><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"
					></line></svg
				>
			</button>
		</div>
	</div>
{/if}
