package service

// helpers, that used in accounting_entry_test, royalty_report_test, vat_reports_test and order_view_test

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/globalsign/mgo/bson"
	"github.com/golang/protobuf/ptypes"
	"github.com/paysuper/paysuper-billing-server/pkg"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/billing"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/grpc"
	"github.com/paysuper/paysuper-recurring-repository/pkg/constant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"math/rand"
	"strconv"
	"time"
)

func helperCreateEntitiesForTests(suite suite.Suite, service *Service) (
	*billing.Merchant,
	*billing.Project,
	*billing.PaymentMethod,
	*billing.PaymentSystem,
) {

	paymentSystem := &billing.PaymentSystem{
		Id:                 bson.NewObjectId().Hex(),
		Name:               "CardPay",
		AccountingCurrency: "RUB",
		AccountingPeriod:   "every-day",
		Country:            "",
		IsActive:           true,
		Handler:            paymentSystemHandlerCardPayMock,
	}

	pmBankCard := &billing.PaymentMethod{
		Id:               bson.NewObjectId().Hex(),
		Name:             "Bank card",
		Group:            "BANKCARD",
		MinPaymentAmount: 10,
		MaxPaymentAmount: 15000,
		ExternalId:       "BANKCARD",
		ProductionSettings: map[string]*billing.PaymentMethodParams{
			"RUB": {
				TerminalId:     "15985",
				Secret:         "A1tph4I6BD0f",
				SecretCallback: "0V1rJ7t4jCRv",
			},
			"USD": {
				TerminalId:     "15985",
				Secret:         "A1tph4I6BD0f",
				SecretCallback: "0V1rJ7t4jCRv",
			},
		},
		TestSettings: map[string]*billing.PaymentMethodParams{
			"RUB": {
				TerminalId:     "15985",
				Secret:         "A1tph4I6BD0f",
				SecretCallback: "0V1rJ7t4jCRv",
			},
			"USD": {
				TerminalId:     "15985",
				Secret:         "A1tph4I6BD0f",
				SecretCallback: "0V1rJ7t4jCRv",
			},
		},
		Type:            "bank_card",
		IsActive:        true,
		AccountRegexp:   "^(?:4[0-9]{12}(?:[0-9]{3})?|[25][1-7][0-9]{14}|6(?:011|5[0-9][0-9])[0-9]{12}|3[47][0-9]{13}|3(?:0[0-5]|[68][0-9])[0-9]{11}|(?:2131|1800|35\\d{3})\\d{11})$",
		PaymentSystemId: paymentSystem.Id,
	}

	merchant := helperCreateMerchant(suite, service, "USD", "RU", pmBankCard, 0)

	projectFixedAmount := &billing.Project{
		Id:                       bson.NewObjectId().Hex(),
		CallbackCurrency:         "RUB",
		CallbackProtocol:         "default",
		LimitsCurrency:           "USD",
		MaxPaymentAmount:         15000,
		MinPaymentAmount:         1,
		Name:                     map[string]string{"en": "test project 1"},
		IsProductsCheckout:       false,
		AllowDynamicRedirectUrls: true,
		SecretKey:                "test project 1 secret key",
		Status:                   pkg.ProjectStatusDraft,
		MerchantId:               merchant.Id,
	}

	bin := &BinData{
		Id:                 bson.NewObjectId(),
		CardBin:            400000,
		CardBrand:          "MASTERCARD",
		CardType:           "DEBIT",
		CardCategory:       "WORLD",
		BankName:           "ALFA BANK",
		BankCountryName:    "UKRAINE",
		BankCountryIsoCode: "US",
	}

	err := service.db.Collection(collectionBinData).Insert(bin)

	if err != nil {
		suite.FailNow("Insert BIN test data failed", "%v", err)
	}

	pms := []*billing.PaymentMethod{pmBankCard}
	if err := service.paymentMethod.MultipleInsert(pms); err != nil {
		suite.FailNow("Insert payment methods test data failed", "%v", err)
	}

	projects := []*billing.Project{projectFixedAmount}
	if err := service.project.MultipleInsert(projects); err != nil {
		suite.FailNow("Insert project test data failed", "%v", err)
	}

	ps := []*billing.PaymentSystem{paymentSystem}
	if err := service.paymentSystem.MultipleInsert(ps); err != nil {
		suite.FailNow("Insert payment system test data failed", "%v", err)
	}

	sysCost := &billing.MoneyBackCostSystem{
		Name:           "MASTERCARD",
		PayoutCurrency: "USD",
		UndoReason:     "reversal",
		Region:         "Russia",
		Country:        "RU",
		DaysFrom:       0,
		PaymentStage:   1,
		Percent:        0.10,
		FixAmount:      0.15,
	}
	sysCost2 := &billing.MoneyBackCostSystem{
		Name:           "MASTERCARD",
		PayoutCurrency: "RUB",
		UndoReason:     "reversal",
		Region:         "Russia",
		Country:        "RU",
		DaysFrom:       0,
		PaymentStage:   1,
		Percent:        0.10,
		FixAmount:      0.15,
	}
	sysCost3 := &billing.MoneyBackCostSystem{
		Name:           "MASTERCARD",
		PayoutCurrency: "USD",
		UndoReason:     "reversal",
		Region:         "North America",
		Country:        "US",
		DaysFrom:       0,
		PaymentStage:   1,
		Percent:        0.10,
		FixAmount:      0.15,
	}
	sysCost4 := &billing.MoneyBackCostSystem{
		Name:           "MASTERCARD",
		PayoutCurrency: "USD",
		UndoReason:     "reversal",
		Region:         "EU",
		Country:        "FI",
		DaysFrom:       0,
		PaymentStage:   1,
		Percent:        0.10,
		FixAmount:      0.15,
	}
	sysCost5 := &billing.MoneyBackCostSystem{
		Name:           "MASTERCARD",
		PayoutCurrency: "USD",
		UndoReason:     "chargeback",
		Region:         "Russia",
		Country:        "RU",
		DaysFrom:       0,
		PaymentStage:   1,
		Percent:        0.10,
		FixAmount:      0.15,
	}
	sysCost6 := &billing.MoneyBackCostSystem{
		Name:           "MASTERCARD",
		PayoutCurrency: "RUB",
		UndoReason:     "chargeback",
		Region:         "Russia",
		Country:        "RU",
		DaysFrom:       0,
		PaymentStage:   1,
		Percent:        0.10,
		FixAmount:      0.15,
	}
	sysCost7 := &billing.MoneyBackCostSystem{
		Name:           "MASTERCARD",
		PayoutCurrency: "USD",
		UndoReason:     "chargeback",
		Region:         "North America",
		Country:        "US",
		DaysFrom:       0,
		PaymentStage:   1,
		Percent:        0.10,
		FixAmount:      0.15,
	}
	sysCost8 := &billing.MoneyBackCostSystem{
		Name:           "MASTERCARD",
		PayoutCurrency: "USD",
		UndoReason:     "chargeback",
		Region:         "EU",
		Country:        "FI",
		DaysFrom:       0,
		PaymentStage:   1,
		Percent:        0.10,
		FixAmount:      0.15,
	}

	err = service.moneyBackCostSystem.MultipleInsert([]*billing.MoneyBackCostSystem{sysCost, sysCost2, sysCost3, sysCost4, sysCost5, sysCost6, sysCost7, sysCost8})

	if err != nil {
		suite.FailNow("Insert MoneyBackCostSystem test data failed", "%v", err)
	}

	paymentSysCost1 := &billing.PaymentChannelCostSystem{
		Name:              "MASTERCARD",
		Region:            "Russia",
		Country:           "RU",
		Percent:           0.015,
		FixAmount:         0.01,
		FixAmountCurrency: "USD",
	}
	paymentSysCost2 := &billing.PaymentChannelCostSystem{
		Name:              "MASTERCARD",
		Region:            "North America",
		Country:           "US",
		Percent:           0.015,
		FixAmount:         0.01,
		FixAmountCurrency: "USD",
	}
	paymentSysCost3 := &billing.PaymentChannelCostSystem{
		Name:              "MASTERCARD",
		Region:            "EU",
		Country:           "FI",
		Percent:           0.015,
		FixAmount:         0.01,
		FixAmountCurrency: "USD",
	}
	paymentSysCost4 := &billing.PaymentChannelCostSystem{
		Name:              "VISA",
		Region:            "Russia",
		Country:           "RU",
		Percent:           0.015,
		FixAmount:         0.01,
		FixAmountCurrency: "USD",
	}

	err = service.paymentChannelCostSystem.MultipleInsert([]*billing.PaymentChannelCostSystem{
		paymentSysCost1,
		paymentSysCost2,
		paymentSysCost3,
		paymentSysCost4,
	})

	if err != nil {
		suite.FailNow("Insert PaymentChannelCostSystem test data failed", "%v", err)
	}

	return merchant, projectFixedAmount, pmBankCard, paymentSystem
}

