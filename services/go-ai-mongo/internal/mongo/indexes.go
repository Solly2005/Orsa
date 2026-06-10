package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func EnsureTTLIndex(ctx context.Context, collection *mongo.Collection, field string, ttl time.Duration) error {
	_, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: field, Value: 1}},
		Options: options.Index().
			SetName(field + "_ttl").
			SetExpireAfterSeconds(int32(ttl.Seconds())),
	})
	return err
}

func EnsureThreadIndexes(ctx context.Context, collection *mongo.Collection) error {
	_, err := collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "updated_at", Value: -1}},
			Options: options.Index().SetName("user_threads_updated"),
		},
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "title", Value: "text"}},
			Options: options.Index().SetName("user_threads_search"),
		},
	})
	return err
}
