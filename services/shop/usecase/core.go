package usecase

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log/slog"
	"market/pkg/models"
	communication "market/pkg/requests"
	"market/pkg/variables"
	"market/services/authorization/proto/authorization"
)

type IAdvertRepository interface {
	AdvertsList(userId int64, sortedBy string, sortDirection string, start uint64, end uint64) ([]communication.AdvertItemResponse, error)
	AdvertItem(id int64) (*communication.AdvertItemResponse, error)
	AddAdvert(advert models.AdvertItem, id uint64) error
}

type Core struct {
	logger           *slog.Logger
	advertRepository IAdvertRepository
	grpcClient       authorization.AuthorizationClient
}

func GetGrpcClient(port string) (authorization.AuthorizationClient, error) {
	conn, err := grpc.Dial(port, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf(variables.GrpcConnectError, ": %w", err)
	}
	client := authorization.NewAuthorizationClient(conn)

	return client, nil
}

func GetCore(configGrpc variables.GrpcConfig, adverts IAdvertRepository, logger *slog.Logger) *Core {
	client, err := GetGrpcClient(configGrpc.Port)
	if err != nil {
		logger.Error(variables.GrpcConnectError, ": %w", err)
		return nil
	}
	return &Core{
		advertRepository: adverts,
		grpcClient:       client,
		logger:           logger,
	}
}

func (core *Core) AdvertsList(sid string, sortedBy string, sortDirection string, start uint64, end uint64) ([]communication.AdvertItemResponse, error) {
	if sortedBy == "" || (sortedBy != "date" && sortedBy != "price") {
		sortedBy = "date"
	}

	if sortDirection == "" || (sortDirection != "asc" && sortDirection != "desc") {
		sortDirection = "desc"
	}

	userId, err := core.GetUserId(context.Background(), sid)

	adverts, err := core.advertRepository.AdvertsList(userId, sortedBy, sortDirection, start, end)
	if err != nil {
		core.logger.Error(variables.AdvertNotFoundError, err)
		return nil, err
	}

	return adverts, nil
}

func (core *Core) AdvertItem(id int64) (*communication.AdvertItemResponse, error) {
	advert, err := core.advertRepository.AdvertItem(id)
	if err != nil {
		core.logger.Error(variables.AdvertNotFoundError, err)
		return nil, err
	}
	return advert, nil
}

func (core *Core) AddAdvert(advert models.AdvertItem, id uint64) error {
	err := core.advertRepository.AddAdvert(advert, id)
	if err != nil {
		core.logger.Error(variables.AdvertNotCreatedError, err)
		return err
	}
	return nil
}

func (core *Core) GetUserRole(ctx context.Context, id int64) (string, error) {
	grpcRequest := authorization.RoleRequest{Id: id}

	grpcResponse, err := core.grpcClient.GetRole(ctx, &grpcRequest)
	if err != nil {
		core.logger.Error(variables.GrpcRecievError, err)
		return "", fmt.Errorf(variables.GrpcRecievError, err)
	}
	return grpcResponse.GetRole(), nil
}

func (core *Core) GetUserId(ctx context.Context, sid string) (int64, error) {
	grpcRequest := authorization.FindIdRequest{Sid: sid}

	grpcResponse, err := core.grpcClient.GetId(ctx, &grpcRequest)
	if err != nil {
		core.logger.Error(variables.GrpcRecievError, err)
		return 0, fmt.Errorf(variables.GrpcRecievError, err)
	}
	return grpcResponse.Value, nil
}
