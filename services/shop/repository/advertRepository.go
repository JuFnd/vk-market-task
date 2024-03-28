package repository

import (
	"database/sql"
	"fmt"
	"log/slog"
	"market/pkg/models"
	communication "market/pkg/requests"
	"market/pkg/variables"
	"time"

	_ "github.com/jackc/pgx/stdlib"
)

type AdvertRepository struct {
	db *sql.DB
}

func GetAdvertRepository(configDatabase variables.RelationalDataBaseConfig, logger *slog.Logger) (*AdvertRepository, error) {
	dsn := fmt.Sprintf("user=%s dbname=%s password= %s host=%s port=%d sslmode=%s",
		configDatabase.User, configDatabase.DbName, configDatabase.Password, configDatabase.Host, configDatabase.Port, configDatabase.Sslmode)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		logger.Error(variables.SqlOpenError, err.Error())
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		logger.Error(variables.SqlPingError, err.Error())
		return nil, err
	}

	db.SetMaxOpenConns(configDatabase.MaxOpenConns)

	advertRepository := &AdvertRepository{db: db}

	errs := make(chan error)
	go func() {
		errs <- advertRepository.pingDb(configDatabase.Timer, logger)
	}()

	if err := <-errs; err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	return advertRepository, nil
}

func (repository *AdvertRepository) pingDb(timer uint32, logger *slog.Logger) error {
	var err error
	var retries int

	for retries < variables.MaxRetries {
		err = repository.db.Ping()
		if err == nil {
			return nil
		}

		retries++
		logger.Error(variables.SqlPingError, err.Error())
		time.Sleep(time.Duration(timer) * time.Second)
	}

	logger.Error(variables.SqlMaxPingRetriesError, err)
	return fmt.Errorf(variables.SqlMaxPingRetriesError, err.Error())
}

func (repository *AdvertRepository) AdvertsList(userId int64, sortedBy string, sortDirection string, start uint64, end uint64) ([]communication.AdvertItemResponse, error) {
	var advert []communication.AdvertItemResponse

	var query string
	switch {
	case sortedBy == "price" && sortDirection == "desc":
		query = "SELECT id, title, description, created_date, price, image_path, profile_id FROM adverts WHERE profile_id = $1 ORDER BY price DESC LIMIT $2 OFFSET $3"
	case sortedBy == "date" && sortDirection == "asc":
		query = "SELECT id, title, description, created_date, price, image_path, profile_id FROM adverts WHERE profile_id = $1 ORDER BY created_date ASC LIMIT $2 OFFSET $3"
	case sortedBy == "price" && sortDirection == "asc":
		query = "SELECT id, title, description, created_date, price, image_path, profile_id FROM adverts WHERE profile_id = $1 ORDER BY price ASC LIMIT $2 OFFSET $3"
	case sortedBy == "date" && sortDirection == "desc":
		query = "SELECT id, title, description, created_date, price, image_path, profile_id FROM adverts WHERE profile_id = $1 ORDER BY created_date DESC LIMIT $2 OFFSET $3"
	default:
		query = "SELECT id, title, description, created_date, price, image_path, profile_id FROM adverts WHERE profile_id = $1 ORDER BY created_date DESC LIMIT $2 OFFSET $3"
	}

	rows, err := repository.db.Query(query, userId, end-start, start)
	if err != nil {
		return []communication.AdvertItemResponse{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var item communication.AdvertItemResponse

		err := rows.Scan(&item.ID, &item.Title, &item.Description, &item.Date, &item.Price, &item.ImagePath, &item.ProfileId)

		if err != nil {
			return []communication.AdvertItemResponse{}, err
		}

		if item.ProfileId == userId {
			item.IsAuthor = true
		} else {
			item.IsAuthor = false
		}
		advert = append(advert, item)
	}
	return advert, nil
}

func (repository *AdvertRepository) AdvertItem(id int64) (*communication.AdvertItemResponse, error) {
	query := "SELECT id, title, description, created_date, price, image_path, profile_id FROM adverts WHERE id = $1"

	row := repository.db.QueryRow(query, id)

	var item communication.AdvertItemResponse
	err := row.Scan(&item.ID, &item.Title, &item.Description, &item.Date, &item.Price, &item.ImagePath, &item.ProfileId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf(variables.AdvertNotFoundError)
		}
		return nil, err
	}

	return &item, nil
}

func (repository *AdvertRepository) AddAdvert(advert models.AdvertItem, id uint64) error {
	query := "INSERT INTO adverts (title, description, price, image_path, profile_id) VALUES ($1, $2, $3, $4, $5)"

	_, err := repository.db.Exec(query, advert.Title, advert.Description, advert.Price, advert.ImagePath, id)
	if err != nil {
		return err
	}

	return nil
}
