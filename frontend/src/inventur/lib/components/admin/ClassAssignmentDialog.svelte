<script>
	import { apiFetch } from '../../../../lib/apiFetch.js';
	import { toastStore } from '../../../../lib/stores/toastStore.svelte.js';
	import { vorauswahlAusGruppe } from './klassensatzVorauswahl.js';
	import { erzeugeBuecherListe } from './klassensatzBuecher.svelte.js';
	import { onMount } from 'svelte';
	import ClassAssignmentSelector from './ClassAssignmentSelector.svelte';
	import ClassAssignmentBookGrid from './ClassAssignmentBookGrid.svelte';
	import ClassAssignmentSummary from './ClassAssignmentSummary.svelte';
	import { TriangleAlert, X } from '@lucide/svelte';
	import Modal from '../../../../lib/Modal.svelte';

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
	const buecher = erzeugeBuecherListe();
	let isSaving = $state(false);

	$effect(() => {
		if (selectedClasses.length === 0 && selectedBookIds.size > 0) {
			selectedBookIds = new Set();
		}
	});

	onMount(async () => {
		if (initialGroup) {
			selectedClasses = [initialGroup.className];
			// Nur Handgepflegtes vorgewählt (Regel samt Begründung: klassensatzVorauswahl.js).
			selectedBookIds = vorauswahlAusGruppe(initialGroup);
		}

		await buecher.laden();
	});

	const selectedBooksList = $derived(
		buecher.liste.filter((/** @type {any} */ b) => selectedBookIds.has(b.id))
	);

	// Der Speicherpfad ersetzt: UpdateClassBooks löscht die Zuweisungen aller Zielklassen und
	// schreibt danach die Auswahl hinein (inventur/datenbank_klassen.go). Wer zu 05F1 noch 06A2
	// dazunimmt, löscht deren Satz; die Warnung unten nennt die betroffenen Klassen.
	const ueberschriebeneKlassen = $derived(
		selectedClasses
			.filter((/** @type {string} */ name) => name !== initialGroup?.className)
			.map((/** @type {string} */ name) =>
				vorhandeneGruppen.find((/** @type {any} */ g) => g.className === name)
			)
			.filter((/** @type {any} */ g) => g && g.books?.length > 0)
	);

	// Aufzählung im Skript statt im Markup: {#each} bräuchte für Komma und „und" Mustaches mit
	// reinen Zeichenketten, und die lehnt ESLint ab (svelte/no-useless-mustaches).
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
				toastStore.addToast('Ein Fehler ist aufgetreten. Bitte erneut versuchen.', 'error');
			}
		} catch (e) {
			console.error('Netzwerkfehler', e);
			toastStore.addToast('Fehler beim Speichern der Zuweisung.', 'error');
		} finally {
			isSaving = false;
		}
	}
</script>

{#if isOpen}
	<!-- Modal.svelte in Größe „voll" (M3 full-screen dialog) stellt Hintergrund, Fokusfalle,
	     Escape und Hintergrundklick; die zweispaltige Arbeitsfläche samt eigenem
	     Schließen-Knopf ist Inhalt. -->
	<Modal open={true} onclose={() => onClose()} size="voll" beschriftetDurch="zuweisung-titel">
		<div
			class="h-full p-4 sm:p-6 lg:p-8 flex flex-col lg:flex-row gap-6 lg:gap-8 relative overflow-hidden"
		>
			<!-- Left Content Area -->
			<div class="grow flex flex-col gap-4 sm:gap-6 relative z-10 w-full overflow-hidden">
				<div class="shrink-0">
					<h2
						id="zuweisung-titel"
						class="text-2xl sm:text-3xl font-bold tracking-tight text-on-surface leading-none"
					>
						Klasse & Bücher zuweisen
					</h2>
					<p class="mt-1 sm:mt-2 text-on-surface-variant font-medium text-sm sm:text-lg">
						Wähle Zielklassen und die entsprechenden Schulbücher aus.
					</p>
				</div>

				<div
					class="flex-1 overflow-y-auto [&::-webkit-scrollbar]:w-1.5 [&::-webkit-scrollbar-track]:bg-transparent [&::-webkit-scrollbar-thumb]:bg-outline-variant [&::-webkit-scrollbar-thumb]:rounded-full pr-4 pb-4"
				>
					<ClassAssignmentSelector bind:selectedClasses {vorhandeneGruppen} />

					{#if ueberschriebeneKlassen.length > 0}
						<!-- Warnung statt Verbot: Meist ist es gewollt (ein Jahrgang bekommt denselben
						     Satz). Es darf nur nicht unbemerkt passieren. -->
						<div
							class="mb-4 flex items-start gap-2.5 rounded-xl bg-warning-container px-4 py-3 text-sm text-on-warning-container"
						>
							<TriangleAlert class="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
							<p>{ueberschriebenText}</p>
						</div>
					{/if}

					<ClassAssignmentBookGrid
						books={buecher.liste}
						buecherFehler={buecher.fehler}
						bind:selectedBookIds
					/>
				</div>
			</div>

			<!-- Right Sidebar Area -->
			<aside
				class="w-full lg:w-85 flex-none lg:shrink-0 flex flex-col gap-4 relative z-10 border-t lg:border-t-0 lg:border-l border-outline-variant pt-4 lg:pt-0 lg:pl-8 h-[40dvh] lg:h-auto"
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
				class="icon-btn absolute top-4 sm:top-6 right-4 sm:right-6 z-20 text-on-surface-variant"
			>
				<X class="w-4 h-4" aria-hidden="true" />
			</button>
		</div>
	</Modal>
{/if}
