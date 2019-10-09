package service

import (
	"context"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/paysuper/paysuper-billing-server/internal/config"
	"github.com/paysuper/paysuper-billing-server/internal/mocks"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/billing"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/grpc"
	mongodb "github.com/paysuper/paysuper-database-mongo"
	reportingMocks "github.com/paysuper/paysuper-reporter/pkg/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"gopkg.in/ProtocolONE/rabbitmq.v1/pkg"
	"gopkg.in/mgo.v2/bson"
	"testing"
)

type KeyProductTestSuite struct {
	suite.Suite
	service *Service
	log     *zap.Logger
	cache   CacheInterface

	project    *billing.Project
	pmBankCard *billing.PaymentMethod
}

func Test_KeyProduct(t *testing.T) {
	suite.Run(t, new(KeyProductTestSuite))
}

func (suite *KeyProductTestSuite) SetupTest() {
	cfg, err := config.NewConfig()
	assert.NoError(suite.T(), err, "Config load failed")

	cfg.AccountingCurrency = "RUB"

	db, err := mongodb.NewDatabase()
	if err != nil {
		suite.FailNow("Database connection failed", "%v", err)
	}

	m, err := migrate.New(
		"file://../../migrations/tests/keys",
		cfg.MongoDsn)
	assert.NoError(suite.T(), err, "Migrate init failed")
	if err != nil {
		suite.FailNow("Migrations failed", "%v", err)
	}

	err = m.Up()
	if err != nil && err.Error() != "no change" {
		suite.FailNow("Migrations failed", "%v", err)
	}

	pgRub := &billing.PriceGroup{
		Id:       bson.NewObjectId().Hex(),
		Region:   "RUB",
		Currency: "RUB",
		IsActive: true,
	}
	pgUsd := &billing.PriceGroup{
		Id:       bson.NewObjectId().Hex(),
		Region:   "USD",
		Currency: "USD",
		IsActive: true,
	}
	pgEur := &billing.PriceGroup{
		Id:       bson.NewObjectId().Hex(),
		Region:   "EUR",
		Currency: "EUR",
		IsActive: true,
	}
	if err != nil {
		suite.FailNow("Insert currency test data failed", "%v", err)
	}

	suite.log, err = zap.NewProduction()
	assert.NoError(suite.T(), err, "Logger initialization failed")

	broker, err := rabbitmq.NewBroker(cfg.BrokerAddress)
	assert.NoError(suite.T(), err, "Creating RabbitMQ publisher failed")

	redisdb := mocks.NewTestRedis()
	suite.cache = NewCacheRedis(redisdb)
	suite.service = NewBillingService(db, cfg, mocks.NewGeoIpServiceTestOk(), mocks.NewRepositoryServiceOk(), mocks.NewTaxServiceOkMock(), broker, nil, suite.cache, mocks.NewCurrencyServiceMockOk(), mocks.NewDocumentSignerMockOk(), &reportingMocks.ReporterService{}, mocks.NewFormatterOK())

	if err := suite.service.Init(); err != nil {
		suite.FailNow("Billing service initialization failed", "%v", err)
	}

	suite.NoError(suite.service.merchant.Insert(&billing.Merchant{Id: merchantId}))

	pgs := []*billing.PriceGroup{pgRub, pgUsd, pgEur}
	if err := suite.service.priceGroup.MultipleInsert(pgs); err != nil {
		suite.FailNow("Insert price group test data failed", "%v", err)
	}
}

func (suite *KeyProductTestSuite) TearDownTest() {
	if err := suite.service.db.Drop(); err != nil {
		suite.FailNow("Database deletion failed", "%v", err)
	}

	suite.service.db.Close()
}

