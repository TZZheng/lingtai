package i18n

import (
	"encoding/json"
	"testing"
)

func loadLocaleForTest(t *testing.T, locale string) map[string]string {
	t.Helper()
	data, err := localeFS.ReadFile(locale + ".json")
	if err != nil {
		t.Fatalf("read %s locale: %v", locale, err)
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("parse %s locale: %v", locale, err)
	}
	return m
}

func copyStringMap(m map[string]string) map[string]string {
	cp := make(map[string]string, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp
}

func withActiveStringsForTest(t *testing.T, m map[string]string) {
	t.Helper()
	mu.Lock()
	old := activeStrings
	activeStrings = m
	mu.Unlock()
	t.Cleanup(func() {
		mu.Lock()
		activeStrings = old
		mu.Unlock()
	})
}

func TestSetLang_SwitchesLanguage(t *testing.T) {
	t.Cleanup(func() { SetLang("en") })

	if err := SetLang("zh"); err != nil {
		t.Fatalf("SetLang(\"zh\") returned error: %v", err)
	}
	if got := T("settings.language"); got != "语言" {
		t.Fatalf("after SetLang(\"zh\"), T(settings.language) = %q, want %q", got, "语言")
	}
}

func TestSetLang_InvalidLanguageLeavesStateUnchanged(t *testing.T) {
	t.Cleanup(func() { SetLang("en") })

	if err := SetLang("zh"); err != nil {
		t.Fatalf("SetLang(\"zh\") returned error: %v", err)
	}
	beforeLang := Lang()
	beforeText := T("settings.language")

	if err := SetLang("fr"); err == nil {
		t.Fatal("SetLang(\"fr\") returned nil error")
	}
	if got := Lang(); got != beforeLang {
		t.Fatalf("Lang() after invalid SetLang = %q, want %q", got, beforeLang)
	}
	if got := T("settings.language"); got != beforeText {
		t.Fatalf("T(settings.language) after invalid SetLang = %q, want %q", got, beforeText)
	}
}

func TestSetLang_ValidSwitchWorksAfterInvalidLanguage(t *testing.T) {
	t.Cleanup(func() { SetLang("en") })

	if err := SetLang("zh"); err != nil {
		t.Fatalf("SetLang(\"zh\") returned error: %v", err)
	}
	if err := SetLang("fr"); err == nil {
		t.Fatal("SetLang(\"fr\") returned nil error")
	}
	if err := SetLang("wen"); err != nil {
		t.Fatalf("SetLang(\"wen\") after invalid language returned error: %v", err)
	}
	if got := T("settings.language"); got != "言" {
		t.Fatalf("T(settings.language) after SetLang(\"wen\") = %q, want %q", got, "言")
	}
}

func TestT_FallsBackToEnglishWhenActiveLocaleMissesKey(t *testing.T) {
	t.Cleanup(func() { SetLang("en") })

	if err := SetLang("zh"); err != nil {
		t.Fatalf("SetLang(\"zh\") returned error: %v", err)
	}
	key := "settings.language"
	zh := loadLocaleForTest(t, "zh")
	gap := copyStringMap(zh)
	delete(gap, key)
	withActiveStringsForTest(t, gap)

	if got, want := T(key), "Language"; got != want {
		t.Fatalf("T(%q) = %q, want English fallback %q", key, got, want)
	}
}

func TestT_MissingEverywhereReturnsKey(t *testing.T) {
	t.Cleanup(func() { SetLang("en") })

	if err := SetLang("zh"); err != nil {
		t.Fatalf("SetLang(\"zh\") returned error: %v", err)
	}
	key := "__i18n_missing_everywhere__"
	if got := T(key); got != key {
		t.Fatalf("T(%q) = %q, want raw key", key, got)
	}
}

func TestLocaleJSONKeySetsAreComplete(t *testing.T) {
	en := loadLocaleForTest(t, "en")
	for _, locale := range []string{"en", "zh", "wen"} {
		m := loadLocaleForTest(t, locale)
		for key, value := range m {
			if value == "" {
				t.Fatalf("%s locale has empty value for key %q", locale, key)
			}
		}
		if len(m) != len(en) {
			t.Fatalf("%s locale has %d keys, want %d", locale, len(m), len(en))
		}
		for key := range en {
			if _, ok := m[key]; !ok {
				t.Fatalf("%s locale missing key %q", locale, key)
			}
		}
		for key := range m {
			if _, ok := en[key]; !ok {
				t.Fatalf("%s locale has extra key %q", locale, key)
			}
		}
	}
}
