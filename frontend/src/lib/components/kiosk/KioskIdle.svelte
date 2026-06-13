<script>
  import { kioskStore } from "../../stores/kiosk.svelte.js";

  // ── Svelte Action: Autofocus-Lock ───────────────────────────────────
  function keepFocus(node, options) {
    let active = options.active;

    function enforceFocus() {
      if (active && !node.disabled) {
        node.focus();
      }
    }

    requestAnimationFrame(enforceFocus);

    function onWindowClick(e) {
      if (!active || node.disabled) return;
      const isInteractive = e.target.closest('button, a, input, select, textarea, [role="button"], dialog, .modal');
      if (!isInteractive) {
        enforceFocus();
      }
    }

    function onBlur() {
      if (!active || node.disabled) return;
      setTimeout(() => {
        if (active && !node.disabled && (document.activeElement === document.body || document.activeElement === null)) {
          enforceFocus();
        }
      }, 50);
    }

    window.addEventListener('click', onWindowClick, { capture: true });
    node.addEventListener('blur', onBlur);

    return {
      update(newOptions) {
        active = newOptions.active;
        if (active) {
          requestAnimationFrame(enforceFocus);
        }
      },
      destroy() {
        window.removeEventListener('click', onWindowClick, { capture: true });
        node.removeEventListener('blur', onBlur);
      }
    };
  }
</script>

<div class="bg-white p-8 rounded-2xl shadow-sm border border-slate-200 text-center max-w-xl mx-auto {kioskStore.isShaking ? 'animate-shake' : ''}">
  <div class="w-16 h-16 bg-blue-100 text-blue-600 rounded-full flex items-center justify-center mx-auto mb-6">
    <svg class="w-8 h-8" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4"/></svg>
  </div>
  <h2 class="text-2xl font-bold text-slate-800 mb-2">Ausleihe starten</h2>
  <p class="text-slate-500 mb-8">Scanne zuerst den Schülerausweis, um das Profil aufzurufen.</p>
  <form onsubmit={(e) => { e.preventDefault(); kioskStore.handleStudentSubmit(); }}>
    <input type="text" id="kiosk-student-input" bind:value={kioskStore.studentInputVal} disabled={kioskStore.isScanningStudent}
           use:keepFocus={{ active: !kioskStore.isScanningStudent }}
           placeholder="S-XXXXXX scannen..." autocomplete="off"
           class="w-full bg-slate-50 border-2 border-blue-200 focus:border-blue-500 focus:ring-4 focus:ring-blue-500/20 rounded-xl px-5 py-4 text-xl font-medium outline-none transition-all text-center placeholder:text-slate-400" />
  </form>
</div>

<style>
  @keyframes shake {
    0%, 100% { transform: translateX(0); }
    25% { transform: translateX(-8px); }
    75% { transform: translateX(8px); }
  }
  .animate-shake {
    animation: shake 0.3s cubic-bezier(.36,.07,.19,.97) both;
  }
</style>
