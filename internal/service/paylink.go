package service

import (
	"context"
	"fmt"
	u "github.com/PuerkitoBio/purell"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/golang/protobuf/ptypes"
	"github.com/google/go-querystring/query"
	"github.com/jinzhu/now"
	"github.com/paysuper/paysuper-billing-server/pkg"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/billing"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/grpc"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/paylink"
	"go.uber.org/zap"
	"time"
)

type orderViewPaylinkStatFunc func(OrderViewServiceInterface, string, string, int64, int64) (*paylink.GroupStatCommon, error)

type paylinkVisits struct {
	PaylinkId bson.ObjectId `bson:"paylink_id"`
	Date      time.Time     `bson:"date"`
}

type utmQueryParams struct {
	UtmSource   string `url:"utm_source,omitempty"`
	UtmMedium   string `url:"utm_medium,omitempty"`
	UtmCampaign string `url:"utm_campaign,omitempty"`
}

type PaylinkServiceInterface interface {
	CountByQuery(query bson.M) (n int, err error)
	GetListByQuery(query bson.M, limit, offset int) (result []*paylink.Paylink, err error)
	GetById(id string) (pl *paylink.Paylink, err error)
	GetByIdAndMerchant(id, merchantId string) (pl *paylink.Paylink, err error)
	IncrVisits(id string) error
	GetUrl(id, merchantId, urlMask, utmSource, utmMedium, utmCampaign string) (string, error)
	Delete(id, merchantId string) error
	Insert(pl *paylink.Paylink) error
	Update(pl *paylink.Paylink) error
	UpdatePaylinkTotalStat(id, merchantId string) error
	GetPaylinkVisits(id string, from, to int64) (int, error)
}

const (
	collectionPaylinks      = "paylinks"
	collectionPaylinkVisits = "paylink_visits"

	cacheKeyPaylink         = "paylink:id:%s"
	cacheKeyPaylinkMerchant = "paylink:id:%s:merhcant_id:%s"
)

var (
	errorPaylinkExpired                      = newBillingServerErrorMsg("pl000001", "payment link expired")
	errorPaylinkNotFound                     = newBillingServerErrorMsg("pl000002", "paylink not found")
	errorPaylinkProjectMismatch              = newBillingServerErrorMsg("pl000003", "projectId mismatch for existing paylink")
	errorPaylinkExpiresInPast                = newBillingServerErrorMsg("pl000004", "paylink expiry date in past")
	errorPaylinkProductsLengthInvalid        = newBillingServerErrorMsg("pl000005", "paylink products length invalid")
	errorPaylinkProductsTypeInvalid          = newBillingServerErrorMsg("pl000006", "paylink products type invalid")
	errorPaylinkProductNotBelongToMerchant   = newBillingServerErrorMsg("pl000007", "at least one of paylink products is not belongs to merchant")
	errorPaylinkProductNotBelongToProject    = newBillingServerErrorMsg("pl000008", "at least one of paylink products is not belongs to project")
	errorPaylinkStatDataInconsistent         = newBillingServerErrorMsg("pl000009", "paylink stat data inconsistent")
	errorPaylinkProductNotFoundOrInvalidType = newBillingServerErrorMsg("pl000010", "at least one of paylink products is not found or have type differ from given products_type value")

	orderViewPaylinkStatFuncMap = map[string]orderViewPaylinkStatFunc{
		"GetPaylinkStatByCountry":  OrderViewServiceInterface.GetPaylinkStatByCountry,
		"GetPaylinkStatByReferrer": OrderViewServiceInterface.GetPaylinkStatByReferrer,
		"GetPaylinkStatByDate":     OrderViewServiceInterface.GetPaylinkStatByDate,
		"GetPaylinkStatByUtm":      OrderViewServiceInterface.GetPaylinkStatByUtm,
	}
)

func newPaylinkService(svc *Service) *Paylink {
	s := &Paylink{svc: svc}
	return s
}

