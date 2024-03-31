//go build update_registry
//go:build update_registry

package cmd

import (
	"context"
	"fmt"
	"log"

	"actionforge/graph-runner/core"
	"actionforge/graph-runner/utils"
	u "actionforge/graph-runner/utils"

	// initialize all nodes
	_ "actionforge/graph-runner/nodes"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var cmdUpdate = &cobra.Command{
	Use: "update",
	Run: func(cmd *cobra.Command, args []string) {
		err := UpdateRegistry()
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	utils.LoadEnvOnce()

	cmdRoot.AddCommand(cmdUpdate)
}

func UpdateRegistry() error {

	mongoDbUrl := u.GetVariable("mongodb_url", "MongoDB URL", u.GetVariableOpts{
		Env: true,
	})

	mongoDbAuthSource := u.GetVariable("mongodb_auth_source", "MongoDB Auth Source", u.GetVariableOpts{
		Env: true,
	})

	mongoDbUsername := u.GetVariable("mongodb_username", "MongoDB Username", u.GetVariableOpts{
		Env: true,
	})

	mongoDbPassword := u.GetVariable("mongodb_password", "MongoDB Password", u.GetVariableOpts{
		Env: true,
	})

	m, err := CreateMongoDbClientWithCredentials(Root, mongoDbUrl, mongoDbUsername, mongoDbPassword, mongoDbAuthSource)
	if err != nil {
		panic(err)
	}
	defer m.Close()

	registryDb := m.Client.Database(MONGODB_DATABASE_NODES)
	registryBuiltinsCol := registryDb.Collection(MONGODB_COLLECTION_BUILTINS)

	opts := options.Update().SetUpsert(true)

	fmt.Println("Updating registry...")

	for nodeId, nodeDef := range core.GetRegistries() {
		filter := bson.M{"_id": nodeId}

		fmt.Printf("  %s@v%v\n", nodeDef.Id, nodeDef.Version)
		// On insert or update, MongoDB combines the `filter` and `update` instructions. Due to
		// the `bson:"_id"` tag in NodeDef, MongoDB would receive two `_id`s, resulting in a failure.
		// I could nullify the `_id` in the node definition and add an `omitempty` tag to its field,
		// but to keep the sourec code changes local, I simply remove `_id` as it is already provided
		// by the `filter` from above.
		byteData, _ := bson.Marshal(nodeDef)
		var nodeDefMap bson.M
		err = bson.Unmarshal(byteData, &nodeDefMap)
		if err != nil {
			return err
		}
		delete(nodeDefMap, "_id") // already set by filter, see comment above

		update := bson.M{
			"$set": nodeDefMap,
		}
		_, err = registryBuiltinsCol.UpdateOne(context.Background(), filter, update, opts)
		if err != nil {
			return err
		}
	}
	return nil
}