func helperCreateMerchant(
	suite suite.Suite,
	service *Service,
	currency string,
	country string,
	paymentMethod *billing.PaymentMethod,
	minPayoutAmount float64,
) *billing.Merchant {
	date, err := ptypes.TimestampProto(time.Now().Add(time.Hour * -360))

	if err != nil {
		suite.FailNow("Generate merchant date failed", "%v", err)
	}

	merchant := &billing.Merchant{
		Id: bson.NewObjectId().Hex(),
		Company: &billing.MerchantCompanyInfo{
			Name:    "Unit test",
			Country: country,
		},
		Contacts: &billing.MerchantContact{
			Authorized: &billing.MerchantContactAuthorized{
				Name:     "Unit Test",
				Email:    "test@unit.test",
				Phone:    "123456789",
				Position: "Unit Test",
			},
			Technical: &billing.MerchantContactTechnical{
				Name:  "Unit Test",
				Email: "test@unit.test",
				Phone: "123456789",
			},
		},
		Banking: &billing.MerchantBanking{
			Currency: currency,
			Name:     "Bank name",
		},
		IsVatEnabled:              true,
		MinPayoutAmount:           minPayoutAmount,
		IsCommissionToUserEnabled: true,
		Status:                    pkg.MerchantStatusDraft,
		LastPayout: &billing.MerchantLastPayout{
			Date:   date,
			Amount: 999999,
		},
		IsSigned:       true,
		PaymentMethods: map[string]*billing.MerchantPaymentMethod{},
		Tariff: &billing.MerchantTariffRates{
			Region: "USD",
			Chargeback: &billing.TariffRatesItem{
				FixedFee:         1,
				FixedFeeCurrency: "USD",
				IsPaidByMerchant: true,
			},
			MoneyBack: []*billing.MerchantTariffRatesMoneyBack{
				{Method: "VISA"},
			},
			Payment: []*billing.MerchantTariffRatesPayments{
				{Method: "VISA"},
			},
			Payout: &billing.TariffRatesItem{
				FixedFee:         1,
				FixedFeeCurrency: "USD",
				IsPaidByMerchant: true,
			},
		},
	}

	if paymentMethod != nil {
		merchant.PaymentMethods[paymentMethod.Id] = &billing.MerchantPaymentMethod{
			PaymentMethod: &billing.MerchantPaymentMethodIdentification{
				Id:   paymentMethod.Id,
				Name: paymentMethod.Name,
			},
			Commission: &billing.MerchantPaymentMethodCommissions{
				Fee: 2.5,
				PerTransaction: &billing.MerchantPaymentMethodPerTransactionCommission{
					Fee:      30,
					Currency: "RUB",
				},
			},
			Integration: &billing.MerchantPaymentMethodIntegration{
				TerminalId:               "15985",
				TerminalPassword:         "A1tph4I6BD0f",
				TerminalCallbackPassword: "0V1rJ7t4jCRv",
				Integrated:               true,
			},
			IsActive: true,
		}
	}

	merchants := []*billing.Merchant{merchant}
	if err := service.merchant.MultipleInsert(merchants); err != nil {
		suite.FailNow("Insert merchant test data failed", "%v", err)
	}

	merCost1 := &billing.MoneyBackCostMerchant{
		Id:                bson.NewObjectId().Hex(),
		MerchantId:        merchant.Id,
		Name:              "MASTERCARD",
		PayoutCurrency:    "USD",
		UndoReason:        "reversal",
		Region:            "Russia",
		Country:           "RU",
		DaysFrom:          0,
		PaymentStage:      1,
		Percent:           0.2,
		FixAmount:         0.15,
		FixAmountCurrency: "EUR",
		IsPaidByMerchant:  false,
	}

	merCost2 := &billing.MoneyBackCostMerchant{
		Id:                bson.NewObjectId().Hex(),
		MerchantId:        merchant.Id,
		Name:              "MASTERCARD",
		PayoutCurrency:    "RUB",
		UndoReason:        "reversal",
		Region:            "Russia",
		Country:           "RU",
		DaysFrom:          0,
		PaymentStage:      1,
		Percent:           0.2,
		FixAmount:         0.15,
		FixAmountCurrency: "EUR",
		IsPaidByMerchant:  false,
	}

	merCost3 := &billing.MoneyBackCostMerchant{
		Id:                bson.NewObjectId().Hex(),
		MerchantId:        merchant.Id,
		Name:              "MASTERCARD",
		PayoutCurrency:    "USD",
		UndoReason:        "reversal",
		Region:            "North America",
		Country:           "US",
		DaysFrom:          0,
		PaymentStage:      1,
		Percent:           0.2,
		FixAmount:         0.15,
		FixAmountCurrency: "EUR",
		IsPaidByMerchant:  false,
	}

	merCost4 := &billing.MoneyBackCostMerchant{
		Id:                bson.NewObjectId().Hex(),
		MerchantId:        merchant.Id,
		Name:              "MASTERCARD",
		PayoutCurrency:    "USD",
		UndoReason:        "reversal",
		Region:            "EU",
		Country:           "FI",
		DaysFrom:          0,
		PaymentStage:      1,
		Percent:           0.2,
		FixAmount:         0.15,
		FixAmountCurrency: "EUR",
		IsPaidByMerchant:  false,
	}

	merCost5 := &billing.MoneyBackCostMerchant{
		Id:                bson.NewObjectId().Hex(),
		MerchantId:        merchant.Id,
		Name:              "MASTERCARD",
		PayoutCurrency:    "USD",
		UndoReason:        "chargeback",
		Region:            "Russia",
		Country:           "RU",
		DaysFrom:          0,
		PaymentStage:      1,
		Percent:           0.2,
		FixAmount:         0.15,
		FixAmountCurrency: "EUR",
		IsPaidByMerchant:  true,
	}

	merCost6 := &billing.MoneyBackCostMerchant{
		Id:                bson.NewObjectId().Hex(),
		MerchantId:        merchant.Id,
		Name:              "MASTERCARD",
		PayoutCurrency:    "RUB",
		UndoReason:        "chargeback",
		Region:            "Russia",
		Country:           "RU",
		DaysFrom:          0,
		PaymentStage:      1,
		Percent:           0.2,
		FixAmount:         0.15,
		FixAmountCurrency: "EUR",
		IsPaidByMerchant:  true,
	}

	merCost7 := &billing.MoneyBackCostMerchant{
		Id:                bson.NewObjectId().Hex(),
		MerchantId:        merchant.Id,
		Name:              "MASTERCARD",
		PayoutCurrency:    "USD",
		UndoReason:        "chargeback",
		Region:            "North America",
		Country:           "US",
		DaysFrom:          0,
		PaymentStage:      1,
		Percent:           0.2,
		FixAmount:         0.15,
		FixAmountCurrency: "EUR",
		IsPaidByMerchant:  true,
	}

	merCost8 := &billing.MoneyBackCostMerchant{
		Id:                bson.NewObjectId().Hex(),
		MerchantId:        merchant.Id,
		Name:              "MASTERCARD",
		PayoutCurrency:    "USD",
		UndoReason:        "chargeback",
		Region:            "EU",
		Country:           "FI",
		DaysFrom:          0,
		PaymentStage:      1,
		Percent:           0.2,
		FixAmount:         0.15,
		FixAmountCurrency: "EUR",
		IsPaidByMerchant:  true,
	}

	err = service.moneyBackCostMerchant.MultipleInsert([]*billing.MoneyBackCostMerchant{merCost1, merCost2, merCost3, merCost4, merCost5, merCost6, merCost7, merCost8})

	if err != nil {
		suite.FailNow("Insert MoneyBackCostMerchant test data failed", "%v", err)
	}

	paymentMerCost1 := &billing.PaymentChannelCostMerchant{
		MerchantId:              merchant.Id,
		Name:                    "MASTERCARD",
		PayoutCurrency:          "USD",
		MinAmount:               0,
		Region:                  "Russia",
		Country:                 "RU",
		MethodPercent:           0.025,
		MethodFixAmount:         0.02,
		MethodFixAmountCurrency: "EUR",
		PsPercent:               0.05,
		PsFixedFee:              0.05,
		PsFixedFeeCurrency:      "EUR",
	}
	paymentMerCost2 := &billing.PaymentChannelCostMerchant{
		MerchantId:              merchant.Id,
		Name:                    "MASTERCARD",
		PayoutCurrency:          "RUB",
		MinAmount:               0,
		Region:                  "Russia",
		Country:                 "RU",
		MethodPercent:           0.025,
		MethodFixAmount:         0.02,
		MethodFixAmountCurrency: "EUR",
		PsPercent:               0.05,
		PsFixedFee:              0.05,
		PsFixedFeeCurrency:      "EUR",
	}
	paymentMerCost3 := &billing.PaymentChannelCostMerchant{
		MerchantId:              merchant.Id,
		Name:                    "MASTERCARD",
		PayoutCurrency:          "USD",
		MinAmount:               0,
		Region:                  "North America",
		Country:                 "US",
		MethodPercent:           0.025,
		MethodFixAmount:         0.02,
		MethodFixAmountCurrency: "EUR",
		PsPercent:               0.05,
		PsFixedFee:              0.05,
		PsFixedFeeCurrency:      "EUR",
	}
	paymentMerCost4 := &billing.PaymentChannelCostMerchant{
		MerchantId:              merchant.Id,
		Name:                    "MASTERCARD",
		PayoutCurrency:          "USD",
		MinAmount:               0,
		Region:                  "EU",
		Country:                 "FI",
		MethodPercent:           0.025,
		MethodFixAmount:         0.02,
		MethodFixAmountCurrency: "EUR",
		PsPercent:               0.05,
		PsFixedFee:              0.05,
		PsFixedFeeCurrency:      "EUR",
	}

	err = service.paymentChannelCostMerchant.MultipleInsert([]*billing.PaymentChannelCostMerchant{paymentMerCost1, paymentMerCost2, paymentMerCost3, paymentMerCost4})

	if err != nil {
		suite.FailNow("Insert PaymentChannelCostMerchant test data failed", "%v", err)
	}

	return merchant
}

