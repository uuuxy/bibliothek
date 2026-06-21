<!--
  @component
  ActiveStudentList

  Diese Komponente rendert die Liste der aktiven Schüler mit Filter- und Sortierfunktionen.
  Sie zeigt ein Profilbild, den Namen, die Klasse, die Anzahl der ausgeliehenen Bücher und den Status an.
-->
<script>
  let { filteredStudents = [], students = [], loading = false, onSelectStudent = () => {} } = $props();
</script>

{#snippet avatar(s)}
  <div class="relative w-8 h-8 rounded-full overflow-hidden border border-slate-100/80 bg-slate-50 flex items-center justify-center shrink-0">
    {#if s.foto_url}
      <img src={s.foto_url} alt="Passbild von {s.vorname} {s.nachname}" class="w-full h-full object-cover" />
    {:else}
      <div class="w-full h-full flex items-center justify-center bg-slate-100 text-slate-500 font-bold text-xs uppercase" aria-hidden="true">
        {s.vorname.charAt(0)}{s.nachname.charAt(0)}
      </div>
    {/if}
  </div>
{/snippet}

{#snippet statusBadge(s)}
  <div class="inline-flex items-center justify-end gap-1.5 py-1">
    {#if s.ueberfaellig_count > 0}
      <span class="w-1.5 h-1.5 rounded-full bg-rose-500 animate-pulse" aria-hidden="true"></span>
      <span class="text-xs font-semibold text-rose-600">Überfällig</span>
    {:else if s.ist_gesperrt}
      <span class="w-1.5 h-1.5 rounded-full bg-amber-500" aria-hidden="true"></span>
      <span class="text-xs font-semibold text-amber-600">Gesperrt</span>
    {:else}
      <span class="w-1.5 h-1.5 rounded-full bg-emerald-500" aria-hidden="true"></span>
      <span class="text-xs font-semibold text-emerald-600">Alles ok</span>
    {/if}
  </div>
{/snippet}

<div class="bg-white rounded-2xl border border-slate-100 overflow-hidden shadow-xs w-full">
  {#if loading}
    <div class="py-16 flex justify-center items-center">
      <div class="w-8 h-8 border-4 border-t-blue-600 border-slate-200 rounded-full animate-spin" aria-hidden="true"></div>
    </div>
  {:else if filteredStudents.length === 0}
    <div class="py-16 flex flex-col items-center justify-center text-slate-400 space-y-2">
      <svg xmlns="http://www.w3.org/2000/svg" class="h-10 w-10 text-slate-300" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" /></svg>
      <span class="text-xs font-semibold">Keine Schüler im Verzeichnis gefunden.</span>
    </div>
  {:else}
    <div class="overflow-x-auto w-full text-left">
      <table class="w-full text-base text-slate-700">
        <thead class="bg-slate-50 border-b border-slate-100 uppercase tracking-wider text-sm font-bold text-slate-500 font-sans">
          <tr>
            <th class="px-6 py-4 w-16">Foto</th>
            <th class="px-6 py-4">Name</th>
            <th class="px-6 py-4 w-24">Klasse</th>
            <th class="px-6 py-4 w-44 text-right">Geliehene Bücher</th>
            <th class="px-6 py-4 w-36 text-right">Status</th>
            <th class="px-6 py-4 w-10"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-100">
          {#each filteredStudents as s}
            <tr 
              onclick={() => onSelectStudent(s)} 
              onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onSelectStudent(s); } }}
              tabindex="0"
              role="button"
              aria-label="Profil von {s.vorname} {s.nachname} (Klasse {s.klasse || 'N/A'}) anzeigen"
              class="hover:bg-slate-50/50 cursor-pointer transition-colors group focus-visible:outline-2 focus-visible:outline-blue-600 focus-visible:-outline-offset-2"
            >
              <td class="px-6 py-3">
                {@render avatar(s)}
              </td>
              <td class="px-6 py-3 font-semibold text-slate-800">
                {s.vorname} {s.nachname}
                <div class="text-[9px] text-slate-400 font-normal mt-0.5">{s.barcode_id}</div>
              </td>
              <td class="px-6 py-3 font-medium text-slate-600">
                Kl. {s.klasse || 'N/A'}
              </td>
              <td class="px-6 py-3 text-right">
                <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-bold {s.ausgeliehen_count > 0 ? 'bg-blue-50 text-blue-700' : 'bg-slate-100 text-slate-500'}">
                  {s.ausgeliehen_count || 0}
                </span>
              </td>
              <td class="px-6 py-3 text-right">
                {@render statusBadge(s)}
              </td>
              <td class="px-6 py-3 text-right">
                <svg class="w-4 h-4 text-slate-300 opacity-0 group-hover:opacity-100 transition-opacity ml-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
                </svg>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>
