package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/globalsign/mgo/bson"
	"github.com/golang/protobuf/ptypes"
	our "github.com/paysuper/paysuper-billing-server/pkg"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/billing"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/grpc"
	"github.com/paysuper/paysuper-currencies/pkg"
	"github.com/paysuper/paysuper-currencies/pkg/proto/currencies"
	"go.uber.org/zap"
	"math"
	"strconv"
)

const (
	cachePriceGroupId     = "price_group:id:%s"
	cachePriceGroupAll    = "price_group:all"
	cachePriceGroupRegion = "price_group:region:%s"

	collectionPriceGroup = "price_group"
	collectionPriceTable = "price_table"

	defaultRecommendedCurrency = "USD"
)

var (
	priceGroupErrorNotFound = newBillingServerErrorMsg("pg000001", "price group not found")
)

func (s *Service) GetPriceGroup(
	ctx context.Context,
	req *billing.GetPriceGroupRequest,
	res *billing.PriceGroup,
) error {
	pg, err := s.priceGroup.GetById(req.Id)
	if err != nil {
		return err
	}

	res.Id = pg.Id
	res.Currency = pg.Currency
	res.Region = pg.Region
	res.InflationRate = pg.InflationRate
	res.Fraction = pg.Fraction
	res.CreatedAt = pg.CreatedAt
	res.UpdatedAt = pg.UpdatedAt

	return nil
}

func (s *Service) UpdatePriceGroup(
	ctx context.Context,
	req *billing.PriceGroup,
	res *billing.PriceGroup,
) error {

	pg := &billing.PriceGroup{
		Currency:      req.Currency,
		Region:        req.Region,
		InflationRate: req.InflationRate,
		Fraction:      req.Fraction,
		IsActive:      req.IsActive,
		UpdatedAt:     ptypes.TimestampNow(),
	}

	var err error

	if req.Id != "" {
		data, err := s.priceGroup.GetById(req.Id)
		if err != nil {
			return err
		}
		pg.Id = data.Id
		pg.CreatedAt = data.CreatedAt
		pg.UpdatedAt = ptypes.TimestampNow()
		err = s.priceGroup.Update(pg)
	} else {
		pg.Id = bson.NewObjectId().Hex()
		pg.CreatedAt = ptypes.TimestampNow()
		err = s.priceGroup.Insert(pg)
	}

	if err != nil {
		zap.S().Errorf("create/update price group failed", "err", err.Error(), "data", req)
		return err
	}

	res.Id = pg.Id
	res.Currency = pg.Currency
	res.Region = pg.Region
	res.InflationRate = pg.InflationRate
	res.Fraction = pg.Fraction
	res.CreatedAt = pg.CreatedAt
	res.UpdatedAt = pg.UpdatedAt

	return nil
}

func (s *Service) GetPriceGroupByCountry(
	ctx context.Context,
	req *grpc.PriceGroupByCountryRequest,
	res *billing.PriceGroup,
) error {
	country, err := s.country.GetByIsoCodeA2(req.Country)

	if err != nil {
		zap.S().Errorw("Country not found", "req", req)
		return err
	}

	group, err := s.priceGroup.GetById(country.PriceGroupId)

	if err != nil {
		zap.S().Errorw("Price group not found", "error", err, "price_group_id", country.PriceGroupId)
		return err
	}

	*res = *group

	return nil
}

func (s *Service) GetPriceGroupCurrencies(
	ctx context.Context,
	req *grpc.EmptyRequest,
	res *grpc.PriceGroupCurrenciesResponse,
) error {
	regions, err := s.priceGroup.GetAll()

	if err != nil {
		zap.S().Errorw("Unable to load price groups", "error", err)
		return err
	}

	countries, err := s.country.GetAll()

	if err != nil {
		zap.S().Errorw("Unable to get countries", "error", err)
		return err
	}

	res.Region = s.priceGroup.MakeCurrencyList(regions, countries)

	return nil
}

func (s *Service) GetPriceGroupCurrencyByRegion(
	ctx context.Context,
	req *grpc.PriceGroupByRegionRequest,
	res *grpc.PriceGroupCurrenciesResponse,
) error {
	region, err := s.priceGroup.GetByRegion(req.Region)

	if err != nil {
		zap.S().Errorw("Price group not found", "req", req)
		return err
	}

	countries, err := s.country.GetAll()

	if err != nil {
		zap.S().Errorw("Unable to get countries", "error", err)
		return err
	}

	regions := []*billing.PriceGroup{region}
	list := s.priceGroup.MakeCurrencyList(regions, countries)
	res.Region = list

	return nil
}