func helperCreateAndPayOrder(
	suite suite.Suite,
	service *Service,
	amount float64,
	currency, country string,
	project *billing.Project,
	paymentMethod *billing.PaymentMethod,
) *billing.Order {
	req := &billing.OrderCreateRequest{
		Type:        billing.OrderType_simple,
		ProjectId:   project.Id,
		Amount:      amount,
		Currency:    currency,
		Account:     "unit test",
		Description: "unit test",
		OrderId:     bson.NewObjectId().Hex(),
		User: &billing.OrderUser{
			Email: "test@unit.unit",
			Ip:    "127.0.0.1",
			Address: &billing.OrderBillingAddress{
				Country: country,
			},
		},
	}

	rsp := &grpc.OrderCreateProcessResponse{}
	err := service.OrderCreateProcess(context.TODO(), req, rsp)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), rsp.Status, pkg.ResponseStatusOk)

	req1 := &grpc.PaymentCreateRequest{
		Data: map[string]string{
			pkg.PaymentCreateFieldOrderId:         rsp.Item.Uuid,
			pkg.PaymentCreateFieldPaymentMethodId: paymentMethod.Id,
			pkg.PaymentCreateFieldEmail:           "test@unit.unit",
			pkg.PaymentCreateFieldPan:             "4000000000000002",
			pkg.PaymentCreateFieldCvv:             "123",
			pkg.PaymentCreateFieldMonth:           "02",
			pkg.PaymentCreateFieldYear:            time.Now().AddDate(1, 0, 0).Format("2006"),
			pkg.PaymentCreateFieldHolder:          "MR. CARD HOLDER",
		},
		Ip: "127.0.0.1",
	}

	rsp1 := &grpc.PaymentCreateResponse{}
	err = service.PaymentCreateProcess(context.TODO(), req1, rsp1)
	assert.NoError(suite.T(), err)
	assert.Equalf(suite.T(), pkg.ResponseStatusOk, rsp1.Status, "%v", rsp1.Message)

	var order *billing.Order
	err = service.db.Collection(collectionOrder).FindId(bson.ObjectIdHex(rsp.Item.Id)).One(&order)
	assert.NotNil(suite.T(), order)
	assert.IsType(suite.T(), &billing.Order{}, order)

	callbackRequest := &billing.CardPayPaymentCallback{
		PaymentMethod: paymentMethod.ExternalId,
		CallbackTime:  time.Now().Format("2006-01-02T15:04:05Z"),
		MerchantOrder: &billing.CardPayMerchantOrder{
			Id:          rsp.Item.Id,
			Description: rsp.Item.Description,
		},
		CardAccount: &billing.CallbackCardPayBankCardAccount{
			Holder:             order.PaymentRequisites[pkg.PaymentCreateFieldHolder],
			IssuingCountryCode: country,
			MaskedPan:          order.PaymentRequisites[pkg.PaymentCreateFieldPan],
			Token:              bson.NewObjectId().Hex(),
		},
		Customer: &billing.CardPayCustomer{
			Email:  rsp.Item.User.Email,
			Ip:     rsp.Item.User.Ip,
			Id:     rsp.Item.ProjectAccount,
			Locale: "Europe/Moscow",
		},
		PaymentData: &billing.CallbackCardPayPaymentData{
			Id:          bson.NewObjectId().Hex(),
			Amount:      order.TotalPaymentAmount,
			Currency:    order.Currency,
			Description: order.Description,
			Is_3D:       true,
			Rrn:         bson.NewObjectId().Hex(),
			Status:      pkg.CardPayPaymentResponseStatusCompleted,
		},
	}

	buf, err := json.Marshal(callbackRequest)
	assert.NoError(suite.T(), err)

	hash := sha512.New()
	hash.Write([]byte(string(buf) + order.PaymentMethod.Params.SecretCallback))

	callbackData := &grpc.PaymentNotifyRequest{
		OrderId:   order.Id,
		Request:   buf,
		Signature: hex.EncodeToString(hash.Sum(nil)),
	}

	callbackResponse := &grpc.PaymentNotifyResponse{}
	err = service.PaymentCallbackProcess(context.TODO(), callbackData, callbackResponse)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), pkg.StatusOK, callbackResponse.Status)

	err = service.db.Collection(collectionOrder).FindId(bson.ObjectIdHex(order.Id)).One(&order)
	assert.NotNil(suite.T(), order)
	assert.IsType(suite.T(), &billing.Order{}, order)
	assert.Equal(suite.T(), int32(constant.OrderStatusPaymentSystemComplete), order.PrivateStatus)

	return order
}

