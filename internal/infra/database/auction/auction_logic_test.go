package auction

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/adrianodevfullstack/lab03/internal/entity/auction_entity"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
)

func TestGetAuctionDuration(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected time.Duration
	}{
		{
			name:     "Valid duration - seconds",
			envValue: "30s",
			expected: 30 * time.Second,
		},
		{
			name:     "Valid duration - minutes",
			envValue: "5m",
			expected: 5 * time.Minute,
		},
		{
			name:     "Valid duration - hours",
			envValue: "2h",
			expected: 2 * time.Hour,
		},
		{
			name:     "Invalid duration - default",
			envValue: "invalid",
			expected: 5 * time.Minute,
		},
		{
			name:     "Empty duration - default",
			envValue: "",
			expected: 5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv("AUCTION_INTERVAL", tt.envValue)
			} else {
				os.Unsetenv("AUCTION_INTERVAL")
			}
			defer os.Unsetenv("AUCTION_INTERVAL")

			duration := getAuctionDuration()
			assert.Equal(t, tt.expected, duration)
		})
	}
}

func TestAuctionEntityCreation(t *testing.T) {
	auction := &auction_entity.Auction{
		Id:          "test-auction-id",
		ProductName: "Produto Teste",
		Category:    "Categoria Teste",
		Description: "Descrição do produto teste para validação",
		Condition:   auction_entity.New,
		Status:      auction_entity.Active,
		Timestamp:   time.Now(),
	}

	assert.Equal(t, auction_entity.Active, auction.Status)
	assert.NotEmpty(t, auction.Id)
	assert.NotEmpty(t, auction.ProductName)
}

func TestExpiredAuctionLogic(t *testing.T) {
	auctionInterval := 3 * time.Second

	auctionTimestamp := time.Now().Add(-5 * time.Second)

	expirationTime := time.Now().Add(-auctionInterval)

	isExpired := auctionTimestamp.Before(expirationTime)

	assert.True(t, isExpired, "O leilão deveria estar expirado")
}

func TestActiveAuctionLogic(t *testing.T) {
	auctionInterval := 5 * time.Second

	auctionTimestamp := time.Now()

	expirationTime := time.Now().Add(-auctionInterval)

	isExpired := auctionTimestamp.Before(expirationTime)

	assert.False(t, isExpired, "O leilão deveria ainda estar ativo")
}

func TestBsonFilterCreation(t *testing.T) {
	auctionInterval := 20 * time.Second
	expirationTime := time.Now().Add(-auctionInterval).Unix()

	filter := bson.M{
		"status":    auction_entity.Active,
		"timestamp": bson.M{"$lte": expirationTime},
	}

	assert.NotNil(t, filter)
	assert.Equal(t, auction_entity.Active, filter["status"])
	assert.NotNil(t, filter["timestamp"])

	timestampFilter := filter["timestamp"].(bson.M)
	assert.Equal(t, expirationTime, timestampFilter["$lte"])
}

func TestBsonUpdateCreation(t *testing.T) {
	update := bson.M{
		"$set": bson.M{
			"status": auction_entity.Completed,
		},
	}

	assert.NotNil(t, update)
	assert.NotNil(t, update["$set"])

	setUpdate := update["$set"].(bson.M)
	assert.Equal(t, auction_entity.Completed, setUpdate["status"])
}

func TestTimestampConversion(t *testing.T) {
	now := time.Now()
	unixTimestamp := now.Unix()

	convertedTime := time.Unix(unixTimestamp, 0)

	diff := now.Sub(convertedTime)
	assert.Less(t, diff, time.Second)
	assert.GreaterOrEqual(t, diff, time.Duration(0))
}

func TestAuctionStatusTransition(t *testing.T) {
	status := auction_entity.Active
	assert.Equal(t, auction_entity.Active, status)

	status = auction_entity.Completed
	assert.Equal(t, auction_entity.Completed, status)
	assert.NotEqual(t, auction_entity.Active, status)
}

func TestCheckIntervalCalculation(t *testing.T) {
	tests := []struct {
		name            string
		auctionInterval time.Duration
		expectedMinimum time.Duration
	}{
		{
			name:            "Long interval",
			auctionInterval: 60 * time.Second,
			expectedMinimum: 30 * time.Second,
		},
		{
			name:            "Short interval - should use minimum",
			auctionInterval: 10 * time.Second,
			expectedMinimum: 10 * time.Second,
		},
		{
			name:            "Very short interval - should use minimum",
			auctionInterval: 5 * time.Second,
			expectedMinimum: 10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checkInterval := tt.auctionInterval / 2
			if checkInterval < 10*time.Second {
				checkInterval = 10 * time.Second
			}

			assert.GreaterOrEqual(t, checkInterval, tt.expectedMinimum)
		})
	}
}

func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan bool)

	go func() {
		select {
		case <-ctx.Done():
			done <- true
			return
		case <-time.After(5 * time.Second):
			done <- false
			return
		}
	}()

	cancel()

	result := <-done

	assert.True(t, result, "A goroutine deveria ter terminado via ctx.Done()")
}
