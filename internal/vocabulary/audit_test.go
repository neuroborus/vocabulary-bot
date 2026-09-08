package vocabulary_test

import (
	"testing"

	"github.com/neuroborus/vocabulary-bot/internal/vocabulary"
)

func TestClusterItemFormsHealthyItemHasOneCluster(t *testing.T) {
	t.Parallel()

	item := vocabulary.Item{
		Forms: []vocabulary.Form{
			{Value: "decelerate"},
			{Value: "to decelerate"},
		},
	}

	clusters := vocabulary.ClusterItemForms(item)
	if len(clusters) != 1 {
		t.Fatalf("clusters = %d, want 1: %#v", len(clusters), clusters)
	}
	if vocabulary.IsCollapsed(item) {
		t.Fatal("healthy item must not be reported as collapsed")
	}
}

func TestClusterItemFormsSplitsPhrasesSharingOnlyFunctionWord(t *testing.T) {
	t.Parallel()

	item := vocabulary.Item{
		Forms: []vocabulary.Form{
			{Value: "fit the bill"},
			{Value: "In the beginning"},
			{Value: "get the sack"},
		},
	}

	clusters := vocabulary.ClusterItemForms(item)
	if len(clusters) != 3 {
		t.Fatalf("clusters = %d, want 3: %#v", len(clusters), clusters)
	}
	if !vocabulary.IsCollapsed(item) {
		t.Fatal("phrases sharing only \"the\" must be reported as collapsed")
	}
	if clusters[0].Forms[0] != "fit the bill" {
		t.Fatalf("first cluster = %q, want first-appearance order", clusters[0].Forms[0])
	}
}

func TestClusterItemFormsKeepsRelatedFormsTogether(t *testing.T) {
	t.Parallel()

	item := vocabulary.Item{
		Forms: []vocabulary.Form{
			{Value: "the same thing"},
			{Value: "the same"},
			{Value: "get the sack"},
		},
	}

	clusters := vocabulary.ClusterItemForms(item)
	if len(clusters) != 2 {
		t.Fatalf("clusters = %d, want 2: %#v", len(clusters), clusters)
	}
	if len(clusters[0].Forms) != 2 {
		t.Fatalf("first cluster forms = %#v, want the two 'same' forms together", clusters[0].Forms)
	}
}

func TestClusterItemFormsEmpty(t *testing.T) {
	t.Parallel()

	if clusters := vocabulary.ClusterItemForms(vocabulary.Item{}); clusters != nil {
		t.Fatalf("clusters = %#v, want nil for an item with no forms", clusters)
	}
}