func helperMakeRefund(suite suite.Suite, service *Service, order *billing.Order, amount float64, isChargeback bool) *billing.Refund {
	req2 := &grpc.CreateRefundRequest{
		OrderId:      order.Uuid,
		Amount:       amount,
		CreatorId:    bson.NewObjectId().Hex(),
		Reason:       "unit test",
		IsChargeback: isChargeback,
	}
	rsp2 := &grpc.CreateRefundResponse{}
	err := service.CreateRefund(context.TODO(), req2, rsp2)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), pkg.ResponseStatusOk, rsp2.Status)
	assert.Empty(suite.T(), rsp2.Message)

	err = service.updateOrder(order)
	assert.NoError(suite.T(), err)

	refundReq := &billing.CardPayRefundCallback{
		MerchantOrder: &billing.CardPayMerchantOrder{
			Id: rsp2.Item.Id,
		},
		PaymentMethod: order.PaymentMethod.Group,
		PaymentData: &billing.CardPayRefundCallbackPaymentData{
			Id:              rsp2.Item.Id,
			RemainingAmount: 90,
		},
		RefundData: &billing.CardPayRefundCallbackRefundData{
			Amount:   10,
			Created:  time.Now().Format(cardPayDateFormat),
			Id:       bson.NewObjectId().Hex(),
			Currency: rsp2.Item.Currency,
			Status:   pkg.CardPayPaymentResponseStatusCompleted,
			AuthCode: bson.NewObjectId().Hex(),
			Is_3D:    true,
			Rrn:      bson.NewObjectId().Hex(),
		},
		CallbackTime: time.Now().Format(cardPayDateFormat),
		Customer: &billing.CardPayCustomer{
			Email: order.User.Email,
			Id:    order.User.Email,
		},
	}

	b, err := json.Marshal(refundReq)
	assert.NoError(suite.T(), err)

	hash := sha512.New()
	hash.Write([]byte(string(b) + order.PaymentMethod.Params.SecretCallback))

	req3 := &grpc.CallbackRequest{
		Handler:   pkg.PaymentSystemHandlerCardPay,
		Body:      b,
		Signature: hex.EncodeToString(hash.Sum(nil)),
	}
	rsp3 := &grpc.PaymentNotifyResponse{}
	err = service.ProcessRefundCallback(context.TODO(), req3, rsp3)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), pkg.ResponseStatusOk, rsp3.Status)
	assert.Empty(suite.T(), rsp3.Error)

	var refund *billing.Refund
	err = service.db.Collection(collectionRefund).FindId(bson.ObjectIdHex(rsp2.Item.Id)).One(&refund)
	assert.NotNil(suite.T(), refund)
	assert.Equal(suite.T(), pkg.RefundStatusCompleted, refund.Status)

	return refund
}