func (suite *KeyProductTestSuite) Test_GetKeyProductInfo() {
	shouldBe := require.New(suite.T())

	req := &grpc.CreateOrUpdateKeyProductRequest{
		Object:          "product",
		Sku:             "ru_double_yeti",
		Name:            map[string]string{"en": initialName},
		DefaultCurrency: "USD",
		Description:     map[string]string{"en": "blah-blah-blah"},
		LongDescription: map[string]string{"en": "Super game steam keys"},
		Url:             "http://test.ru/dffdsfsfs",
		Cover: &billing.ImageCollection{
			UseOneForAll: false,
			Images: &billing.LocalizedUrl{
				En: "/home/image.jpg",
			},
		},
		MerchantId: merchantId,
		ProjectId:  projectId,
		Platforms: []*grpc.PlatformPrice{
			{
				Id: "steam",
				Prices: []*billing.ProductPrice{
					{Region: "USD", Currency: "USD", Amount: 10},
					{Region: "EUR", Currency: "EUR", Amount: 20},
				},
			},
		},
		Metadata: map[string]string{
			"SomeKey": "SomeValue",
		},
	}
	response := grpc.KeyProductResponse{}
	err := suite.service.CreateOrUpdateKeyProduct(context.TODO(), req, &response)
	shouldBe.Nil(err)
	shouldBe.Nil(response.Message)

	res := grpc.GetKeyProductInfoResponse{}
	err = suite.service.GetKeyProductInfo(context.TODO(), &grpc.GetKeyProductInfoRequest{Currency: "USD", KeyProductId: response.Product.Id, Language: "en"}, &res)
	shouldBe.Nil(err)
	shouldBe.NotNil(res.Message)
	shouldBe.EqualValues(400, res.Status)

	publishRsp := &grpc.KeyProductResponse{}
	err = suite.service.PublishKeyProduct(context.TODO(), &grpc.PublishKeyProductRequest{MerchantId: merchantId, KeyProductId: response.Product.Id}, publishRsp)
	shouldBe.Nil(err)
	shouldBe.EqualValues(200, publishRsp.Status)

	res = grpc.GetKeyProductInfoResponse{}
	err = suite.service.GetKeyProductInfo(context.TODO(), &grpc.GetKeyProductInfoRequest{Currency: "USD", KeyProductId: response.Product.Id, Language: "en"}, &res)
	shouldBe.Nil(err)
	shouldBe.Nil(res.Message)
	shouldBe.NotNil(res.KeyProduct)
	shouldBe.Equal(response.Product.Id, res.KeyProduct.Id)
	shouldBe.Equal(initialName, res.KeyProduct.Name)
	shouldBe.Equal("blah-blah-blah", res.KeyProduct.Description)
	shouldBe.Equal(1, len(res.KeyProduct.Platforms))
	shouldBe.Equal("steam", res.KeyProduct.Platforms[0].Id)
	shouldBe.EqualValues(10, res.KeyProduct.Platforms[0].Price.Amount)
	shouldBe.Equal("USD", res.KeyProduct.Platforms[0].Price.Currency)
	shouldBe.False(res.KeyProduct.Platforms[0].Price.IsFallback)

	res = grpc.GetKeyProductInfoResponse{}
	err = suite.service.GetKeyProductInfo(context.TODO(), &grpc.GetKeyProductInfoRequest{Currency: "EUR", KeyProductId: response.Product.Id, Language: "ru"}, &res)
	shouldBe.Nil(err)
	shouldBe.Nil(res.Message)
	shouldBe.NotNil(res.KeyProduct)
	shouldBe.Equal(response.Product.Id, res.KeyProduct.Id)
	shouldBe.Equal(initialName, res.KeyProduct.Name)
	shouldBe.Equal("blah-blah-blah", res.KeyProduct.Description)
	shouldBe.Equal(1, len(res.KeyProduct.Platforms))
	shouldBe.Equal("steam", res.KeyProduct.Platforms[0].Id)
	shouldBe.EqualValues(20, res.KeyProduct.Platforms[0].Price.Amount)
	shouldBe.Equal("EUR", res.KeyProduct.Platforms[0].Price.Currency)
	shouldBe.False(res.KeyProduct.Platforms[0].Price.IsFallback)

	res = grpc.GetKeyProductInfoResponse{}
	err = suite.service.GetKeyProductInfo(context.TODO(), &grpc.GetKeyProductInfoRequest{Currency: "UNK", KeyProductId: response.Product.Id, Language: "ru"}, &res)
	shouldBe.Nil(err)
	shouldBe.Nil(res.Message)
	shouldBe.NotNil(res.KeyProduct)
	shouldBe.Equal(response.Product.Id, res.KeyProduct.Id)
	shouldBe.Equal(initialName, res.KeyProduct.Name)
	shouldBe.Equal("blah-blah-blah", res.KeyProduct.Description)
	shouldBe.Equal(1, len(res.KeyProduct.Platforms))
	shouldBe.Equal("steam", res.KeyProduct.Platforms[0].Id)
	shouldBe.EqualValues(10, res.KeyProduct.Platforms[0].Price.Amount)
	shouldBe.Equal("USD", res.KeyProduct.Platforms[0].Price.Currency)
	shouldBe.True(res.KeyProduct.Platforms[0].Price.IsFallback)

	res = grpc.GetKeyProductInfoResponse{}
	err = suite.service.GetKeyProductInfo(context.TODO(), &grpc.GetKeyProductInfoRequest{Currency: "RUB", KeyProductId: response.Product.Id, Language: "ru"}, &res)
	shouldBe.Nil(err)
	shouldBe.Nil(res.Message)
	shouldBe.NotNil(res.KeyProduct)
	shouldBe.Equal(response.Product.Id, res.KeyProduct.Id)
	shouldBe.Equal(initialName, res.KeyProduct.Name)
	shouldBe.Equal("blah-blah-blah", res.KeyProduct.Description)
	shouldBe.Equal(1, len(res.KeyProduct.Platforms))
	shouldBe.Equal("steam", res.KeyProduct.Platforms[0].Id)
	shouldBe.EqualValues(10, res.KeyProduct.Platforms[0].Price.Amount)
	shouldBe.Equal("USD", res.KeyProduct.Platforms[0].Price.Currency)
	shouldBe.True(res.KeyProduct.Platforms[0].Price.IsFallback)

	res = grpc.GetKeyProductInfoResponse{}
	err = suite.service.GetKeyProductInfo(context.TODO(), &grpc.GetKeyProductInfoRequest{Country: "RUS", KeyProductId: response.Product.Id, Language: "ru"}, &res)
	shouldBe.Nil(err)
	shouldBe.Nil(res.Message)
	shouldBe.NotNil(res.KeyProduct)
	shouldBe.Equal(response.Product.Id, res.KeyProduct.Id)
	shouldBe.Equal(initialName, res.KeyProduct.Name)
	shouldBe.Equal("blah-blah-blah", res.KeyProduct.Description)
	shouldBe.Equal(1, len(res.KeyProduct.Platforms))
	shouldBe.Equal("steam", res.KeyProduct.Platforms[0].Id)
	shouldBe.EqualValues(10, res.KeyProduct.Platforms[0].Price.Amount)
	shouldBe.Equal("USD", res.KeyProduct.Platforms[0].Price.Currency)
	shouldBe.True(res.KeyProduct.Platforms[0].Price.IsFallback)
}