// GetPaylinks returns list of all payment links
func (s *Service) GetPaylinks(
	ctx context.Context,
	req *grpc.GetPaylinksRequest,
	res *grpc.GetPaylinksResponse,
) error {

	dbQuery := bson.M{
		"deleted":     false,
		"merchant_id": bson.ObjectIdHex(req.MerchantId),
	}

	if req.ProjectId != "" {
		dbQuery["project_id"] = bson.ObjectIdHex(req.ProjectId)
	}

	n, err := s.paylinkService.CountByQuery(dbQuery)
	if err != nil {
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			res.Status = pkg.ResponseStatusBadData
			res.Message = e
			return nil
		}
		return err
	}

	res.Data = &grpc.PaylinksPaginate{}

	if n > 0 {
		res.Data.Items, err = s.paylinkService.GetListByQuery(dbQuery, int(req.Limit), int(req.Offset))
		if err != nil {
			if e, ok := err.(*grpc.ResponseErrorMessage); ok {
				res.Status = pkg.ResponseStatusBadData
				res.Message = e
				return nil
			}
			return err
		}

		for _, pl := range res.Data.Items {
			visits, err := s.paylinkService.GetPaylinkVisits(pl.Id, 0, 0)
			if err == nil {
				pl.Visits = int32(visits)
			}
			pl.UpdateConversion()
			pl.IsExpired = pl.GetIsExpired()
		}
	}

	res.Data.Count = int32(n)
	res.Status = pkg.ResponseStatusOk

	return nil
}

// GetPaylink returns one payment link
func (s *Service) GetPaylink(
	ctx context.Context,
	req *grpc.PaylinkRequest,
	res *grpc.GetPaylinkResponse,
) (err error) {

	res.Item, err = s.paylinkService.GetByIdAndMerchant(req.Id, req.MerchantId)
	if err != nil {
		if err == mgo.ErrNotFound {
			res.Status = pkg.ResponseStatusNotFound
			res.Message = errorPaylinkNotFound
			return nil
		}
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			res.Status = pkg.ResponseStatusBadData
			res.Message = e
			return nil
		}
		return err
	}

	visits, err := s.paylinkService.GetPaylinkVisits(res.Item.Id, 0, 0)
	if err == nil {
		res.Item.Visits = int32(visits)
	}
	res.Item.UpdateConversion()

	res.Status = pkg.ResponseStatusOk
	return nil
}

// IncrPaylinkVisits adds a visit hit to stat
func (s *Service) IncrPaylinkVisits(
	ctx context.Context,
	req *grpc.PaylinkRequestById,
	res *grpc.EmptyResponse,
) error {
	err := s.paylinkService.IncrVisits(req.Id)
	if err != nil {
		return err
	}
	return nil
}

// GetPaylinkURL returns public url for Paylink
func (s *Service) GetPaylinkURL(
	ctx context.Context,
	req *grpc.GetPaylinkURLRequest,
	res *grpc.GetPaylinkUrlResponse,
) (err error) {

	res.Url, err = s.paylinkService.GetUrl(req.Id, req.MerchantId, req.UrlMask, req.UtmMedium, req.UtmMedium, req.UtmCampaign)
	if err != nil {
		if err == mgo.ErrNotFound {
			res.Status = pkg.ResponseStatusNotFound
			res.Message = errorPaylinkNotFound
			return nil
		}
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			if err == errorPaylinkExpired {
				res.Status = pkg.ResponseStatusGone
			} else {
				res.Status = pkg.ResponseStatusBadData
			}

			res.Message = e
			return nil
		}
		return err
	}

	res.Status = pkg.ResponseStatusOk
	return nil
}

// DeletePaylink deletes payment link
func (s *Service) DeletePaylink(
	ctx context.Context,
	req *grpc.PaylinkRequest,
	res *grpc.EmptyResponseWithStatus,
) error {

	err := s.paylinkService.Delete(req.Id, req.MerchantId)
	if err != nil {
		if err == mgo.ErrNotFound {
			res.Status = pkg.ResponseStatusNotFound
			res.Message = errorPaylinkNotFound
			return nil
		}
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			res.Status = pkg.ResponseStatusBadData
			res.Message = e
			return nil
		}
		return err
	}

	res.Status = pkg.ResponseStatusOk
	return nil
}

