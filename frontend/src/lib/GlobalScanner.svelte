<script>
  /**
   * GlobalScanner.svelte — Globaler Barcode-Scanner-Listener
   *
   * Erkennt Scanner-Eingaben (viele Tastenanschläge in < 30 ms, abgeschlossen mit Enter)
   * und unterscheidet sie von normalem Tipp-Verhalten. Leitet erkannte Barcodes an die
   * zentrale Kiosk-Logik weiter, ohne aktive Input-Felder zu stören.
   *
   * Kommunikation:
   *   - `appState.triggerStudentScan`  → Omnibox + KioskMode reagieren auf diesen Wert
   *   - `onBarcode` prop               → optionaler Callback für Parent-Komponenten
   *
   * Props:
   *   - isLoggedIn:  Boolean — Listener ist nur aktiv, wenn eingeloggt
   *   - activeTab:   String  — aktuell aktiver Tab, wird ggf. auf "kiosk" gesetzt
   *   - onNavigate:  (tab: string) => void — Callback zum Tab-Wechsel im Parent
   */

  import { appState } from "../inventur/lib/store.svelte.js";

  let {
    isLoggedIn = false,
    activeTab = "kiosk",
    onNavigate = /** @type {(tab: string) => void} */ (() => {}),
  } = $props();

  // ── Scanner-Buffer ──────────────────────────────────────────────────────────
  /** Zeichenpuffer für den aktuellen Scanner-Eingabestrahl */
  let buffer = "";
  /** Zeitstempel des letzten Tastendrucks (ms) */
  let lastKeyTime = 0;
  /**
   * Maximaler Zeitabstand zwischen zwei Tastendrücken, um als Scanner erkannt zu werden.
   * Handscanner feuern typischerweise < 20ms zwischen Zeichen; 50ms gibt Puffer für
   * langsamere Scanner und USB-Hubs mit Latenz.
   */
  const SCANNER_INTERVAL_MS = 50;
  /**
   * Minimale Barcode-Länge, um Fehl-Trigger durch kurze Tipp-Sequenzen zu vermeiden.
   * Alle Barcodes im System (S-XXXXX, B-XXXXX, L-XXXXX, G-XXXXX) sind ≥ 3 Zeichen.
   */
  const MIN_BARCODE_LENGTH = 3;

  // ── Flash-Feedback ──────────────────────────────────────────────────────────
  /** Zeigt kurzen visuellen Hinweis, wenn ein globaler Scan erkannt wurde */
  let flashVisible = $state(false);
  let flashBarcode = $state("");

  function showFlash(barcode = "") {
    flashBarcode = barcode;
    flashVisible = true;
    setTimeout(() => {
      flashVisible = false;
      flashBarcode = "";
    }, 1800);
  }

  // ── Hilfsfunktionen ─────────────────────────────────────────────────────────

  /**
   * Prüft, ob der aktuelle Fokus in einem Texteingabe-Element liegt.
   * In diesem Fall soll der globale Listener nicht eingreifen, damit
   * der Nutzer normal tippen kann (z. B. Suche im Katalog).
   * @returns {boolean}
   */
  function isFocusedOnTextInput() {
    const el = document.activeElement;
    if (!el) return false;
    const tag = el.tagName.toLowerCase();
    if (tag === "input" || tag === "textarea") {
      // Ausnahme: Login-Input und Kiosk-Inputs sollen globale Scans DURCHLASSEN,
      // weil sie für Scanner-Eingaben gebaut sind.
      const allowedIds = [
        "login-input",
        "kiosk-student-input",
        "kiosk-book-input",
        "omnibox-input",
      ];
      if (el.id && allowedIds.includes(el.id)) return false;
      // Alle anderen Inputs (z.B. Suchfelder im Katalog) blockieren den Listener
      return true;
    }
    // ContentEditable-Elemente (z. B. Markdown-Editoren) auch blockieren
    if (/** @type {HTMLElement} */ (el).isContentEditable) return true;
    return false;
  }

  /**
   * Verarbeitet einen vollständig erkannten Barcode.
   * Navigiert ggf. in den Kiosk-Tab und übergibt den Barcode via appState.
   * @param {string} barcode
   */
  function dispatchBarcode(barcode) {
    const trimmed = barcode.trim();
    if (!trimmed || trimmed.length < MIN_BARCODE_LENGTH) return;

    // Visuelles Feedback anzeigen
    showFlash(trimmed);

    // Zum Kiosk-Modus navigieren, falls wir uns nicht bereits dort befinden
    if (activeTab !== "kiosk") {
      onNavigate("kiosk");
    }

    // Barcode via reaktiven appState-Kanal an Omnibox/KioskMode übergeben.
    // Beide Komponenten reagieren auf appState.triggerStudentScan via $effect.
    appState.triggerStudentScan = trimmed;
  }

  // ── Globaler keydown-Listener ───────────────────────────────────────────────
  $effect(() => {
    if (!isLoggedIn) {
      // Nicht eingeloggt: Kein globaler Scanner-Listener (Login-Input übernimmt selbst)
      return;
    }

    /** @param {KeyboardEvent} e */
    function handleKeyDown(e) {
      // Modifier-Tasten ignorieren (Ctrl+C, Alt+Tab, etc.)
      if (e.ctrlKey || e.altKey || e.metaKey) return;

      // Wenn Fokus auf einem "echten" Texteingabefeld liegt → nicht eingreifen
      if (isFocusedOnTextInput()) return;

      const now = Date.now();
      const timeSinceLast = now - lastKeyTime;
      lastKeyTime = now;

      if (e.key === "Enter") {
        // Enter beendet immer den Scan-Puffer
        const captured = buffer;
        buffer = "";

        // Nur als Barcode werten, wenn:
        // 1. Der Buffer nicht leer ist
        // 2. Mindestlänge erfüllt
        // Die Zeitbedingung wurde bereits beim Aufbau des Buffers geprüft
        if (captured.length >= MIN_BARCODE_LENGTH) {
          dispatchBarcode(captured);
        }
        return;
      }

      // Nur druckbare Einzelzeichen in den Buffer aufnehmen
      if (e.key.length !== 1) {
        // Funktionstasten, Backspace, Tab etc. → Buffer zurücksetzen, kein Scanner
        if (e.key !== "Shift") {
          // Shift ist Teil normaler Scanner-Sequenzen (für Großbuchstaben), nicht resetten
          buffer = "";
        }
        return;
      }

      // Heuristik: Wenn das Zeichen zu langsam nach dem vorherigen kam,
      // handelt es sich um normales Tippen → Buffer zurücksetzen.
      // Ausnahme: Erstes Zeichen (lastKeyTime war 0 oder Buffer ist leer)
      if (buffer.length > 0 && timeSinceLast > SCANNER_INTERVAL_MS) {
        // Zu langsam für einen Scanner → als manuelles Tippen werten, Buffer verwerfen
        buffer = "";
        return;
      }

      buffer += e.key;
    }

    window.addEventListener("keydown", handleKeyDown, { capture: true });
    return () => {
      window.removeEventListener("keydown", handleKeyDown, { capture: true });
      buffer = "";
    };
  });
