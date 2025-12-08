package tui

import (
	"github.com/ktr0731/go-fuzzyfinder"
)

// SelectOne prompts the user to select one item from a list.
// items is the list of items to display.
// labelFunc returns the string representation of an item.
// previewFunc (optional) returns the preview string for an item.
func SelectOne[T any](items []T, labelFunc func(T) string, previewFunc func(T) string) (int, error) {
	idx, err := fuzzyfinder.Find(
		items,
		func(i int) string {
			return labelFunc(items[i])
		},
		fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			if i == -1 {
				return ""
			}
			if previewFunc != nil {
				return previewFunc(items[i])
			}
			return ""
		}),
	)
	if err != nil {
		return -1, err
	}
	return idx, nil
}
