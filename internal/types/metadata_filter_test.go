package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetadataFiltersValidate(t *testing.T) {
	filters := MetadataFilters{
		Must: []MetadataFilter{{
			Field: "goods_id",
			Op:    MetadataFilterOpEq,
			Value: "1001",
		}},
		MustNot: []MetadataFilter{{
			Field:  "sku_id",
			Op:     MetadataFilterOpIn,
			Values: []string{"disabled"},
		}},
	}

	require.NoError(t, filters.Validate())
	assert.Equal(t, "metadata.goods_id", filters.Must[0].FieldPath())
	assert.Equal(t, []string{"1001"}, filters.Must[0].MatchValues())
}

func TestMetadataFiltersRejectInvalidField(t *testing.T) {
	filters := MetadataFilters{
		Must: []MetadataFilter{{
			Field: "metadata.goods_id",
			Op:    MetadataFilterOpEq,
			Value: "1001",
		}},
	}

	require.Error(t, filters.Validate())
}

func TestMetadataFiltersSplitNegativeOperators(t *testing.T) {
	filters := MetadataFilters{
		Must: []MetadataFilter{
			{Field: "goods_id", Op: MetadataFilterOpEq, Value: "1001"},
			{Field: "status", Op: MetadataFilterOpNotEq, Value: "archived"},
		},
		MustNot: []MetadataFilter{
			{Field: "sku_id", Op: MetadataFilterOpIn, Values: []string{"sku-9"}},
		},
	}

	require.NoError(t, filters.Validate())
	require.Len(t, filters.IncludeFilters(), 1)
	require.Len(t, filters.ExcludeFilters(), 2)
	assert.Equal(t, "goods_id", filters.IncludeFilters()[0].Field)
	assert.Equal(t, "status", filters.ExcludeFilters()[0].Field)
}

func TestMetadataFiltersRejectNegativeMustNot(t *testing.T) {
	filters := MetadataFilters{
		MustNot: []MetadataFilter{{
			Field: "goods_id",
			Op:    MetadataFilterOpNotEq,
			Value: "1001",
		}},
	}

	require.Error(t, filters.Validate())
}

func TestMergeScalarMetadataAllowsChunkOverride(t *testing.T) {
	merged := MergeScalarMetadata(
		map[string]string{"goods_id": "1001", "category": "shoe"},
		map[string]string{"goods_id": "1002"},
	)

	assert.Equal(t, map[string]string{
		"goods_id": "1002",
		"category": "shoe",
	}, merged)
}

func TestDocumentChunkMetadataKeepsScalarMetadataWithGeneratedQuestions(t *testing.T) {
	chunk := &Chunk{}
	err := chunk.SetDocumentMetadata(&DocumentChunkMetadata{
		GeneratedQuestions: []GeneratedQuestion{{ID: "q1", Question: "hello?"}},
		ScalarMetadata:     map[string]string{"goods_id": "1001"},
	})
	require.NoError(t, err)

	meta, err := chunk.DocumentMetadata()
	require.NoError(t, err)
	require.Len(t, meta.GeneratedQuestions, 1)
	assert.Equal(t, "1001", meta.ScalarMetadata["goods_id"])
}

func TestChatManageCloneKeepsMetadataFilters(t *testing.T) {
	cm := &ChatManage{
		PipelineRequest: PipelineRequest{
			MetadataFilters: MetadataFilters{
				Must: []MetadataFilter{{
					Field: "goods_id",
					Op:    MetadataFilterOpEq,
					Value: "1001",
				}},
			},
		},
	}

	cloned := cm.Clone()

	require.Len(t, cloned.MetadataFilters.Must, 1)
	assert.Equal(t, "goods_id", cloned.MetadataFilters.Must[0].Field)
}