func (suite *KeyProductTestSuite) Test_GetPlatforms() {
	shouldBe := require.New(suite.T())

	rsp := &grpc.ListPlatformsResponse{}
	shouldBe.Nil(suite.service.GetPlatforms(context.TODO(), &grpc.ListPlatformsRequest{
		Limit:  100,
		Offset: 0,
	}, rsp))
	shouldBe.EqualValues(200, rsp.Status)
	shouldBe.NotEmpty(rsp.Platforms)
	shouldBe.EqualValues(9, len(rsp.Platforms))
	shouldBe.Equal(rsp.Platforms[0], availablePlatforms["steam"])

	rsp = &grpc.ListPlatformsResponse{}
	shouldBe.Nil(suite.service.GetPlatforms(context.TODO(), &grpc.ListPlatformsRequest{
		Limit:  1,
		Offset: 0,
	}, rsp))
	shouldBe.EqualValues(200, rsp.Status)
	shouldBe.Equal(1, len(rsp.Platforms))

	rsp = &grpc.ListPlatformsResponse{}
	shouldBe.Nil(suite.service.GetPlatforms(context.TODO(), &grpc.ListPlatformsRequest{
		Limit:  100,
		Offset: 100,
	}, rsp))
	shouldBe.EqualValues(200, rsp.Status)
	shouldBe.Empty(rsp.Platforms)
}

