package auction

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/adrianodevfullstack/lab03/internal/entity/auction_entity"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func setupTestDB(t *testing.T) (*mongo.Database, func()) {
	mongoURL := os.Getenv("MONGODB_URL")
	if mongoURL == "" {
		mongoURL = "mongodb://admin:admin@localhost:27017/auctions_test?authSource=admin"
	}

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURL))
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	db := client.Database("auctions_test")

	cleanup := func() {
		if err := db.Drop(context.Background()); err != nil {
			t.Logf("Warning: failed to drop test database: %v", err)
		}
		if err := client.Disconnect(context.Background()); err != nil {
			t.Logf("Warning: failed to disconnect from MongoDB: %v", err)
		}
	}

	return db, cleanup
}

func TestAutoCloseExpiredAuctions(t *testing.T) {
	os.Setenv("AUCTION_INTERVAL", "3s")
	defer os.Unsetenv("AUCTION_INTERVAL")

	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewAuctionRepository(db)

	expiredAuction := &auction_entity.Auction{
		Id:          "expired-auction-id",
		ProductName: "Produto Teste Expirado",
		Category:    "Categoria Teste",
		Description: "Descrição do produto teste que está expirado",
		Condition:   auction_entity.New,
		Status:      auction_entity.Active,
		Timestamp:   time.Now().Add(-5 * time.Second),
	}

	internalErr := repo.CreateAuction(context.Background(), expiredAuction)
	assert.Nil(t, internalErr)

	activeAuction := &auction_entity.Auction{
		Id:          "active-auction-id",
		ProductName: "Produto Teste Ativo",
		Category:    "Categoria Teste",
		Description: "Descrição do produto teste que está ativo",
		Condition:   auction_entity.Used,
		Status:      auction_entity.Active,
		Timestamp:   time.Now(),
	}

	internalErr = repo.CreateAuction(context.Background(), activeAuction)
	assert.Nil(t, internalErr)

	time.Sleep(12 * time.Second)

	var expiredResult AuctionEntityMongo
	mongoErr := repo.Collection.FindOne(context.Background(), bson.M{"_id": "expired-auction-id"}).Decode(&expiredResult)
	assert.Nil(t, mongoErr)
	assert.Equal(t, auction_entity.Completed, expiredResult.Status, "O leilão expirado deveria estar com status Completed")

	var activeResult AuctionEntityMongo
	mongoErr = repo.Collection.FindOne(context.Background(), bson.M{"_id": "active-auction-id"}).Decode(&activeResult)
	assert.Nil(t, mongoErr)
	assert.Equal(t, auction_entity.Active, activeResult.Status, "O leilão ativo deveria continuar com status Active")
}

func TestAutoCloseMultipleExpiredAuctions(t *testing.T) {
	os.Setenv("AUCTION_INTERVAL", "2s")
	defer os.Unsetenv("AUCTION_INTERVAL")

	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewAuctionRepository(db)

	for i := 0; i < 5; i++ {
		auction := &auction_entity.Auction{
			Id:          "expired-auction-" + string(rune(i+'0')),
			ProductName: "Produto Teste " + string(rune(i+'0')),
			Category:    "Categoria Teste",
			Description: "Descrição do produto teste expirado",
			Condition:   auction_entity.New,
			Status:      auction_entity.Active,
			Timestamp:   time.Now().Add(-4 * time.Second),
		}

		internalErr := repo.CreateAuction(context.Background(), auction)
		assert.Nil(t, internalErr)
	}

	time.Sleep(12 * time.Second)

	cursor, err := repo.Collection.Find(context.Background(), bson.M{"status": auction_entity.Completed})
	assert.Nil(t, err)
	defer cursor.Close(context.Background())

	var closedAuctions []AuctionEntityMongo
	err = cursor.All(context.Background(), &closedAuctions)
	assert.Nil(t, err)

	assert.GreaterOrEqual(t, len(closedAuctions), 5, "Pelo menos 5 leilões deveriam estar fechados")
}
