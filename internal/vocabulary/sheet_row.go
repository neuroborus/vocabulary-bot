package vocabulary

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (s *Service) applySheetRowUpdate(ctx context.Context, item Item, draft Draft, now time.Time) (Item, MergeOutcome, error) {
	previousNormalizedKey := item.NormalizedKey
	anchorIndex := sheetAnchorIndex(item, draft.Anchor.SheetName, draft.Anchor.RowNumber)
	if anchorIndex < 0 {
		return Item{}, MergeOutcome{}, fmt.Errorf("sheet row anchor not found on item %q", item.NormalizedKey)
	}

	oldSnapshot := sheetSnapshotFromAnchor(item.Anchors[anchorIndex])
	lookupKeys := BuildLookupKeys(draft.RawWord)
	if len(lookupKeys) == 0 {
		return Item{}, MergeOutcome{}, ErrEmptyWord
	}

	matches, err := s.repository.FindByLookupKeys(ctx, lookupKeys)
	if err != nil {
		return Item{}, MergeOutcome{}, fmt.Errorf("find vocabulary item: %w", err)
	}

	otherItems := make([]Item, 0, len(matches))
	for _, match := range matches {
		if match.NormalizedKey == item.NormalizedKey {
			continue
		}
		otherItems = append(otherItems, match)
	}

	if len(otherItems) > 1 {
		return Item{}, MergeOutcome{
			LookupKeys: lookupKeys,
			Ambiguous:  true,
		}, ErrAmbiguousMatch
	}

	if len(otherItems) == 1 {
		target := otherItems[0]
		if err := s.reassignSheetRow(ctx, &item, target, draft, oldSnapshot, anchorIndex, now); err != nil {
			return Item{}, MergeOutcome{}, err
		}

		return item, MergeOutcome{
			NormalizedKey: item.NormalizedKey,
			LookupKeys:    append([]string(nil), item.LookupKeys...),
			Updated:       true,
		}, nil
	}

	subtractSheetContribution(&item, oldSnapshot)
	MergeIntoItem(&item, draft, now)
	updateSheetAnchorState(&item, anchorIndex, draft, now)
	if sheetRowWordChanged(oldSnapshot, draft) {
		item.DisplayWord = strings.TrimSpace(draft.RawWord)
	}
	rebuildItemLookupState(&item)

	if err := s.saveItem(ctx, item, previousNormalizedKey); err != nil {
		return Item{}, MergeOutcome{}, err
	}

	return item, MergeOutcome{
		NormalizedKey: item.NormalizedKey,
		LookupKeys:    append([]string(nil), item.LookupKeys...),
		Updated:       true,
	}, nil
}

func (s *Service) reassignSheetRow(
	ctx context.Context,
	source *Item,
	target Item,
	draft Draft,
	oldSnapshot SheetRowSnapshot,
	anchorIndex int,
	now time.Time,
) error {
	previousSourceKey := source.NormalizedKey
	previousTargetKey := target.NormalizedKey

	subtractSheetContribution(source, oldSnapshot)
	source.Anchors = removeAnchorAt(source.Anchors, anchorIndex)
	rebuildItemLookupState(source)
	if err := s.saveItem(ctx, *source, previousSourceKey); err != nil {
		return fmt.Errorf("update source item after sheet row move: %w", err)
	}

	MergeIntoItem(&target, draft, now)
	targetAnchorIndex := sheetAnchorIndex(target, draft.Anchor.SheetName, draft.Anchor.RowNumber)
	if targetAnchorIndex >= 0 {
		updateSheetAnchorState(&target, targetAnchorIndex, draft, now)
	} else {
		anchor := normalizeAnchor(draft, now)
		anchor.RowSnapshot = sheetRowSnapshotFromDraft(draft)
		target.Anchors = append(target.Anchors, anchor)
	}
	rebuildItemLookupState(&target)

	if err := s.saveItem(ctx, target, previousTargetKey); err != nil {
		return fmt.Errorf("update target item after sheet row move: %w", err)
	}

	return nil
}

func (s *Service) saveItem(ctx context.Context, item Item, previousNormalizedKey string) error {
	if previousNormalizedKey != "" && previousNormalizedKey != item.NormalizedKey {
		if err := s.repository.Replace(ctx, item, previousNormalizedKey); err != nil {
			return fmt.Errorf("replace vocabulary item: %w", err)
		}
		return nil
	}

	if err := s.repository.Update(ctx, item); err != nil {
		return fmt.Errorf("update vocabulary item: %w", err)
	}

	return nil
}

func sheetRowSnapshotFromDraft(draft Draft) *SheetRowSnapshot {
	return &SheetRowSnapshot{
		RawWord:      strings.TrimSpace(draft.RawWord),
		Translations: append([]string(nil), cleanTextList(draft.Translations, translationKey)...),
		Contexts:     append([]string(nil), cleanTextList(draft.Contexts, contextKey)...),
		Notes:        append([]string(nil), cleanTextList(draft.Notes, comparableTextKey)...),
		Tags:         append([]string(nil), cleanTextList(draft.Tags, comparableTextKey)...),
	}
}