// CreateOrUpdatePaylink create or modify payment link
func (s *Service) CreateOrUpdatePaylink(
	ctx context.Context,
	req *paylink.CreatePaylinkRequest,
	res *grpc.GetPaylinkResponse,
) (err error) {

	isNew := req.GetId() == ""

	pl := &paylink.Paylink{}

	if isNew {
		pl.Id = bson.NewObjectId().Hex()
		pl.CreatedAt = ptypes.TimestampNow()
		pl.Object = "paylink"
		pl.MerchantId = req.MerchantId
		pl.ProjectId = req.ProjectId
		pl.ProductsType = req.ProductsType
	} else {
		pl, err = s.paylinkService.GetByIdAndMerchant(req.Id, req.MerchantId)
		if err != nil {
			if err == mgo.ErrNotFound {
				res.Status = pkg.ResponseStatusNotFound
				res.Message = errorPaylinkNotFound
				return nil
			}
			if e, ok := err.(*grpc.ResponseErrorMessage); ok {
				res.Status = pkg.ResponseStatusBadData
				res.Message = e
				return nil
			}
			return err
		}

		if pl.ProjectId != req.ProjectId {
			res.Status = pkg.ResponseStatusBadData
			res.Message = errorPaylinkProjectMismatch
			return nil
		}
	}

	pl.UpdatedAt = ptypes.TimestampNow()
	pl.Name = req.Name
	pl.NoExpiryDate = req.NoExpiryDate

	dbQuery := bson.M{
		"_id":         bson.ObjectIdHex(pl.ProjectId),
		"merchant_id": bson.ObjectIdHex(pl.MerchantId),
	}
	_, err = s.getProjectBy(dbQuery)
	if err != nil {
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			res.Status = pkg.ResponseStatusBadData
			res.Message = e
			return nil
		}
		return err
	}

	if pl.NoExpiryDate == false {
		expiresAt := now.New(time.Unix(req.ExpiresAt, 0)).EndOfDay()

		if time.Now().After(expiresAt) {
			res.Status = pkg.ResponseStatusBadData
			res.Message = errorPaylinkExpiresInPast
			return nil
		}

		pl.ExpiresAt, err = ptypes.TimestampProto(expiresAt)
		if err != nil {
			zap.L().Error(
				pkg.ErrorTimeConversion,
				zap.Any(pkg.ErrorTimeConversionMethod, "ptypes.TimestampProto"),
				zap.Any(pkg.ErrorTimeConversionValue, expiresAt),
				zap.Error(err),
			)
			return err
		}
	}

	productsLength := len(req.Products)
	if productsLength < s.cfg.PaylinkMinProducts || productsLength > s.cfg.PaylinkMaxProducts {
		res.Status = pkg.ResponseStatusBadData
		res.Message = errorPaylinkProductsLengthInvalid
		return nil
	}

	for _, productId := range req.Products {
		switch req.ProductsType {

		case billing.OrderType_product:
			product, err := s.productService.GetById(productId)
			if err != nil {
				if err.Error() == "product not found" || err == mgo.ErrNotFound {
					res.Status = pkg.ResponseStatusNotFound
					res.Message = errorPaylinkProductNotFoundOrInvalidType
					return nil
				}

				if e, ok := err.(*grpc.ResponseErrorMessage); ok {
					res.Status = pkg.ResponseStatusBadData
					res.Message = e
					return nil
				}
				return err
			}

			if product.MerchantId != pl.MerchantId {
				res.Status = pkg.ResponseStatusBadData
				res.Message = errorPaylinkProductNotBelongToMerchant
				return nil
			}

			if product.ProjectId != pl.ProjectId {
				res.Status = pkg.ResponseStatusBadData
				res.Message = errorPaylinkProductNotBelongToProject
				return nil
			}

			break

		case billing.OrderType_key:
			product, err := s.getKeyProductById(productId)
			if err != nil {
				if err.Error() == "key_product not found" || err == mgo.ErrNotFound {
					res.Status = pkg.ResponseStatusNotFound
					res.Message = errorPaylinkProductNotFoundOrInvalidType
					return nil
				}

				if e, ok := err.(*grpc.ResponseErrorMessage); ok {
					res.Status = pkg.ResponseStatusBadData
					res.Message = e
					return nil
				}
				return err
			}

			if product.MerchantId != pl.MerchantId {
				res.Status = pkg.ResponseStatusBadData
				res.Message = errorPaylinkProductNotBelongToMerchant
				return nil
			}

			if product.ProjectId != pl.ProjectId {
				res.Status = pkg.ResponseStatusBadData
				res.Message = errorPaylinkProductNotBelongToProject
				return nil
			}
			break

		default:
			res.Status = pkg.ResponseStatusBadData
			res.Message = errorPaylinkProductsTypeInvalid
			return nil
		}
	}

	pl.ProductsType = req.ProductsType
	pl.Products = req.Products

	if isNew {
		err = s.paylinkService.Insert(pl)
	} else {
		err = s.paylinkService.Update(pl)
	}
	if err != nil {
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			res.Status = pkg.ResponseStatusBadData
			res.Message = e
			return nil
		}
		return err
	}

	res.Item = pl
	res.Status = pkg.ResponseStatusOk

	return nil
}