func (suite *KeyProductTestSuite) Test_GetKeyProduct() {
	shouldBe := require.New(suite.T())

	req := &grpc.CreateOrUpdateKeyProductRequest{
		Object:          "product",
		Sku:             "ru_double_yeti",
		Name:            map[string]string{"en": initialName},
		DefaultCurrency: "USD",
		Description:     map[string]string{"en": "blah-blah-blah"},
		LongDescription: map[string]string{"en": "Super game steam keys"},
		Url:             "http://test.ru/dffdsfsfs",
		Cover: &billing.ImageCollection{
			UseOneForAll: false,
			Images: &billing.LocalizedUrl{
				En: "/home/image.jpg",
			},
		},
		MerchantId: merchantId,
		ProjectId:  projectId,
		Metadata: map[string]string{
			"SomeKey": "SomeValue",
		},
	}

	response := grpc.KeyProductResponse{}
	err := suite.service.CreateOrUpdateKeyProduct(context.TODO(), req, &response)
	shouldBe.Nil(err)
	shouldBe.Nil(response.Message)
	res := response.Product

	response = grpc.KeyProductResponse{}
	err = suite.service.GetKeyProduct(context.TODO(), &grpc.RequestKeyProductMerchant{Id: res.Id, MerchantId: res.MerchantId}, &response)
	shouldBe.Nil(err)
	shouldBe.Nil(response.Message)

	product := response.Product

	shouldBe.Equal(res.Name["en"], product.Name["en"])
	shouldBe.Equal(res.DefaultCurrency, product.DefaultCurrency)
	shouldBe.Equal(res.Sku, product.Sku)
	shouldBe.Equal(res.Object, product.Object)
	shouldBe.Equal(res.Enabled, product.Enabled)
	shouldBe.Equal(res.Description, product.Description)
	shouldBe.Equal(res.LongDescription, product.LongDescription)
	shouldBe.Equal(res.Url, product.Url)
	shouldBe.Equal(res.Cover.Images.En, product.Cover.Images.En)
	shouldBe.Equal(res.Metadata, product.Metadata)
	shouldBe.NotNil(product.UpdatedAt)
	shouldBe.NotNil(product.CreatedAt)
	shouldBe.Nil(product.PublishedAt)
	shouldBe.False(product.Enabled)

	err = suite.service.GetKeyProduct(context.TODO(), &grpc.RequestKeyProductMerchant{Id: res.Id, MerchantId: res.MerchantId}, &response)
	shouldBe.Nil(err)
	shouldBe.Nil(response.Message)

	err = suite.service.GetKeyProduct(context.TODO(), &grpc.RequestKeyProductMerchant{Id: res.Id, MerchantId: res.MerchantId}, &response)
	shouldBe.Nil(err)
	shouldBe.Nil(response.Message)

	err = suite.service.GetKeyProduct(context.TODO(), &grpc.RequestKeyProductMerchant{Id: res.Id, MerchantId: res.MerchantId}, &response)
	shouldBe.Nil(err)
	shouldBe.Nil(response.Message)
}