func createProductsForProject(
	suite suite.Suite,
	service *Service,
	project *billing.Project,
	productsCount int,
) []*grpc.Product {
	products := make([]*grpc.Product, 1)

	for i := 0; i < productsCount; i++ {
		name := "ru_test_product_" + strconv.Itoa(i)
		req := &grpc.Product{
			Object:          "product",
			Type:            "simple_product",
			Sku:             name,
			Name:            map[string]string{"en": name},
			DefaultCurrency: "USD",
			Enabled:         true,
			Description:     map[string]string{"en": name + " description"},
			MerchantId:      project.MerchantId,
			ProjectId:       project.Id,
		}

		baseAmount := 37.00 * float64(i+1)

		req.Prices = append(req.Prices, &billing.ProductPrice{
			Currency: "USD",
			Region:   "USD",
			Amount:   baseAmount,
		})
		req.Prices = append(req.Prices, &billing.ProductPrice{
			Currency: "RUB",
			Region:   "RUB",
			Amount:   baseAmount * 65.13,
		})

		prod := &grpc.Product{}
		err := service.CreateOrUpdateProduct(context.TODO(), req, prod)

		if err != nil {
			suite.FailNow("Add products for project failed", "%v", err)
		}

		products = append(products, prod)
	}

	return products
}

