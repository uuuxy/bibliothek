<script>
  let { stopCamera, queryVal = $bindable(), submitAction } = $props();
  /** @type {import('html5-qrcode').Html5Qrcode | null} */
  let cameraScanner = $state(null);
  import { onMount, onDestroy } from "svelte";

  onMount(async () => {
    try {
      const { Html5Qrcode } = await import("html5-qrcode");
      cameraScanner = new Html5Qrcode("camera-scan-region");
      await cameraScanner.start(
        { facingMode: "environment" },
        { fps: 10, qrbox: { width: 260, height: 120 } },
        (decodedText) => {
          queryVal = decodedText.trim();
          stopScanner();
          submitAction();
        },
        () => {}
      );
    } catch {
      // Init Error is ignored here
    }
  });

  onDestroy(() => {
    stopScanner();
  });

  async function stopScanner() {
    if (cameraScanner) {
      try { await cameraScanner.stop(); } catch {}
      try { cameraScanner.clear(); } catch {}
      cameraScanner = null;
    }
    stopCamera();
  }
</script>

<div class="w-full rounded-2xl overflow-hidden border border-blue-200 shadow-lg animate-slide-up bg-black relative">
  <div class="absolute top-3 right-3 z-10">
    <button type="button" onclick={stopScanner} class="p-1.5 rounded-full bg-white/80 text-slate-700 hover:bg-white shadow transition-colors cursor-pointer" title="Kamera schließen">
      <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/></svg>
    </button>
  </div>
  <div class="px-4 pt-3 pb-1 text-xs text-blue-200 font-semibold text-center">Kamera auf Barcode richten</div>
  <div id="camera-scan-region" class="w-full min-h-[240px]"></div>
</div>
