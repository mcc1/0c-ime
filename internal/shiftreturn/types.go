//go:build shiftreturneval

// Package shiftreturn implements the isolated v2 offline exchange contract.
package shiftreturn

type Segment struct {
	Kind    string `json:"kind"`
	Begin   int    `json:"begin"`
	End     int    `json:"end"`
	Reading string `json:"reading"`
}

type Gold struct {
	Segments         []Segment `json:"segments"`
	AcceptedSurfaces []string  `json:"accepted_surfaces"`
}

type Fixture struct {
	Version         string   `json:"version"`
	Kind            string   `json:"kind"`
	Demo            bool     `json:"demo"`
	ID              string   `json:"id"`
	PrototypeID     string   `json:"prototype_id"`
	Intent          string   `json:"intent"`
	Category        string   `json:"category"`
	Stratum         string   `json:"stratum"`
	ContextClass    string   `json:"context_class"`
	SourceRef       string   `json:"source_ref"`
	SourcePriority  int      `json:"source_priority"`
	Families        []string `json:"families"`
	PreludeEvents   []Event  `json:"prelude_events"`
	ExpectedPrelude string   `json:"expected_prelude"`
	Events          []Event  `json:"events"`
	ExpectedRaw     string   `json:"expected_raw"`
	Gold            []Gold   `json:"gold"`
}

type Query struct {
	Version      string `json:"version"`
	Kind         string `json:"kind"`
	Demo         bool   `json:"demo"`
	Raw          string `json:"raw"`
	Eligible     bool   `json:"eligible"`
	ContextUnits []struct {
		Text         string `json:"text"`
		Reading      string `json:"reading"`
		Literal      bool   `json:"literal"`
		OriginalKeys string `json:"original_keys"`
	} `json:"context_units"`
	CustomTokens []string `json:"custom_tokens"`
}

type Hypothesis struct {
	Version         string    `json:"version"`
	Kind            string    `json:"kind"`
	Demo            bool      `json:"demo"`
	ID              string    `json:"id"`
	Identity        string    `json:"identity"`
	DisplayIdentity string    `json:"display_identity"`
	Cut             int       `json:"cut"`
	FirstReading    string    `json:"first_reading"`
	Readings        []string  `json:"readings"`
	ParsedEnd       int       `json:"parsed_end"`
	K               int       `json:"k"`
	TailStart       int       `json:"tail_start"`
	Segments        []Segment `json:"segments"`
	DisplaySegments []Segment `json:"display_segments"`
	Display         string    `json:"display"`
	Features        []float64 `json:"features"`
	Kept            bool      `json:"kept"`
}

type Snapshot struct {
	Version          string    `json:"version"`
	Kind             string    `json:"kind"`
	Demo             bool      `json:"demo"`
	EpisodeID        string    `json:"episode_id"`
	Condition        string    `json:"condition"`
	EventIndex       int       `json:"event_index"`
	KeyOffset        int       `json:"key_offset"`
	App              string    `json:"app"`
	Prelude          string    `json:"prelude"`
	Display          string    `json:"display"`
	View             string    `json:"view"`
	Commit           string    `json:"commit"`
	Handled          bool      `json:"handled"`
	Eligible         bool      `json:"eligible"`
	Closed           bool      `json:"closed"`
	SelectedID       *string   `json:"selected_id"`
	Freeze           []Segment `json:"freeze"`
	GenerateCalls    *int      `json:"generate_calls"`
	ScoreCalls       *int      `json:"score_calls"`
	LearningUpdates  *int      `json:"learning_updates"`
	PersistentWrites *int      `json:"persistent_writes"`
	Protected        bool      `json:"protected"`
	ResolutionCount  int       `json:"resolution_count"`
	RevisionCount    int       `json:"revision_count"`
	SafetyStatus     string    `json:"safety_status"`
	SafetyReason     string    `json:"safety_reason"`
	QueryID          *string   `json:"query_id"`
	Handoff          *Handoff  `json:"handoff"`
}

type Model struct {
	Version        string    `json:"version"`
	Kind           string    `json:"kind"`
	Demo           bool      `json:"demo"`
	Name           string    `json:"name"`
	FeatureNames   []string  `json:"feature_names"`
	Active         []int     `json:"active"`
	Coefficients   []float64 `json:"coefficients"`
	Means          []float64 `json:"means"`
	Stds           []float64 `json:"stds"`
	Counts         []int     `json:"counts"`
	Margin         float64   `json:"margin"`
	CodeLockSHA256 string    `json:"code_lock_sha256"`
	FitLockSHA256  string    `json:"fit_lock_sha256"`
	TrainingLoss   float64   `json:"training_loss"`
	GradientL2     float64   `json:"gradient_l2"`
	SelectedOn     string    `json:"selected_on"`
}

