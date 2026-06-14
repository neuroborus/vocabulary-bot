package source

// Details holds optional per-source counters reported after Sync.
type Details struct {
	BookContextSkippedStored int
	BookContextEnriched      int
	RowsSkippedUnchanged     int
	BooksSkipped             int
	BooksFailed              int
	NotesFailed              int
	RowParseErrors           int
}

// DetailsProvider is implemented by adapters that expose post-sync counters.
type DetailsProvider interface {
	SyncDetails() Details
}
