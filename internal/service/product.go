package service

import (
	"context"
	"fmt"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/golang/protobuf/ptypes"
	"github.com/paysuper/paysuper-billing-server/pkg"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/billing"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/grpc"
	"go.uber.org/zap"
)

const (
	cacheProductId    = "product:id:%s"
	collectionProduct = "product"
)

var (
	productErrorUnknown                    = newBillingServerErrorMsg("pd000001", "unknown error with product")
	productErrorNotFound                   = newBillingServerErrorMsg("pd000002", "products with specified ID not found")
	productErrorNotFoundBySku              = newBillingServerErrorMsg("pd000003", "products with specified SKUs not found")
	productErrorCountNotMatch              = newBillingServerErrorMsg("pd000004", "request products count and products in system count not match")
	productErrorAmountNotMatch             = newBillingServerErrorMsg("pd000005", "one or more products amount not match")
	productErrorCurrencyNotMatch           = newBillingServerErrorMsg("pd000006", "one or more products currency not match")
	productErrorIdListEmpty                = newBillingServerErrorMsg("pd000007", "ids list is empty")
	productErrorMerchantNotEqual           = newBillingServerErrorMsg("pd000008", "merchant id is not equal in a product")
	productErrorProjectNotEqual            = newBillingServerErrorMsg("pd000009", "project id is not equal in a product")
	productErrorPriceDefaultCurrency       = newBillingServerErrorMsg("pd000010", "no price in default currency")
	productErrorNameDefaultLanguage        = newBillingServerErrorMsg("pd000011", "no name in default language")
	productErrorDescriptionDefaultLanguage = newBillingServerErrorMsg("pd000012", "no description in default language")
	productErrorUpsert                     = newBillingServerErrorMsg("pd000013", "query to insert/update product failed")
	productErrorDelete                     = newBillingServerErrorMsg("pd000014", "query to delete product failed")
	productErrorProjectAndSkuAlreadyExists = newBillingServerErrorMsg("pd000015", "pair projectId+Sku already exists")
	productErrorListPrices                 = newBillingServerErrorMsg("pd000017", "list of prices is empty")
	productErrorPricesUpdate               = newBillingServerErrorMsg("pd000017", "query to update product prices is failed")
	productSkuMismatch                     = newBillingServerErrorMsg("pd000008", "sku mismatch")
)

func (s *Service) CreateOrUpdateProduct(ctx context.Context, req *grpc.Product, res *grpc.Product) error {
	var (
		err     error
		product = &grpc.Product{}
		isNew   = req.Id == ""
		now     = ptypes.TimestampNow()
	)

	if isNew {
		req.Id = bson.NewObjectId().Hex()
		req.CreatedAt = now
	} else {
		product, err = s.productService.GetById(req.Id)
		if err != nil {
			zap.S().Errorf("Product that requested to change is not found", "err", err.Error(), "data", req)
			return productErrorNotFound
		}

		if req.Sku != "" && req.Sku != product.Sku {
			zap.S().Errorf("SKU mismatch", "data", req)
			return productSkuMismatch
		}

		if req.MerchantId != product.MerchantId {
			zap.S().Errorf("MerchantId mismatch", "data", req)
			return productErrorMerchantNotEqual
		}

		if req.ProjectId != product.ProjectId {
			zap.S().Errorf("ProjectId mismatch", "data", req)
			return productErrorProjectNotEqual
		}

		req.CreatedAt = product.CreatedAt
	}
	req.UpdatedAt = now
	req.Deleted = false

	if !req.IsPricesContainDefaultCurrency() {
		zap.S().Errorf(productErrorPriceDefaultCurrency.Message, "data", req)
		return productErrorPriceDefaultCurrency
	}

	if _, err := req.GetLocalizedName(DefaultLanguage); err != nil {
		zap.S().Errorf("No name in default language", "data", req)
		return productErrorNameDefaultLanguage
	}

	if _, err := req.GetLocalizedDescription(DefaultLanguage); err != nil {
		zap.S().Errorf("No description in default language", "data", req)
		return productErrorDescriptionDefaultLanguage
	}

	count, err := s.productService.CountByProjectSku(req.ProjectId, req.Sku)

	if err != nil {
		zap.S().Errorf("Query to find duplicates failed", "err", err.Error(), "data", req)
		return productErrorUnknown
	}

	allowed := 1
	if isNew {
		allowed = 0
	}
	if count > allowed {
		zap.S().Errorf("Pair projectId+Sku already exists", "data", req)
		return productErrorProjectAndSkuAlreadyExists
	}

	if err = s.productService.Upsert(req); err != nil {
		zap.S().Errorf("Query to create/update product failed", "err", err.Error(), "data", req)
		return productErrorUpsert
	}

	res.Id = req.Id
	res.Object = req.Object
	res.Type = req.Type
	res.Sku = req.Sku
	res.Name = req.Name
	res.DefaultCurrency = req.DefaultCurrency
	res.Enabled = req.Enabled
	res.Prices = req.Prices
	res.Description = req.Description
	res.LongDescription = req.LongDescription
	res.Images = req.Images
	res.Url = req.Url
	res.Metadata = req.Metadata
	res.CreatedAt = req.CreatedAt
	res.UpdatedAt = req.UpdatedAt
	res.Deleted = req.Deleted
	res.MerchantId = req.MerchantId
	res.ProjectId = req.ProjectId
	res.Pricing = req.Pricing

	return nil
}