</script>

<!--
  Globaler Scanner-Erkennungs-Toast
  Erscheint kurz am oberen Bildschirmrand, wenn ein Barcode außerhalb des Kiosk-Modus
  erkannt und weitergeleitet wurde. Gibt dem Personal visuelles Feedback ohne Ablenkung.
-->
{#if flashVisible}
  <div
    class="scanner-toast"
    role="status"
    aria-live="polite"
    aria-label="Barcode erkannt: {flashBarcode}"
  >
    <span class="scanner-toast__icon" aria-hidden="true">
      <!-- Barcode-Icon -->
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M3 5v14M7 5v14M13 5v14M17 5v14M21 5v14M11 5v14"/>
      </svg>
    </span>
    <span class="scanner-toast__text">
      <span class="scanner-toast__label">Scanner erkannt</span>
      <span class="scanner-toast__barcode">{flashBarcode}</span>
    </span>
    <span class="scanner-toast__arrow" aria-hidden="true">→ Kiosk</span>
  </div>
{/if}

<style>
  .scanner-toast {
    position: fixed;
    top: 1.25rem;
    left: 50%;
    transform: translateX(-50%);
    z-index: 9999;

    display: flex;
    align-items: center;
    gap: 0.625rem;

    padding: 0.625rem 1.125rem;
    border-radius: 9999px;
    background: rgba(15, 23, 42, 0.92);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
    border: 1px solid rgba(255, 255, 255, 0.08);
    box-shadow:
      0 4px 24px rgba(0, 0, 0, 0.28),
      0 0 0 1px rgba(99, 102, 241, 0.25);

    color: #f1f5f9;
    font-family: inherit;
    font-size: 0.8125rem;
    font-weight: 600;
    letter-spacing: 0.01em;
    white-space: nowrap;
    pointer-events: none;

    animation: scanner-toast-in 0.22s cubic-bezier(0.16, 1, 0.3, 1) forwards,
               scanner-toast-out 0.3s cubic-bezier(0.4, 0, 1, 1) 1.4s forwards;
  }

  .scanner-toast__icon {
    display: flex;
    align-items: center;
    color: #818cf8; /* indigo-400 */
  }

  .scanner-toast__text {
    display: flex;
    flex-direction: column;
    gap: 0.0625rem;
    line-height: 1.2;
  }

  .scanner-toast__label {
    font-size: 0.6875rem;
    font-weight: 500;
    color: #94a3b8; /* slate-400 */
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }

  .scanner-toast__barcode {
    font-size: 0.875rem;
    font-weight: 700;
    font-variant-numeric: tabular-nums;
    color: #e2e8f0;
    letter-spacing: 0.04em;
  }

  .scanner-toast__arrow {
    font-size: 0.75rem;
    color: #6366f1; /* indigo-500 */
    font-weight: 700;
    letter-spacing: 0.04em;
    padding-left: 0.25rem;
    border-left: 1px solid rgba(255, 255, 255, 0.1);
    padding-left: 0.625rem;
  }

  @keyframes scanner-toast-in {
    from {
      opacity: 0;
      transform: translateX(-50%) translateY(-0.75rem) scale(0.95);
    }
    to {
      opacity: 1;
      transform: translateX(-50%) translateY(0) scale(1);
    }
  }

  @keyframes scanner-toast-out {
    from {
      opacity: 1;
      transform: translateX(-50%) translateY(0) scale(1);
    }
    to {
      opacity: 0;
      transform: translateX(-50%) translateY(-0.5rem) scale(0.97);
    }
  }
</style>