func (s *Service) GetRecommendedPriceByPriceGroup(
	ctx context.Context,
	req *grpc.RecommendedPriceRequest,
	res *grpc.RecommendedPriceResponse,
) error {
	regions, err := s.priceGroup.GetAll()

	if err != nil {
		zap.S().Errorw("Unable to get price regions", "err", err, "req", req)
		return err
	}

	priceTable, err := s.findPriceTableByAmount(req.Amount)

	if err != nil {
		zap.S().Errorw("Unable to get price table for amount", "err", err, "req", req)
		return err
	}

	for _, region := range regions {
		price, err := s.priceGroup.GetRecommendedPriceForRegion(priceTable, region, req.Amount)

		if err != nil {
			zap.S().Errorw("Unable to get recommended price for region", "err", err, "region", region)
			return err
		}

		res.RecommendedPrice = append(res.RecommendedPrice, &billing.RecommendedPrice{
			Amount:   price,
			Region:   region.Region,
			Currency: region.Currency,
		})
	}

	return nil
}

func (s *Service) GetRecommendedPriceByConversion(
	ctx context.Context,
	req *grpc.RecommendedPriceRequest,
	res *grpc.RecommendedPriceResponse,
) error {
	regions, err := s.priceGroup.GetAll()

	if err != nil {
		zap.S().Errorw("Unable to get price regions", "err", err, "req", req)
		return err
	}

	for _, region := range regions {
		amount, err := s.getPriceInCurrencyByAmount(region.Currency, req.Amount)

		if err != nil {
			zap.S().Errorw("Unable to get amount for region", "err", err, "region", region)
			return err
		}

		res.RecommendedPrice = append(res.RecommendedPrice, &billing.RecommendedPrice{
			Amount:   s.priceGroup.CalculatePriceWithFraction(region.Fraction, amount),
			Region:   region.Region,
			Currency: region.Currency,
		})
	}

	return nil
}

func (s *Service) GetPriceGroupByRegion(ctx context.Context, req *grpc.GetPriceGroupByRegionRequest, rsp *grpc.GetPriceGroupByRegionResponse) error {
	group, err := s.priceGroup.GetByRegion(req.Region)
	rsp.Status = our.ResponseStatusOk

	if err != nil {
		zap.L().Error(
			our.ErrorGrpcServiceCallFailed,
			zap.Error(err),
			zap.Any("region", req.Region),
		)
		rsp.Status = our.ResponseStatusBadData
		rsp.Message = priceGroupErrorNotFound
		return nil
	}

	rsp.Group = group

	return nil
}

func (s *Service) findPriceTableByAmount(amount float64) (*billing.PriceTable, error) {
	priceTable, _ := s.priceTable.GetByAmount(amount)

	if priceTable != nil {
		return priceTable, nil
	}

	priceTable, err := s.priceTable.GetLatest()

	if err != nil {
		return nil, err
	}

	return s.priceTable.InterpolateByAmount(priceTable, amount), nil
}

func (s *Service) getPriceInCurrencyByAmount(currency string, amount float64) (float64, error) {
	req := &currencies.ExchangeCurrencyCurrentCommonRequest{
		Amount:   amount,
		From:     "USD",
		To:       currency,
		RateType: pkg.RateTypeOxr,
	}
	res, err := s.curService.ExchangeCurrencyCurrentCommon(context.Background(), req)

	if err != nil {
		return 0, err
	}

	return res.ExchangedAmount, nil
}

type PriceGroupServiceInterface interface {
	Insert(*billing.PriceGroup) error
	MultipleInsert([]*billing.PriceGroup) error
	Update(*billing.PriceGroup) error
	GetById(string) (*billing.PriceGroup, error)
	GetByRegion(string) (*billing.PriceGroup, error)
	GetAll() ([]*billing.PriceGroup, error)
	MakeCurrencyList([]*billing.PriceGroup, *billing.CountriesList) []*grpc.PriceGroupRegions
	GetRecommendedPriceForRegion(*billing.PriceTable, *billing.PriceGroup, float64) (float64, error)
	CalculatePriceWithFraction(float64, float64) float64
}

func newPriceGroupService(svc *Service) *PriceGroup {
	s := &PriceGroup{svc: svc}
	return s
}

func (h *PriceGroup) Insert(pg *billing.PriceGroup) error {
	if err := h.svc.db.Collection(collectionPriceGroup).Insert(pg); err != nil {
		return err
	}

	if err := h.updateCache(pg); err != nil {
		return err
	}

	return nil
}

func (h PriceGroup) MultipleInsert(pg []*billing.PriceGroup) error {
	c := make([]interface{}, len(pg))
	for i, v := range pg {
		c[i] = v
	}

	if err := h.svc.db.Collection(collectionPriceGroup).Insert(c...); err != nil {
		return err
	}

	return nil
}

func (h PriceGroup) Update(pg *billing.PriceGroup) error {
	if err := h.svc.db.Collection(collectionPriceGroup).UpdateId(bson.ObjectIdHex(pg.Id), pg); err != nil {
		return err
	}

	if err := h.updateCache(pg); err != nil {
		return err
	}

	return nil
}

