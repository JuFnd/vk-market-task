package main

import (
	"fmt"
	"log/slog"
	"market/configs"
	"market/pkg/variables"
	"market/services/shop/delivery"
	"market/services/shop/repository"
	"market/services/shop/usecase"
	"os"
)

// @title Market service
// @version 1.0
// @description VK Market films service

// @host localhost:8081
// @BasePath /

// @in header
// @name Films

func main() {
	logFile, err := os.Create("market.log")
	if err != nil {
		fmt.Println("Error creating log file")
		return
	}

	logger := slog.New(slog.NewJSONHandler(logFile, nil))

	configFilms, err := configs.ReadMarketAppConfig()
	if err != nil {
		logger.Error(err.Error())
		return
	}

	relationalDataBaseConfig, err := configs.ReadRelationalMarketDataBaseConfig()
	if err != nil {
		logger.Error(variables.ReadMarketSqlConfigError, err.Error())
		return
	}

	grpcConfig, err := configs.ReadGrpcConfig()

	advertsRepository, err := repository.GetAdvertRepository(*relationalDataBaseConfig, logger)
	core := usecase.GetCore(*grpcConfig, advertsRepository, logger)
	if err != nil {
		logger.Error(variables.CoreInitializeError, err)
		return
	}

	api := delivery.GetMarketApi(core, logger)

	err = api.ListenAndServe(configFilms)
	if err != nil {
		logger.Error(err.Error())
	}
}