func (s *Service) GetProductsForOrder(ctx context.Context, req *grpc.GetProductsForOrderRequest, res *grpc.ListProductsResponse) error {
	if len(req.Ids) == 0 {
		zap.S().Errorf("Ids list is empty", "data", req)
		return productErrorIdListEmpty
	}

	var found []*grpc.Product
	for _, id := range req.Ids {
		p, err := s.productService.GetById(id)

		if err != nil {
			zap.S().Errorf("Unable to get product", "err", err.Error(), "req", req)
			continue
		}

		if p.Enabled != true || p.ProjectId != req.ProjectId {
			continue
		}

		found = append(found, p)
	}

	res.Limit = int32(len(found))
	res.Offset = 0
	res.Total = res.Limit
	res.Products = found
	return nil
}

func (s *Service) ListProducts(ctx context.Context, req *grpc.ListProductsRequest, res *grpc.ListProductsResponse) error {
	res.Total, res.Products = s.productService.List(
		req.MerchantId,
		req.ProjectId,
		req.Sku,
		req.Name,
		int(req.Offset),
		int(req.Limit),
		req.Enable,
	)

	res.Limit = req.Limit
	res.Offset = req.Offset

	return nil
}

func (s *Service) GetProduct(
	ctx context.Context,
	req *grpc.RequestProduct,
	rsp *grpc.GetProductResponse,
) error {
	product, err := s.productService.GetById(req.Id)

	if err != nil {
		zap.L().Error(
			"Unable to get product",
			zap.Error(err),
			zap.Any(pkg.LogFieldRequest, req),
		)

		rsp.Status = pkg.ResponseStatusNotFound
		rsp.Message = productErrorNotFound

		return nil
	}

	if req.MerchantId != product.MerchantId {
		zap.L().Error(
			"Merchant id mismatch",
			zap.Any("product", product),
			zap.Any(pkg.LogFieldRequest, req),
		)

		rsp.Status = pkg.ResponseStatusBadData
		rsp.Message = productErrorMerchantNotEqual

		return nil
	}

	rsp.Status = pkg.ResponseStatusOk
	rsp.Item = product

	return nil
}

