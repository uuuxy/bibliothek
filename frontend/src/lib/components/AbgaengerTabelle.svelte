<!-- @component AbgaengerTabelle — die Liste der Abgänger mit offenen Posten in vier
     Zuständen desselben Bildschirmbereichs: Ladefehler, außerhalb der Saison (Hinweis mit
     den Daten), in der Saison ohne Posten („alle entlastet"), in der Saison mit Zeilen
     (Tabelle). Welcher Zustand gilt, sagt der Server über `fenster` — die Oberfläche
     rechnet keinen eigenen Kalender. Der Ladefehler schlägt alle anderen: Eine leere
     Liste nach gescheitertem Abruf ist nicht „alle entlastet" (Bestands-Durchgang
     10.09.2026 — sonst entlässt die Bibliothek Schüler, ohne Bücher einzusammeln). -->
<script>
	import { CalendarClock, Check } from '@lucide/svelte';
	import Tabelle from './ui/Tabelle.svelte';
	import LadeFehler from './ui/LadeFehler.svelte';
	/** @type {{ zeilen: any[], leer: boolean, fenster: { offen: boolean, von: string, bis: string }, ladefehler?: string | null, onErneut?: () => void, onProfil: (student: any) => void }} */
	let { zeilen, leer, fenster, ladefehler = null, onErneut = () => {}, onProfil } = $props();
</script>

{#if ladefehler}
	<LadeFehler onerneut={onErneut} titel="Abgängerliste nicht geladen" text={ladefehler} />
{:else if !fenster.offen}
	<div class="py-12 text-center space-y-3 animate-fade-in">
		<div
			class="w-16 h-16 rounded-full bg-surface-container-low border border-outline-variant flex items-center justify-center text-on-surface-variant mx-auto"
		>
			<CalendarClock class="h-8 w-8" aria-hidden="true" />
		</div>
		<h3 class="font-bold text-on-surface">Abschlussklassen erscheinen hier ab Mai</h3>
		<p class="text-xs text-on-surface-variant max-w-sm mx-auto">
			Vom {fenster.von} bis {fenster.bis} zeigt diese Liste die Abschlussklassen (9H, 10R, 13) mit noch
			offenen Büchern — zum Einsammeln vor der Entlassung. Wer die Schule schon verlassen hat und noch
			Bücher schuldet, steht im Mahnwesen.
		</p>
	</div>
{:else if leer}
	<div class="py-12 text-center space-y-3 animate-fade-in">
		<div
			class="w-16 h-16 rounded-full bg-emerald-50 border border-emerald-100 flex items-center justify-center text-emerald-600 mx-auto"
		>
			<Check class="h-8 w-8" aria-hidden="true" />
		</div>
		<h3 class="font-bold text-slate-800">Alle Abgänger entlastet!</h3>
		<p class="text-xs text-slate-500 max-w-xs mx-auto">
			Kein Schüler der Abschlussklassen hat noch offene Lehrmittel.
		</p>
	</div>
{:else}
	<div class="overflow-x-auto">
		<Tabelle beschriftung="Abgänger mit offenen Büchern">
			<thead>
				<tr>
					<th>Klasse</th>
					<th>Name</th>
					<th>Offene Bücher</th>
					<th>Sperr-Status</th>
				</tr>
			</thead>
			<tbody>
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
						class="cursor-pointer animate-slide-up focus-visible:outline-2 focus-visible:outline-blue-600 focus-visible:-outline-offset-2"
					>
						<td>{student.klasse}</td>
						<td class="font-medium">{student.vorname} {student.nachname}</td>
						<td>
							{student.offene_buecher}
							{student.offene_buecher === 1 ? 'Buch' : 'Bücher'}
							{#if student.ueberfaellig > 0}
								<span class="font-medium text-rose-600">
									· {student.ueberfaellig} überfällig
								</span>
							{/if}
						</td>
						<td>
							{#if student.ist_gesperrt}
								<span class="text-sm font-medium text-rose-600">Sperre aktiv</span>
							{/if}
						</td>
					</tr>
				{/each}
			</tbody>
		</Tabelle>
	</div>
{/if}