// GetPaylinkStatTotal returns total stat for requested paylink and period
func (s *Service) GetPaylinkStatTotal(
	ctx context.Context,
	req *grpc.GetPaylinkStatCommonRequest,
	res *grpc.GetPaylinkStatCommonResponse,
) (err error) {

	pl, err := s.paylinkService.GetByIdAndMerchant(req.Id, req.MerchantId)
	if err != nil {
		if err == mgo.ErrNotFound {
			res.Status = pkg.ResponseStatusNotFound
			res.Message = errorPaylinkNotFound
			return nil
		}
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			res.Status = pkg.ResponseStatusBadData
			res.Message = e
			return nil
		}
		return err
	}

	visits, err := s.paylinkService.GetPaylinkVisits(pl.Id, req.PeriodFrom, req.PeriodTo)
	if err != nil {
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			res.Status = pkg.ResponseStatusBadData
			res.Message = e
			return nil
		}
		return err
	}

	res.Item, err = s.orderView.GetPaylinkStat(pl.Id, req.MerchantId, req.PeriodFrom, req.PeriodTo)
	if err != nil {
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			res.Status = pkg.ResponseStatusBadData
			res.Message = e
			return nil
		}
		return err
	}

	res.Item.PaylinkId = pl.Id
	res.Item.Visits = int32(visits)
	res.Item.UpdateConversion()

	res.Status = pkg.ResponseStatusOk
	return nil
}

// GetPaylinkStatByCountry returns stat groped by country for requested paylink and period
func (s *Service) GetPaylinkStatByCountry(
	ctx context.Context,
	req *grpc.GetPaylinkStatCommonRequest,
	res *grpc.GetPaylinkStatCommonGroupResponse,
) (err error) {
	err = s.getPaylinkStatGroup(ctx, req, res, "GetPaylinkStatByCountry")
	if err != nil {
		return err
	}
	return nil
}

// GetPaylinkStatByReferrer returns stat grouped by referer hosts for requested paylink and period
func (s *Service) GetPaylinkStatByReferrer(
	ctx context.Context,
	req *grpc.GetPaylinkStatCommonRequest,
	res *grpc.GetPaylinkStatCommonGroupResponse,
) (err error) {
	err = s.getPaylinkStatGroup(ctx, req, res, "GetPaylinkStatByReferrer")
	if err != nil {
		return err
	}
	return nil
}

// GetPaylinkStatByDate returns stat groped by date for requested paylink and period
func (s *Service) GetPaylinkStatByDate(
	ctx context.Context,
	req *grpc.GetPaylinkStatCommonRequest,
	res *grpc.GetPaylinkStatCommonGroupResponse,
) (err error) {
	err = s.getPaylinkStatGroup(ctx, req, res, "GetPaylinkStatByDate")
	if err != nil {
		return err
	}
	return nil
}

// GetPaylinkStatByUtm returns stat groped by utm labels for requested paylink and period
func (s *Service) GetPaylinkStatByUtm(
	ctx context.Context,
	req *grpc.GetPaylinkStatCommonRequest,
	res *grpc.GetPaylinkStatCommonGroupResponse,
) (err error) {
	err = s.getPaylinkStatGroup(ctx, req, res, "GetPaylinkStatByUtm")
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) getPaylinkStatGroup(
	ctx context.Context,
	req *grpc.GetPaylinkStatCommonRequest,
	res *grpc.GetPaylinkStatCommonGroupResponse,
	function string,
) (err error) {
	pl, err := s.paylinkService.GetByIdAndMerchant(req.Id, req.MerchantId)
	if err != nil {
		if err == mgo.ErrNotFound {
			res.Status = pkg.ResponseStatusNotFound
			res.Message = errorPaylinkNotFound
			return nil
		}
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			res.Status = pkg.ResponseStatusBadData
			res.Message = e
			return nil
		}
		return err
	}

	res.Item, err = orderViewPaylinkStatFuncMap[function](s.orderView, pl.Id, pl.MerchantId, req.PeriodFrom, req.PeriodTo)
	if err != nil {
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			res.Status = pkg.ResponseStatusBadData
			res.Message = e
			return nil
		}
		return err
	}
	res.Status = pkg.ResponseStatusOk
	return nil
}

