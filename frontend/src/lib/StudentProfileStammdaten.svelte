<script>
	let { profile, role, onEdit } = $props();

	import Button from './components/ui/Button.svelte';
	import { Folder, SquarePen } from '@lucide/svelte';

	function formatDate(dateString) {
		if (!dateString) return 'Keine Angabe';
		try {
			const d = new Date(dateString);
			return d.toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit', year: 'numeric' });
		} catch {
			return dateString;
		}
	}
</script>

<div class="w-full pt-2 animate-fade-in space-y-8">
	<div class="flex justify-between items-center border-b border-slate-100 pb-4">
		<h3 class="text-xl font-bold text-slate-800 flex items-center gap-2">
			<Folder class="w-6 h-6 text-blue-500" aria-hidden="true" />
			Stammdaten & Adresse
		</h3>
		<div class="flex items-center gap-2">
			{#if role === 'admin'}
				<Button
					variant="secondary"
					size="lg"
					onclick={onEdit}
					class="px-5 bg-blue-50 border-blue-100 text-blue-600 hover:bg-blue-100"
				>
					<SquarePen class="w-4 h-4" aria-hidden="true" />
					Bearbeiten
				</Button>
			{/if}
		</div>
	</div>

	<div class="grid grid-cols-1 md:grid-cols-2 gap-8">
		<div class="space-y-6">
			<div>
				<p class="text-xs font-medium text-slate-600 mb-1">Geburtsdatum</p>
				<p class="text-slate-800 font-semibold">{formatDate(profile.geburtsdatum)}</p>
			</div>
			<div>
				<p class="text-xs font-medium text-slate-600 mb-1">LUSD ID</p>
				<p class="text-slate-800 font-semibold">{profile.lusd_id || 'Keine Angabe'}</p>
			</div>
		</div>

		<div class="space-y-6">
			<div>
				<p class="text-xs font-medium text-slate-600 mb-1">Postanschrift</p>
				{#if profile.strasse}
					<p class="text-slate-800 font-semibold">{profile.strasse} {profile.hausnummer}</p>
					<p class="text-slate-800 font-semibold">{profile.plz} {profile.ort}</p>
				{:else}
					<p class="text-slate-600 italic text-sm">Keine Adresse hinterlegt</p>
				{/if}
			</div>
			<div>
				<p class="text-xs font-medium text-slate-600 mb-1">Eltern E-Mail</p>
				{#if profile.eltern_email}
					<a
						href="mailto:{profile.eltern_email}"
						class="text-blue-600 hover:underline font-semibold">{profile.eltern_email}</a
					>
				{:else}
					<p class="text-slate-600 italic text-sm">Keine E-Mail hinterlegt</p>
				{/if}
			</div>
		</div>
	</div>
</div>