func (suite *KeyProductTestSuite) Test_CreateOrUpdateKeyProduct() {
	shouldBe := require.New(suite.T())

	req := &grpc.CreateOrUpdateKeyProductRequest{
		Object:          "product",
		Sku:             "ru_double_yeti",
		Name:            map[string]string{"en": initialName},
		DefaultCurrency: "USD",
		Description:     map[string]string{"en": "blah-blah-blah"},
		LongDescription: map[string]string{"en": "Super game steam keys"},
		Url:             "http://test.ru/dffdsfsfs",
		Cover: &billing.ImageCollection{
			UseOneForAll: false,
			Images: &billing.LocalizedUrl{
				En: "/home/image.jpg",
			},
		},
		MerchantId: merchantId,
		ProjectId:  projectId,
		Metadata: map[string]string{
			"SomeKey": "SomeValue",
		},
	}

	response := grpc.KeyProductResponse{}
	err := suite.service.CreateOrUpdateKeyProduct(context.TODO(), req, &response)
	res := response.Product

	shouldBe.Nil(err)
	shouldBe.EqualValuesf(200, response.Status, "%s", response.Message)
	shouldBe.Equal(res.Name["en"], req.Name["en"])
	shouldBe.Equal(res.DefaultCurrency, req.DefaultCurrency)
	shouldBe.Equal(res.Sku, req.Sku)
	shouldBe.Equal(res.Object, req.Object)
	shouldBe.Equal(res.Description, req.Description)
	shouldBe.Equal(res.LongDescription, req.LongDescription)
	shouldBe.Equal(res.Url, req.Url)
	shouldBe.Equal(res.Cover.Images.En, req.Cover.Images.En)
	shouldBe.Equal(res.Metadata, req.Metadata)
	shouldBe.NotNil(res.UpdatedAt)
	shouldBe.NotNil(res.CreatedAt)
	shouldBe.Nil(res.PublishedAt)
	shouldBe.False(res.Enabled)
	shouldBe.NotEmpty(res.Id)

	req.Id = res.Id
	res2 := grpc.KeyProductResponse{}
	err = suite.service.CreateOrUpdateKeyProduct(context.TODO(), req, &res2)
	shouldBe.Nil(err)
	shouldBe.Nil(res2.Message)

	res2 = grpc.KeyProductResponse{}
	req.Id = bson.NewObjectId().Hex()
	err = suite.service.CreateOrUpdateKeyProduct(context.TODO(), req, &res2)
	shouldBe.Nil(err)
	shouldBe.NotNil(res2.Message)
	shouldBe.EqualValuesf(400, res2.Status, "%s", res2.Message)

	req.Sku = "NEW SKU"
	req.Id = res.Id
	res2 = grpc.KeyProductResponse{}
	err = suite.service.CreateOrUpdateKeyProduct(context.TODO(), req, &res2)
	shouldBe.Nil(err)
	shouldBe.NotNil(res2.Message)
	shouldBe.EqualValues(400, res2.Status)

	req.Sku = res.Sku
	req.MerchantId = bson.NewObjectId().Hex()
	res2 = grpc.KeyProductResponse{}
	err = suite.service.CreateOrUpdateKeyProduct(context.TODO(), req, &res2)
	shouldBe.Nil(err)
	shouldBe.NotNil(res2.Message)
	shouldBe.EqualValues(400, res2.Status)

	req.MerchantId = res.MerchantId
	req.ProjectId = bson.NewObjectId().Hex()
	res2 = grpc.KeyProductResponse{}
	err = suite.service.CreateOrUpdateKeyProduct(context.TODO(), req, &res2)
	shouldBe.Nil(err)
	shouldBe.NotNil(res2.Message)
	shouldBe.EqualValues(400, res2.Status)
}

func (suite *KeyProductTestSuite) Test_GetKeyProducts() {
	shouldBe := require.New(suite.T())

	req := &grpc.ListKeyProductsRequest{
		MerchantId: merchantId,
		ProjectId:  projectId,
	}
	res := &grpc.ListKeyProductsResponse{}
	err := suite.service.GetKeyProducts(context.TODO(), req, res)
	shouldBe.Nil(err)
	shouldBe.EqualValues(0, res.Count)
	shouldBe.EqualValues(0, res.Offset)
	shouldBe.EqualValues(0, len(res.Products))

	for i := 0; i < 10; i++ {
		suite.createKeyProduct()
	}

	err = suite.service.GetKeyProducts(context.TODO(), req, res)
	shouldBe.Nil(err)
	shouldBe.EqualValuesf(200, res.Status, "%s", res.Message)
	shouldBe.EqualValues(10, res.Count)
	shouldBe.EqualValues(0, res.Offset)
	shouldBe.EqualValues(10, len(res.Products))
	shouldBe.EqualValues(0, res.Products[0].Platforms[0].Count)

	req.Offset = 9
	err = suite.service.GetKeyProducts(context.TODO(), req, res)
	shouldBe.Nil(err)
	shouldBe.EqualValues(10, res.Count)
	shouldBe.EqualValues(1, len(res.Products))

	req.Offset = 0
	req.Limit = 2
	err = suite.service.GetKeyProducts(context.TODO(), req, res)
	shouldBe.Nil(err)
	shouldBe.EqualValues(10, res.Count)
	shouldBe.EqualValues(2, len(res.Products))

	req.Offset = 0
	req.Limit = 0
	req.Sku = "some sku"
	req.Name = "some name"
	err = suite.service.GetKeyProducts(context.TODO(), req, res)
	shouldBe.Nil(err)

	req.Offset = 0
	req.Limit = 100
	req.Sku = ""
	req.Name = ""
	req.Enabled = "true"
	err = suite.service.GetKeyProducts(context.TODO(), req, res)
	shouldBe.Nil(err)
	shouldBe.EqualValues(0, len(res.Products))

	req.Offset = 0
	req.Limit = 100
	req.Sku = ""
	req.Name = ""
	req.Enabled = "false"
	err = suite.service.GetKeyProducts(context.TODO(), req, res)
	shouldBe.Nil(err)
	shouldBe.EqualValues(10, len(res.Products))
}

