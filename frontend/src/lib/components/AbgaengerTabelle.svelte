<!-- @component AbgaengerTabelle — die Liste der Abgänger mit offenen Posten,
     einschließlich des leeren Falls („alle entlastet"). Beides gehört zusammen:
     Es ist derselbe Bildschirmbereich in zwei Zuständen. -->
<script>
	import { Check } from '@lucide/svelte';
	/** @type {{ zeilen: any[], leer: boolean, onProfil: (student: any) => void }} */
	let { zeilen, leer, onProfil } = $props();
</script>

{#if leer}
	<div class="py-12 text-center space-y-3 animate-fade-in">
		<div
			class="w-16 h-16 rounded-full bg-emerald-50 border border-emerald-100 flex items-center justify-center text-emerald-600 mx-auto"
		>
			<Check class="h-8 w-8" aria-hidden="true" />
		</div>
		<h3 class="font-bold text-slate-800">Alle Abgänger entlastet!</h3>
		<p class="text-xs text-slate-500 max-w-xs mx-auto">
			Kein Abgänger hat mehr offene Lehrmittel oder unbezahlte Schadensfälle.
		</p>
	</div>
{:else}
	<div class="overflow-x-auto">
		<table class="w-full text-left text-base border-collapse">
			<thead>
				<tr class="border-b border-slate-100 text-slate-500 text-sm">
					<th class="py-2 px-4">Klasse</th>
					<th class="py-2 px-4">Name</th>
					<th class="py-2 px-4">Offene Bücher</th>
					<th class="py-2 px-4">Sperr-Status</th>
				</tr>
			</thead>
			<tbody class="divide-y divide-slate-50">
				{#each zeilen as student (student.id)}
					<tr
						onclick={() => onProfil(student)}
						onkeydown={(e) => {
							if (e.key === 'Enter' || e.key === ' ') {
								e.preventDefault();
								onProfil(student);
							}
						}}
						tabindex="0"
						role="button"
						aria-label="Profil von {student.vorname} {student.nachname} (Klasse {student.klasse}) anzeigen"
						class="hover:bg-slate-50/85 cursor-pointer transition-colors animate-slide-up focus-visible:outline-2 focus-visible:outline-blue-600 focus-visible:-outline-offset-2"
					>
						<td class="py-2 px-4 text-slate-500">{student.klasse}</td>
						<td class="py-2 px-4 font-medium text-slate-800"
							>{student.vorname} {student.nachname}</td
						>
						<td class="py-2 px-4 text-slate-600">
							{student.offene_buecher}
							{student.offene_buecher === 1 ? 'Buch' : 'Bücher'}
							{#if student.ueberfaellig > 0}
								<span class="font-medium text-rose-600">
									· {student.ueberfaellig} überfällig
								</span>
							{/if}
						</td>
						<td class="py-2 px-4">
							{#if student.ist_gesperrt}
								<span class="text-sm font-medium text-rose-600">Sperre aktiv</span>
							{/if}
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}
