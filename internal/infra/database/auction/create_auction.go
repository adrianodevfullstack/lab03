package auction

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/adrianodevfullstack/lab03/configuration/logger"
	"github.com/adrianodevfullstack/lab03/internal/entity/auction_entity"
	"github.com/adrianodevfullstack/lab03/internal/internal_error"

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
	Collection      *mongo.Collection
	auctionInterval time.Duration
	mu              sync.Mutex
}

func NewAuctionRepository(database *mongo.Database) *AuctionRepository {
	repo := &AuctionRepository{
		Collection:      database.Collection("auctions"),
		auctionInterval: getAuctionDuration(),
	}

	repo.startAutoCloseRoutine(context.Background())

	return repo
}

func (ar *AuctionRepository) CreateAuction(
	ctx context.Context,
	auctionEntity *auction_entity.Auction) *internal_error.InternalError {
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

	return nil
}

func getAuctionDuration() time.Duration {
	auctionInterval := os.Getenv("AUCTION_INTERVAL")
	duration, err := time.ParseDuration(auctionInterval)
	if err != nil {
		logger.Error("Error parsing AUCTION_INTERVAL, using default 5 minutes", err)
		return 5 * time.Minute
	}
	return duration
}

func (ar *AuctionRepository) startAutoCloseRoutine(ctx context.Context) {
	go func() {
		checkInterval := ar.auctionInterval / 2
		if checkInterval < 10*time.Second {
			checkInterval = 10 * time.Second
		}

		ticker := time.NewTicker(checkInterval)
		defer ticker.Stop()

		logger.Info("Auto-close auction routine started")

		for {
			select {
			case <-ctx.Done():
				logger.Info("Auto-close auction routine stopped")
				return
			case <-ticker.C:
				ar.closeExpiredAuctions(context.Background())
			}
		}
	}()
}

func (ar *AuctionRepository) closeExpiredAuctions(ctx context.Context) {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	expirationTime := time.Now().Add(-ar.auctionInterval).Unix()

	filter := bson.M{
		"status":    auction_entity.Active,
		"timestamp": bson.M{"$lte": expirationTime},
	}

	update := bson.M{
		"$set": bson.M{
			"status": auction_entity.Completed,
		},
	}

	result, err := ar.Collection.UpdateMany(ctx, filter, update)
	if err != nil {
		logger.Error("Error trying to close expired auctions", err)
		return
	}

	if result.ModifiedCount > 0 {
		logger.Info("Closed expired auctions")
	}
}
