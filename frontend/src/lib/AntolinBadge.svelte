<script>
  let { isbn = '' } = $props();

  /** @type {{found: boolean, stufen: string, punkte: number} | null} */
  let result = $state(null);

  $effect(() => {
    if (!isbn) { result = null; return; }
    fetch(`/api/antolin?isbn=${encodeURIComponent(isbn)}`)
      .then(r => r.json())
      .then(d => { result = d; })
      .catch(() => { result = null; });
  });
</script>

{#if result?.found}
  <div class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-amber-50 border border-amber-200 text-amber-800 text-xs font-semibold select-none">
    <span>🐝</span>
    <span>Antolin</span>
    {#if result.stufen}
      <span class="text-amber-600">· Kl.&nbsp;{result.stufen}</span>
    {/if}
    {#if result.punkte}
      <span class="text-amber-600">· {result.punkte}&nbsp;Pkt.</span>
    {/if}
  </div>
{/if}