type Component struct {
	ID                    string   `json:"id"`
	RowIDs                []string `json:"row_ids"`
	Split                 string   `json:"split"`
	PrimaryReturn         *string  `json:"primary_return"`
	PrimaryLiteral        *string  `json:"primary_literal"`
	Categories            []string `json:"categories"`
	PrimaryReturnMetadata *struct {
		ID               string `json:"id"`
		Category         string `json:"category"`
		Stratum          string `json:"stratum"`
		ContextClass     string `json:"context_class"`
		ContrastEligible bool   `json:"contrast_eligible"`
	} `json:"primary_return_metadata"`
	PrimaryLiteralMetadata *struct {
		ID               string `json:"id"`
		Category         string `json:"category"`
		Stratum          string `json:"stratum"`
		ContextClass     string `json:"context_class"`
		ContrastEligible bool   `json:"contrast_eligible"`
	} `json:"primary_literal_metadata"`
}

type Assignment struct {
	Version        string      `json:"version"`
	Kind           string      `json:"kind"`
	Demo           bool        `json:"demo"`
	Components     []Component `json:"components"`
	FixturesSHA256 string      `json:"fixtures_sha256"`
	SeenSHA256     string      `json:"seen_sha256"`
}

type Lock struct {
	Version         string            `json:"version"`
	Kind            string            `json:"kind"`
	Demo            bool              `json:"demo"`
	BaselineCommit  string            `json:"baseline_commit"`
	SourceCommit    string            `json:"source_commit"`
	DirtyDiffSHA256 string            `json:"dirty_diff_sha256"`
	Files           map[string]string `json:"files"`
	Commands        []string          `json:"commands"`
	Versions        struct {
		Go      string  `json:"go"`
		Python  string  `json:"python"`
		Numpy   string  `json:"numpy"`
		Threads float64 `json:"threads"`
	} `json:"versions"`
	MaxRawBytes      float64 `json:"max_raw_bytes"`
	MaxKeptChinese   float64 `json:"max_kept_chinese"`
	MaxLexQueries    float64 `json:"max_lex_queries"`
	MaxDecoderCalls  float64 `json:"max_decoder_calls"`
	MaxPairs         float64 `json:"max_pairs"`
	MaxArtifactBytes float64 `json:"max_artifact_bytes"`
}

type Row struct {
	Version                     string `json:"version"`
	Kind                        string `json:"kind"`
	Demo                        bool   `json:"demo"`
	FixtureID                   string `json:"fixture_id"`
	ComponentID                 string `json:"component_id"`
	Condition                   string `json:"condition"`
	Intent                      string `json:"intent"`
	Primary                     bool   `json:"primary"`
	Support                     string `json:"support"`
	FinalBoundaryError          *bool  `json:"final_boundary_error"`
	FinalSurfaceError           *bool  `json:"final_surface_error"`
	FinalEnglishError           *bool  `json:"final_english_error"`
	GoldSurvived                *bool  `json:"gold_survived"`
	FirstCorrect                *int   `json:"first_correct"`
	LastStable                  *int   `json:"last_stable"`
	TemporaryErrorLengths       []int  `json:"temporary_error_lengths"`
	RevisionCount               int    `json:"revision_count"`
	ResolutionCount             int    `json:"resolution_count"`
	KeyFailures                 *int   `json:"key_failures"`
	CommitFailures              *int   `json:"commit_failures"`
	AppRewrites                 *int   `json:"app_rewrites"`
	LockViolations              *int   `json:"lock_violations"`
	HypotheticalLearningUpdates *int   `json:"hypothetical_learning_updates"`
	PersistentWrites            *int   `json:"persistent_writes"`
	ProtectedCalls              *int   `json:"protected_calls"`
	SnapshotsPath               string `json:"snapshots_path"`
	CandidatesPath              string `json:"candidates_path"`
	PrimaryCategory             string `json:"primary_category"`
	ContextClass                string `json:"context_class"`
	ContrastEligible            bool   `json:"contrast_eligible"`
	SafetyStatus                string `json:"safety_status"`
	SafetyReason                string `json:"safety_reason"`
	PrimaryMetadata             *struct {
		ID               string `json:"id"`
		Category         string `json:"category"`
		Stratum          string `json:"stratum"`
		ContextClass     string `json:"context_class"`
		ContrastEligible bool   `json:"contrast_eligible"`
	} `json:"primary_metadata"`
	GeneratorsPath string `json:"generators_path"`
}

