<script>
  import { apiFetch } from "./apiFetch.js";
  import { onMount, onDestroy } from 'svelte';

  /** @type {{buch_des_monats: any, neu_eingetroffen: any[], beliebt: any[]} | null} */
  let slides = $state(null);
  let currentSlide = $state(0);
  let coverIndex = $state(0);
  let progressKey = $state(0); // forces progress bar restart

  /** @type {ReturnType<typeof setInterval> | undefined} */
  let slideTimer;
  /** @type {ReturnType<typeof setInterval> | undefined} */
  let coverTimer;

  async function fetchSlides() {
    try {
      const res = await apiFetch('/api/monitor/slides');
      if (res.ok) slides = await res.json();
    } catch { /* network unavailable, keep old data */ }
  }

  function nextSlide() {
    currentSlide = (currentSlide + 1) % 3;
    coverIndex = 0;
    progressKey++;
  }

  onMount(() => {
    fetchSlides();
    slideTimer = setInterval(nextSlide, 15_000);
    coverTimer = setInterval(() => {
      if (currentSlide === 1 && slides?.neu_eingetroffen?.length) {
        coverIndex = (coverIndex + 1) % slides.neu_eingetroffen.length;
      }
    }, 2_500);
  });

  onDestroy(() => {
    clearInterval(slideTimer);
    clearInterval(coverTimer);
  });

  const slideLabels = ['⭐ Buch des Monats', '🆕 Neu eingetroffen', '🔥 Beliebt diese Woche'];
</script>

<style>
  @keyframes progress {
    from { width: 0%; }
    to   { width: 100%; }
  }
  .progress-bar {
    animation: progress 15s linear infinite;
  }
</style>

<div class="fixed inset-0 bg-slate-900 text-white flex flex-col overflow-hidden select-none">

  <!-- Slide indicator dots -->
  <div class="absolute top-4 left-1/2 -translate-x-1/2 flex gap-2 z-10">
    {#each [0, 1, 2] as i}
      <button
        onclick={() => { currentSlide = i; coverIndex = 0; progressKey++; }}
        aria-label="Slide {i + 1} anzeigen"
        class="rounded-full transition-all duration-300 cursor-pointer {currentSlide === i ? 'bg-white w-6 h-2' : 'bg-slate-600 w-2 h-2'}"
      ></button>
    {/each}
  </div>

  <!-- Slide content -->
  <div class="flex-1 flex items-center justify-center px-8 py-16">

    {#if !slides}
      <div class="text-slate-500 text-xl animate-pulse">📡 Lade Daten …</div>

    {:else if currentSlide === 0}
      <!-- Buch des Monats -->
      <div class="flex flex-col items-center text-center gap-6 max-w-sm">
        <span class="text-sm font-bold tracking-widest uppercase text-amber-400">⭐ Buch des Monats</span>
        {#if slides.buch_des_monats}
          {#if slides.buch_des_monats.cover_url}
            <img src={slides.buch_des_monats.cover_url} alt="Cover"
              class="w-48 h-64 object-cover rounded-2xl shadow-2xl ring-4 ring-amber-400/30" />
          {:else}
            <div class="w-48 h-64 rounded-2xl bg-slate-700 flex items-center justify-center shadow-2xl">
              <span class="text-6xl font-extrabold text-slate-500">
                {slides.buch_des_monats.titel.charAt(0)}
              </span>
            </div>
          {/if}
          <div>
            <h2 class="text-3xl font-extrabold leading-tight">{slides.buch_des_monats.titel}</h2>
            {#if slides.buch_des_monats.autor}
              <p class="text-slate-400 mt-2 text-lg">{slides.buch_des_monats.autor}</p>
            {/if}
          </div>
        {:else}
          <p class="text-slate-500">Kein Buch verfügbar</p>
        {/if}
      </div>

    {:else if currentSlide === 1}
      <!-- Neu eingetroffen -->
      <div class="flex flex-col items-center gap-8 w-full max-w-4xl">
        <span class="text-sm font-bold tracking-widest uppercase text-cyan-400">🆕 Neu eingetroffen</span>
        {#if slides.neu_eingetroffen.length > 0}
          <div class="flex gap-4 items-end justify-center flex-wrap">
            {#each slides.neu_eingetroffen as book, i}
              <div class="flex flex-col items-center gap-2 transition-all duration-500"
                   class:scale-110={i === coverIndex}
                   class:opacity-50={i !== coverIndex}>
                {#if book.cover_url}
                  <img src={book.cover_url} alt="Cover"
                    class="rounded-xl shadow-lg object-cover transition-all duration-500
                           {i === coverIndex ? 'w-32 h-44' : 'w-20 h-28'}" />
                {:else}
                  <div class="rounded-xl bg-slate-700 flex items-center justify-center transition-all duration-500
                              {i === coverIndex ? 'w-32 h-44' : 'w-20 h-28'}">
                    <span class="{i === coverIndex ? 'text-2xl' : 'text-base'} text-slate-500 font-extrabold">
                      {book.titel.charAt(0)}
                    </span>
                  </div>
                {/if}
                {#if i === coverIndex}
                  <div class="text-center max-w-32">
                    <p class="text-sm font-bold leading-tight text-white truncate">{book.titel}</p>
                    {#if book.autor}
                      <p class="text-xs text-slate-400 truncate">{book.autor}</p>
                    {/if}
                  </div>
                {/if}
              </div>
            {/each}
          </div>
        {:else}
          <p class="text-slate-500">Keine neuen Medien</p>
        {/if}
      </div>

    {:else if currentSlide === 2}
      <!-- Beliebt -->
      <div class="flex flex-col items-center gap-6 w-full max-w-lg">
        <span class="text-sm font-bold tracking-widest uppercase text-rose-400">🔥 Beliebt diese Woche</span>
        {#if slides.beliebt.length > 0}
          <ol class="w-full flex flex-col gap-3">
            {#each slides.beliebt as book, i}
              <li class="flex items-center gap-4 bg-slate-800/60 rounded-2xl p-3 shadow-md">
                <span class="text-2xl font-black w-8 text-center text-slate-500">#{i + 1}</span>
                {#if book.cover_url}
                  <img src={book.cover_url} alt="Cover" class="w-12 h-16 object-cover rounded-xl shadow" />
                {:else}
                  <div class="w-12 h-16 rounded-xl bg-slate-700 flex items-center justify-center">
                    <span class="text-lg font-extrabold text-slate-500">{book.titel.charAt(0)}</span>
                  </div>
                {/if}
                <div class="flex-1 min-w-0">
                  <p class="font-bold truncate">{book.titel}</p>
                  {#if book.autor}
                    <p class="text-xs text-slate-400 truncate">{book.autor}</p>
                  {/if}
                </div>
              </li>
            {/each}
          </ol>
        {:else}
          <p class="text-slate-500">Keine Daten verfügbar</p>
        {/if}
      </div>
    {/if}
  </div>

  <!-- Slide label bar -->
  <div class="bg-slate-800 px-6 py-3 flex items-center justify-between text-xs text-slate-400 font-semibold tracking-wide">
    <span>{slideLabels[currentSlide]}</span>
    <span class="text-slate-600">Schulbibliothek</span>
  </div>

  <!-- Progress bar -->
  {#key progressKey}
    <div class="h-1 bg-slate-800">
      <div class="h-full bg-slate-400 progress-bar"></div>
    </div>
  {/key}
</div>
