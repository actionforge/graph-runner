//go build update_registry
//go:build update_registry

package cmd

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDBClient struct {
	Client *mongo.Client
}

var MongoDbUrl string        // flag, env
var MongoDbAuthSource string // flag, env
var MongoDbUsername string   // flag, env, docker secret
var MongoDbPassword string   // flag, env, docker secret

const MONGODB_DATABASE_NODES = "nodes"
const MONGODB_COLLECTION_BUILTINS = "builtins"

type AuthType int64

const (
	Root AuthType = iota
)

func CreateMongoDbClient(dbType AuthType) (*MongoDBClient, error) {
	return CreateMongoDbClientWithCredentials(dbType, MongoDbUrl, MongoDbUsername, MongoDbPassword, MongoDbAuthSource)
}

func CreateMongoDbClientWithCredentials(dbType AuthType, mongoDbUrl string, mongoDbUsername string, mongoDbPassword string, mongoDbAuthSource string) (*MongoDBClient, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var credentials options.Credential

	switch dbType {
	case Root:
		credentials = options.Credential{
			Username:   mongoDbUsername,
			Password:   mongoDbPassword,
			AuthSource: mongoDbAuthSource,
		}
	default:
		return nil, fmt.Errorf("unknown database type: %v", dbType)
	}

	clientOptions := options.Client().ApplyURI(mongoDbUrl).SetAuth(credentials)

	if clientOptions.Validate() != nil {
		return nil, fmt.Errorf("error validating MongoDB client options: %v", clientOptions.Validate())
	}

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("error creating MongoDB client: %v", err)
	}

	return &MongoDBClient{
		Client: client,
	}, nil
}

func (m *MongoDBClient) Close() error {
	return m.Client.Disconnect(context.Background())
}
