package tencentvectordb

import (
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/tencent/vectordatabase-sdk-go/tcvdbtext/encoder"
	"github.com/tencent/vectordatabase-sdk-go/tcvectordb"
)

func TestSupportIncludesKeywordAndVectorRetrieval(t *testing.T) {
	repo := NewTencentVectorDBRetrieveEngineRepository(nil, "", nil)

	supports := repo.Support()

	assert.Contains(t, supports, types.KeywordsRetrieverType)
	assert.Contains(t, supports, types.VectorRetrieverType)
}

func TestToDocumentIncludesSparseVector(t *testing.T) {
	embedding := &vectorEmbedding{
		ID:              "chunk-1",
		Content:         "腾讯云向量数据库支持关键词检索",
		SourceID:        "source-1",
		SourceType:      int(types.ChunkSourceType),
		ChunkID:         "chunk-1",
		KnowledgeID:     "knowledge-1",
		KnowledgeBaseID: "kb-1",
		TagID:           "tag-1",
		Embedding:       []float32{0.1, 0.2},
		SparseVector: []encoder.SparseVecItem{
			{TermId: 10, Score: 0.3},
			{TermId: 20, Score: 0.7},
		},
		IsEnabled: true,
	}

	doc := toDocument(embedding)

	assert.Equal(t, embedding.ID, doc.Id)
	assert.Equal(t, embedding.Embedding, doc.Vector)
	assert.Equal(t, embedding.SparseVector, doc.SparseVector)
	assert.Equal(t, "腾讯云向量数据库支持关键词检索", doc.Fields[fieldContent].String())
	assert.Equal(t, uint64(1), doc.Fields[fieldIsEnabled].Uint64())
}

func TestToVectorEmbeddingUsesStableIDPriority(t *testing.T) {
	tests := []struct {
		name string
		info *types.IndexInfo
		want string
	}{
		{
			name: "explicit id wins",
			info: &types.IndexInfo{
				ID:       "index-id",
				SourceID: "source-id",
				ChunkID:  "chunk-id",
			},
			want: "index-id",
		},
		{
			name: "source id differentiates generated questions",
			info: &types.IndexInfo{
				SourceID: "chunk-id-question-id",
				ChunkID:  "chunk-id",
			},
			want: "chunk-id-question-id",
		},
		{
			name: "chunk id remains the final fallback",
			info: &types.IndexInfo{
				ChunkID: "chunk-id",
			},
			want: "chunk-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			embedding := toVectorEmbedding(tt.info, nil)
			assert.Equal(t, tt.want, embedding.ID)
		})
	}
}

func TestRemapCopiedEmbeddingsPreservesQuestionSourceIDs(t *testing.T) {
	docs := []tcvectordb.Document{
		toDocument(&vectorEmbedding{
			ID:              "source-chunk",
			SourceID:        "source-chunk",
			ChunkID:         "source-chunk",
			KnowledgeID:     "source-knowledge",
			KnowledgeBaseID: "source-kb",
		}),
		toDocument(&vectorEmbedding{
			ID:              "source-chunk-question-1",
			SourceID:        "source-chunk-question-1",
			ChunkID:         "source-chunk",
			KnowledgeID:     "source-knowledge",
			KnowledgeBaseID: "source-kb",
		}),
		toDocument(&vectorEmbedding{
			ID:              "other-source-id",
			SourceID:        "other-source-id",
			ChunkID:         "source-chunk",
			KnowledgeID:     "source-knowledge",
			KnowledgeBaseID: "source-kb",
		}),
	}

	embeddings := remapCopiedEmbeddings(
		docs,
		map[string]string{"source-knowledge": "target-knowledge"},
		map[string]string{"source-chunk": "target-chunk"},
		"target-kb",
	)

	assert.Len(t, embeddings, 3)
	assert.Equal(t, "target-chunk", embeddings[0].ID)
	assert.Equal(t, "target-chunk", embeddings[0].SourceID)
	assert.Equal(t, "target-chunk", embeddings[0].ChunkID)
	assert.Equal(t, "target-knowledge", embeddings[0].KnowledgeID)
	assert.Equal(t, "target-kb", embeddings[0].KnowledgeBaseID)

	assert.Equal(t, "target-chunk-question-1", embeddings[1].ID)
	assert.Equal(t, "target-chunk-question-1", embeddings[1].SourceID)
	assert.Equal(t, "target-chunk", embeddings[1].ChunkID)

	assert.NotEmpty(t, embeddings[2].ID)
	assert.NotEqual(t, "other-source-id", embeddings[2].ID)
	assert.Equal(t, embeddings[2].ID, embeddings[2].SourceID)
	assert.Equal(t, "target-chunk", embeddings[2].ChunkID)
}

func TestCopySourceQueryParamsScopesByKnowledgeBase(t *testing.T) {
	chunkIDs := []string{"chunk-1", "chunk-2"}
	params := copySourceQueryParams(
		"kb-1",
		chunkIDs,
		int64(len(chunkIDs)*copyIndicesMaxEntriesPerChunk),
	)

	assert.Equal(t, int64(22), params.Limit)
	assert.Equal(t, int64(0), params.Offset)
	cond := params.Filter.Cond()
	assert.Contains(t, cond, `knowledge_base_id in ("kb-1")`)
	assert.Contains(t, cond, `chunk_id in ("chunk-1","chunk-2")`)
}

func TestBaseFilterBuildsTencentVectorDBCondition(t *testing.T) {
	repo := &repository{}

	filter := repo.baseFilter(types.RetrieveParams{
		KnowledgeBaseIDs:    []string{"kb-1"},
		KnowledgeIDs:        []string{"knowledge-1", "knowledge-2"},
		TagIDs:              []string{"tag-1"},
		ExcludeKnowledgeIDs: []string{"knowledge-9"},
		ExcludeChunkIDs:     []string{"chunk-9"},
	})
	cond := filter.Cond()

	for _, want := range []string{
		"is_enabled=1",
		"knowledge_base_id in (\"kb-1\")",
		"knowledge_id in (\"knowledge-1\",\"knowledge-2\")",
		"tag_id in (\"tag-1\")",
		"knowledge_id not in (\"knowledge-9\")",
		"chunk_id not in (\"chunk-9\")",
	} {
		assert.True(t, strings.Contains(cond, want), "condition %q should contain %q", cond, want)
	}
}
