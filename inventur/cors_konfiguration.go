package inventur

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
)

func loadAllowedOriginsFromEnv() map[string]struct{} {
	allowed := map[string]struct{}{}

	addOrigin := func(raw string) {
		normalized, err := normalizeOrigin(raw)
		if err != nil {
			log.Fatalf("Ungültiger CORS-Origin in Umgebungsvariablen: %q (%v)", raw, err)
		}
		allowed[normalized] = struct{}{}
	}

	single := strings.TrimSpace(os.Getenv("ALLOWED_ORIGIN"))
	if single != "" {
		addOrigin(single)
	}

	for _, origin := range strings.Split(strings.TrimSpace(os.Getenv("ALLOWED_ORIGINS")), ",") {
		trimmed := strings.TrimSpace(origin)
		if trimmed == "" {
			continue
		}
		addOrigin(trimmed)
	}

	addOrigin("http://localhost:5173")
	addOrigin("http://127.0.0.1:5173")

	return allowed
}

func normalizeOrigin(origin string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(origin))
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("nur http/https erlaubt")
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("host fehlt")
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return "", fmt.Errorf("path ist nicht erlaubt")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("query/fragment ist nicht erlaubt")
	}

	return strings.ToLower(parsed.Scheme + "://" + parsed.Host), nil
}
