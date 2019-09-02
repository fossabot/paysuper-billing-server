package service

import (
	"context"
	"github.com/go-redis/redis"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mongodb"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang/protobuf/ptypes"
	"github.com/jinzhu/now"
	"github.com/paysuper/paysuper-billing-server/internal/config"
	"github.com/paysuper/paysuper-billing-server/internal/database"
	"github.com/paysuper/paysuper-billing-server/internal/mock"
	"github.com/paysuper/paysuper-billing-server/pkg"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/billing"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/grpc"
	mongodb "github.com/paysuper/paysuper-database-mongo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	rabbitmq "gopkg.in/ProtocolONE/rabbitmq.v1/pkg"
	"testing"
	"time"
)

type VatReportsTestSuite struct {
	suite.Suite
	service *Service
	log     *zap.Logger
	cache   CacheInterface

	projectFixedAmount *billing.Project
	paymentMethod      *billing.PaymentMethod
	paymentSystem      *billing.PaymentSystem
}

func Test_VatReports(t *testing.T) {
	suite.Run(t, new(VatReportsTestSuite))
}

func (suite *VatReportsTestSuite) SetupTest() {
	cfg, err := config.NewConfig()
	if err != nil {
		suite.FailNow("Config load failed", "%v", err)
	}
	cfg.AccountingCurrency = "RUB"
	cfg.CardPayApiUrl = "https://sandbox.cardpay.com"
	cfg.OrderViewUpdateBatchSize = 20

	m, err := migrate.New(
		"file://../../migrations/tests",
		cfg.MongoDsn)
	assert.NoError(suite.T(), err, "Migrate init failed")

	err = m.Up()
	if err != nil && err.Error() != "no change" {
		suite.FailNow("Migrations failed", "%v", err)
	}

	db, err := mongodb.NewDatabase()
	if err != nil {
		suite.FailNow("Database connection failed", "%v", err)
	}

	suite.log, err = zap.NewProduction()

	if err != nil {
		suite.FailNow("Logger initialization failed", "%v", err)
	}

	broker, err := rabbitmq.NewBroker(cfg.BrokerAddress)

	if err != nil {
		suite.FailNow("Creating RabbitMQ publisher failed", "%v", err)
	}

	redisClient := database.NewRedis(
		&redis.Options{
			Addr:     cfg.RedisHost,
			Password: cfg.RedisPassword,
		},
	)

	redisdb := mock.NewTestRedis()
	suite.cache = NewCacheRedis(redisdb)
	suite.service = NewBillingService(
		db,
		cfg,
		mock.NewGeoIpServiceTestOk(),
		mock.NewRepositoryServiceOk(),
		mock.NewTaxServiceOkMock(),
		broker,
		redisClient,
		suite.cache,
		mock.NewCurrencyServiceMockOk(),
		mock.NewDocumentSignerMockOk(),
	)

	if err := suite.service.Init(); err != nil {
		suite.FailNow("Billing service initialization failed", "%v", err)
	}

	_, suite.projectFixedAmount, suite.paymentMethod, suite.paymentSystem = helperCreateEntitiesForTests(suite.Suite, suite.service)
}

func (suite *VatReportsTestSuite) TearDownTest() {
	if err := suite.service.db.Drop(); err != nil {
		suite.FailNow("Database deletion failed", "%v", err)
	}

	suite.service.db.Close()
}

func (suite *VatReportsTestSuite) TestVatReports_getLastVatReportTime() {
	_, _, err := suite.service.getLastVatReportTime(0)
	assert.NoError(suite.T(), err)

	from, to, err := suite.service.getLastVatReportTime(int32(3))
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), from, now.BeginningOfDay())
	assert.Equal(suite.T(), to, now.EndOfDay())

	from, to, err = suite.service.getLastVatReportTime(int32(1))
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), from, now.BeginningOfDay())
	assert.Equal(suite.T(), to, now.EndOfDay())

	from, to, err = suite.service.getLastVatReportTime(int32(2))
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), from, now.BeginningOfDay())
	assert.Equal(suite.T(), to, now.EndOfDay())
}