func (suite *KeyProductTestSuite) getKeyProduct(id string) *grpc.KeyProduct {
	suite.T().Helper()

	res := &grpc.KeyProductResponse{}
	err := suite.service.GetKeyProduct(context.TODO(), &grpc.RequestKeyProductMerchant{MerchantId: merchantId, Id: id}, res)
	assert.Nil(suite.T(), err)
	assert.Nil(suite.T(), res.Message)
	return res.Product
}

func (suite *KeyProductTestSuite) createKeyProduct() *grpc.KeyProduct {
	suite.T().Helper()

	req := &grpc.CreateOrUpdateKeyProductRequest{
		Object:          "product",
		Sku:             bson.NewObjectId().Hex(),
		Name:            map[string]string{"en": initialName},
		DefaultCurrency: "USD",
		Description:     map[string]string{"en": "blah-blah-blah"},
		LongDescription: map[string]string{"en": "Super game steam keys"},
		Url:             "http://test.ru/dffdsfsfs",
		Cover: &billing.ImageCollection{
			UseOneForAll: false,
			Images: &billing.LocalizedUrl{
				En: "/home/image.jpg",
			},
		},
		MerchantId: merchantId,
		ProjectId:  projectId,
		Platforms: []*grpc.PlatformPrice{
			{
				Id: "steam",
				Prices: []*billing.ProductPrice{
					{Region: "USD", Currency: "USD", Amount: 66.66},
				},
			},
		},
		Metadata: map[string]string{
			"SomeKey": "SomeValue",
		},
	}

	res := &grpc.KeyProductResponse{}
	err := suite.service.CreateOrUpdateKeyProduct(context.TODO(), req, res)
	assert.Nil(suite.T(), err)
	assert.Nil(suite.T(), res.Message)
	return res.Product
}

func (suite *KeyProductTestSuite) Test_UpdatePlatformPrices_WithBadPrice_Error() {
	shouldBe := require.New(suite.T())
	product := suite.createKeyProduct()
	req := &grpc.CreateOrUpdateKeyProductRequest{
		Id:              product.Id,
		MerchantId:      product.MerchantId,
		Name:            product.Name,
		Description:     product.Description,
		ProjectId:       product.MerchantId,
		DefaultCurrency: product.DefaultCurrency,
		Platforms: []*grpc.PlatformPrice{
			{
				Id: "steam",
				Prices: []*billing.ProductPrice{
					{Region: "USD", Currency: "RUB", Amount: 66.66},
				},
			},
		},
	}

	res := &grpc.KeyProductResponse{}
	err := suite.service.CreateOrUpdateKeyProduct(context.TODO(), req, res)
	shouldBe.Nil(err)
	shouldBe.EqualValues(400, res.Status)
	shouldBe.NotEmpty(res.Message)
}

