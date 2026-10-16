package repository

import (
	"context"

	"bibliothek/db"
	"bibliothek/pkg/mitteltopf"
)

// Lieferant ist ein Händler, bei dem die Schule bestellt, etwa eine Buchhandlung.
type Lieferant struct {
	ID           string
	Name         string
	Email        string
	Kundennummer string

	// IstHauptlieferant kennzeichnet den einen Händler, über den die Schule bestellt. Er
	// bekommt mit der Bestellmail den Bestelllink, wählt darüber die Größe der Etiketten,
	// beklebt die Bücher selbst und bestätigt damit die Bestellung. An dem Merkmal hängen
	// deshalb drei Dinge:
	//
	//   - der Token zur Bestätigung und das große Lernmittel-Etikett im Anhang der Mail,
	//   - die Exemplare entstehen als „Etikett vorhanden" und stehen nicht auf der Liste zum
	//     Nachdrucken,
	//   - die Vorauswahl im Bestellformular.
	//
	// Es ist ein Merkmal und nicht drei Schalter, weil alle drei denselben Händler
	// beschreiben: Mit „Bestelllink, aber nicht beklebt" beklebte der Händler, und die
	// Bibliothek druckte dieselben Etiketten noch einmal.
	IstHauptlieferant bool

	// KundennummerSchultraeger ist das zweite Kundenkonto beim selben Händler, für
	// Bestellungen aus Mitteln des Schulträgers (Schülerbücherei). Leer heißt: dieselbe
	// Nummer wie Kundennummer (KundennummerFuer).
	KundennummerSchultraeger string
}

// KundennummerFuer liefert die Kundennummer, die auf einer Bestellung aus dem genannten Topf
// steht: für den Schulträger die zweite Nummer, wenn eine hinterlegt ist, sonst und für Mittel
// des Landes immer die erste.
func (l *Lieferant) KundennummerFuer(mittel string) string {
	if mittel == mitteltopf.Schultraeger && l.KundennummerSchultraeger != "" {
		return l.KundennummerSchultraeger
	}
	return l.Kundennummer
}

// LieferantenAbfragen liest die Lieferanten aus der Datenbank.
type LieferantenAbfragen struct {
	db db.Pool
}

// NewLieferantenAbfragen liefert die Abfragen der Lieferanten an Postgres.
func NewLieferantenAbfragen(pool db.Pool) *LieferantenAbfragen {
	return &LieferantenAbfragen{db: pool}
}

// HatHauptlieferant meldet, ob ein Hauptlieferant eingerichtet ist. Die Oberfläche braucht
// nur diese Antwort und keinen Datensatz: Ohne Hauptlieferant bekommt niemand einen Link zur
// Bestätigung, und der Hinweis auf die fehlende öffentliche Adresse entfällt.
func (a *LieferantenAbfragen) HatHauptlieferant(ctx context.Context) (bool, error) {
	var vorhanden bool
	err := a.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM lieferanten WHERE ist_hauptlieferant)`).Scan(&vorhanden)
	return vorhanden, err
}

// Lade lädt einen Lieferanten über seine ID.
func (a *LieferantenAbfragen) Lade(ctx context.Context, id string) (*Lieferant, error) {
	var l Lieferant
	l.ID = id
	err := a.db.QueryRow(ctx, `
		SELECT name, email, kundennummer, ist_hauptlieferant, kundennummer_schultraeger
		FROM lieferanten
		WHERE id = $1
	`, id).Scan(&l.Name, &l.Email, &l.Kundennummer, &l.IstHauptlieferant, &l.KundennummerSchultraeger)
	if err != nil {
		return nil, err
	}
	return &l, nil
}