func (s *Service) DeleteProduct(ctx context.Context, req *grpc.RequestProduct, res *grpc.EmptyResponse) error {
	product, err := s.productService.GetById(req.Id)

	if err != nil {
		zap.S().Errorf("Unable to get product", "err", err.Error(), "req", req)
		return productErrorNotFound
	}

	if req.MerchantId != product.MerchantId {
		zap.S().Errorf("MerchantId mismatch", "product", product, "data", req)
		return productErrorMerchantNotEqual
	}

	product.Deleted = true
	product.UpdatedAt = ptypes.TimestampNow()

	err = s.productService.Upsert(product)

	if err != nil {
		zap.S().Errorf("Query to delete product failed", "err", err.Error(), "data", req)
		return productErrorDelete
	}

	return nil
}

func (s *Service) GetProductPrices(ctx context.Context, req *grpc.RequestProduct, res *grpc.ProductPricesResponse) error {
	product, err := s.productService.GetById(req.Id)

	if err != nil {
		zap.S().Errorf("Unable to get product", "err", err.Error(), "req", req)
		return productErrorNotFound
	}

	res.ProductPrice = product.Prices

	return nil
}

func (s *Service) UpdateProductPrices(ctx context.Context, req *grpc.UpdateProductPricesRequest, res *grpc.ResponseError) error {
	if len(req.Prices) == 0 {
		zap.S().Errorf("List of product prices is empty", "data", req)
		return productErrorListPrices
	}

	product, err := s.productService.GetById(req.ProductId)

	if err != nil {
		zap.S().Errorf("Unable to get product", "err", err.Error(), "req", req)
		return productErrorNotFound
	}

	product.Prices = req.Prices

	if !product.IsPricesContainDefaultCurrency() {
		zap.S().Errorf(productErrorPriceDefaultCurrency.Message, "data", req)
		return productErrorPriceDefaultCurrency
	}

	if err := s.productService.Upsert(product); err != nil {
		zap.S().Errorf("Query to create/update product failed", "err", err.Error(), "data", req)
		return productErrorPricesUpdate
	}

	return nil
}

func (s *Service) getProductsCountByProject(projectId string) int32 {
	query := bson.M{"project_id": bson.ObjectIdHex(projectId), "deleted": false}
	count, err := s.db.Collection(collectionProduct).Find(query).Count()

	if err != nil {
		zap.S().Errorf("Query to get project products count failed", "err", err.Error(), "query", query)
	}

	return int32(count)
}

func (s *Service) processTokenProducts(req *grpc.TokenRequest) ([]string, error) {
	var sku []string
	var products []*grpc.Product
	skuItemsMap := make(map[string]*billing.TokenSettingsItem, len(req.Settings.Items))

	for _, v := range req.Settings.Items {
		sku = append(sku, v.Sku)
		skuItemsMap[v.Sku] = v
	}

	query := bson.M{
		"project_id": bson.ObjectIdHex(req.Settings.ProjectId),
		"sku":        bson.M{"$in": sku},
		"deleted":    false,
	}
	err := s.db.Collection(collectionProduct).Find(query).All(&products)

	if err != nil && err != mgo.ErrNotFound {
		zap.S().Errorf("Query to find project products failed", "err", err.Error(), "query", query)
		return nil, productErrorUnknown
	}

	if len(products) <= 0 {
		return nil, productErrorNotFoundBySku
	}

	if len(sku) != len(products) {
		return nil, productErrorCountNotMatch
	}

	var productsIds []string

	for _, v := range products {
		item, _ := skuItemsMap[v.Sku]

		matchAmount := false
		matchCurrency := false

		for _, v1 := range v.Prices {
			if item.Amount != v1.Amount && item.Currency != v1.Currency {
				continue
			}

			if item.Amount == v1.Amount {
				matchAmount = true
			}

			if item.Currency == v1.Currency {
				matchCurrency = true
			}
		}

		if matchAmount == false {
			return nil, productErrorAmountNotMatch
		}

		if matchCurrency == false && v.DefaultCurrency != item.Currency {
			return nil, productErrorCurrencyNotMatch
		}

		productsIds = append(productsIds, v.Id)
	}

	return productsIds, nil
}

