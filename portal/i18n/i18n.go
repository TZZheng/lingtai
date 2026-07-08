package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"sync"
)

//go:embed en.json zh.json wen.json
var localeFS embed.FS

var (
	mu             sync.RWMutex
	lang           = "en"
	activeStrings  map[string]string
	englishStrings map[string]string
)

var supportedLocales = map[string]bool{
	"en":  true,
	"zh":  true,
	"wen": true,
}

func init() {
	m, err := load("en")
	if err != nil {
		panic(err)
	}
	activeStrings = m
	englishStrings = m
}

func SetLang(l string) error {
	m, err := load(l)
	if err != nil {
		return err
	}
	mu.Lock()
	defer mu.Unlock()
	lang = l
	activeStrings = m
	return nil
}

func load(l string) (map[string]string, error) {
	if !supportedLocales[l] {
		return nil, fmt.Errorf("unsupported language %q", l)
	}
	data, err := localeFS.ReadFile(l + ".json")
	if err != nil {
		return nil, fmt.Errorf("load language %q: %w", l, err)
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse language %q: %w", l, err)
	}
	return m, nil
}

func T(key string) string {
	mu.RLock()
	defer mu.RUnlock()
	if s, ok := activeStrings[key]; ok {
		return s
	}
	if s, ok := englishStrings[key]; ok {
		return s
	}
	return key
}

func TF(key string, args ...any) string {
	return fmt.Sprintf(T(key), args...)
}

func Lang() string {
	mu.RLock()
	defer mu.RUnlock()
	return lang
}
