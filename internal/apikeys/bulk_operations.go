package apikeys

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// GetKeysByTag retrieves all API keys with a specific tag value
// Example: GetKeysByTag(ctx, "department", "A") returns all keys with tags.department = "A"
func (s *Store) GetKeysByTag(ctx context.Context, tagKey, tagValue string) ([]*APIKey, error) {
	if tagKey == "" || tagValue == "" {
		return nil, fmt.Errorf("tagKey and tagValue are required")
	}

	// Scan entire table (no filter expression due to DynamoDB attribute limitations)
	result, err := s.client.Scan(ctx, &dynamodb.ScanInput{
		TableName: &s.tableName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan table: %w", err)
	}

	// Unmarshal results into APIKey structs
	var allKeys []*APIKey
	err = attributevalue.UnmarshalListOfMaps(result.Items, &allKeys)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal results: %w", err)
	}

	// Filter in application (simple approach)
	var filtered []*APIKey
	for _, key := range allKeys {
		if key.Tags != nil {
			if val, ok := key.Tags[tagKey]; ok && val == tagValue {
				filtered = append(filtered, key)
			}
		}
	}

	return filtered, nil
}

// UpdateKeysByTag updates all keys with a specific tag to use a new actual key
// Example: UpdateKeysByTag(ctx, "department", "A", "fw_new_key") updates all keys in dept A
func (s *Store) UpdateKeysByTag(ctx context.Context, tagKey, tagValue, newActualKey string) (int, error) {
	if tagKey == "" || tagValue == "" {
		return 0, fmt.Errorf("tagKey and tagValue are required")
	}

	// First, get all keys with this tag
	keys, err := s.GetKeysByTag(ctx, tagKey, tagValue)
	if err != nil {
		return 0, err
	}

	if len(keys) == 0 {
		s.logger.Warn("No keys found with tag", "tag_key", tagKey, "tag_value", tagValue)
		return 0, nil
	}

	// Update each key
	updated := 0
	for _, key := range keys {
		_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
			TableName: &s.tableName,
			Key: map[string]types.AttributeValue{
				"pk": &types.AttributeValueMemberS{Value: key.PK},
			},
			UpdateExpression: &[]string{"SET actual_key = :key, updated_at = :now"}[0],
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":key": &types.AttributeValueMemberS{Value: newActualKey},
				":now": &types.AttributeValueMemberS{Value: key.UpdatedAt.String()},
			},
		})
		if err != nil {
			s.logger.Error("Failed to update key", "pk", key.PK, "error", err)
			continue
		}
		updated++
	}

	s.logger.Info("Updated keys by tag", "tag_key", tagKey, "tag_value", tagValue, "count", updated)
	return updated, nil
}

// DeleteKeysByTag deletes all keys with a specific tag
func (s *Store) DeleteKeysByTag(ctx context.Context, tagKey, tagValue string) (int, error) {
	if tagKey == "" || tagValue == "" {
		return 0, fmt.Errorf("tagKey and tagValue are required")
	}

	// Get all keys with this tag
	keys, err := s.GetKeysByTag(ctx, tagKey, tagValue)
	if err != nil {
		return 0, err
	}

	deleted := 0
	for _, key := range keys {
		_, err := s.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
			TableName: &s.tableName,
			Key: map[string]types.AttributeValue{
				"pk": &types.AttributeValueMemberS{Value: key.PK},
			},
		})
		if err != nil {
			s.logger.Error("Failed to delete key", "pk", key.PK, "error", err)
			continue
		}
		deleted++
	}

	s.logger.Info("Deleted keys by tag", "tag_key", tagKey, "tag_value", tagValue, "count", deleted)
	return deleted, nil
}