func (suite *KeyProductTestSuite) Test_UpdatePlatformPrices() {
	shouldBe := require.New(suite.T())
	product := suite.createKeyProduct()
	req := &grpc.CreateOrUpdateKeyProductRequest{
		Id:              product.Id,
		MerchantId:      product.MerchantId,
		Name:            product.Name,
		Description:     product.Description,
		ProjectId:       product.MerchantId,
		DefaultCurrency: product.DefaultCurrency,
		Platforms: []*grpc.PlatformPrice{
			{
				Id: "steam",
				Prices: []*billing.ProductPrice{
					{Region: "USD", Currency: "USD", Amount: 66.66},
				},
			},
		},
	}

	res := &grpc.KeyProductResponse{}
	err := suite.service.CreateOrUpdateKeyProduct(context.TODO(), req, res)
	shouldBe.Nil(err)
	shouldBe.Nil(res.Message)

	prices := res.Product.Platforms[0].Prices
	shouldBe.Equal(1, len(prices))
	shouldBe.Equal(66.66, prices[0].Amount)
	shouldBe.Equal("USD", prices[0].Currency)

	req = &grpc.CreateOrUpdateKeyProductRequest{
		Id:              product.Id,
		MerchantId:      product.MerchantId,
		ProjectId:       product.MerchantId,
		DefaultCurrency: product.DefaultCurrency,
		Name:            product.Name,
		Description:     product.Description,
		Platforms: []*grpc.PlatformPrice{
			{
				Id: "steam",
				Prices: []*billing.ProductPrice{
					{Region: "USD", Currency: "USD", Amount: 77.66},
					{Region: "EUR", Currency: "EUR", Amount: 77.77},
				},
			},
		},
	}

	res = &grpc.KeyProductResponse{}
	err = suite.service.CreateOrUpdateKeyProduct(context.TODO(), req, res)

	shouldBe.Nil(err)
	shouldBe.Nil(res.Message)

	req = &grpc.CreateOrUpdateKeyProductRequest{
		Id:              product.Id,
		MerchantId:      product.MerchantId,
		ProjectId:       product.MerchantId,
		DefaultCurrency: product.DefaultCurrency,
		Name:            product.Name,
		Description:     product.Description,
		Platforms: []*grpc.PlatformPrice{
			{
				Id:            "best_store_ever",
				EulaUrl:       "http://www.example.com",
				ActivationUrl: "http://www.example.com",
				Prices: []*billing.ProductPrice{
					{Region: "RUB", Currency: "RUB", Amount: 0.01},
					{Region: "USD", Currency: "USD", Amount: 66.66},
				},
			},
		},
	}

	res = &grpc.KeyProductResponse{}
	err = suite.service.CreateOrUpdateKeyProduct(context.TODO(), req, res)
	shouldBe.Nil(err)
	shouldBe.NotNil(res.Message)
	shouldBe.EqualValues(400, res.Status)

	req = &grpc.CreateOrUpdateKeyProductRequest{
		Id:              product.Id,
		MerchantId:      product.MerchantId,
		ProjectId:       product.MerchantId,
		DefaultCurrency: product.DefaultCurrency,
		Name:            product.Name,
		Description:     product.Description,
		Platforms: []*grpc.PlatformPrice{
			{
				Id:            "best_store_ever",
				EulaUrl:       "http://www.example.com",
				ActivationUrl: "http://www.example.com",
				Prices: []*billing.ProductPrice{
					{Region: "RUB", Currency: "RUB", Amount: 0.01},
					{Region: "USD", Currency: "USD", Amount: 66.66},
				},
			},
			{
				Id:            "another_best_store_ever",
				EulaUrl:       "http://www.example.com",
				ActivationUrl: "http://www.example.com",
				Prices: []*billing.ProductPrice{
					{Region: "RUB", Currency: "RUB", Amount: 0.01},
					{Region: "USD", Currency: "USD", Amount: 66.66},
				},
			},
		},
	}
	res = &grpc.KeyProductResponse{}
	err = suite.service.CreateOrUpdateKeyProduct(context.TODO(), req, res)
	shouldBe.Nil(err)
	shouldBe.NotNil(res.Message)
	shouldBe.EqualValues(400, res.Status)
}

func (suite *KeyProductTestSuite) Test_PublishKeyProduct() {
	shouldBe := require.New(suite.T())

	product := suite.createKeyProduct()
	req := &grpc.PublishKeyProductRequest{
		KeyProductId: product.Id,
		MerchantId:   merchantId,
	}
	res := &grpc.KeyProductResponse{}
	err := suite.service.PublishKeyProduct(context.TODO(), req, res)
	shouldBe.Nil(err)
	shouldBe.Nil(res.Message)
	shouldBe.EqualValues(200, res.Status)
	shouldBe.True(res.Product.Enabled)
	shouldBe.NotNil(res.Product.PublishedAt)
}

func (suite *KeyProductTestSuite) Test_DeleteKeyProduct() {
	shouldBe := require.New(suite.T())
	product := suite.createKeyProduct()

	res := &grpc.EmptyResponseWithStatus{}
	err := suite.service.DeleteKeyProduct(context.TODO(), &grpc.RequestKeyProductMerchant{Id: product.Id, MerchantId: merchantId}, res)
	shouldBe.Nil(err)
	shouldBe.Nil(res.Message)

	res = &grpc.EmptyResponseWithStatus{}
	err = suite.service.DeleteKeyProduct(context.TODO(), &grpc.RequestKeyProductMerchant{Id: product.Id, MerchantId: merchantId}, res)
	shouldBe.Nil(err)
	shouldBe.NotNil(res.Message)
}

