package linter

import (
	"fmt"
	"sort"
	"sync"
)

var (
	registry = &Registry{
		linters: make(map[string]Linter),
	}
)

type Registry struct {
	mu      sync.RWMutex
	linters map[string]Linter
}

func Register(linter Linter) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.linters[linter.Name()] = linter
}

func Get(name string) (Linter, error) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	linter, ok := registry.linters[name]
	if !ok {
		return nil, fmt.Errorf("linter %q not found", name)
	}
	return linter, nil
}

func All() []Linter {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	names := make([]string, 0, len(registry.linters))
	for name := range registry.linters {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]Linter, 0, len(names))
	for _, name := range names {
		result = append(result, registry.linters[name])
	}
	return result
}

func Names() []string {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	names := make([]string, 0, len(registry.linters))
	for name := range registry.linters {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

type Factory interface {
	Create(name string, description string) Linter
}

var factories = make(map[string]Factory)

func RegisterFactory(linterType string, factory Factory) {
	factories[linterType] = factory
}

func CreateLinter(linterType string, name string, description string) (Linter, error) {
	factory, ok := factories[linterType]
	if !ok {
		return nil, fmt.Errorf("no factory registered for linter type %q", linterType)
	}
	return factory.Create(name, description), nil
}
