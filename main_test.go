package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func setenv(key, value string) {
	if err := os.Setenv(key, value); err != nil {
		panic(err)
	}
}

func unsetenv(key string) {
	if err := os.Unsetenv(key); err != nil {
		panic(err)
	}
}

func setupSuccessEnv() {
	os.Clearenv()
	setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")
	setenv("JWT_SECRET", "this-is-a-very-secret-key-that-is-at-least-32-bytes")
	setenv("APP_ENCRYPTION_KEY", "12345678901234567890123456789012")
	setenv("PORT", "8080")
	setenv("COOKIE_SECURE", "true")
	setenv("ENFORCE_PROD_SECRETS", "false")
}

func restoreEnv(originalEnv []string) {
	os.Clearenv()
	for _, e := range originalEnv {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			setenv(parts[0], parts[1])
		}
	}
}

func TestLoadConfig_Success(t *testing.T) {
	originalEnv := os.Environ()
	defer restoreEnv(originalEnv)

	setupSuccessEnv()
	dsn, jwt, port, secure := loadConfig()

	if dsn != "postgres://user:pass@localhost:5432/db" {
		t.Errorf("Expected dsn, got %s", dsn)
	}
	if jwt != "this-is-a-very-secret-key-that-is-at-least-32-bytes" {
		t.Errorf("Expected jwt, got %s", jwt)
	}
	if port != "8080" {
		t.Errorf("Expected port, got %s", port)
	}
	if !secure {
		t.Errorf("Expected secure to be true, got %v", secure)
	}
}

func testFatal(t *testing.T, testName string, setupEnv func()) {
	if os.Getenv("BE_CRASHER") == "1" {
		setupEnv()
		loadConfig()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run="+testName)
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}

func TestLoadConfig_MissingDatabaseURL(t *testing.T) {
	testFatal(t, "TestLoadConfig_MissingDatabaseURL", func() {
		setupSuccessEnv()
		unsetenv("DATABASE_URL")
	})
}

func TestLoadConfig_ShortJWT(t *testing.T) {
	testFatal(t, "TestLoadConfig_ShortJWT", func() {
		setupSuccessEnv()
		setenv("JWT_SECRET", "short")
	})
}

func TestLoadConfig_InvalidAESKey(t *testing.T) {
	testFatal(t, "TestLoadConfig_InvalidAESKey", func() {
		setupSuccessEnv()
		setenv("APP_ENCRYPTION_KEY", "invalid")
	})
}

func TestLoadConfig_MissingPort(t *testing.T) {
	testFatal(t, "TestLoadConfig_MissingPort", func() {
		setupSuccessEnv()
		unsetenv("PORT")
	})
}

func TestLoadConfig_EnforceProdSecrets_KnownJWT(t *testing.T) {
	testFatal(t, "TestLoadConfig_EnforceProdSecrets_KnownJWT", func() {
		setupSuccessEnv()
		setenv("JWT_SECRET", "super-secret-default-key-at-least-32-bytes")
		setenv("ENFORCE_PROD_SECRETS", "true")
	})
}

func TestLoadConfig_EnforceProdSecrets_KnownAES(t *testing.T) {
	testFatal(t, "TestLoadConfig_EnforceProdSecrets_KnownAES", func() {
		setupSuccessEnv()
		setenv("APP_ENCRYPTION_KEY", "super-secure-aes-key-32-chars-ok")
		setenv("ENFORCE_PROD_SECRETS", "true")
	})
}
