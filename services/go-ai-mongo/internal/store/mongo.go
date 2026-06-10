package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	orsamongo "orsa.ai/go-ai-mongo/internal/mongo"
)

// MongoStore persists threads in a MongoDB Atlas collection.
type MongoStore struct {
	client     *mongo.Client
	collection *mongo.Collection
}

// NewMongoStore connects to Atlas, pings to verify reachability, and ensures the
// thread indexes exist. Callers fall back to the in-memory store on error.
func NewMongoStore(ctx context.Context, uri, database string) (*MongoStore, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	pingCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, err
	}
	collection := client.Database(database).Collection("threads")
	_ = orsamongo.EnsureThreadIndexes(ctx, collection)
	return &MongoStore{client: client, collection: collection}, nil
}

func (m *MongoStore) ListConversations(ctx context.Context, userID string) ([]ConversationSummary, error) {
	cursor, err := m.collection.Find(ctx,
		bson.M{"user_id": userID, "deleted": bson.M{"$ne": true}},
		options.Find().SetSort(bson.D{{Key: "updated_at", Value: -1}}).SetProjection(bson.M{"state": 0}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var threads []Thread
	if err := cursor.All(ctx, &threads); err != nil {
		return nil, err
	}
	out := make([]ConversationSummary, 0, len(threads))
	for _, t := range threads {
		out = append(out, ConversationSummary{ID: t.ThreadID, Title: t.Title, UpdatedAt: t.UpdatedAt, Deleted: t.Deleted})
	}
	return out, nil
}

func (m *MongoStore) GetThread(ctx context.Context, userID, threadID string) (*Thread, error) {
	var thread Thread
	err := m.collection.FindOne(ctx, bson.M{"user_id": userID, "thread_id": threadID}).Decode(&thread)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &thread, nil
}

func (m *MongoStore) SaveThread(ctx context.Context, thread *Thread) error {
	_, err := m.collection.ReplaceOne(ctx,
		bson.M{"user_id": thread.UserID, "thread_id": thread.ThreadID},
		thread,
		options.Replace().SetUpsert(true))
	return err
}

func (m *MongoStore) SetDeleted(ctx context.Context, userID, threadID string, deleted bool) error {
	_, err := m.collection.UpdateOne(ctx,
		bson.M{"user_id": userID, "thread_id": threadID},
		bson.M{"$set": bson.M{"deleted": deleted, "updated_at": time.Now().UTC()}})
	return err
}

func (m *MongoStore) ChangedThreads(ctx context.Context, userID string, since time.Time) ([]Thread, error) {
	cursor, err := m.collection.Find(ctx, bson.M{"user_id": userID, "updated_at": bson.M{"$gt": since}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var threads []Thread
	if err := cursor.All(ctx, &threads); err != nil {
		return nil, err
	}
	return threads, nil
}

func (m *MongoStore) Close(ctx context.Context) error {
	return m.client.Disconnect(ctx)
}
