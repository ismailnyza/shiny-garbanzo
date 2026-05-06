package audit

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Log struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ActorType  string         `gorm:"not null"`
	ActorID    *uuid.UUID     `gorm:"type:uuid"`
	TenantID   *uuid.UUID     `gorm:"type:uuid"`
	Action     string         `gorm:"not null"`
	EntityType string         `gorm:"not null"`
	EntityID   *uuid.UUID     `gorm:"type:uuid"`
	Metadata   datatypes.JSON `gorm:"type:jsonb"`
	CreatedAt  time.Time      `gorm:"not null;default:now()"`
}

func (Log) TableName() string {
	return "audit_logs"
}

var db *gorm.DB

func Configure(gormDB *gorm.DB) {
	db = gormDB
}

func Record(c *gin.Context, action, entityType string, entityID uuid.UUID, metadata map[string]interface{}) {
	if db == nil {
		return
	}
	log := &Log{
		ActorType:  "SYSTEM",
		Action:     action,
		EntityType: entityType,
	}
	if entityID != uuid.Nil {
		log.EntityID = &entityID
	}
	if c != nil {
		if userID, err := uuid.Parse(c.GetString("user_id")); err == nil {
			log.ActorType = "USER"
			log.ActorID = &userID
		} else if c.Param("sessionToken") != "" {
			log.ActorType = "CUSTOMER_SESSION"
		}
		if tenantID, err := uuid.Parse(c.Param("restaurantId")); err == nil {
			log.TenantID = &tenantID
		}
	}
	if metadata != nil {
		b, _ := json.Marshal(metadata)
		log.Metadata = datatypes.JSON(b)
	}
	_ = db.Create(log).Error
}
