<!-- @component AbgaengerTabelle — die Liste der Abgänger mit offenen Posten in vier
     Zuständen desselben Bildschirmbereichs: Ladefehler, außerhalb der Saison (Hinweis mit
     den Daten), in der Saison ohne Posten („alle entlastet"), in der Saison mit Zeilen
     (Tabelle). Welcher Zustand gilt, sagt der Server über `fenster` — die Oberfläche
     rechnet keinen eigenen Kalender. Der Ladefehler schlägt alle anderen: Eine leere
     Liste nach gescheitertem Abruf ist nicht „alle entlastet", sonst entlässt die
     Bibliothek Schüler, ohne Bücher einzusammeln. -->
<script>
	import { CalendarClock, Check } from '@lucide/svelte';
	import SuchZustand from '../ui/SuchZustand.svelte';
	import Tabelle from '../ui/Tabelle.svelte';
	import LadeFehler from '../ui/LadeFehler.svelte';
	import { fokusAuffangen } from '../ui/fokusAuffangen.svelte.js';
	/** @type {{ zeilen: any[], leer: boolean, fenster: { offen: boolean, von: string, bis: string }, ladefehler?: string | null, onErneut?: () => void, onProfil: (student: any) => void }} */
	let { zeilen, leer, fenster, ladefehler = null, onErneut = () => {}, onProfil } = $props();

	/** @type {HTMLElement | undefined} */
	let liste = $state();
	// Eine Rückgabe am anderen Platz nimmt die Zeile aus der Liste, auf der vielleicht der Fokus steht.
	fokusAuffangen(
		() => zeilen,
		() => liste
	);
</script>

{#if ladefehler}
	<LadeFehler onerneut={onErneut} titel="Abgängerliste nicht geladen" text={ladefehler} />
{:else if !fenster.offen}
	<SuchZustand
		symbol={CalendarClock}
		titel="Abschlussklassen erscheinen hier ab Mai"
		hinweis="Vom {fenster.von} bis {fenster.bis} zeigt diese Liste die Abschlussklassen (9H, 10R, 13) mit noch offenen Büchern — zum Einsammeln vor der Entlassung. Wer die Schule schon verlassen hat und noch Bücher schuldet, steht im Mahnwesen."
	/>
{:else if leer}
	<SuchZustand
		symbol={Check}
		titel="Alle Abgänger entlastet"
		hinweis="Kein Schüler der Abschlussklassen hat noch offene Lehrmittel."
	/>
{:else}
	<div bind:this={liste} class="overflow-x-auto">
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
						class="cursor-pointer animate-slide-up focus-visible:outline-2 focus-visible:outline-primary focus-visible:-outline-offset-2"
					>
						<td>{student.klasse}</td>
						<td class="font-medium">{student.vorname} {student.nachname}</td>
						<td>
							{student.offene_buecher}
							{student.offene_buecher === 1 ? 'Buch' : 'Bücher'}
							{#if student.ueberfaellig > 0}
								<span class="font-medium text-error">
									· {student.ueberfaellig} überfällig
								</span>
							{/if}
						</td>
						<td>
							{#if student.ist_gesperrt}
								<span class="text-sm font-medium text-error">Sperre aktiv</span>
							{/if}
							<span class="sr-only">Profil anzeigen</span>
						</td>
					</tr>
				{/each}
			</tbody>
		</Tabelle>
	</div>
{/if}