func (p Paylink) CountByQuery(query bson.M) (n int, err error) {
	n, err = p.svc.db.Collection(collectionPaylinks).Find(query).Count()
	if err != nil {
		zap.L().Error(
			pkg.ErrorDatabaseQueryFailed,
			zap.Error(err),
			zap.String(pkg.ErrorDatabaseFieldCollection, collectionPaylinks),
			zap.String(pkg.ErrorDatabaseFieldOperation, pkg.ErrorDatabaseFieldOperationCount),
			zap.Any(pkg.ErrorDatabaseFieldQuery, query),
		)
	}
	return
}

func (p Paylink) GetListByQuery(query bson.M, limit, offset int) (result []*paylink.Paylink, err error) {
	if limit <= 0 {
		limit = pkg.DatabaseRequestDefaultLimit
	}

	if offset <= 0 {
		offset = 0
	}

	err = p.svc.db.Collection(collectionPaylinks).
		Find(query).
		Sort("_id").
		Limit(limit).
		Skip(offset).
		All(&result)

	if err != nil {
		zap.L().Error(
			pkg.ErrorDatabaseQueryFailed,
			zap.Error(err),
			zap.String(pkg.ErrorDatabaseFieldCollection, collectionPaylinks),
			zap.Any(pkg.ErrorDatabaseFieldQuery, query),
		)
	}

	return
}

func (p Paylink) GetById(id string) (pl *paylink.Paylink, err error) {
	key := fmt.Sprintf(cacheKeyPaylink, id)
	dbQuery := bson.M{
		"_id":     bson.ObjectIdHex(id),
		"deleted": false,
	}
	return p.getBy(key, dbQuery)
}

func (p Paylink) GetByIdAndMerchant(id, merchantId string) (pl *paylink.Paylink, err error) {
	key := fmt.Sprintf(cacheKeyPaylinkMerchant, id, merchantId)
	dbQuery := bson.M{
		"_id":         bson.ObjectIdHex(id),
		"merchant_id": bson.ObjectIdHex(merchantId),
		"deleted":     false,
	}
	return p.getBy(key, dbQuery)
}

func (p Paylink) getBy(key string, dbQuery bson.M) (pl *paylink.Paylink, err error) {
	if err = p.svc.cacher.Get(key, &pl); err == nil {
		pl.IsExpired = pl.GetIsExpired()
		return pl, nil
	}

	err = p.svc.db.Collection(collectionPaylinks).Find(dbQuery).One(&pl)
	if err != nil {
		zap.L().Error(
			pkg.ErrorDatabaseQueryFailed,
			zap.Error(err),
			zap.String(pkg.ErrorDatabaseFieldCollection, collectionPaylinks),
			zap.Any(pkg.ErrorDatabaseFieldQuery, dbQuery),
		)
		return
	}

	pl.IsExpired = pl.GetIsExpired()

	if pl.Deleted == false {
		err = p.updateCaches(pl)
	}
	return
}

func (p Paylink) IncrVisits(id string) (err error) {
	visit := &paylinkVisits{
		PaylinkId: bson.ObjectIdHex(id),
		Date:      time.Now(),
	}
	err = p.svc.db.Collection(collectionPaylinkVisits).Insert(visit)
	if err != nil {
		zap.L().Error(
			pkg.ErrorDatabaseQueryFailed,
			zap.Error(err),
			zap.String(pkg.ErrorDatabaseFieldCollection, collectionPaylinkVisits),
			zap.String(pkg.ErrorDatabaseFieldOperation, pkg.ErrorDatabaseFieldOperationInsert),
			zap.Any(pkg.ErrorDatabaseFieldDocument, visit),
		)
	}

	return
}

