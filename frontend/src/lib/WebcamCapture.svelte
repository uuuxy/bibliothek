<script>
  import { apiFetch } from "./apiFetch.js";
  import { onMount } from "svelte";

  /** @type {{ studentId: string, onCapture: (url: string) => void, onClose: () => void }} */
  let { studentId, onCapture, onClose } = $props();

  /** @type {HTMLVideoElement | null} */
  let videoEl = $state(null);
  /** @type {MediaStream | null} */
  let stream = $state(null);
  /** @type {string | null} */
  let errorMsg = $state(null);
  let isCapturing = $state(false);

  async function startCamera() {
    try {
      errorMsg = null;
      stream = await navigator.mediaDevices.getUserMedia({
        video: {
          width: { ideal: 1920 }, // High resolution for sharp details
          height: { ideal: 1080 },
          aspectRatio: { ideal: 1.7777777778 }, // Widescreen native
          facingMode: "user"
        }
      });
      if (videoEl) {
        videoEl.srcObject = stream;
      }
    } catch (err) {
      const error = /** @type {any} */ (err);
      errorMsg = "Kamera-Zugriff fehlgeschlagen: " + error.message;
    }
  }

  async function capturePhoto() {
    if (!videoEl || !stream || isCapturing) return;
    isCapturing = true;

    try {
      const videoWidth = videoEl.videoWidth;
      const videoHeight = videoEl.videoHeight;

      // Passport photo aspect ratio is 3:4 (e.g. 300x400)
      const targetHeight = videoHeight;
      const targetWidth = Math.round(videoHeight * 0.75); // 3:4
      const startX = Math.max(0, Math.round((videoWidth - targetWidth) / 2));

      // Create high-res in-memory canvas
      const canvas = document.createElement("canvas");
      canvas.width = targetWidth;
      canvas.height = targetHeight;
      const ctx = canvas.getContext("2d");

      if (ctx) {
        // Draw cropped area from video stream
        ctx.drawImage(videoEl, startX, 0, targetWidth, targetHeight, 0, 0, targetWidth, targetHeight);
        
        // Export high-quality JPEG
        const dataUrl = canvas.toDataURL("image/jpeg", 0.95);

        // Upload to backend
        const res = await apiFetch(`/api/schueler/${studentId}/photo`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ photo_data: dataUrl })
        });

        if (!res.ok) {
          throw new Error(await res.text() || "Upload fehlgeschlagen");
        }

        const data = await res.json();
        onCapture(data.url);
      }
    } catch (err) {
      const error = /** @type {any} */ (err);
      errorMsg = "Aufnahme fehlgeschlagen: " + error.message;
      isCapturing = false;
    }
  }

  function stopCamera() {
    if (stream) {
      for (const track of stream.getTracks()) {
        track.stop();
      }
      stream = null;
    }
  }

  onMount(() => {
    startCamera();
    return () => stopCamera();
  });
</script>

<div class="fixed inset-0 z-50 flex items-center justify-center bg-zinc-950/80 backdrop-blur-md p-4 no-print">
  <div class="w-full max-w-lg p-6 rounded-3xl bg-zinc-900 border border-zinc-800 shadow-2xl flex flex-col space-y-4">
    
    <div class="flex items-center justify-between border-b border-zinc-800 pb-3">
      <h3 class="text-sm font-bold text-zinc-100 tracking-wide">📸 HQ Schülerfoto aufnehmen</h3>
      <button onclick={onClose} class="text-xs font-bold text-zinc-500 hover:text-zinc-300 transition-colors cursor-pointer">Schließen</button>
    </div>

    {#if errorMsg}
      <div class="p-8 text-center text-xs text-red-400 font-medium bg-red-950/20 border border-red-900/30 rounded-2xl">
        {errorMsg}
      </div>
      <div class="flex justify-end pt-2">
        <button onclick={startCamera} class="px-4 py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 text-xs font-bold rounded-xl cursor-pointer">Erneut versuchen</button>
      </div>
    {:else}
      <!-- Camera screen viewport with aspect ratio guidelines overlay -->
      <div class="relative w-full aspect-video bg-black rounded-2xl overflow-hidden border border-zinc-800">
        <!-- svelte-ignore a11y_media_has_caption -->
        <video bind:this={videoEl} autoplay playsinline class="w-full h-full object-cover"></video>
        
        <!-- Passport Overlay (3:4 Ratio & Face Guide Ellipse) -->
        <div class="absolute inset-0 pointer-events-none flex items-center justify-center bg-zinc-950/20">
          <div class="w-[50%] h-[90%] border-2 border-dashed border-emerald-400 rounded-[20px] flex items-center justify-center relative">
            <div class="w-[85%] h-[80%] border border-dashed border-emerald-400/40 rounded-full"></div>
            <span class="absolute bottom-2 text-[8px] bg-zinc-950/90 px-2 py-0.5 text-emerald-400 rounded-full font-bold tracking-wider">Gesichtsrahmen</span>
          </div>
        </div>
      </div>

      <div class="flex items-center justify-between pt-2">
        <span class="text-[10px] text-zinc-500">1080p Stream · Automatischer 3:4 Zuschnitt</span>
        <button onclick={capturePhoto} disabled={isCapturing} class="px-5 py-2.5 bg-emerald-500 hover:bg-emerald-400 disabled:bg-zinc-800 text-zinc-950 disabled:text-zinc-500 font-bold text-xs rounded-xl shadow-lg cursor-pointer transition-all flex items-center gap-1.5">
          {#if isCapturing}
            <span class="w-3.5 h-3.5 border-2 border-t-zinc-950 border-zinc-950/20 rounded-full animate-spin"></span>
            Speichert...
          {:else}
            📸 Foto aufnehmen
          {/if}
        </button>
      </div>
    {/if}
  </div>
</div>