func (suite *VatReportsTestSuite) TestVatReports_getVatReportTimeForDate() {

	t, err := time.Parse(time.RFC3339, "2019-06-29T11:45:26.371Z")
	assert.NoError(suite.T(), err)

	from, to, err := suite.service.getVatReportTimeForDate(int32(3), t)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), from.Format(time.RFC3339), "2019-06-29T00:00:00Z")
	assert.Equal(suite.T(), to.Format(time.RFC3339), "2019-06-29T23:59:59Z")

	from, to, err = suite.service.getVatReportTimeForDate(int32(1), t)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), from.Format(time.RFC3339), "2019-06-29T00:00:00Z")
	assert.Equal(suite.T(), to.Format(time.RFC3339), "2019-06-29T23:59:59Z")

	from, to, err = suite.service.getVatReportTimeForDate(int32(2), t)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), from.Format(time.RFC3339), "2019-06-29T00:00:00Z")
	assert.Equal(suite.T(), to.Format(time.RFC3339), "2019-06-29T23:59:59Z")

	t, err = time.Parse(time.RFC3339, "2019-05-29T11:45:26.371Z")
	assert.NoError(suite.T(), err)
	from, to, err = suite.service.getVatReportTimeForDate(int32(2), t)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), from.Format(time.RFC3339), "2019-05-29T00:00:00Z")
	assert.Equal(suite.T(), to.Format(time.RFC3339), "2019-05-29T23:59:59Z")

	t, err = time.Parse(time.RFC3339, "2019-07-29T11:45:26.371Z")
	assert.NoError(suite.T(), err)
	from, to, err = suite.service.getVatReportTimeForDate(int32(2), t)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), from.Format(time.RFC3339), "2019-07-29T00:00:00Z")
	assert.Equal(suite.T(), to.Format(time.RFC3339), "2019-07-29T23:59:59Z")

	t, err = time.Parse(time.RFC3339, "2019-08-29T11:45:26.371Z")
	assert.NoError(suite.T(), err)
	from, to, err = suite.service.getVatReportTimeForDate(int32(2), t)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), from.Format(time.RFC3339), "2019-08-29T00:00:00Z")
	assert.Equal(suite.T(), to.Format(time.RFC3339), "2019-08-29T23:59:59Z")

	t, err = time.Parse(time.RFC3339, "2019-04-01T00:00:00Z")
	assert.NoError(suite.T(), err)
	from, to, err = suite.service.getVatReportTimeForDate(int32(3), t)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), from.Format(time.RFC3339), "2019-04-01T00:00:00Z")
	assert.Equal(suite.T(), to.Format(time.RFC3339), "2019-04-01T23:59:59Z")

	t, err = time.Parse(time.RFC3339, "2019-06-30T23:59:59Z")
	assert.NoError(suite.T(), err)
	from, to, err = suite.service.getVatReportTimeForDate(int32(3), t)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), from.Format(time.RFC3339), "2019-06-30T00:00:00Z")
	assert.Equal(suite.T(), to.Format(time.RFC3339), "2019-06-30T23:59:59Z")
}

func (suite *VatReportsTestSuite) TestVatReports_ProcessVatReports() {

	amounts := []float64{100, 10}
	currencies := []string{"RUB", "USD"}
	countries := []string{"RU", "FI"}
	var orders []*billing.Order
	numberOfOrders := 30

	count := 0
	for count < numberOfOrders {
		order := helperCreateAndPayOrder(
			suite.Suite,
			suite.service,
			amounts[count%2],
			currencies[count%2],
			countries[count%2],
			suite.projectFixedAmount,
			suite.paymentMethod,
		)
		assert.NotNil(suite.T(), order)
		orders = append(orders, order)

		count++
	}

	suite.paymentSystem.Handler = "mock_ok"
	err := suite.service.paymentSystem.Update(suite.paymentSystem)
	assert.NoError(suite.T(), err)

	for _, order := range orders {
		refund := helperMakeRefund(suite.Suite, suite.service, order, order.TotalPaymentAmount*0.5, false)
		assert.NotNil(suite.T(), refund)
	}

	req := &grpc.ProcessVatReportsRequest{
		Date: ptypes.TimestampNow(),
	}
	err = suite.service.ProcessVatReports(context.TODO(), req, &grpc.EmptyResponse{})
	assert.NoError(suite.T(), err)

	repRes := grpc.VatReportsResponse{}

	err = suite.service.GetVatReportsForCountry(context.TODO(), &grpc.VatReportsRequest{Country: "RU"}, &repRes)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), repRes.Status, pkg.ResponseStatusOk)
	assert.NotNil(suite.T(), repRes.Data)
	assert.Equal(suite.T(), repRes.Data.Count, int32(1))

	report := repRes.Data.Items[0]
	assert.NotNil(suite.T(), report)
	assert.Equal(suite.T(), report.Country, "RU")
	assert.Equal(suite.T(), report.Currency, "RUB")
	assert.Equal(suite.T(), report.TransactionsCount, int32(numberOfOrders))
	assert.Equal(suite.T(), report.GrossRevenue, float64(900))
	assert.Equal(suite.T(), report.VatAmount, float64(150))
	assert.Equal(suite.T(), report.FeesAmount, float64(144.39))
	assert.Equal(suite.T(), report.DeductionAmount, float64(0))
	assert.Equal(suite.T(), report.CountryAnnualTurnover, float64(1800))
	assert.Equal(suite.T(), report.WorldAnnualTurnover, float64(13183.80))
	assert.Equal(suite.T(), report.Status, pkg.VatReportStatusThreshold)

	err = suite.service.GetVatReportsForCountry(context.TODO(), &grpc.VatReportsRequest{Country: "FI"}, &repRes)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), repRes.Status, pkg.ResponseStatusOk)
	assert.NotNil(suite.T(), repRes.Data)
	assert.Equal(suite.T(), repRes.Data.Count, int32(1))

	report = repRes.Data.Items[0]
	assert.NotNil(suite.T(), report)
	assert.Equal(suite.T(), report.Country, "FI")
	assert.Equal(suite.T(), report.Currency, "EUR")
	assert.Equal(suite.T(), report.TransactionsCount, int32(numberOfOrders))
	assert.Equal(suite.T(), report.GrossRevenue, float64(81.32))
	assert.Equal(suite.T(), report.VatAmount, float64(13.56))
	assert.Equal(suite.T(), report.FeesAmount, float64(8.9))
	assert.Equal(suite.T(), report.DeductionAmount, float64(0))
	assert.Equal(suite.T(), report.CountryAnnualTurnover, float64(162))
	assert.Equal(suite.T(), report.WorldAnnualTurnover, float64(188.34))
	assert.Equal(suite.T(), report.Status, pkg.VatReportStatusThreshold)

	assert.NoError(suite.T(), err)
}