func sheetSnapshotFromAnchor(anchor SourceAnchor) SheetRowSnapshot {
	if anchor.RowSnapshot != nil {
		return SheetRowSnapshot{
			RawWord:      anchor.RowSnapshot.RawWord,
			Translations: append([]string(nil), anchor.RowSnapshot.Translations...),
			Contexts:     append([]string(nil), anchor.RowSnapshot.Contexts...),
			Notes:        append([]string(nil), anchor.RowSnapshot.Notes...),
			Tags:         append([]string(nil), anchor.RowSnapshot.Tags...),
		}
	}

	if anchor.RowFingerprint == "" {
		return SheetRowSnapshot{}
	}

	parts := strings.Split(anchor.RowFingerprint, sheetFingerprintSeparator)
	if len(parts) == 0 {
		return SheetRowSnapshot{}
	}

	return SheetRowSnapshot{RawWord: parts[0]}
}

func subtractSheetContribution(item *Item, snapshot SheetRowSnapshot) {
	if snapshot.RawWord != "" && !hasNonSheetAnchors(*item) {
		removeKey := formKey(snapshot.RawWord)
		forms := item.Forms[:0]
		for _, form := range item.Forms {
			if formKey(form.Value) != removeKey {
				forms = append(forms, form)
			}
		}
		item.Forms = forms
	}

	item.Translations = subtractStringsByKey(item.Translations, snapshot.Translations, translationKey)
	item.Contexts = subtractStringsByKey(item.Contexts, snapshot.Contexts, contextKey)
	item.Notes = subtractStringsByKey(item.Notes, snapshot.Notes, comparableTextKey)
	item.Tags = subtractStringsByKey(item.Tags, snapshot.Tags, comparableTextKey)
}

func hasNonSheetAnchors(item Item) bool {
	for _, anchor := range item.Anchors {
		if anchor.Source != SourceGoogleSheet {
			return true
		}
	}

	return false
}

func sheetRowWordChanged(previous SheetRowSnapshot, draft Draft) bool {
	return strings.TrimSpace(previous.RawWord) != strings.TrimSpace(draft.RawWord)
}

func subtractStringsByKey(existing []string, remove []string, key func(string) string) []string {
	if len(remove) == 0 {
		return append([]string(nil), existing...)
	}

	removeKeys := make(map[string]struct{}, len(remove))
	for _, value := range remove {
		if normalized := key(value); normalized != "" {
			removeKeys[normalized] = struct{}{}
		}
	}

	result := make([]string, 0, len(existing))
	for _, value := range existing {
		if _, ok := removeKeys[key(value)]; ok {
			continue
		}
		result = append(result, value)
	}

	return result
}

func updateSheetAnchorState(item *Item, anchorIndex int, draft Draft, now time.Time) {
	item.Anchors[anchorIndex].LastSeenAt = now
	item.Anchors[anchorIndex].RowFingerprint = DraftFingerprint(draft)
	item.Anchors[anchorIndex].RowSnapshot = sheetRowSnapshotFromDraft(draft)
	if draft.Anchor.SourceLabel != "" {
		item.Anchors[anchorIndex].SourceLabel = draft.Anchor.SourceLabel
	}
}

func rebuildItemLookupState(item *Item) {
	lookupKeys := make([]string, 0)
	seen := make(map[string]struct{})

	for _, form := range item.Forms {
		for _, lookupKey := range form.LookupKeys {
			if lookupKey == "" {
				continue
			}
			if _, ok := seen[lookupKey]; ok {
				continue
			}
			seen[lookupKey] = struct{}{}
			lookupKeys = append(lookupKeys, lookupKey)
		}
	}

	item.LookupKeys = lookupKeys
	if len(lookupKeys) == 0 {
		return
	}

	displayWord := strings.TrimSpace(item.DisplayWord)
	if displayWord == "" && len(item.Forms) > 0 {
		displayWord = item.Forms[len(item.Forms)-1].Value
	}

	candidateKeys := BuildLookupKeys(displayWord)
	if len(candidateKeys) > 0 {
		item.NormalizedKey = candidateKeys[0]
		item.DisplayWord = displayWord
		return
	}

	item.NormalizedKey = lookupKeys[0]
}

func sheetAnchorIndex(item Item, sheetName string, rowNumber int) int {
	sheetName = strings.TrimSpace(sheetName)
	for index, anchor := range item.Anchors {
		if anchor.Source != SourceGoogleSheet {
			continue
		}
		if anchor.RowNumber != rowNumber {
			continue
		}
		if strings.TrimSpace(anchor.SheetName) != sheetName {
			continue
		}

		return index
	}

	return -1
}

func removeAnchorAt(anchors []SourceAnchor, index int) []SourceAnchor {
	if index < 0 || index >= len(anchors) {
		return anchors
	}

	return append(append([]SourceAnchor(nil), anchors[:index]...), anchors[index+1:]...)
}
