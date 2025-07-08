package qdrant

import (
	"context"
	"fmt"

	"github.com/anunay/mentis/internal/core/domain"
	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
)

// Repository uses the official Qdrant Go client (gRPC)
type Repository struct {
	client     *qdrant.Client
	collection string
}

func NewRepository(client *qdrant.Client, collection string) *Repository {
	return &Repository{
		client:     client,
		collection: collection,
	}
}

func (r *Repository) ensureCollection(ctx context.Context) error {
	// Check if collection exists
	collections, err := r.client.ListCollections(ctx)
	if err != nil {
		return fmt.Errorf("failed to list collections: %w", err)
	}

	// Check if our collection exists - collections is a slice of strings
	for _, collectionName := range collections {
		if collectionName == r.collection {
			return nil // Collection already exists
		}
	}

	// Create collection with configurable dimensions (defaulting to 1536 for OpenAI)
	// TODO: This should be configurable based on embedding provider
	err = r.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: r.collection,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     1536,
			Distance: qdrant.Distance_Cosine,
		}),
	})
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	return nil
}

func (r *Repository) Store(ctx context.Context, id uuid.UUID, embedding []float32, metadata map[string]interface{}) error {
	if err := r.ensureCollection(ctx); err != nil {
		return err
	}

	// Create point with UUID, vector, and metadata
	point := &qdrant.PointStruct{
		Id:      qdrant.NewID(id.String()),
		Vectors: qdrant.NewVectors(embedding...),
		Payload: qdrant.NewValueMap(metadata),
	}

	// Upsert the point
	_, err := r.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: r.collection,
		Points:         []*qdrant.PointStruct{point},
	})
	if err != nil {
		return fmt.Errorf("failed to store vector: %w", err)
	}

	return nil
}

func (r *Repository) Search(ctx context.Context, query []float32, topK int, minScore float32, filter map[string]interface{}) ([]domain.LookupResult, error) {
	// Build the query request
	request := &qdrant.QueryPoints{
		CollectionName: r.collection,
		Query:          qdrant.NewQuery(query...),
		Limit:          qdrant.PtrOf(uint64(topK)),
		WithPayload:    qdrant.NewWithPayload(true),
	}

	// Add score threshold if provided
	if minScore > 0 {
		request.ScoreThreshold = qdrant.PtrOf(minScore)
	}

	// Add filter if provided
	if len(filter) > 0 {
		// Convert filter to Qdrant filter format
		conditions := make([]*qdrant.Condition, 0, len(filter))
		for key, value := range filter {
			// Type assert value to string for match condition
			if strValue, ok := value.(string); ok {
				conditions = append(conditions, qdrant.NewMatch(key, strValue))
			}
		}
		if len(conditions) > 0 {
			request.Filter = &qdrant.Filter{
				Must: conditions,
			}
		}
	}

	// Execute the query
	response, err := r.client.Query(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to search vectors: %w", err)
	}

	// Convert results to domain types
	results := make([]domain.LookupResult, 0, len(response))
	for _, result := range response {
		// Parse UUID from ID - handle different point ID types
		var idStr string
		switch pointId := result.Id.PointIdOptions.(type) {
		case *qdrant.PointId_Uuid:
			idStr = pointId.Uuid
		case *qdrant.PointId_Num:
			idStr = fmt.Sprintf("%d", pointId.Num)
		default:
			continue // Skip if ID type is not supported
		}

		id, err := uuid.Parse(idStr)
		if err != nil {
			continue // Skip if ID is not a valid UUID
		}

		// Extract payload as metadata
		metadata := make(map[string]interface{})
		if result.Payload != nil {
			for key, value := range result.Payload {
				metadata[key] = extractValue(value)
			}
		}

		lookupResult := domain.LookupResult{
			Score: result.Score,
			Artifact: &domain.Artifact{
				ID:       id,
				Metadata: metadata,
			},
		}
		results = append(results, lookupResult)
	}

	return results, nil
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	// Delete the point by ID
	_, err := r.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: r.collection,
		Points:         qdrant.NewPointsSelector(qdrant.NewID(id.String())),
	})
	if err != nil {
		return fmt.Errorf("failed to delete vector: %w", err)
	}

	return nil
}

func (r *Repository) Update(ctx context.Context, id uuid.UUID, embedding []float32, metadata map[string]interface{}) error {
	// Update is the same as store in Qdrant (upsert behavior)
	return r.Store(ctx, id, embedding, metadata)
}

// extractValue converts Qdrant Value to Go interface{}
func extractValue(value *qdrant.Value) interface{} {
	if value == nil {
		return nil
	}

	switch v := value.Kind.(type) {
	case *qdrant.Value_NullValue:
		return nil
	case *qdrant.Value_BoolValue:
		return v.BoolValue
	case *qdrant.Value_IntegerValue:
		return v.IntegerValue
	case *qdrant.Value_DoubleValue:
		return v.DoubleValue
	case *qdrant.Value_StringValue:
		return v.StringValue
	case *qdrant.Value_ListValue:
		result := make([]interface{}, len(v.ListValue.Values))
		for i, item := range v.ListValue.Values {
			result[i] = extractValue(item)
		}
		return result
	case *qdrant.Value_StructValue:
		result := make(map[string]interface{})
		for key, item := range v.StructValue.Fields {
			result[key] = extractValue(item)
		}
		return result
	default:
		return nil
	}
}