func createKeyProductsFroProject(
	suite suite.Suite,
	service *Service,
	project *billing.Project,
	productsCount int,
) []*grpc.KeyProduct {
	products := make([]*grpc.KeyProduct, 1)

	for i := 0; i < productsCount; i++ {
		baseAmount := 37.00 * float64(i+1)

		name := "ru_test_key_product_" + strconv.Itoa(i)
		req := &grpc.CreateOrUpdateKeyProductRequest{
			Object:          "key_product",
			Sku:             name,
			Name:            map[string]string{"en": name},
			DefaultCurrency: "USD",
			Description:     map[string]string{"en": name + " description"},
			MerchantId:      project.MerchantId,
			ProjectId:       project.Id,
			Platforms: []*grpc.PlatformPrice{
				{
					Id: "steam",
					Prices: []*billing.ProductPrice{
						{
							Currency: "USD",
							Region:   "USD",
							Amount:   baseAmount,
						},
						{
							Currency: "RUB",
							Region:   "RUB",
							Amount:   baseAmount * 65.13,
						},
					},
				},
			},
		}
		rsp := &grpc.KeyProductResponse{}
		err := service.CreateOrUpdateKeyProduct(context.TODO(), req, rsp)

		if err != nil {
			suite.FailNow("Add key products for project failed", "%v", err)
		}

		req2 := &grpc.PublishKeyProductRequest{
			MerchantId:   project.MerchantId,
			KeyProductId: rsp.Product.Id,
		}
		rsp2 := &grpc.KeyProductResponse{}
		err = service.PublishKeyProduct(context.TODO(), req2, rsp2)

		if err != nil {
			suite.FailNow("Publishing key product for project failed", "%v", err)
		}

		fileContent := fmt.Sprintf("%s-%s-%s-%s", GenerateRandomString(4), GenerateRandomString(4), GenerateRandomString(4), GenerateRandomString(4))
		file := []byte(fileContent)

		req3 := &grpc.PlatformKeysFileRequest{
			KeyProductId: rsp.Product.Id,
			PlatformId:   "steam",
			MerchantId:   project.MerchantId,
			File:         file,
		}
		rsp3 := &grpc.PlatformKeysFileResponse{}
		err = service.UploadKeysFile(context.TODO(), req3, rsp3)

		if err != nil {
			suite.FailNow("Upload keys to key product for project failed", "%v", err)
		}

		products = append(products, rsp.Product)
	}

	return products
}

