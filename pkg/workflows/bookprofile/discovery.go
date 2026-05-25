package bookprofile

import (
	"os"

	"gopkg.in/yaml.v3"
)

type DiscoveryState struct {
	BookID               string                 `yaml:"book_id" json:"book_id"`
	SourceProfile        string                 `yaml:"source_profile,omitempty" json:"source_profile,omitempty"`
	RunID                string                 `yaml:"run_id,omitempty" json:"run_id,omitempty"`
	Updated              string                 `yaml:"updated,omitempty" json:"updated,omitempty"`
	ObservedPages        []ObservedPage         `yaml:"observed_pages,omitempty" json:"observed_pages,omitempty"`
	Figures              []DiscoveredFigure     `yaml:"figures,omitempty" json:"figures,omitempty"`
	VocabularyCandidates []VocabularyCandidate  `yaml:"vocabulary_candidates,omitempty" json:"vocabulary_candidates,omitempty"`
	QAFindingSummary     []DiscoveryQAFinding   `yaml:"qa_findings,omitempty" json:"qa_findings,omitempty"`
	Metadata             map[string]interface{} `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

type ObservedPage struct {
	Page         int      `yaml:"page" json:"page"`
	InferredType PageType `yaml:"inferred_type" json:"inferred_type"`
	Confidence   float64  `yaml:"confidence,omitempty" json:"confidence,omitempty"`
	Evidence     []string `yaml:"evidence,omitempty" json:"evidence,omitempty"`
}

type DiscoveredFigure struct {
	Page         int      `yaml:"page" json:"page"`
	Caption      string   `yaml:"caption,omitempty" json:"caption,omitempty"`
	Description  string   `yaml:"description,omitempty" json:"description,omitempty"`
	MarkerSource string   `yaml:"marker_source,omitempty" json:"marker_source,omitempty"`
	ImagePath    string   `yaml:"image_path,omitempty" json:"image_path,omitempty"`
	Warnings     []string `yaml:"warnings,omitempty" json:"warnings,omitempty"`
}

type VocabularyCandidate struct {
	Term   string `yaml:"term" json:"term"`
	Reason string `yaml:"reason,omitempty" json:"reason,omitempty"`
	Pages  []int  `yaml:"pages,omitempty" json:"pages,omitempty"`
}

type DiscoveryQAFinding struct {
	Code    string `yaml:"code" json:"code"`
	Page    int    `yaml:"page,omitempty" json:"page,omitempty"`
	Message string `yaml:"message,omitempty" json:"message,omitempty"`
}

type ProfilePatch struct {
	SourceProfile   string         `yaml:"source_profile,omitempty" json:"source_profile,omitempty"`
	SourceDiscovery string         `yaml:"source_discovery,omitempty" json:"source_discovery,omitempty"`
	Proposals       PatchProposals `yaml:"proposals,omitempty" json:"proposals,omitempty"`
	Reasons         []string       `yaml:"reasons,omitempty" json:"reasons,omitempty"`
}

type PatchProposals struct {
	PageTypes  PageTypePatch   `yaml:"page_types,omitempty" json:"page_types,omitempty"`
	QA         QAPatch         `yaml:"qa,omitempty" json:"qa,omitempty"`
	Vocabulary VocabularyPatch `yaml:"vocabulary,omitempty" json:"vocabulary,omitempty"`
}

type PageTypePatch struct {
	Add map[int]PageType `yaml:"add,omitempty" json:"add,omitempty"`
}

type QAPatch struct {
	ExpectedFigureCount int `yaml:"expected_figure_count,omitempty" json:"expected_figure_count,omitempty"`
}

type VocabularyPatch struct {
	ProtectedTermsAdd []string `yaml:"protected_terms_add,omitempty" json:"protected_terms_add,omitempty"`
}

func LoadDiscovery(path string) (DiscoveryState, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return DiscoveryState{}, err
	}
	var state DiscoveryState
	if err := yaml.Unmarshal(body, &state); err != nil {
		return DiscoveryState{}, err
	}
	return state, nil
}

func SaveDiscovery(path string, state DiscoveryState) error {
	body, err := yaml.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o644)
}

func SavePatch(path string, patch ProfilePatch) error {
	body, err := yaml.Marshal(patch)
	if err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o644)
}

func BuildPatch(profile Profile, state DiscoveryState) ProfilePatch {
	patch := ProfilePatch{SourceProfile: profile.ID, Proposals: PatchProposals{PageTypes: PageTypePatch{Add: map[int]PageType{}}}}
	known := profile.PageTypes.KnownPages
	for _, page := range state.ObservedPages {
		if page.InferredType == "" {
			continue
		}
		if known == nil || known[page.Page] != page.InferredType {
			patch.Proposals.PageTypes.Add[page.Page] = page.InferredType
		}
	}
	if len(state.Figures) > 0 && profile.QA.ExpectedFigureCount != len(state.Figures) {
		patch.Proposals.QA.ExpectedFigureCount = len(state.Figures)
	}
	if len(patch.Proposals.PageTypes.Add) == 0 {
		patch.Proposals.PageTypes.Add = nil
	}
	if len(state.Figures) > 0 {
		patch.Reasons = append(patch.Reasons, "Figure extraction discovered embedded figures that may be worth promoting into stable page-type and QA policy.")
	}
	return patch
}
