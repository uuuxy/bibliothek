<!-- @component Schuljahr — die Seite „Schuljahreswechsel": die Aufgaben, die ein- bis
     zweimal im Jahr anfallen, als Reiter an einem Ort (Peter, 05.09.2026: der LMF-Plan
     „lenkt nur ab", wenn er dauerhaft im Bibliotheks-Menü steht) — LMF-Plan (Rückgabe
     vor den Sommerferien, Ausgabe danach), Abgänger (Mai–Juli) und LUSD-Abgleich mit
     Versetzung (August). Nicht in den Einstellungen: Das sind Arbeitsdokumente und
     Läufe, keine Konfiguration.

     Jeder Reiter hängt an dem Recht seiner Routen (hatRecht, nicht Rollenvergleich):
     LMF-Plan edit_books, Abgänger view_graduates, LUSD/Versetzung import_students bzw.
     manage_students_admin. Der Menüpunkt öffnet sich mit einem davon (menu.js). -->
<script>
	import PageShell from './components/layout/PageShell.svelte';
	import Reiter from './components/ui/Reiter.svelte';
	import LmfPlan from './LmfPlan.svelte';
	import Graduates from './Graduates.svelte';
	import SchuljahreswechselBereich from './components/admin/SchuljahreswechselBereich.svelte';
	import { authStore } from './stores/authStore.svelte.js';
	import { uiStore } from './stores/uiStore.svelte.js';
	import { hatRecht } from './menu.js';

	const reiter = $derived(
		[
			{ id: 'lmf-plan', label: 'LMF-Plan', recht: ['edit_books'] },
			{ id: 'abgaenger', label: 'Abgänger', recht: ['view_graduates'] },
			{
				id: 'versetzung',
				label: 'LUSD & Versetzung',
				recht: ['import_students', 'manage_students_admin']
			}
		].filter((r) => r.recht.some((p) => hatRecht(authStore.currentUser, p)))
	);

	// Der Router setzt den gewünschten Reiter aus der URL (/schuljahr/abgaenger) und
	// liest ihn für die Adresszeile zurück — so sind die Reiter verlinkbar, und die
	// alten Adressen /abgaenger und /lmf-plan landen weiter am richtigen Ort.
	$effect(() => {
		if (reiter.length > 0 && !reiter.some((r) => r.id === uiStore.schuljahrReiter)) {
			uiStore.schuljahrReiter = reiter[0].id;
		}
	});
</script>

<PageShell>
	<Reiter
		etikett="Schuljahreswechsel-Bereiche"
		{reiter}
		aktiv={uiStore.schuljahrReiter}
		onwahl={(id) => (uiStore.schuljahrReiter = id)}
	/>

	<div class="mt-4">
		{#if uiStore.schuljahrReiter === 'lmf-plan' && reiter.some((r) => r.id === 'lmf-plan')}
			<LmfPlan />
		{:else if uiStore.schuljahrReiter === 'abgaenger' && reiter.some((r) => r.id === 'abgaenger')}
			<Graduates />
		{:else if uiStore.schuljahrReiter === 'versetzung' && reiter.some((r) => r.id === 'versetzung')}
			<SchuljahreswechselBereich />
		{/if}
	</div>
</PageShell>