type Metric struct {
	Errors                int       `json:"errors"`
	N                     int       `json:"n"`
	Value                 *float64  `json:"value"`
	Wilson95              []float64 `json:"wilson95"`
	IndependentComponents bool      `json:"independent_components"`
}

type Gate struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type Report struct {
	Version          string `json:"version"`
	Kind             string `json:"kind"`
	Demo             bool   `json:"demo"`
	Outcome          string `json:"outcome"`
	PrimaryCondition string `json:"primary_condition"`
	Gates            struct {
		G0 Gate `json:"G0"`
		G1 Gate `json:"G1"`
		G2 Gate `json:"G2"`
		G3 Gate `json:"G3"`
		G4 Gate `json:"G4"`
		G5 Gate `json:"G5"`
	} `json:"gates"`
	Conditions map[string]map[string]Metric `json:"conditions"`
	Paired     []struct {
		Baseline                   string    `json:"baseline"`
		Challenger                 string    `json:"challenger"`
		Cohort                     string    `json:"cohort"`
		N                          int       `json:"n"`
		Repaired                   int       `json:"repaired"`
		Worsened                   int       `json:"worsened"`
		Net                        int       `json:"net"`
		RelativeErrorReduction     *float64  `json:"relative_error_reduction"`
		ValidBootstrap             int       `json:"valid_bootstrap"`
		AbsoluteErrorReductionCi95 []float64 `json:"absolute_error_reduction_ci95"`
		DiscordantComponents       int       `json:"discordant_components"`
	} `json:"paired"`
	Qualification struct {
		TestReturnComponents  int            `json:"test_return_components"`
		TestLiteralComponents int            `json:"test_literal_components"`
		TestEnglishComponents int            `json:"test_english_components"`
		ContrastComponents    int            `json:"contrast_components"`
		CategoryComponents    map[string]int `json:"category_components"`
		SourceComplete        bool           `json:"source_complete"`
		SurfaceGoldComplete   bool           `json:"surface_gold_complete"`
	} `json:"qualification"`
	SourceLockSHA256 string            `json:"source_lock_sha256"`
	ModelsSHA256     map[string]string `json:"models_sha256"`
	DiagnosticsPath  string            `json:"diagnostics_path"`
	Safety           struct {
		KeyFailures                 *int `json:"key_failures"`
		CommitFailures              *int `json:"commit_failures"`
		AppRewrites                 *int `json:"app_rewrites"`
		LockViolations              *int `json:"lock_violations"`
		HypotheticalLearningUpdates *int `json:"hypothetical_learning_updates"`
		PersistentWrites            *int `json:"persistent_writes"`
		ProtectedCalls              *int `json:"protected_calls"`
	} `json:"safety"`
	Derived      []Derived `json:"derived"`
	SafetyStatus string    `json:"safety_status"`
	SafetyReason string    `json:"safety_reason"`
}

type Sources struct {
	Version string `json:"version"`
	Kind    string `json:"kind"`
	Demo    bool   `json:"demo"`
	Entries []struct {
		ID               string   `json:"id"`
		Kind             string   `json:"kind"`
		Reference        string   `json:"reference"`
		Version          string   `json:"version"`
		SHA256           string   `json:"sha256"`
		License          string   `json:"license"`
		AnnotationPasses float64  `json:"annotation_passes"`
		AdjudicationLog  string   `json:"adjudication_log"`
		AnnotatorIDs     []string `json:"annotator_ids"`
		CachedPath       string   `json:"cached_path"`
	} `json:"entries"`
	SeenPrefixes []string `json:"seen_prefixes"`
	SeenContents []string `json:"seen_contents"`
	SeenRaw      []string `json:"seen_raw"`
}

type Derived struct {
	Name           string    `json:"name"`
	Model          *string   `json:"model"`
	Cohort         string    `json:"cohort"`
	Value          *float64  `json:"value"`
	Bootstrap95    []float64 `json:"bootstrap95"`
	ValidBootstrap int       `json:"valid_bootstrap"`
}

