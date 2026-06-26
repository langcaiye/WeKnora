package types

import (
	"fmt"
	"regexp"
)

const (
	// MetadataFieldPrefix keeps business metadata isolated from built-in
	// retriever fields such as knowledge_id and chunk_id.
	MetadataFieldPrefix = "metadata."
	// MetadataPayloadFieldPrefix is used by vector stores that store metadata
	// as flat payload fields rather than nested JSON objects.
	MetadataPayloadFieldPrefix = "metadata__"
)

var metadataFilterFieldPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,63}$`)

// MetadataFilterOp is the portable operator set accepted by the retrieval API.
type MetadataFilterOp string

const (
	MetadataFilterOpEq    MetadataFilterOp = "eq"
	MetadataFilterOpIn    MetadataFilterOp = "in"
	MetadataFilterOpNotEq MetadataFilterOp = "not_eq"
	MetadataFilterOpNotIn MetadataFilterOp = "not_in"
)

// MetadataFilter is a typed, portable filter over chunk-level business metadata.
type MetadataFilter struct {
	Field  string           `json:"field"`
	Op     MetadataFilterOp `json:"op"`
	Value  string           `json:"value,omitempty"`
	Values []string         `json:"values,omitempty"`
}

// MetadataFilters groups filters by boolean semantics. The first release keeps
// the DSL intentionally small: all Must clauses are ANDed, all MustNot clauses
// are ANDed as negated conditions.
type MetadataFilters struct {
	Must    []MetadataFilter `json:"must,omitempty"`
	MustNot []MetadataFilter `json:"must_not,omitempty"`
}

// Empty reports whether no metadata filters are present.
func (f MetadataFilters) Empty() bool {
	return len(f.Must) == 0 && len(f.MustNot) == 0
}

// Validate rejects invalid field names and malformed values before any backend
// can translate the filter into its native query language.
func (f MetadataFilters) Validate() error {
	for _, filter := range f.Must {
		if err := filter.Validate(); err != nil {
			return err
		}
	}
	for _, filter := range f.MustNot {
		if err := filter.Validate(); err != nil {
			return err
		}
		if filter.IsNegative() {
			return fmt.Errorf("metadata must_not filter %q cannot use negative operator %s", filter.Field, filter.Op)
		}
	}
	return nil
}

// IncludeFilters returns positive membership filters that should be translated
// to backend must/filter clauses.
func (f MetadataFilters) IncludeFilters() []MetadataFilter {
	out := make([]MetadataFilter, 0, len(f.Must))
	for _, filter := range f.Must {
		if !filter.IsNegative() {
			out = append(out, filter)
		}
	}
	return out
}

// ExcludeFilters returns filters that should be translated to backend must_not
// clauses. A negative operator in Must is equivalent to a positive operator in
// MustNot; MustNot itself rejects negative operators during validation.
func (f MetadataFilters) ExcludeFilters() []MetadataFilter {
	out := make([]MetadataFilter, 0, len(f.Must)+len(f.MustNot))
	for _, filter := range f.Must {
		if filter.IsNegative() {
			out = append(out, filter)
		}
	}
	out = append(out, f.MustNot...)
	return out
}

// Validate ensures a single filter is safe and portable across retrievers.
func (f MetadataFilter) Validate() error {
	if !metadataFilterFieldPattern.MatchString(f.Field) {
		return fmt.Errorf("invalid metadata filter field %q", f.Field)
	}
	switch f.Op {
	case MetadataFilterOpEq, MetadataFilterOpNotEq:
		if f.Value == "" {
			return fmt.Errorf("metadata filter %q requires value", f.Field)
		}
		if len(f.Values) > 0 {
			return fmt.Errorf("metadata filter %q cannot use values with %s", f.Field, f.Op)
		}
	case MetadataFilterOpIn, MetadataFilterOpNotIn:
		if len(f.Values) == 0 {
			return fmt.Errorf("metadata filter %q requires values", f.Field)
		}
		for _, value := range f.Values {
			if value == "" {
				return fmt.Errorf("metadata filter %q contains an empty value", f.Field)
			}
		}
		if f.Value != "" {
			return fmt.Errorf("metadata filter %q cannot use value with %s", f.Field, f.Op)
		}
	default:
		return fmt.Errorf("unsupported metadata filter operator %q", f.Op)
	}
	return nil
}

// FieldPath returns the namespaced backend field name for this metadata key.
func (f MetadataFilter) FieldPath() string {
	return MetadataFieldPrefix + f.Field
}

// IsNegative reports whether this filter represents a negated membership test.
func (f MetadataFilter) IsNegative() bool {
	return f.Op == MetadataFilterOpNotEq || f.Op == MetadataFilterOpNotIn
}

// MatchValues returns the values used by eq/in style filters.
func (f MetadataFilter) MatchValues() []string {
	if f.Op == MetadataFilterOpEq || f.Op == MetadataFilterOpNotEq {
		return []string{f.Value}
	}
	out := make([]string, len(f.Values))
	copy(out, f.Values)
	return out
}

// MetadataFieldName returns the backend field name for a metadata key.
func MetadataFieldName(field string) string {
	return MetadataFieldPrefix + field
}

// MetadataPayloadFieldName returns a flat backend payload field name.
func MetadataPayloadFieldName(field string) string {
	return MetadataPayloadFieldPrefix + field
}

// MergeScalarMetadata copies document-level metadata and lets chunk-level
// metadata override matching keys.
func MergeScalarMetadata(base, override map[string]string) map[string]string {
	if len(base) == 0 && len(override) == 0 {
		return nil
	}
	merged := make(map[string]string, len(base)+len(override))
	for k, v := range base {
		if metadataFilterFieldPattern.MatchString(k) && v != "" {
			merged[k] = v
		}
	}
	for k, v := range override {
		if metadataFilterFieldPattern.MatchString(k) && v != "" {
			merged[k] = v
		}
	}
	if len(merged) == 0 {
		return nil
	}
	return merged
}
