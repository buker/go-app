package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	// Timeout operations after N seconds
	connectTimeout           = 5
	connectionStringTemplate = "mongodb://%s:%s@%s"
)

// GetConnection Retrieves a client to the MongoDB
func getConnection() (*mongo.Client, context.Context, context.CancelFunc) {
	username := os.Getenv("MONGODB_USERNAME")
	password := os.Getenv("MONGODB_PASSWORD")
	clusterEndpoint := os.Getenv("MONGODB_ENDPOINT")

	connectionURI := fmt.Sprintf(connectionStringTemplate, username, password, clusterEndpoint)

	client, err := mongo.NewClient(options.Client().ApplyURI(connectionURI))
	if err != nil {
		log.Error("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout*time.Second)

	err = client.Connect(ctx)
	if err != nil {
		log.Error("Failed to connect to cluster: %v", err)
	}

	// Force a connection to verify our connection string
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Error("Failed to ping cluster: %v", err)
	}

	log.Info("Connected to MongoDB!")
	return client, ctx, cancel
}

// GetAllRecords Retrives all records from the db
func GetAllRecords() ([]*Record, error) {
	var records []*Record

	client, ctx, cancel := getConnection()
	defer cancel()
	defer client.Disconnect(ctx)
	db := client.Database("records")
	collection := db.Collection("records")
	cursor, err := collection.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	err = cursor.All(ctx, &records)
	if err != nil {
		log.Error("Failed marshalling %v", err)
		return nil, err
	}
	return records, nil
}

// GetRecordByID Retrives a record by its id from the db
func GetRecordByID(id primitive.ObjectID) (*Record, error) {
	var record *Record

	client, ctx, cancel := getConnection()
	defer cancel()
	defer client.Disconnect(ctx)
	db := client.Database("records")
	collection := db.Collection("records")
	result := collection.FindOne(ctx, bson.D{})
	if result == nil {
		return nil, errors.New("Could not find a Record")
	}
	err := result.Decode(&record)

	if err != nil {
		log.Error("Failed marshalling %v", err)
		return nil, err
	}
	log.Info("Records: %v", record)
	return record, nil
}

//Create creating a record in a mongo
func Create(record *Record) (primitive.ObjectID, error) {
	client, ctx, cancel := getConnection()
	defer cancel()
	defer client.Disconnect(ctx)
	record.ID = primitive.NewObjectID()

	result, err := client.Database("records").Collection("records").InsertOne(ctx, record)
	if err != nil {
		log.Error("Could not create Record: %v", err)
		return primitive.NilObjectID, err
	}
	oid := result.InsertedID.(primitive.ObjectID)
	return oid, nil
}

//Update updating an existing record in a mongo
func Update(record *Record) (*Record, error) {
	var updatedRecord *Record
	client, ctx, cancel := getConnection()
	defer cancel()
	defer client.Disconnect(ctx)

	update := bson.M{
		"$set": record,
	}

	upsert := true
	after := options.After
	opt := options.FindOneAndUpdateOptions{
		Upsert:         &upsert,
		ReturnDocument: &after,
	}

	err := client.Database("records").Collection("records").FindOneAndUpdate(ctx, bson.M{"_id": record.ID}, update, &opt).Decode(&updatedRecord)
	if err != nil {
		log.Error("Could not save Record: %v", err)
		return nil, err
	}
	return updatedRecord, nil
}
