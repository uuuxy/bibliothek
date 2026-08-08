<script>
	import { apiFetch } from '../../../../lib/apiFetch.js';
	import { onMount } from 'svelte';

	import ClassAssignmentSelector from './ClassAssignmentSelector.svelte';
	import ClassAssignmentBookGrid from './ClassAssignmentBookGrid.svelte';
	import ClassAssignmentSummary from './ClassAssignmentSummary.svelte';
	import { TriangleAlert, X } from '@lucide/svelte';

	/**
	 * @type {{
	 *   isOpen?: boolean,
	 *   onClose?: (event?: MouseEvent) => void,
	 *   onSaved?: (res: { classes: string[], count: number }) => void,
	 *   initialGroup?: any,
	 *   vorhandeneGruppen?: any[]
	 * }}
	 */
	let {
		isOpen = true,
		onClose = () => {},
		onSaved = () => {},
		initialGroup = null,
		vorhandeneGruppen = []
	} = $props();

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

	// Der Speicherpfad ERSETZT: UpdateClassBooks loescht die Zuweisungen ALLER Zielklassen
	// und schreibt danach die Auswahl hinein (inventur/datenbank_klassen.go). Mehrere
	// Klassen gleichzeitig zu bearbeiten ist also seit jeher moeglich — nur sagte nichts,
	// dass die zusaetzlich gewaehlte Klasse dabei ihren bisherigen Satz verliert. Wer zu
	// 05F1 noch 06A2 dazunimmt, loescht deren neun Buecher, ohne es zu sehen.
	const ueberschriebeneKlassen = $derived(
		selectedClasses
			.filter((/** @type {string} */ name) => name !== initialGroup?.className)
			.map((/** @type {string} */ name) =>
				vorhandeneGruppen.find((/** @type {any} */ g) => g.className === name)
			)
			.filter((/** @type {any} */ g) => g && g.books?.length > 0)
	);

	// Aufzaehlung im Skript statt im Markup: Die Variante mit {#each} brauchte fuer das
	// Komma und das „und" Mustaches mit reinen Zeichenketten — ESLint lehnt die zu Recht
	// ab (svelte/no-useless-mustaches), und lesbar war sie auch nicht.
	const ueberschriebenText = $derived.by(() => {
		const teile = ueberschriebeneKlassen.map(
			(/** @type {any} */ g) =>
				`${g.className} (${g.books.length} ${g.books.length === 1 ? 'Buch' : 'Bücher'})`
		);
		if (teile.length === 0) return '';
		const liste =
			teile.length === 1 ? teile[0] : `${teile.slice(0, -1).join(', ')} und ${teile.at(-1)}`;
		return teile.length === 1
			? `${liste} hat bereits einen Klassensatz. Beim Speichern wird er durch die Auswahl hier ersetzt.`
			: `${liste} haben bereits einen Klassensatz. Beim Speichern werden sie durch die Auswahl hier ersetzt.`;
	});

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

<!-- Escape schliesst den Dialog; vorher gab es dafuer keinen Weg ausser dem Klick auf den
     Hintergrund, den man mit der Tastatur nicht erreicht. `isOpen` wird im Handler
     geprueft, weil <svelte:window> auf der obersten Ebene stehen muss. -->
<svelte:window
	onkeydown={(e) => {
		if (isOpen && e.key === 'Escape') onClose();
	}}
/>

{#if isOpen}
	<!-- role="presentation": Der Hintergrund ist Dekoration. Das Schliessen per Klick
	     darauf ist eine Maus-Bequemlichkeit, der Dialog selbst traegt die Semantik. -->
	<div
		class="fixed inset-0 z-50 flex items-center justify-center p-0 sm:p-4 bg-black/30 backdrop-blur-sm animate-in fade-in duration-200"
		role="presentation"
		onclick={(e) => {
			if (e.target === e.currentTarget) onClose();
		}}
	>
		<div
			class="bg-white rounded-none sm:rounded-3xl shadow-2xl w-full lg:w-300 max-w-[100vw] lg:max-w-[90vw] h-dvh sm:h-[90vh] lg:h-212.5 max-h-dvh lg:max-h-[95vh] p-4 sm:p-6 lg:p-8 flex flex-col lg:flex-row gap-6 lg:gap-8 relative overflow-hidden animate-in zoom-in-95 duration-200"
		>
			<!-- Left Content Area -->
			<div class="grow flex flex-col gap-4 sm:gap-6 relative z-10 w-full overflow-hidden">
				<div class="shrink-0">
					<h2 class="text-2xl sm:text-3xl font-bold tracking-tight text-slate-900 leading-none">
						Klasse & Bücher zuweisen
					</h2>
					<p class="mt-1 sm:mt-2 text-slate-500 font-medium text-sm sm:text-lg">
						Wähle Zielklassen und die entsprechenden Schulbücher aus.
					</p>
				</div>

				<div
					class="flex-1 overflow-y-auto [&::-webkit-scrollbar]:w-1.5 [&::-webkit-scrollbar-track]:bg-transparent [&::-webkit-scrollbar-thumb]:bg-outline-variant [&::-webkit-scrollbar-thumb]:rounded-full pr-4 pb-4"
				>
					<ClassAssignmentSelector bind:selectedClasses />

					{#if ueberschriebeneKlassen.length > 0}
						<!-- Warnung statt Verbot: Genau das WILL man meistens (ein Jahrgang
						     bekommt denselben Satz). Es darf nur nicht unbemerkt passieren. -->
						<div
							class="mb-4 flex items-start gap-2.5 rounded-xl bg-amber-50 px-4 py-3 text-sm text-amber-900"
						>
							<TriangleAlert class="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
							<p>{ueberschriebenText}</p>
						</div>
					{/if}

					<ClassAssignmentBookGrid {books} bind:selectedBookIds />
				</div>
			</div>

			<!-- Right Sidebar Area -->
			<aside
				class="w-full lg:w-85 flex-none lg:shrink-0 flex flex-col gap-4 relative z-10 border-t lg:border-t-0 lg:border-l border-slate-100 pt-4 lg:pt-0 lg:pl-8 h-[40dvh] lg:h-auto"
			>
				<ClassAssignmentSummary
					{selectedClasses}
					{selectedBookIds}
					{selectedBooksList}
					{isSaving}
					isUpdate={!!initialGroup}
					onToggleBook={toggleBook}
					onsave={saveAssignments}
					oncancel={onClose}
				/>
			</aside>

			<!-- Close Button (Absolute Top Right) -->
			<button
				aria-label="Schließen"
				onclick={onClose}
				class="absolute top-4 sm:top-6 right-4 sm:right-6 p-2 hover:bg-slate-100 hover:text-slate-900 rounded-full transition-all duration-200 text-slate-400 z-20 cursor-pointer border-none bg-transparent"
			>
				<X class="w-4 h-4" aria-hidden="true" />
			</button>
		</div>
	</div>
{/if}