func (p *Paylink) GetUrl(id, merchantId, urlMask, utmSource, utmMedium, utmCampaign string) (string, error) {
	pl, err := p.GetByIdAndMerchant(id, merchantId)
	if err != nil {
		return "", err
	}
	if pl.GetIsExpired() {
		return "", errorPaylinkExpired
	}

	if urlMask == "" {
		urlMask = pkg.PaylinkUrlDefaultMask
	}

	urlString := fmt.Sprintf(urlMask, id)

	utmQuery := &utmQueryParams{
		UtmSource:   utmSource,
		UtmMedium:   utmMedium,
		UtmCampaign: utmCampaign,
	}

	q, err := query.Values(utmQuery)
	if err != nil {
		zap.L().Error(
			"Failed to serialize utm query params",
			zap.Error(err),
		)
		return "", err
	}
	encodedQuery := q.Encode()
	if encodedQuery != "" {
		urlString += "?" + encodedQuery
	}

	return u.NormalizeURLString(urlString, u.FlagsUsuallySafeGreedy|u.FlagRemoveDuplicateSlashes)
}

func (p Paylink) Delete(id, merchantId string) error {
	pl, err := p.GetByIdAndMerchant(id, merchantId)
	if err != nil {
		return err
	}

	pl.Deleted = true
	err = p.svc.db.Collection(collectionPaylinks).UpdateId(bson.ObjectIdHex(pl.Id), pl)
	if err != nil {
		zap.L().Error(
			pkg.ErrorDatabaseQueryFailed,
			zap.Error(err),
			zap.String(pkg.ErrorDatabaseFieldCollection, collectionPaylinks),
			zap.String(pkg.ErrorDatabaseFieldOperation, pkg.ErrorDatabaseFieldOperationUpdate),
			zap.Any(pkg.ErrorDatabaseFieldDocument, pl),
		)

		return err
	}

	err = p.updateCaches(pl)

	return nil
}

func (p Paylink) Update(pl *paylink.Paylink) (err error) {

	dbQuery := bson.M{"_id": bson.ObjectIdHex(pl.Id)}

	expiresAt := int64(0)
	if pl.ExpiresAt != nil {
		expiresAt, err := ptypes.Timestamp(pl.ExpiresAt)
		if err != nil {
			zap.L().Error(
				pkg.ErrorTimeConversion,
				zap.Any(pkg.ErrorTimeConversionMethod, "ptypes.Timestamp"),
				zap.Any(pkg.ErrorTimeConversionValue, expiresAt),
				zap.Error(err),
			)
			return err
		}
	}

	updatedAt, err := ptypes.Timestamp(pl.UpdatedAt)
	if err != nil {
		zap.L().Error(
			pkg.ErrorTimeConversion,
			zap.Any(pkg.ErrorTimeConversionMethod, "ptypes.Timestamp"),
			zap.Any(pkg.ErrorTimeConversionValue, expiresAt),
			zap.Error(err),
		)
		return err
	}

	set := bson.M{"$set": bson.M{
		"expires_at":     expiresAt,
		"updated_at":     updatedAt,
		"products":       pl.Products,
		"name":           pl.Name,
		"no_expiry_date": pl.NoExpiryDate,
		"products_type":  pl.ProductsType,
		"is_expired":     pl.GetIsExpired(),
	}}
	err = p.svc.db.Collection(collectionPaylinks).Update(dbQuery, set)
	if err != nil {
		zap.S().Error(
			pkg.ErrorDatabaseQueryFailed,
			zap.Error(err),
			zap.String(pkg.ErrorDatabaseFieldCollection, collectionPaylinks),
			zap.Any(pkg.ErrorDatabaseFieldQuery, dbQuery),
			zap.Any(pkg.ErrorDatabaseFieldSet, set),
		)

		return err
	}

	err = p.updateCaches(pl)

	return
}

func (p Paylink) Insert(pl *paylink.Paylink) (err error) {
	_, err = p.svc.db.Collection(collectionPaylinks).UpsertId(bson.ObjectIdHex(pl.Id), pl)

	if err != nil {
		zap.S().Error(
			pkg.ErrorDatabaseQueryFailed,
			zap.Error(err),
			zap.String(pkg.ErrorDatabaseFieldCollection, collectionPaylinks),
			zap.String(pkg.ErrorDatabaseFieldOperation, pkg.ErrorDatabaseFieldOperationUpsert),
			zap.Any(pkg.ErrorDatabaseFieldDocument, pl),
		)
	}

	err = p.updateCaches(pl)

	return
}