func (suite *KeyProductTestSuite) Test_UploadKey() {
	shouldBe := require.New(suite.T())
	product := suite.createKeyProduct()
	fileContent := fmt.Sprintf("%s-%s-%s-%s", RandomString(4), RandomString(4), RandomString(4), RandomString(4))
	file := []byte(fileContent)

	keysRsp := &grpc.PlatformKeysFileResponse{}
	keysReq := &grpc.PlatformKeysFileRequest{
		KeyProductId: product.Id,
		PlatformId:   "steam",
		MerchantId:   product.MerchantId,
		File:         file,
	}
	shouldBe.NoError(suite.service.UploadKeysFile(context.TODO(), keysReq, keysRsp))
	shouldBe.EqualValues(200, keysRsp.Status)
	shouldBe.EqualValues(1, keysRsp.TotalCount)
	shouldBe.EqualValues(1, keysRsp.KeysProcessed)

	keysRsp = &grpc.PlatformKeysFileResponse{}
	keysReq = &grpc.PlatformKeysFileRequest{
		KeyProductId: product.Id,
		PlatformId:   "steam",
		MerchantId:   product.MerchantId,
		File:         file,
	}
	shouldBe.NoError(suite.service.UploadKeysFile(context.TODO(), keysReq, keysRsp))
	shouldBe.EqualValues(200, keysRsp.Status)
	shouldBe.EqualValues(1, keysRsp.TotalCount)
	shouldBe.EqualValues(0, keysRsp.KeysProcessed)
}

func (suite *KeyProductTestSuite) Test_CheckSkuAndKeyProject() {
	shouldBe := require.New(suite.T())
	rsp := &grpc.EmptyResponseWithStatus{}
	err := suite.service.CheckSkuAndKeyProject(context.TODO(), &grpc.CheckSkuAndKeyProjectRequest{ProjectId: projectId, Sku: "TEST_SKU"}, rsp)
	shouldBe.NoError(err)
	shouldBe.EqualValues(200, rsp.Status)

	product := suite.createKeyProduct()
	err = suite.service.CheckSkuAndKeyProject(context.TODO(), &grpc.CheckSkuAndKeyProjectRequest{ProjectId: product.ProjectId, Sku: product.Sku}, rsp)
	shouldBe.NoError(err)
	shouldBe.EqualValues(400, rsp.Status)
	shouldBe.NotNil(rsp.Message)
}

func (suite *KeyProductTestSuite) Test_UnPublishKeyProduct() {
	shouldBe := require.New(suite.T())
	product := suite.createKeyProduct()

	res := &grpc.KeyProductResponse{}
	err := suite.service.PublishKeyProduct(context.TODO(), &grpc.PublishKeyProductRequest{KeyProductId: product.Id, MerchantId: product.MerchantId}, res)
	shouldBe.Nil(err)
	shouldBe.EqualValues(200, res.Status)

	err = suite.service.UnPublishKeyProduct(context.TODO(), &grpc.UnPublishKeyProductRequest{KeyProductId: product.Id}, res)
	shouldBe.Nil(err)
	shouldBe.EqualValues(200, res.Status)

	err = suite.service.UnPublishKeyProduct(context.TODO(), &grpc.UnPublishKeyProductRequest{KeyProductId: product.Id}, res)
	shouldBe.Nil(err)
	shouldBe.EqualValues(400, res.Status)
	shouldBe.Equal(keyProductNotPublished, res.Message)

	emptyRes := &grpc.EmptyResponseWithStatus{}
	err = suite.service.DeleteKeyProduct(context.TODO(), &grpc.RequestKeyProductMerchant{Id: product.Id, MerchantId: product.MerchantId}, emptyRes)
	shouldBe.Nil(err)
	shouldBe.EqualValuesf(200, emptyRes.Status, "%v", emptyRes.Message)

	err = suite.service.UnPublishKeyProduct(context.TODO(), &grpc.UnPublishKeyProductRequest{KeyProductId: product.Id}, res)
	shouldBe.Nil(err)
	shouldBe.EqualValues(400, res.Status)
	shouldBe.Equal(keyProductNotFound, res.Message)
}
