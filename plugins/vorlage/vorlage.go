package vorlage

import (
	"context"
	"log"

	"bibliothek/plugins"
)

// Init registers the template plugin hooks into the global event registry.
func Init() {
	plugins.RegisterHook(plugins.EventBookReturned, func(ctx context.Context, payload any) error {
		p, ok := payload.(plugins.BookReturnedPayload)
		if !ok {
			return nil
		}

		log.Printf("Vorlage Plugin: Event OnBookReturned received! Copy ID: %s, Title: %s, Barcode: %s, Operator ID: %s",
			p.CopyID, p.Titel, p.BarcodeID, p.BearbeiterID)

		// Plugin logic here (e.g. sending a Slack webhook, printing a label, or triggering notifications)
		return nil
	})
}
