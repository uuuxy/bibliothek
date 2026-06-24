// Kategorien und ihre zugeordneten Berechtigungen für den PermissionManager.
// Aus der Komponente ausgelagert (Daten statt Markup) — hält PermissionManager schlank.
export const permissionsMetadata = [
  {
    category: "Schülerverwaltung",
    icon: "👤",
    items: [
      { key: "view_students", label: "Schülerdatei anzeigen", desc: "Erlaubt das Suchen und Einsehen von Schülerdaten und Klassen" },
      { key: "create_students", label: "Schüler hinzufügen", desc: "Ermöglicht das manuelle Anlegen neuer Schüler" },
      { key: "delete_students", label: "Schüler löschen", desc: "Erlaubt das Entfernen von Schülern aus der Datenbank" },
      { key: "import_students", label: "LUSD / CSV Import", desc: "Ermöglicht den Import von Schülerdaten per CSV-Datei" },
      { key: "upload_photos", label: "Ausweisfotos hochladen", desc: "Erlaubt die Aufnahme und Zuweisung von Ausweisfotos per Webcam" }
    ]
  },
  {
    category: "Medien & Inventar",
    icon: "📚",
    items: [
      { key: "view_books", label: "Medienkatalog anzeigen", desc: "Erlaubt das Suchen und Anzeigen von Buchtiteln und Exemplaren" },
      { key: "edit_books", label: "Bücher / Notizen bearbeiten", desc: "Ermöglicht das Hinzufügen von Schadensnotizen an Exemplaren" },
      { key: "delete_books", label: "Bücher & Exemplare löschen", desc: "Erlaubt das Löschen von Exemplaren und Buchtiteln" },
      { key: "inventory_scan", label: "Inventur durchführen", desc: "Ermöglicht das Einscannen von Büchern während einer aktiven Inventur" }
    ]
  },
  {
    category: "Bestellungen & Kiosk",
    icon: "🛒",
    items: [
      { key: "view_orders", label: "Bestellungen anzeigen", desc: "Erlaubt das Einsehen von Buchbestellungen und Lieferanten-Order" },
      { key: "create_orders", label: "Bestellungen verwalten", desc: "Ermöglicht das Bestellen neuer Bücher und Freigeben von Lieferungen" },
      { key: "view_graduates", label: "Abgängerliste einsehen", desc: "Erlaubt das Einsehen von Schulabgängern mit ausstehenden Büchern" }
    ]
  },
  {
    category: "Administration & System",
    icon: "⚙️",
    items: [
      { key: "view_stats", label: "Statistiken anzeigen", desc: "Zeigt Systemstatistiken und Ausleih-Auswertungen" },
      { key: "audit_logs", label: "Sicherheits-Logbuch einsehen", desc: "Ermöglicht den Zugriff auf das Enterprise Audit-Logbuch" },
      { key: "manage_users", label: "Benutzer & Rechte verwalten", desc: "Ermöglicht die Verwaltung von Benutzern und Berechtigungen" }
    ]
  }
];