func (h PriceGroup) GetById(id string) (*billing.PriceGroup, error) {
	var c billing.PriceGroup
	key := fmt.Sprintf(cachePriceGroupId, id)
	err := h.svc.cacher.Get(key, c)

	if err == nil {
		return &c, nil
	}

	query := bson.M{"_id": bson.ObjectIdHex(id), "is_active": true}
	err = h.svc.db.Collection(collectionPriceGroup).Find(query).One(&c)

	if err != nil {
		zap.L().Error(
			our.ErrorDatabaseQueryFailed,
			zap.Error(err),
			zap.String(our.ErrorDatabaseFieldCollection, collectionPriceGroup),
			zap.Any(our.ErrorDatabaseFieldQuery, query),
		)
		return nil, fmt.Errorf(errorNotFound, collectionPriceGroup)
	}

	err = h.svc.cacher.Set(key, c, 0)

	if err != nil {
		zap.L().Error(
			our.ErrorCacheQueryFailed,
			zap.Error(err),
			zap.String(our.ErrorCacheFieldCmd, "SET"),
			zap.String(our.ErrorCacheFieldKey, key),
			zap.Any(our.ErrorDatabaseFieldQuery, c),
		)
	}

	return &c, nil
}

func (h PriceGroup) GetByRegion(region string) (*billing.PriceGroup, error) {
	var c billing.PriceGroup
	key := fmt.Sprintf(cachePriceGroupRegion, region)

	if err := h.svc.cacher.Get(key, c); err == nil {
		return &c, nil
	}

	err := h.svc.db.Collection(collectionPriceGroup).Find(bson.M{"region": region, "is_active": true}).One(&c)

	if err != nil {
		return nil, fmt.Errorf(errorNotFound, collectionPriceGroup)
	}

	if err := h.svc.cacher.Set(key, c, 0); err != nil {
		zap.S().Errorf("Unable to set cache", "err", err.Error(), "key", key, "data", c)
	}

	return &c, nil
}

func (h PriceGroup) GetAll() ([]*billing.PriceGroup, error) {
	var c []*billing.PriceGroup

	if err := h.svc.cacher.Get(cachePriceGroupAll, c); err == nil {
		return c, nil
	}

	err := h.svc.db.Collection(collectionPriceGroup).Find(bson.M{"is_active": true}).All(&c)
	if err != nil {
		return nil, err
	}

	err = h.svc.cacher.Set(cachePriceGroupAll, c, 0)
	if err != nil {
		zap.S().Errorf("Unable to set cache", "err", err.Error(), "key", cachePriceGroupAll, "data", c)
	}

	return c, nil
}

func (h *PriceGroup) updateCache(pg *billing.PriceGroup) error {
	if err := h.svc.cacher.Set(fmt.Sprintf(cachePriceGroupId, pg.Id), pg, 0); err != nil {
		return err
	}

	if err := h.svc.cacher.Set(fmt.Sprintf(cachePriceGroupRegion, pg.Region), pg, 0); err != nil {
		return err
	}

	if err := h.svc.cacher.Delete(cachePriceGroupAll); err != nil {
		return err
	}

	return nil
}

func (h *PriceGroup) MakeCurrencyList(regions []*billing.PriceGroup, countries *billing.CountriesList) []*grpc.PriceGroupRegions {
	curr := map[string]*grpc.PriceGroupRegions{}

	for _, region := range regions {
		if region.Region == defaultRecommendedCurrency {
			continue
		}

		if curr[region.Currency] == nil {
			curr[region.Currency] = &grpc.PriceGroupRegions{
				Currency: region.Currency,
				Regions:  []*grpc.PriceGroupRegion{},
			}
		}

		var c []string
		for _, country := range countries.Countries {
			if country.PriceGroupId == region.Id {
				c = append(c, country.IsoCodeA2)
			}
		}

		curr[region.Currency].Regions = append(
			curr[region.Currency].Regions,
			&grpc.PriceGroupRegion{
				Region:  region.Region,
				Country: c,
			},
		)
	}

	var list []*grpc.PriceGroupRegions
	for _, entry := range curr {
		list = append(list, entry)
	}

	return list
}

func (h *PriceGroup) GetRecommendedPriceForRegion(pt *billing.PriceTable, region *billing.PriceGroup, amount float64) (float64, error) {
	local, ok := pt.Currencies[region.Currency]

	if ok != true {
		return 0, errors.New("currency in price table not found")
	}

	ratio := (pt.To - amount) / (pt.To - pt.From)

	if ratio == 0 {
		ratio = 1
	} else if ratio == 1 {
		ratio = 0
	}

	price := local.From + (local.To-local.From)*ratio
	priceFrac := h.CalculatePriceWithFraction(region.Fraction, price)

	return priceFrac, nil
}

func (h *PriceGroup) CalculatePriceWithFraction(fraction float64, price float64) float64 {
	i, frac := math.Modf(price)
	frac, _ = strconv.ParseFloat(fmt.Sprintf("%.2f", frac), 64)

	if fraction == 0.05 || fraction == 0.5 {
		divider := math.Ceil(frac / fraction)
		s, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", divider*fraction), 64)
		p, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", i+s), 64)

		return p
	}

	if fraction == 0.95 {
		if frac > 0.95 {
			i = i + 1
		}

		return i + 0.95
	}

	if fraction == 0.09 {
		i2, _ := math.Modf(frac * 10)
		p, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", i+(i2/10)+fraction), 64)
		return p
	}

	if fraction == 0 {
		i = math.Ceil(price)
	}

	return i
}
