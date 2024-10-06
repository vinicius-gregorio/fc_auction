package auction

import (
	"context"
	"fmt"
	"fullcycle-auction_go/configuration/logger"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/internal_error"
	"log"
	"os"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type AuctionEntityMongo struct {
	Id          string                          `bson:"_id"`
	ProductName string                          `bson:"product_name"`
	Category    string                          `bson:"category"`
	Description string                          `bson:"description"`
	Condition   auction_entity.ProductCondition `bson:"condition"`
	Status      auction_entity.AuctionStatus    `bson:"status"`
	Timestamp   int64                           `bson:"timestamp"`
}
type AuctionRepository struct {
	Collection          *mongo.Collection
	auctionEndTimeMap   map[string]time.Time
	auctionEndTimeMutex *sync.Mutex
}

func NewAuctionRepository(database *mongo.Database) *AuctionRepository {
	return &AuctionRepository{
		Collection:          database.Collection("auctions"),
		auctionEndTimeMutex: &sync.Mutex{},
		auctionEndTimeMap:   make(map[string]time.Time),
	}
}

func (ar *AuctionRepository) CreateAuction(
	ctx context.Context,
	auctionEntity *auction_entity.Auction) *internal_error.InternalError {

	log.Println("Starting auction creation process...")

	go func(ctx context.Context) {
		log.Println("starting go routine")

		ar.auctionEndTimeMutex.Lock()
		auctionEndTime := auctionEntity.Timestamp.Add(getAuctionDuration())
		ar.auctionEndTimeMap[auctionEntity.Id] = auctionEndTime
		ar.auctionEndTimeMutex.Unlock()

		log.Printf("Auction will end at: %v\n", auctionEndTime)

		time.Sleep(getAuctionDuration())

		log.Println("Auction time is over. Closing auction...")

		ar.auctionEndTimeMutex.Lock()
		delete(ar.auctionEndTimeMap, auctionEntity.Id)
		ar.auctionEndTimeMutex.Unlock()

		log.Println("auction status is - BEFORE:", auctionEntity.Status)

		auctionEntity.Status = auction_entity.Completed
		log.Println("auction status is - AFTER:", auctionEntity.Status)

		filter := bson.M{"_id": auctionEntity.Id}
		update := bson.M{"$set": bson.M{"status": auction_entity.Completed}}

		log.Println("filter is:", filter)
		log.Println("update is:", update)

		updateCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		updated, err := ar.Collection.UpdateOne(updateCtx, filter, update)
		if err != nil {
			log.Println("Error updating auction status:", err)
			return
		} else if updated.ModifiedCount == 0 {
			log.Println("No document found with the given ID for update")
		} else {
			log.Println("Auction status updated successfully")
		}

		log.Printf("Modified count: %d\n", updated.ModifiedCount)

	}(ctx)

	auctionEntityMongo := &AuctionEntityMongo{
		Id:          auctionEntity.Id,
		ProductName: auctionEntity.ProductName,
		Category:    auctionEntity.Category,
		Description: auctionEntity.Description,
		Condition:   auctionEntity.Condition,
		Status:      auctionEntity.Status,
		Timestamp:   auctionEntity.Timestamp.Unix(),
	}

	_, err := ar.Collection.InsertOne(ctx, auctionEntityMongo)
	if err != nil {
		logger.Error("Error trying to insert auction", err)
		return internal_error.NewInternalServerError("Error trying to insert auction")
	}

	log.Println("Auction successfully created and inserted into MongoDB.")

	return nil
}

func getAuctionDuration() time.Duration {
	auctionInterval := os.Getenv("AUCTION_DURATION")
	duration, err := time.ParseDuration(auctionInterval)
	if err != nil {
		return time.Second * 30
	}
	fmt.Println("duration is:", duration)
	return duration
}
