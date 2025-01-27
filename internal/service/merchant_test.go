package service

import (
	"errors"
	"github.com/globalsign/mgo/bson"
	"github.com/paysuper/paysuper-billing-server/internal/config"
	"github.com/paysuper/paysuper-billing-server/internal/mocks"
	internalPkg "github.com/paysuper/paysuper-billing-server/internal/pkg"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/billing"
	mongodb "github.com/paysuper/paysuper-database-mongo"
	reportingMocks "github.com/paysuper/paysuper-reporter/pkg/mocks"
	"github.com/stretchr/testify/assert"
	mock2 "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"testing"
)

type MerchantTestSuite struct {
	suite.Suite
	service    *Service
	log        *zap.Logger
	cache      internalPkg.CacheInterface
	merchant   *billing.Merchant
	pmBankCard *billing.PaymentMethod
}

func Test_Merchant(t *testing.T) {
	suite.Run(t, new(MerchantTestSuite))
}

func (suite *MerchantTestSuite) SetupTest() {
	cfg, err := config.NewConfig()
	if err != nil {
		suite.FailNow("Config load failed", "%v", err)
	}

	db, err := mongodb.NewDatabase()
	if err != nil {
		suite.FailNow("Database connection failed", "%v", err)
	}

	suite.log, err = zap.NewProduction()

	if err != nil {
		suite.FailNow("Logger initialization failed", "%v", err)
	}

	redisdb := mocks.NewTestRedis()
	suite.cache = NewCacheRedis(redisdb)
	suite.service = NewBillingService(
		db,
		cfg,
		nil,
		nil,
		nil,
		nil,
		nil,
		suite.cache,
		mocks.NewCurrencyServiceMockOk(),
		mocks.NewDocumentSignerMockOk(),
		&reportingMocks.ReporterService{},
		mocks.NewFormatterOK(),
		mocks.NewBrokerMockOk(),
	)

	if err := suite.service.Init(); err != nil {
		suite.FailNow("Billing service initialization failed", "%v", err)
	}

	suite.pmBankCard = &billing.PaymentMethod{
		Id:   bson.NewObjectId().Hex(),
		Name: "Bank card",
	}
	suite.merchant = &billing.Merchant{
		Id: bson.NewObjectId().Hex(),
		PaymentMethods: map[string]*billing.MerchantPaymentMethod{
			suite.pmBankCard.Id: {
				PaymentMethod: &billing.MerchantPaymentMethodIdentification{
					Id:   suite.pmBankCard.Id,
					Name: suite.pmBankCard.Name,
				},
				Commission: &billing.MerchantPaymentMethodCommissions{
					Fee: 2.5,
					PerTransaction: &billing.MerchantPaymentMethodPerTransactionCommission{
						Fee:      30,
						Currency: "RUB",
					},
				},
				Integration: &billing.MerchantPaymentMethodIntegration{
					TerminalId:       "1234567890",
					TerminalPassword: "0987654321",
					Integrated:       true,
				},
				IsActive: true,
			},
		},
	}
	if err := suite.service.merchant.Insert(suite.merchant); err != nil {
		suite.FailNow("Insert country test data failed", "%v", err)
	}
}

func (suite *MerchantTestSuite) TearDownTest() {
	if err := suite.service.db.Drop(); err != nil {
		suite.FailNow("Database deletion failed", "%v", err)
	}

	suite.service.db.Close()
}

func (suite *MerchantTestSuite) TestMerchant_Insert_Ok() {
	assert.NoError(suite.T(), suite.service.merchant.Insert(&billing.Merchant{}))
}

func (suite *MerchantTestSuite) TestMerchant_Insert_ErrorCacheUpdate() {
	id := bson.NewObjectId().Hex()
	ci := &mocks.CacheInterface{}
	ci.On("Set", "merchant:id:"+id, mock2.Anything, mock2.Anything).
		Return(errors.New("service unavailable"))
	suite.service.cacher = ci
	err := suite.service.merchant.Insert(&billing.Merchant{Id: id})

	assert.Error(suite.T(), err)
	assert.EqualError(suite.T(), err, "service unavailable")
}

func (suite *MerchantTestSuite) TestMerchant_Update_Ok() {
	assert.NoError(suite.T(), suite.service.merchant.Update(&billing.Merchant{Id: suite.merchant.Id}))
}

func (suite *MerchantTestSuite) TestMerchant_Update_NotFound() {
	err := suite.service.merchant.Update(&billing.Merchant{Id: bson.NewObjectId().Hex()})

	assert.Error(suite.T(), err)
	assert.EqualError(suite.T(), err, "not found")
}

func (suite *MerchantTestSuite) TestMerchant_Update_ErrorCacheUpdate() {
	id := bson.NewObjectId().Hex()
	ci := &mocks.CacheInterface{}
	ci.On("Set", "merchant:id:"+id, mock2.Anything, mock2.Anything).
		Return(errors.New("service unavailable"))
	suite.service.cacher = ci
	_ = suite.service.merchant.Insert(&billing.Merchant{Id: id})
	err := suite.service.merchant.Update(&billing.Merchant{Id: id})

	assert.Error(suite.T(), err)
	assert.EqualError(suite.T(), err, "service unavailable")
}

func (suite *MerchantTestSuite) TestMerchant_GetById_Ok() {
	merchant := &billing.Merchant{
		Id: bson.NewObjectId().Hex(),
	}
	if err := suite.service.merchant.Insert(merchant); err != nil {
		suite.Assert().NoError(err)
	}
	c, err := suite.service.merchant.GetById(merchant.Id)

	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), c)
	assert.Equal(suite.T(), merchant.Id, c.Id)
}

func (suite *MerchantTestSuite) TestMerchant_GetById_Ok_ByCache() {
	ci := &mocks.CacheInterface{}
	ci.On("Get", "merchant:id:"+suite.merchant.Id, mock2.Anything).
		Return(nil)
	suite.service.cacher = ci
	c, err := suite.service.merchant.GetById(suite.merchant.Id)

	assert.Nil(suite.T(), err)
	assert.IsType(suite.T(), &billing.Merchant{}, c)
}

func (suite *MerchantTestSuite) TestMerchant_GetPaymentMethod_Ok() {
	pm, err := suite.service.merchant.GetPaymentMethod(suite.merchant.Id, suite.pmBankCard.Id)

	assert.NoError(suite.T(), err)
	assert.IsType(suite.T(), &billing.MerchantPaymentMethod{}, pm)
}

func (suite *MerchantTestSuite) TestMerchant_GetPaymentMethod_ErrorByMerchantNotFound() {
	_, err := suite.service.merchant.GetPaymentMethod(bson.NewObjectId().Hex(), "")

	assert.Error(suite.T(), err)
	assert.EqualError(suite.T(), err, "merchant not found")
}