type CandidateOccurrence struct {
	Version         string     `json:"version"`
	Kind            string     `json:"kind"`
	Demo            bool       `json:"demo"`
	FixtureID       string     `json:"fixture_id"`
	ComponentID     string     `json:"component_id"`
	Condition       string     `json:"condition"`
	EventIndex      int        `json:"event_index"`
	KeyOffset       int        `json:"key_offset"`
	Raw             string     `json:"raw"`
	QueryID         string     `json:"query_id"`
	Hypothesis      Hypothesis `json:"candidate"`
	Score           *float64   `json:"score"`
	Allowed         bool       `json:"allowed"`
	ExclusionReason string     `json:"exclusion_reason"`
}

type BankTask struct {
	Version             string   `json:"version"`
	Kind                string   `json:"kind"`
	Demo                bool     `json:"demo"`
	ID                  string   `json:"id"`
	PrototypeID         string   `json:"prototype_id"`
	VariantIndex        int      `json:"variant_index"`
	SourceRef           string   `json:"source_ref"`
	SourcePriority      float64  `json:"source_priority"`
	Prompt              string   `json:"prompt"`
	Intent              string   `json:"intent"`
	Category            string   `json:"category"`
	Stratum             string   `json:"stratum"`
	ContextClass        string   `json:"context_class"`
	Raw                 string   `json:"raw"`
	Gold                []Gold   `json:"gold"`
	Families            []string `json:"families"`
	PreludeEvents       []Event  `json:"prelude_events"`
	ExpectedPrelude     string   `json:"expected_prelude"`
	DesignatedTypoSpans []struct {
		Begin int `json:"begin"`
		End   int `json:"end"`
	} `json:"designated_typo_spans"`
	SourceEdits []struct {
		Begin       int    `json:"begin"`
		End         int    `json:"end"`
		Replacement string `json:"replacement"`
	} `json:"source_edits"`
	Annotations []struct {
		AnnotatorID     string `json:"annotator_id"`
		Intent          string `json:"intent"`
		ReadingVerified bool   `json:"reading_verified"`
		Accepted        bool   `json:"accepted"`
		LabelsSHA256    string `json:"labels_sha256"`
		AdjudicationRef string `json:"adjudication_ref"`
	} `json:"annotations"`
}

type TrainingObservation struct {
	Version     string     `json:"version"`
	Kind        string     `json:"kind"`
	Demo        bool       `json:"demo"`
	FixtureID   string     `json:"fixture_id"`
	ComponentID string     `json:"component_id"`
	EventIndex  int        `json:"event_index"`
	KeyOffset   int        `json:"key_offset"`
	Raw         string     `json:"raw"`
	Hypothesis  Hypothesis `json:"candidate"`
	GoldIDs     []string   `json:"gold_ids"`
}

type Handoff struct {
	Event      Event  `json:"event"`
	Action     string `json:"action"`
	TargetMode string `json:"target_mode"`
	Prelude    string `json:"prelude"`
}

type GeneratorDiagnostic struct {
	Version             string   `json:"version"`
	Kind                string   `json:"kind"`
	Demo                bool     `json:"demo"`
	FixtureID           string   `json:"fixture_id"`
	ComponentID         string   `json:"component_id"`
	Condition           string   `json:"condition"`
	EventIndex          int      `json:"event_index"`
	KeyOffset           int      `json:"key_offset"`
	Raw                 string   `json:"raw"`
	QueryID             string   `json:"query_id"`
	GenerationPerformed bool     `json:"generation_performed"`
	UncappedIDs         []string `json:"uncapped_ids"`
	KeptIDs             []string `json:"kept_ids"`
	UncappedCount       *int     `json:"uncapped_count"`
	KeptCount           int      `json:"kept_count"`
	LiteralID           string   `json:"literal_id"`
	CapPruned           *bool    `json:"cap_pruned"`
	RawOverflow         bool     `json:"raw_overflow"`
	Status              string   `json:"status"`
	Reason              string   `json:"reason"`
	LimitHit            string   `json:"limit_hit"`
	GenerateCalls       *int     `json:"generate_calls"`
	ProbeLexQueries     *int     `json:"probe_lex_queries"`
	BestCalls           *int     `json:"best_calls"`
}

type Event struct {
	Type        string  `json:"type"`
	Key         string  `json:"key,omitempty"`
	To          *int    `json:"to,omitempty"`
	Text        *string `json:"text,omitempty"`
	On          *bool   `json:"on,omitempty"`
	CandidateID string  `json:"candidate_id,omitempty"`
}
type Vector = [13]float64

const Version = "shift-return-v2"
