package storage

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/quanhengzhuang/gtm"
)

func init() {
	db, err := gorm.Open("mysql", "root:root1234@/gtm?charset=utf8&parseTime=True&loc=Local")
	if err != nil {
		log.Fatalf("db open failed: %v", err)
	}
	db.LogMode(true)
}

type TransactionMeta struct {
	*gtm.Transaction

	CreatedAt time.Time
	Result    string
	Cost      time.Duration
}

type PartnerResult struct {
	TransactionID string
	Phase         string
	Offset        int
	Result        gtm.Result
	Cost          time.Duration
	CreatedAt     time.Time
}

type ConsoleStorage interface {
	gtm.Storage

	GetTransactions(page int, pageSize int) (txs []TransactionMeta, err error)
	GetTransaction(id string) (tx TransactionMeta, err error)
	GetPartnerResults(txID string) (results []PartnerResult, err error)
}

type DBConsoleStorage struct {
	*gtm.DBStorage
	db *gorm.DB
}

func NewDBConsoleStorage(db *gorm.DB) *DBConsoleStorage {
	return &DBConsoleStorage{db: db}
}

func (s *DBConsoleStorage) GetTransactions(page int, pageSize int) (txs []TransactionMeta, err error) {
	var rows []gtm.DBStorageTransaction
	err = s.db.Where("1=1").Offset((page - 1) * pageSize).Limit(pageSize).Order("id DESC").Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("find err: %v", err)
	}

	for _, row := range rows {
		tx, err := s.convert(row)
		if err != nil {
			return nil, fmt.Errorf("convert failed: %v", err)
		}

		txs = append(txs, tx)
	}

	return txs, nil
}

func (s *DBConsoleStorage) GetTransaction(id string) (tx TransactionMeta, err error) {
	var row gtm.DBStorageTransaction
	err = s.db.First(&row, "id=?", id).Error
	if err != nil {
		return tx, fmt.Errorf("find err: %v", err)
	}

	return s.convert(row)
}

func (s *DBConsoleStorage) GetPartnerResults(txID string) (results []PartnerResult, err error) {
	var rows []gtm.DBStoragePartnerResult
	err = s.db.Where("transaction_id=?", txID).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("find err: %v", err)
	}

	for _, row := range rows {
		result := PartnerResult{
			TransactionID: strconv.Itoa(row.TransactionID),
			Phase:         row.Phase,
			Offset:        row.Offset,
			Result:        gtm.Result(row.Result),
			Cost:          row.Cost,
			CreatedAt:     row.CreatedAt,
		}
		results = append(results, result)
	}

	return results, nil
}

func (s *DBConsoleStorage) convert(row gtm.DBStorageTransaction) (tx TransactionMeta, err error) {
	tx.Transaction, err = s.Decode(row.Content)
	if err != nil {
		tx.Transaction = &gtm.Transaction{}
	}

	tx.Transaction.ID = strconv.Itoa(row.ID)
	tx.Transaction.Name = row.Name
	tx.Transaction.Times = row.Times
	tx.Transaction.RetryAt = row.RetryAt

	tx.CreatedAt = row.CreatedAt
	tx.Result = row.Result
	tx.Cost = row.Cost

	return tx, nil
}