type ProductServiceInterface interface {
	Upsert(product *grpc.Product) error
	GetById(string) (*grpc.Product, error)
	CountByProjectSku(string, string) (int, error)
	List(string, string, string, string, int, int, int32) (int32, []*grpc.Product)
}

func newProductService(svc *Service) *Product {
	s := &Product{svc: svc}
	return s
}

func (h *Product) Upsert(p *grpc.Product) error {
	_, err := h.svc.db.Collection(collectionProduct).UpsertId(bson.ObjectIdHex(p.Id), p)
	if err != nil {
		return err
	}

	if err := h.updateCache(p); err != nil {
		return err
	}

	return nil
}

func (h *Product) GetById(id string) (*grpc.Product, error) {
	var c grpc.Product
	key := fmt.Sprintf(cacheProductId, id)

	if err := h.svc.cacher.Get(key, c); err == nil {
		return &c, nil
	}

	err := h.svc.db.Collection(collectionProduct).Find(bson.M{"_id": bson.ObjectIdHex(id), "deleted": false}).One(&c)

	if err != nil {
		return nil, fmt.Errorf(errorNotFound, collectionProduct)
	}

	if err := h.svc.cacher.Set(key, c, 0); err != nil {
		zap.S().Errorf("Unable to set cache", "err", err.Error(), "key", key, "data", c)
	}

	return &c, nil
}

func (h *Product) CountByProjectSku(projectId string, sku string) (int, error) {
	query := bson.M{"project_id": bson.ObjectIdHex(projectId), "sku": sku, "deleted": false}
	count, err := h.svc.db.Collection(collectionProduct).Find(query).Count()

	if err != nil {
		zap.S().Errorf("Query to find duplicates failed", "err", err.Error(), "projectId", projectId)
		return 0, productErrorUnknown
	}

	return count, nil
}

func (h *Product) List(
	merchantId string,
	projectId string,
	sku string,
	name string,
	offset int,
	limit int,
	enable int32,
) (int32, []*grpc.Product) {
	query := bson.M{"merchant_id": bson.ObjectIdHex(merchantId), "deleted": false}

	if projectId != "" {
		query["project_id"] = bson.ObjectIdHex(projectId)
	}

	if sku != "" {
		query["sku"] = bson.RegEx{Pattern: sku, Options: "i"}
	}
	if name != "" {
		query["name"] = bson.M{"$elemMatch": bson.M{"value": bson.RegEx{Pattern: name, Options: "i"}}}
	}

	if enable > 0 {
		if enable == 1 {
			query["enabled"] = false
		} else {
			query["enabled"] = true
		}
	}

	count, err := h.svc.db.Collection(collectionProduct).Find(query).Count()

	if err != nil {
		zap.L().Error(
			pkg.ErrorDatabaseQueryFailed,
			zap.Error(err),
			zap.String(pkg.ErrorDatabaseFieldCollection, collectionOrderView),
			zap.Any(pkg.ErrorDatabaseFieldQuery, query),
		)
		return 0, nil
	}

	if count == 0 || offset > count {
		zap.L().Error(
			"total is empty or less then total",
			zap.Error(err),
			zap.Int(pkg.ErrorDatabaseFieldLimit, limit),
			zap.Int(pkg.ErrorDatabaseFieldOffset, offset),
		)
		return 0, nil
	}

	var list []*grpc.Product
	err = h.svc.db.Collection(collectionProduct).Find(query).Skip(offset).Limit(limit).All(&list)

	if err != nil {
		zap.L().Error(
			pkg.ErrorDatabaseQueryFailed,
			zap.Error(err),
			zap.String(pkg.ErrorDatabaseFieldCollection, collectionOrderView),
			zap.Any(pkg.ErrorDatabaseFieldQuery, query),
		)
		return 0, nil
	}

	return int32(count), list
}

func (h *Product) updateCache(p *grpc.Product) error {
	if err := h.svc.cacher.Set(fmt.Sprintf(cacheProductId, p.Id), p, 0); err != nil {
		return err
	}

	return nil
}
