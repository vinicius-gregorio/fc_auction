package auction_test

import (
	"context"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/infra/database/auction"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestAuctionStatusUpdate(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	testAuction := &auction_entity.Auction{
		Id:          "test-auction-id",
		ProductName: "Test Product",
		Category:    "Test Category",
		Description: "Test Description",
		Condition:   auction_entity.New,
		Status:      auction_entity.Active,
		Timestamp:   time.Now(),
	}

	mt.Run("Auction Status Update", func(mt *mtest.T) {
		mt.AddMockResponses(mtest.CreateSuccessResponse())

		updateResponse := mtest.CreateSuccessResponse(
			primitive.E{Key: "nModified", Value: 1},
		)
		mt.AddMockResponses(updateResponse)

		repo := auction.NewAuctionRepository(mt.DB)
		ctx := context.Background()

		os.Setenv("AUCTION_DURATION", "2s")

		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			defer wg.Done()
			_ = repo.CreateAuction(ctx, testAuction)
		}()

		wg.Wait()

		time.Sleep(3 * time.Second)

		assert.Equal(t, auction_entity.Completed, testAuction.Status)
	})
}
