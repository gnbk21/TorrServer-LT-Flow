package flow

// Coverage contains local server observations, never a decoder's playhead.
type Coverage struct {
	SchemaVersion           int                `json:"schema_version"`
	Known                   bool               `json:"known"`
	SampledAtMS             int64              `json:"sampled_at_ms"`
	AvailabilitySampledAtMS int64              `json:"availability_sampled_at_ms"`
	Truncated               bool               `json:"truncated"`
	PieceLength             int64              `json:"piece_length"`
	Pieces                  []CoveragePiece    `json:"pieces"`
	Frontiers               []CoverageFrontier `json:"frontiers"`
}
type CoveragePiece struct {
	Piece          int  `json:"piece"`
	Resident       bool `json:"resident"`
	Verified       bool `json:"verified"`
	Suppliers      *int `json:"suppliers,omitempty"`
	QueueKnown     bool `json:"queue_known"`
	Requested      int  `json:"requested"`
	ReceivingBytes int  `json:"receiving_bytes"`
	Writing        int  `json:"writing"`
	Finished       int  `json:"finished"`
	RequiredHead   bool `json:"required_head"`
	RequiredIndex  bool `json:"required_index"`
	Blocker        bool `json:"blocker"`
}
type CoverageFrontier struct {
	FileIndex     int   `json:"file_index"`
	Piece         int   `json:"piece"`
	ReadableBytes int64 `json:"readable_bytes"`
	VerifiedBytes int64 `json:"verified_bytes"`
	SampledAtMS   int64 `json:"sampled_at_ms"`
}