func (p Paylink) GetPaylinkVisits(id string, from, to int64) (n int, err error) {

	matchQuery := bson.M{
		"paylink_id": bson.ObjectIdHex(id),
	}

	if from > 0 || to > 0 {
		date := bson.M{}
		if from > 0 {
			date["$gte"] = time.Unix(from, 0)
		}
		if to > 0 {
			date["$lte"] = time.Unix(to, 0)
		}
		matchQuery["date"] = date
	}

	n, err = p.svc.db.Collection(collectionPaylinkVisits).Find(matchQuery).Count()
	if err != nil {
		zap.L().Error(
			pkg.ErrorDatabaseQueryFailed,
			zap.Error(err),
			zap.String(pkg.ErrorDatabaseFieldCollection, collectionPaylinks),
			zap.String(pkg.ErrorDatabaseFieldOperation, pkg.ErrorDatabaseFieldOperationCount),
			zap.Any(pkg.ErrorDatabaseFieldQuery, matchQuery),
		)
	}
	return
}

func (p Paylink) UpdatePaylinkTotalStat(id, merchantId string) (err error) {
	pl, err := p.GetByIdAndMerchant(id, merchantId)
	if err != nil {
		return err
	}

	visits, err := p.GetPaylinkVisits(id, 0, 0)
	if err == nil {
		pl.Visits = int32(visits)
	}

	stat, err := p.svc.orderView.GetPaylinkStat(id, merchantId, 0, 0)
	if err != nil {
		return err
	}

	pl.TotalTransactions = stat.TotalTransactions
	pl.ReturnsCount = stat.ReturnsCount
	pl.SalesCount = stat.SalesCount
	pl.TransactionsCurrency = stat.TransactionsCurrency
	pl.GrossTotalAmount = stat.GrossTotalAmount
	pl.GrossSalesAmount = stat.GrossSalesAmount
	pl.GrossReturnsAmount = stat.GrossReturnsAmount
	pl.UpdateConversion()

	dbQuery := bson.M{"_id": bson.ObjectIdHex(pl.Id)}

	set := bson.M{"$set": bson.M{
		"visits":                pl.Visits,
		"conversion":            pl.Conversion,
		"total_transactions":    stat.TotalTransactions,
		"sales_count":           stat.SalesCount,
		"returns_count":         stat.ReturnsCount,
		"gross_sales_amount":    stat.GrossSalesAmount,
		"gross_returns_amount":  stat.GrossReturnsAmount,
		"gross_total_amount":    stat.GrossTotalAmount,
		"transactions_currency": stat.TransactionsCurrency,
		"is_expired":            pl.GetIsExpired(),
	}}
	err = p.svc.db.Collection(collectionPaylinks).Update(dbQuery, set)
	if err != nil {
		zap.S().Error(
			pkg.ErrorDatabaseQueryFailed,
			zap.Error(err),
			zap.String(pkg.ErrorDatabaseFieldCollection, collectionPaylinks),
			zap.Any(pkg.ErrorDatabaseFieldQuery, dbQuery),
			zap.Any(pkg.ErrorDatabaseFieldSet, set),
		)

		return err
	}

	err = p.updateCaches(pl)

	pl, err = p.GetById(pl.Id)
	if err != nil {
		return err
	}

	return
}

func (p Paylink) updateCaches(pl *paylink.Paylink) (err error) {
	key1 := fmt.Sprintf(cacheKeyPaylink, pl.Id)
	key2 := fmt.Sprintf(cacheKeyPaylinkMerchant, pl.Id, pl.MerchantId)

	if pl.Deleted {
		err = p.svc.cacher.Delete(key1)
		if err != nil {
			return
		}

		err = p.svc.cacher.Delete(key2)
		return
	}

	err = p.svc.cacher.Set(key1, pl, 0)
	if err != nil {
		zap.L().Error(
			pkg.ErrorCacheQueryFailed,
			zap.Error(err),
			zap.String(pkg.ErrorCacheFieldCmd, "SET"),
			zap.String(pkg.ErrorCacheFieldKey, key1),
			zap.Any(pkg.ErrorCacheFieldData, pl),
		)
		return
	}

	err = p.svc.cacher.Set(key2, pl, 0)
	if err != nil {
		zap.L().Error(
			pkg.ErrorCacheQueryFailed,
			zap.Error(err),
			zap.String(pkg.ErrorCacheFieldCmd, "SET"),
			zap.String(pkg.ErrorCacheFieldKey, key2),
			zap.Any(pkg.ErrorCacheFieldData, pl),
		)
	}
	return
}
