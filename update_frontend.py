import re

with open('frontend/src/lib/components/BookExemplarStatusEditor.svelte', 'r') as f:
    content = f.read()

new_content = content.replace("""  let editStatusType = $state(
    ex.ist_ausleihbar
      ? "Verfügbar"
      : (ex.ist_ausgesondert || (ex.zustand_notiz && ex.zustand_notiz.toLowerCase().includes("verloren")))
        ? "Verloren"
        : "Gesperrt (Defekt/Reserviert)"
  );""", """  let initialEditStatusType = ex.ist_ausleihbar
    ? "Verfügbar"
    : (ex.ist_ausgesondert || (ex.zustand_notiz && ex.zustand_notiz.toLowerCase().includes("verloren")))
      ? "Verloren"
      : "Gesperrt (Defekt/Reserviert)";
  let editStatusType = $state(initialEditStatusType);""")

with open('frontend/src/lib/components/BookExemplarStatusEditor.svelte', 'w') as f:
    f.write(new_content)
print("Updated frontend")
