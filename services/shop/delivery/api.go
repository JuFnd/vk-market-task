package delivery

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"market/pkg/middleware"
	"market/pkg/models"
	communication "market/pkg/requests"
	"market/pkg/util"
	"market/pkg/variables"
	"market/services/shop/usecase"
	"net/http"
	"os"
	"strconv"
)

type ICore interface {
	AdvertsList(sid string, sortedBy string, sortDirection string, start uint64, end uint64) ([]communication.AdvertItemResponse, error)
	AddAdvert(advert models.AdvertItem, userId uint64) error
	AdvertItem(id int64) (*communication.AdvertItemResponse, error)
	GetUserId(ctx context.Context, sid string) (int64, error)
	GetUserRole(ctx context.Context, id int64) (string, error)
}

type API struct {
	core   ICore
	logger *slog.Logger
	mux    *http.ServeMux
}

func (api *API) ListenAndServe(appConfig *variables.AppConfig) error {
	err := http.ListenAndServe(appConfig.Address, api.mux)
	if err != nil {
		api.logger.Error(variables.ListenAndServeError, err.Error())
		return err
	}
	return nil
}

func GetMarketApi(marketCore *usecase.Core, marketLogger *slog.Logger) *API {
	api := &API{
		core:   marketCore,
		logger: marketLogger,
		mux:    http.NewServeMux(),
	}

	// Adverts list
	api.mux.Handle("/api/v1/adverts", middleware.MethodMiddleware(
		http.HandlerFunc(api.AdvertsList),
		http.MethodGet,
		api.logger))

	// Adverts list
	api.mux.Handle("/api/v1/adverts/item", middleware.MethodMiddleware(
		http.HandlerFunc(api.AdvertItem),
		http.MethodGet,
		api.logger))

	// Add advert handler
	api.mux.Handle("/api/v1/adverts/add", middleware.MethodMiddleware(
		middleware.AuthorizationMiddleware(
			http.HandlerFunc(api.AddAdvert),
			api.core,
			api.logger),
		http.MethodPost,
		api.logger))
	return api
}

func (api *API) AddAdvert(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		util.SendResponse(w, r, http.StatusBadRequest, variables.StatusBadRequestError, variables.StatusBadRequestError, err, api.logger)
		return
	}

	title := r.FormValue("title")
	err = util.ValidateStringSize(title, variables.MinTitleSize, variables.MaxTitleSize, variables.ValidateStringError, api.logger)
	if err != nil {
		util.SendResponse(w, r, http.StatusBadRequest, variables.StatusBadRequestError, variables.StatusBadRequestError, err, api.logger)
		return
	}

	description := r.FormValue("description")
	err = util.ValidateStringSize(title, variables.MinDescriptionSize, variables.MaxDescriptionSize, variables.ValidateStringError, api.logger)
	if err != nil {
		util.SendResponse(w, r, http.StatusBadRequest, variables.StatusBadRequestError, variables.StatusBadRequestError, err, api.logger)
		return
	}

	price, err := strconv.ParseInt(r.FormValue("price"), 10, 64)
	if err != nil {
		util.SendResponse(w, r, http.StatusBadRequest, variables.StatusBadRequestError, variables.StatusBadRequestError, err, api.logger)
		return
	}

	image, handler, err := r.FormFile("image")
	if err != nil && !errors.Is(err, http.ErrMissingFile) {
		util.SendResponse(w, r, http.StatusBadRequest, variables.StatusBadRequestError, variables.StatusBadRequestError, err, api.logger)
		return
	}

	fileType := handler.Header.Get("Content-Type")
	isValidImage := false
	for _, validType := range variables.ValidImageTypes {
		if fileType == validType {
			isValidImage = true
			break
		}
	}

	if err != nil || handler == nil || image == nil || !isValidImage {
		util.SendResponse(w, r, http.StatusBadRequest, variables.StatusBadRequestError, variables.StatusBadRequestError, err, api.logger)
		return
	}

	err = util.ValidateImageType(fileType)
	if err != nil {
		util.SendResponse(w, r, http.StatusBadRequest, variables.InvalidImageError, variables.InvalidImageError, err, api.logger)
		return
	}

	filename := "/advert-images/" + handler.Filename
	if err != nil && handler != nil && image != nil {
		util.SendResponse(w, r, http.StatusBadRequest, variables.StatusBadRequestError, variables.StatusBadRequestError, err, api.logger)
		return
	}

	fileImage, err := os.OpenFile("/home/image-store"+filename, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		util.SendResponse(w, r, http.StatusInternalServerError, variables.StatusInternalServerError, variables.StatusInternalServerError, err, api.logger)
		return
	}
	defer fileImage.Close()

	_, err = io.Copy(fileImage, image)
	if err != nil {
		util.SendResponse(w, r, http.StatusInternalServerError, variables.StatusInternalServerError, variables.StatusInternalServerError, err, api.logger)
		return
	}

	userId, _ := r.Context().Value(variables.UserIDKey).(uint64)

	advert := models.AdvertItem{
		Title:       title,
		Description: description,
		Price:       price,
		ImagePath:   filename,
	}

	err = api.core.AddAdvert(advert, userId)
	if err != nil {
		util.SendResponse(w, r, http.StatusInternalServerError, variables.StatusInternalServerError, variables.StatusInternalServerError, err, api.logger)
		return
	}
	util.SendResponse(w, r, http.StatusOK, advert, variables.StatusOkMessage, nil, api.logger)
}

func (api *API) AdvertsList(w http.ResponseWriter, r *http.Request) {
	var sid string
	pageList, pageSize := util.Pagination(r)

	start := (pageList - 1) * pageSize
	end := pageList * pageSize
	sortedBy := r.URL.Query().Get("sorted_by")
	sortDirection := r.URL.Query().Get("sort_direction")
	session, err := r.Cookie(variables.SessionCookieName)
	if err != nil {
		sid = ""
	} else {
		sid = session.Value
	}

	adverts, err := api.core.AdvertsList(sid, sortedBy, sortDirection, start, end)
	if err != nil {
		util.SendResponse(w, r, http.StatusInternalServerError, variables.StatusInternalServerError, variables.StatusInternalServerError, err, api.logger)
		return
	}

	util.SendResponse(w, r, http.StatusOK, adverts, variables.StatusOkMessage, nil, api.logger)
}

func (api *API) AdvertItem(w http.ResponseWriter, r *http.Request) {
	advertId, err := strconv.ParseInt(r.URL.Query().Get("advert_id"), 10, 64)
	if err != nil {
		util.SendResponse(w, r, http.StatusNotFound, variables.AdvertNotFoundError, variables.AdvertNotFoundError, err, api.logger)
		return
	}

	advert, err := api.core.AdvertItem(advertId)
	if err != nil {
		util.SendResponse(w, r, http.StatusNotFound, variables.AdvertNotFoundError, variables.AdvertNotFoundError, err, api.logger)
		return
	}

	util.SendResponse(w, r, http.StatusOK, advert, variables.StatusOkMessage, nil, api.logger)
}
