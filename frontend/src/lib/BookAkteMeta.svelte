<script>
  import { getSubjectGradient } from '../inventur/lib/bookHelpers.js';

  /** 
   * @typedef {Object} Props
   * @property {any} book
   * @property {any[]} borrowers
   * @property {any[]} exemplare
   * @property {string} coverSrc
   * @property {boolean} coverFailed
   * @property {(e: Event) => void} onCoverError
   * @property {(e: Event) => void} onCoverLoad
   */
  
  /** @type {Props} */
  let { book, borrowers, exemplare, coverSrc, coverFailed, onCoverError, onCoverLoad } = $props();
</script>

<div class="overflow-hidden rounded-2xl">
  <div class="flex flex-col sm:flex-row gap-0">
    <!-- Cover / Spine -->
    <div class="w-full sm:w-48 shrink-0 bg-linear-to-br {getSubjectGradient(book.subject)} flex items-center justify-center min-h-56 relative">
      {#if coverSrc && !coverFailed}
        <img
          src={coverSrc}
          alt={`Cover ${book.title}`}
          class="h-full w-full object-cover absolute inset-0"
          onerror={onCoverError}
          onload={onCoverLoad}
        />
      {:else}
        <div class="text-center p-6 z-10">
          <p class="text-xs font-extrabold text-white/60 uppercase tracking-widest mb-2">{book.subject}</p>
          <p class="text-sm font-bold text-white leading-snug line-clamp-4">{book.title}</p>
        </div>
      {/if}
    </div>

    <!-- Meta -->
    <div class="flex-1 p-6 sm:p-8 flex flex-col justify-between gap-4">
      <div>
        <div class="flex flex-wrap gap-2 mb-3">
          <span class="text-[10px] font-bold px-2 py-0.5 rounded-md bg-blue-50 border border-blue-200 text-blue-700">{book.subject}</span>
          <span class="text-[10px] font-bold px-2 py-0.5 rounded-md bg-slate-100 border border-slate-200 text-slate-600">Klasse {book.gradeLevel}</span>
          {#if book.jahrgangVon && book.jahrgangBis}
            <span class="text-[10px] font-bold px-2 py-0.5 rounded-md bg-slate-100 border border-slate-200 text-slate-600">Verwendbar: Kl. {book.jahrgangVon} - {book.jahrgangBis}</span>
          {/if}
          {#if book.track}
            <span class="text-[10px] font-bold px-2 py-0.5 rounded-md bg-cyan-50 border border-cyan-200 text-cyan-700">{book.track}</span>
          {/if}
          {#if book.medientyp && book.medientyp !== "Buch"}
            <span class="text-[10px] font-bold px-2 py-0.5 rounded-md bg-amber-50 border border-amber-200 text-amber-700">{book.medientyp}</span>
          {/if}
          {#if book.signatur || book.erweiterte_eigenschaften?.signatur}
            <span class="text-[10px] font-bold px-2 py-0.5 rounded-md bg-purple-50 border border-purple-200 text-purple-700">📚 {book.signatur || book.erweiterte_eigenschaften?.signatur}</span>
          {/if}
          {#if book.erweiterte_eigenschaften?.standort}
            <span class="text-[10px] font-bold px-2 py-0.5 rounded-md bg-emerald-50 border border-emerald-200 text-emerald-700">📍 {book.erweiterte_eigenschaften.standort}</span>
          {/if}
        </div>
        <h1 class="text-2xl font-extrabold text-slate-900 leading-tight mb-1">{book.title}</h1>
        <p class="text-sm text-slate-500 font-medium">{book.author || "Unbekannter Autor"}</p>
      </div>

      <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
        <!-- Stock -->
        <div class="bg-slate-50 rounded-xl p-3 border border-slate-100">
          <p class="text-[10px] font-bold text-slate-400 mb-1">Verfügbar</p>
          <p class="text-2xl font-extrabold {(book.verfuegbar) === 0 ? 'text-rose-600' : (book.verfuegbar) < 5 ? 'text-amber-600' : 'text-emerald-600'}">
            {book.verfuegbar}
            <span class="text-sm font-medium text-slate-400">/ {book.gesamt}</span>
          </p>
        </div>
        <!-- Ausleiher -->
        <div class="bg-indigo-50 rounded-xl p-3 border border-indigo-100">
          <p class="text-[10px] font-bold text-indigo-400 mb-1">Ausleiher</p>
          <p class="text-2xl font-extrabold text-indigo-700">{borrowers.length}</p>
        </div>
        <!-- Exemplare -->
        <div class="bg-emerald-50 rounded-xl p-3 border border-emerald-100">
          <p class="text-[10px] font-bold text-emerald-400 mb-1">Exemplare</p>
          <p class="text-2xl font-extrabold text-emerald-700">{exemplare.length}</p>
        </div>
        <!-- ISBN -->
        <div class="bg-slate-50 rounded-xl p-3 border border-slate-100">
          <p class="text-[10px] font-bold text-slate-400 mb-1">ISBN / EAN</p>
          <p class="text-sm font-mono font-semibold text-slate-700 break-all">{book.isbn || "–"}</p>
        </div>
      </div>
    </div>
  </div>
</div>