func helperCreateAndPayOrder2(
	suite suite.Suite,
	service *Service,
	amount float64,
	currency, country string,
	project *billing.Project,
	paymentMethod *billing.PaymentMethod,
	paymentMethodClosedAt time.Time,
	product *grpc.Product,
	keyProduct *grpc.KeyProduct,
	issuerUrl string,
) *billing.Order {
	req := &billing.OrderCreateRequest{
		ProjectId:   project.Id,
		Account:     "unit test",
		Description: "unit test",
		OrderId:     bson.NewObjectId().Hex(),
		User: &billing.OrderUser{
			Email: "test@unit.unit",
			Ip:    "127.0.0.1",
			Address: &billing.OrderBillingAddress{
				Country: country,
			},
		},
		IssuerUrl: issuerUrl,
	}

	if product != nil {
		req.Type = billing.OrderType_product
		req.Products = []string{product.Id}
	} else if keyProduct != nil {
		req.Type = billing.OrderType_key
		req.Products = []string{keyProduct.Id}
	} else {
		if amount <= 0 || currency == "" {
			suite.FailNow("Order creation failed because request hasn't required fields", "%v")
		}

		req.Type = billing.OrderType_simple
		req.Amount = amount
		req.Currency = currency
	}

	rsp := &grpc.OrderCreateProcessResponse{}
	err := service.OrderCreateProcess(context.TODO(), req, rsp)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), rsp.Status, pkg.ResponseStatusOk)

	req1 := &grpc.PaymentCreateRequest{
		Data: map[string]string{
			pkg.PaymentCreateFieldOrderId:         rsp.Item.Uuid,
			pkg.PaymentCreateFieldPaymentMethodId: paymentMethod.Id,
			pkg.PaymentCreateFieldEmail:           "test@unit.unit",
			pkg.PaymentCreateFieldPan:             "4000000000000002",
			pkg.PaymentCreateFieldCvv:             "123",
			pkg.PaymentCreateFieldMonth:           "02",
			pkg.PaymentCreateFieldYear:            time.Now().AddDate(1, 0, 0).Format("2006"),
			pkg.PaymentCreateFieldHolder:          "MR. CARD HOLDER",
		},
		Ip: "127.0.0.1",
	}

	rsp1 := &grpc.PaymentCreateResponse{}
	err = service.PaymentCreateProcess(context.TODO(), req1, rsp1)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), pkg.ResponseStatusOk, rsp1.Status)

	var order *billing.Order
	err = service.db.Collection(collectionOrder).FindId(bson.ObjectIdHex(rsp.Item.Id)).One(&order)
	assert.NotNil(suite.T(), order)
	assert.IsType(suite.T(), &billing.Order{}, order)

	callbackRequest := &billing.CardPayPaymentCallback{
		PaymentMethod: paymentMethod.ExternalId,
		CallbackTime:  paymentMethodClosedAt.Format("2006-01-02T15:04:05Z"),
		MerchantOrder: &billing.CardPayMerchantOrder{
			Id:          rsp.Item.Id,
			Description: rsp.Item.Description,
		},
		CardAccount: &billing.CallbackCardPayBankCardAccount{
			Holder:             order.PaymentRequisites[pkg.PaymentCreateFieldHolder],
			IssuingCountryCode: country,
			MaskedPan:          order.PaymentRequisites[pkg.PaymentCreateFieldPan],
			Token:              bson.NewObjectId().Hex(),
		},
		Customer: &billing.CardPayCustomer{
			Email:  rsp.Item.User.Email,
			Ip:     rsp.Item.User.Ip,
			Id:     rsp.Item.ProjectAccount,
			Locale: "Europe/Moscow",
		},
		PaymentData: &billing.CallbackCardPayPaymentData{
			Id:          bson.NewObjectId().Hex(),
			Amount:      order.TotalPaymentAmount,
			Currency:    order.Currency,
			Description: order.Description,
			Is_3D:       true,
			Rrn:         bson.NewObjectId().Hex(),
			Status:      pkg.CardPayPaymentResponseStatusCompleted,
		},
	}

	buf, err := json.Marshal(callbackRequest)
	assert.NoError(suite.T(), err)

	hash := sha512.New()
	hash.Write([]byte(string(buf) + order.PaymentMethod.Params.SecretCallback))

	callbackData := &grpc.PaymentNotifyRequest{
		OrderId:   order.Id,
		Request:   buf,
		Signature: hex.EncodeToString(hash.Sum(nil)),
	}

	callbackResponse := &grpc.PaymentNotifyResponse{}
	err = service.PaymentCallbackProcess(context.TODO(), callbackData, callbackResponse)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), pkg.StatusOK, callbackResponse.Status)

	err = service.db.Collection(collectionOrder).FindId(bson.ObjectIdHex(order.Id)).One(&order)
	assert.NotNil(suite.T(), order)
	assert.IsType(suite.T(), &billing.Order{}, order)
	assert.Equal(suite.T(), int32(constant.OrderStatusPaymentSystemComplete), order.PrivateStatus)

	return order
}

func GenerateRandomString(n int) string {
	var letter = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	b := make([]rune, n)
	for i := range b {
		b[i] = letter[rand.Intn(len(letter))]
	}
	return string(b)
}
