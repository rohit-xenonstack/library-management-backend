package repository

import (
	"errors"
	"library-management/backend/internal/api/model"
	"library-management/backend/internal/database/transaction"
	"library-management/backend/internal/util"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ReaderRepository struct {
	db        *gorm.DB
	txManager *transaction.TxManager
	mu        sync.RWMutex
}

func NewReaderRepository(db *gorm.DB, txManager *transaction.TxManager) *ReaderRepository {
	return &ReaderRepository{
		db:        db,
		txManager: txManager,
	}
}

func (reader *ReaderRepository) SearchBookByTitle(ctx *gin.Context, title string, books *[]model.BookInventory) error {
	reader.mu.Lock()
	defer reader.mu.Unlock()

	return reader.txManager.ExecuteInTx(ctx, func(tx *gorm.DB) error {
		query := `select * from book_inventories where lower(title) like lower('%` + title + `%')`
		return tx.Set("gorm:query_option", "FOR UPDATE").Model(&model.BookInventory{}).Raw(query).Scan(&books).Error
	})
}

func (reader *ReaderRepository) SearchBookByAuthor(ctx *gin.Context, author string, books *[]model.BookInventory) error {
	reader.mu.Lock()
	defer reader.mu.Unlock()

	return reader.txManager.ExecuteInTx(ctx, func(tx *gorm.DB) error {
		query := `select * from book_inventories where lower(authors) like lower('%` + author + `%')`
		return tx.Set("gorm:query_option", "FOR UPDATE").Model(&model.BookInventory{}).Raw(query).Scan(&books).Error
	})
}

func (reader *ReaderRepository) SearchBookByPublisher(ctx *gin.Context, publisher string, books *[]model.BookInventory) error {
	reader.mu.Lock()
	defer reader.mu.Unlock()

	return reader.txManager.ExecuteInTx(ctx, func(tx *gorm.DB) error {
		query := `select * from book_inventories where lower(publisher) like lower('%` + publisher + `%')`
		return tx.Set("gorm:query_option", "FOR UPDATE").Model(&model.BookInventory{}).Raw(query).Scan(&books).Error
	})
}

func (reader *ReaderRepository) RaiseIssueRequest(ctx *gin.Context, isbn string, email string, readerID string) error {
	reader.mu.Lock()
	defer reader.mu.Unlock()

	return reader.txManager.ExecuteInTx(ctx, func(tx *gorm.DB) error {
		var user model.Users
		result := tx.Set("gorm:query_option", "FOR UPDATE").Where("email = ?", email).First(&user)
		if result.Error != nil {
			return result.Error
		}

		if user.Role != util.ReaderRole {
			return errors.New("access denied, provide a valid Reader email")
		}
		readerID := user.ID

		var existingBook model.BookInventory
		result = tx.Set("gorm:query_option", "FOR UPDATE").Where("isbn = ?", isbn).First(&existingBook)

		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return errors.New("book with supplied ISBN not found in database")
		}

		if existingBook.AvailableCopies < 1 {
			return errors.New("no books available to issue")
		}

		var existingIssueRequest model.IssueRegistry
		result = tx.Set("gorm:query_option", "FOR UPDATE").Model(&model.IssueRegistry{}).Where("reader_id = ?", readerID).Where("book_id = ?", isbn).Where("issue_status = ?", "open").First(&existingIssueRequest)
		if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return result.Error
		}
		if result.RowsAffected > 0 {
			return errors.New("book already issued to reader")
		}

		var existingRequestEvent model.RequestEvents
		result = tx.Set("gorm:query_option", "FOR UPDATE").Model(&model.RequestEvents{}).Where("reader_id = ?", readerID).Where("book_id = ?", isbn).First(&existingRequestEvent)
		if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return result.Error
		}
		if result.RowsAffected > 0 {
			return errors.New("only one request allowed at a time")
		}

		issueRequest := model.RequestEvents{
			ReqID:        util.RandomUUID(),
			BookID:       isbn,
			ReaderID:     readerID,
			RequestDate:  time.Now().Format(time.RFC3339),
			ApprovalDate: nil,
			ApproverID:   nil,
			RequestType:  "issue",
		}
		return tx.Model(&model.RequestEvents{}).Create(&issueRequest).Error
	})
}

func (reader *ReaderRepository) GetLatestBookAvailability(ctx *gin.Context, isbn string, latestDate *string) error {
	reader.mu.Lock()
	defer reader.mu.Unlock()

	return reader.txManager.ExecuteInTx(ctx, func(tx *gorm.DB) error {
		query := `
            SELECT expected_return_date
            FROM issue_registries
            WHERE book_id = ?
            ORDER BY expected_return_date ASC
            LIMIT 1
        `
		return tx.Set("gorm:query_option", "FOR UPDATE").
			Raw(query, isbn).
			Scan(latestDate).
			Error
	})
}

func (reader *ReaderRepository) GetBooks(ctx *gin.Context, books *[]model.BookInventory) error {
	reader.mu.Lock()
	defer reader.mu.Unlock()

	return reader.txManager.ExecuteInTx(ctx, func(tx *gorm.DB) error {
		return tx.Model(&model.BookInventory{}).Find(&books).Error
	})
}
