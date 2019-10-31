package service

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ProtocolONE/geoip-service/pkg/proto"
	"github.com/dgrijalva/jwt-go"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	protobuf "github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/google/uuid"
	"github.com/paysuper/paysuper-billing-server/pkg"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/billing"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/grpc"
	curPkg "github.com/paysuper/paysuper-currencies/pkg"
	"github.com/paysuper/paysuper-currencies/pkg/proto/currencies"
	"github.com/paysuper/paysuper-recurring-repository/pkg/constant"
	"github.com/paysuper/paysuper-recurring-repository/pkg/proto/entity"
	repo "github.com/paysuper/paysuper-recurring-repository/pkg/proto/repository"
	"github.com/paysuper/paysuper-recurring-repository/tools"
	"github.com/paysuper/paysuper-tax-service/proto"
	postmarkSdrPkg "github.com/paysuper/postmark-sender/pkg"
	"github.com/streadway/amqp"
	"github.com/ttacon/libphonenumber"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	paymentRequestIncorrect             = "payment request has incorrect format"
	callbackRequestIncorrect            = "callback request has incorrect format"
	callbackHandlerIncorrect            = "unknown callback type"
	orderErrorPublishNotificationFailed = "publish order notification failed"
	orderErrorUpdateOrderDataFailed     = "update order data failed"

	paymentCreateBankCardFieldBrand         = "card_brand"
	paymentCreateBankCardFieldType          = "card_type"
	paymentCreateBankCardFieldCategory      = "card_category"
	paymentCreateBankCardFieldIssuerName    = "bank_issuer_name"
	paymentCreateBankCardFieldIssuerCountry = "bank_issuer_country"

	orderDefaultDescription = "Payment by order # %s"

	defaultExpireDateToFormInput = 30
	cookieCounterUpdateTime      = 1800

	taxTypeVat      = "vat"
	taxTypeSalesTax = "sales_tax"

	collectionOrder           = "order"
	collectionBinData         = "bank_bin"
	collectionNotifySales     = "notify_sales"
	collectionNotifyNewRegion = "notify_new_region"

	processPaylinkKeyProductsTemplate      = "[processPaylinkKeyProducts] %s"
	processProcessOrderKeyProductsTemplate = "[ProcessOrderKeyProducts] %s"
)

var (
	orderErrorProjectIdIncorrect                              = newBillingServerErrorMsg("fm000001", "project identifier is incorrect")
	orderErrorProjectNotFound                                 = newBillingServerErrorMsg("fm000002", "project with specified identifier not found")
	orderErrorProjectInactive                                 = newBillingServerErrorMsg("fm000003", "project with specified identifier is inactive")
	orderErrorProjectMerchantInactive                         = newBillingServerErrorMsg("fm000004", "merchant for project with specified identifier is inactive")
	orderErrorPaymentMethodNotAllowed                         = newBillingServerErrorMsg("fm000005", "payment method not available for project")
	orderErrorPaymentMethodNotFound                           = newBillingServerErrorMsg("fm000006", "payment method with specified identifier not found")
	orderErrorPaymentMethodInactive                           = newBillingServerErrorMsg("fm000007", "payment method with specified identifier is inactive")
	orderErrorConvertionCurrency                              = newBillingServerErrorMsg("fm000008", "currency convertion error")
	orderErrorPaymentMethodEmptySettings                      = newBillingServerErrorMsg("fm000009", "payment method setting for project is empty")
	orderErrorPaymentSystemInactive                           = newBillingServerErrorMsg("fm000010", "payment system for specified payment method is inactive")
	orderErrorPayerRegionUnknown                              = newBillingServerErrorMsg("fm000011", "payer region can't be found")
	orderErrorProjectOrderIdIsDuplicate                       = newBillingServerErrorMsg("fm000012", "request with specified project order identifier processed early")
	orderErrorDynamicNotifyUrlsNotAllowed                     = newBillingServerErrorMsg("fm000013", "dynamic verify url or notify url not allowed for project")
	orderErrorDynamicRedirectUrlsNotAllowed                   = newBillingServerErrorMsg("fm000014", "dynamic payer redirect urls not allowed for project")
	orderErrorCurrencyNotFound                                = newBillingServerErrorMsg("fm000015", "currency received from request not found")
	orderErrorAmountLowerThanMinAllowed                       = newBillingServerErrorMsg("fm000016", "order amount is lower than min allowed payment amount for project")
	orderErrorAmountGreaterThanMaxAllowed                     = newBillingServerErrorMsg("fm000017", "order amount is greater than max allowed payment amount for project")
	orderErrorAmountLowerThanMinAllowedPaymentMethod          = newBillingServerErrorMsg("fm000018", "order amount is lower than min allowed payment amount for payment method")
	orderErrorAmountGreaterThanMaxAllowedPaymentMethod        = newBillingServerErrorMsg("fm000019", "order amount is greater than max allowed payment amount for payment method")
	orderErrorCanNotCreate                                    = newBillingServerErrorMsg("fm000020", "order can't create. try request later")
	orderErrorNotFound                                        = newBillingServerErrorMsg("fm000021", "order with specified identifier not found")
	orderErrorOrderCreatedAnotherProject                      = newBillingServerErrorMsg("fm000022", "order created for another project")
	orderErrorFormInputTimeExpired                            = newBillingServerErrorMsg("fm000023", "time to enter date on payment form expired")
	orderErrorCurrencyIsRequired                              = newBillingServerErrorMsg("fm000024", "parameter currency in create order request is required")
	orderErrorUnknown                                         = newBillingServerErrorMsg("fm000025", "unknown error. try request later")
	orderCountryPaymentRestrictedError                        = newBillingServerErrorMsg("fm000027", "payments from your country are not allowed")
	orderGetSavedCardError                                    = newBillingServerErrorMsg("fm000028", "saved card data with specified identifier not found")
	orderErrorCountryByPaymentAccountNotFound                 = newBillingServerErrorMsg("fm000029", "information about user country can't be found")
	orderErrorPaymentAccountIncorrect                         = newBillingServerErrorMsg("fm000030", "account in payment system is incorrect")
	orderErrorProductsEmpty                                   = newBillingServerErrorMsg("fm000031", "products set is empty")
	orderErrorProductsInvalid                                 = newBillingServerErrorMsg("fm000032", "some products in set are invalid or inactive")
	orderErrorNoProductsCommonCurrency                        = newBillingServerErrorMsg("fm000033", "no common prices neither in requested currency nor in default currency")
	orderErrorNoNameInDefaultLanguage                         = newBillingServerErrorMsg("fm000034", "no name in default language %s")
	orderErrorNoNameInRequiredLanguage                        = newBillingServerErrorMsg("fm000035", "no name in required language %s")
	orderErrorNoDescriptionInDefaultLanguage                  = newBillingServerErrorMsg("fm000036", "no description in default language %s")
	orderErrorNoDescriptionInRequiredLanguage                 = newBillingServerErrorMsg("fm000037", "no description in required language %s")
	orderErrorProjectMerchantNotFound                         = newBillingServerErrorMsg("fm000038", "merchant for project with specified identifier not found")
	orderErrorRecurringCardNotOwnToUser                       = newBillingServerErrorMsg("fm000039", "you can't use not own bank card for payment")
	orderErrorNotRestricted                                   = newBillingServerErrorMsg("fm000040", "order country not restricted")
	orderErrorEmailRequired                                   = newBillingServerErrorMsg("fm000041", "email is required")
	orderErrorCreatePaymentRequiredFieldIdNotFound            = newBillingServerErrorMsg("fm000042", "required field with order identifier not found")
	orderErrorCreatePaymentRequiredFieldPaymentMethodNotFound = newBillingServerErrorMsg("fm000043", "required field with payment method identifier not found")
	orderErrorCreatePaymentRequiredFieldEmailNotFound         = newBillingServerErrorMsg("fm000044", "required field \"email\" not found")
	orderErrorCreatePaymentRequiredFieldUserCountryNotFound   = newBillingServerErrorMsg("fm000045", "user country is required")
	orderErrorCreatePaymentRequiredFieldUserZipNotFound       = newBillingServerErrorMsg("fm000046", "user zip is required")
	orderErrorOrderAlreadyComplete                            = newBillingServerErrorMsg("fm000047", "order with specified identifier payed early")
	orderErrorSignatureInvalid                                = newBillingServerErrorMsg("fm000048", "request signature is invalid")
	orderErrorZipCodeNotFound                                 = newBillingServerErrorMsg("fm000050", "zip_code not found")
	orderErrorProductsPrice                                   = newBillingServerErrorMsg("fm000051", "can't get product price")
	orderErrorCheckoutWithoutProducts                         = newBillingServerErrorMsg("fm000052", "order products not specified")
	orderErrorCheckoutWithoutAmount                           = newBillingServerErrorMsg("fm000053", "order amount not specified")
	orderErrorUnknownType                                     = newBillingServerErrorMsg("fm000055", "unknown type of order")
	orderErrorMerchantBadTariffs                              = newBillingServerErrorMsg("fm000056", "merchant don't have tariffs")
	orderErrorReceiptNotEquals                                = newBillingServerErrorMsg("fm000057", "receipts not equals")
	orderErrorDuringFormattingCurrency                        = newBillingServerErrorMsg("fm000058", "error during formatting currency")
	orderErrorDuringFormattingDate                            = newBillingServerErrorMsg("fm000059", "error during formatting date")
	orderErrorMerchantForOrderNotFound                        = newBillingServerErrorMsg("fm000060", "merchant for order not found")
	orderErrorPaymentMethodsNotFound                          = newBillingServerErrorMsg("fm000061", "payment methods for payment with specified currency not found")
	orderErrorNoPlatforms                                     = newBillingServerErrorMsg("fm000062", "no available platforms")
	orderCountryPaymentRestrictedEmailRequire                 = newBillingServerErrorMsg("fm000063", "payments from your country are not allowed")
	orderErrorCostsRatesNotFound                              = newBillingServerErrorMsg("fm000064", "settings to calculate commissions not found")
	orderErrorVirtualCurrencyNotFilled                        = newBillingServerErrorMsg("fm000065", "virtual currency is not filled")
	orderErrorVirtualCurrencyFracNotSupported                 = newBillingServerErrorMsg("fm000066", "fractional numbers is not supported for this virtual currency")
	orderErrorVirtualCurrencyLimits                           = newBillingServerErrorMsg("fm000067", "amount of order is more than max amount or less than minimal amount for virtual currency")
	orderErrorVirtualCurrencyUserCountryRequired              = newBillingServerErrorMsg("fm000068", "request for create payment by project virtual currency must contain user data with required field country")
	orderErrorCheckoutWithProducts                            = newBillingServerErrorMsg("fm000069", "request to processing simple payment can't contain products list")

	virtualCurrencyPayoutCurrencyMissed       = newBillingServerErrorMsg("vc000001", "virtual currency don't have price in merchant payout currency")
)

type orderCreateRequestProcessorChecked struct {
	id              string
	project         *billing.Project
	merchant        *billing.Merchant
	currency        string
	amount          float64
	paymentMethod   *billing.PaymentMethod
	products        []string
	items           []*billing.OrderItem
	metadata        map[string]string
	privateMetadata map[string]string
	user            *billing.OrderUser
	virtualAmount   float64
}

type OrderCreateRequestProcessor struct {
	*Service
	checked *orderCreateRequestProcessorChecked
	request *billing.OrderCreateRequest
}

type PaymentFormProcessor struct {
	service *Service
	order   *billing.Order
	request *grpc.PaymentFormJsonDataRequest
}

type PaymentCreateProcessor struct {
	service        *Service
	data           map[string]string
	ip             string
	acceptLanguage string
	userAgent      string
	checked        struct {
		order         *billing.Order
		project       *billing.Project
		paymentMethod *billing.PaymentMethod
	}
}

type BinData struct {
	Id                 bson.ObjectId `bson:"_id"`
	CardBin            int32         `bson:"card_bin"`
	CardBrand          string        `bson:"card_brand"`
	CardType           string        `bson:"card_type"`
	CardCategory       string        `bson:"card_category"`
	BankName           string        `bson:"bank_name"`
	BankCountryName    string        `bson:"bank_country_name"`
	BankCountryIsoCode string        `bson:"bank_country_code_a2"`
	BankSite           string        `bson:"bank_site"`
	BankPhone          string        `bson:"bank_phone"`
}

func (s *Service) OrderCreateByPaylink(
	ctx context.Context,
	req *billing.OrderCreateByPaylink,
	rsp *grpc.OrderCreateProcessResponse,
) error {
	pl, err := s.paylinkService.GetById(req.PaylinkId)
	if err != nil {
		if err == mgo.ErrNotFound {
			rsp.Status = pkg.ResponseStatusNotFound
			rsp.Message = errorPaylinkNotFound
			return nil
		}
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			rsp.Status = pkg.ResponseStatusBadData
			rsp.Message = e
			return nil
		}
		return err
	}

	if pl.GetIsExpired() == true {
		rsp.Status = pkg.ResponseStatusGone
		rsp.Message = errorPaylinkExpired
		return nil
	}

	oReq := &billing.OrderCreateRequest{
		ProjectId: pl.ProjectId,
		PayerIp:   req.PayerIp,
		Products:  pl.Products,
		PrivateMetadata: map[string]string{
			"PaylinkId": pl.Id,
		},
		Type:                pl.ProductsType,
		IssuerUrl:           req.IssuerUrl,
		IsEmbedded:          req.IsEmbedded,
		IssuerReferenceType: pkg.OrderIssuerReferenceTypePaylink,
		IssuerReference:     pl.Id,
		UtmSource:           req.UtmSource,
		UtmMedium:           req.UtmMedium,
		UtmCampaign:         req.UtmCampaign,
	}

	err = s.OrderCreateProcess(ctx, oReq, rsp)
	if err != nil {
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			rsp.Status = pkg.ResponseStatusBadData
			rsp.Message = e
			return nil
		}
		return err
	}

	return nil
}

func (s *Service) OrderCreateProcess(
	ctx context.Context,
	req *billing.OrderCreateRequest,
	rsp *grpc.OrderCreateProcessResponse,
) error {

	rsp.Status = pkg.ResponseStatusOk

	processor := &OrderCreateRequestProcessor{
		Service: s,
		request: req,
		checked: &orderCreateRequestProcessorChecked{},
	}

	if req.Token != "" {
		err := processor.processCustomerToken()

		if err != nil {
			zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
			if e, ok := err.(*grpc.ResponseErrorMessage); ok {
				rsp.Status = pkg.ResponseStatusBadData
				rsp.Message = e
				return nil
			}
			return err
		}
	} else {
		if req.ProjectId == "" || bson.IsObjectIdHex(req.ProjectId) == false {
			rsp.Status = pkg.ResponseStatusBadData
			rsp.Message = orderErrorProjectIdIncorrect
			return nil
		}
	}

	if err := processor.processProject(); err != nil {
		zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			rsp.Status = pkg.ResponseStatusBadData
			rsp.Message = e
			return nil
		}
		return err
	}

	if err := processor.processMerchant(); err != nil {
		zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			rsp.Status = pkg.ResponseStatusBadData
			rsp.Message = e
			return nil
		}
		return err
	}

	if req.Signature != "" || processor.checked.project.SignatureRequired == true {
		if err := processor.processSignature(); err != nil {
			zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
			if e, ok := err.(*grpc.ResponseErrorMessage); ok {
				rsp.Status = pkg.ResponseStatusBadData
				rsp.Message = e
				return nil
			}
			return err
		}
	}

	switch req.Type {
	case billing.OrderType_simple, billing.OrderTypeVirtualCurrency:
		if req.Products != nil {
			rsp.Status = pkg.ResponseStatusBadData
			rsp.Message = orderErrorCheckoutWithProducts
			return nil
		}

		if req.Amount <= 0 {
			rsp.Status = pkg.ResponseStatusBadData
			rsp.Message = orderErrorCheckoutWithoutAmount
			return nil
		}
		break
	case billing.OrderType_product, billing.OrderType_key:
		if req.Amount > float64(0) {
			rsp.Status = pkg.ResponseStatusBadData
			rsp.Message = orderErrorCheckoutWithoutProducts
			return nil
		}
		break
	default:
		rsp.Status = pkg.ResponseStatusBadData
		rsp.Message = orderErrorUnknownType
		return nil
	}

	if req.User != nil {
		err := processor.processUserData()

		if err != nil {
			zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
			if e, ok := err.(*grpc.ResponseErrorMessage); ok {
				rsp.Status = pkg.ResponseStatusBadData
				rsp.Message = e
				return nil
			}
			return err
		}
	}

	if processor.checked.user != nil && processor.checked.user.Ip != "" {
		err := processor.processPayerIp()

		if err != nil {
			zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
			if e, ok := err.(*grpc.ResponseErrorMessage); ok {
				rsp.Status = pkg.ResponseStatusBadData
				rsp.Message = e
				return nil
			}
			return err
		}
	}

	switch req.Type {
	case billing.OrderType_simple:
		if req.Currency != "" {
			if err := processor.processCurrency(); err != nil {
				zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
				if e, ok := err.(*grpc.ResponseErrorMessage); ok {
					rsp.Status = pkg.ResponseStatusBadData
					rsp.Message = e
					return nil
				}
				return err
			}
		}

		if req.Amount != 0 {
			processor.processAmount()
		}
		break
	case billing.OrderTypeVirtualCurrency:
		if !processor.UserCountryExists() {
			rsp.Status = pkg.ResponseStatusBadData
			rsp.Message = orderErrorVirtualCurrencyUserCountryRequired
			return nil
		}

		err := processor.processVirtualCurrency()
        if err != nil {
            zap.L().Error(
                pkg.MethodFinishedWithError,
                zap.Error(err),
            )

            rsp.Status = pkg.ResponseStatusBadData
            rsp.Message = err.(*grpc.ResponseErrorMessage)
            return nil
        }
		break
	case billing.OrderType_product:
		if err := processor.processPaylinkProducts(); err != nil {
			if pid := req.PrivateMetadata["PaylinkId"]; pid != "" {
				s.notifyPaylinkError(ctx, pid, err, req, nil)
			}
			zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
			if e, ok := err.(*grpc.ResponseErrorMessage); ok {
				rsp.Status = pkg.ResponseStatusBadData
				rsp.Message = e
				return nil
			}
			return err
		}
		break
	case billing.OrderType_key:
		if err := processor.processPaylinkKeyProducts(); err != nil {
			if pid := req.PrivateMetadata["PaylinkId"]; pid != "" {
				s.notifyPaylinkError(ctx, pid, err, req, nil)
			}
			zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
			if e, ok := err.(*grpc.ResponseErrorMessage); ok {
				rsp.Status = pkg.ResponseStatusBadData
				rsp.Message = e
				return nil
			}
			return err
		}
		break
	}

	if processor.checked.currency == "" {
		rsp.Status = pkg.ResponseStatusBadData
		rsp.Message = orderErrorCurrencyIsRequired
		return nil
	}

	if req.OrderId != "" {
		if err := processor.processProjectOrderId(); err != nil {
			zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
			if e, ok := err.(*grpc.ResponseErrorMessage); ok {
				rsp.Status = pkg.ResponseStatusBadData
				rsp.Message = e
				return nil
			}
			return err
		}
	}

	if req.PaymentMethod != "" {
		pm, err := s.paymentMethod.GetByGroupAndCurrency(
			processor.checked.project,
			req.PaymentMethod,
			processor.checked.currency,
		)

		if err != nil {
			rsp.Status = pkg.ResponseStatusBadData
			rsp.Message = orderErrorPaymentMethodNotFound
			return nil
		}

		if err := processor.processPaymentMethod(pm); err != nil {
			zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
			if e, ok := err.(*grpc.ResponseErrorMessage); ok {
				rsp.Status = pkg.ResponseStatusBadData
				rsp.Message = e
				return nil
			}
			return err
		}
	}

	if req.Type == billing.OrderType_simple {
		if err := processor.processLimitAmounts(); err != nil {
			zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
			if e, ok := err.(*grpc.ResponseErrorMessage); ok {
				rsp.Status = pkg.ResponseStatusBadData
				rsp.Message = e
				return nil
			}
			return err
		}
	}

	processor.processMetadata()
	processor.processPrivateMetadata()

	order, err := processor.prepareOrder()

	if err != nil {
		zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			rsp.Status = pkg.ResponseStatusBadData
			rsp.Message = e
			return nil
		}
		return err
	}

	err = s.db.Collection(collectionOrder).Insert(order)

	if err != nil {
		zap.S().Errorw(fmt.Sprintf(errorQueryMask, collectionOrder), "err", err.Error(), "inserted_data", order)
		rsp.Status = pkg.ResponseStatusBadData
		rsp.Message = orderErrorCanNotCreate
		return nil
	}

	rsp.Item = order

	return nil
}

func (s *Service) PaymentFormJsonDataProcess(
	ctx context.Context,
	req *grpc.PaymentFormJsonDataRequest,
	rsp *grpc.PaymentFormJsonDataResponse,
) error {

	rsp.Status = pkg.ResponseStatusOk
	rsp.Item = &grpc.PaymentFormJsonData{}

	order, err := s.getOrderByUuid(req.OrderId)

	if err != nil {
		zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			rsp.Status = pkg.ResponseStatusSystemError
			rsp.Message = e
			return nil
		}
		return err
	}

	p := &PaymentFormProcessor{service: s, order: order, request: req}
	p1 := &OrderCreateRequestProcessor{
		Service: s,
		checked: &orderCreateRequestProcessorChecked{
			user: &billing.OrderUser{
				Ip:      req.Ip,
				Address: &billing.OrderBillingAddress{},
			},
		},
	}

	if req.Ip != "" {
		err = p1.processPayerIp()

		if err != nil {
			zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
			if e, ok := err.(*grpc.ResponseErrorMessage); ok {
				rsp.Status = pkg.ResponseStatusSystemError
				rsp.Message = e
				return nil
			}
			return err
		}
	}

	loc, ctr := s.getCountryFromAcceptLanguage(req.Locale)
	isIdentified := order.User.IsIdentified()
	browserCustomer := &BrowserCookieCustomer{
		Ip:             req.Ip,
		UserAgent:      req.UserAgent,
		AcceptLanguage: req.Locale,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if isIdentified == true {
		customer, err := s.processCustomerData(order.User.Id, order, req, browserCustomer, loc)

		if err == nil {
			browserCustomer.CustomerId = customer.Id
		}
	} else {
		if req.Cookie != "" {
			decryptedBrowserCustomer, err := s.decryptBrowserCookie(req.Cookie)

			if err == nil {
				isIdentified = true

				if (time.Now().Unix() - decryptedBrowserCustomer.UpdatedAt.Unix()) <= cookieCounterUpdateTime {
					decryptedBrowserCustomer.SessionCount++
				}

				if decryptedBrowserCustomer.CustomerId != "" {
					customer, err := s.processCustomerData(decryptedBrowserCustomer.CustomerId, order, req, decryptedBrowserCustomer, loc)

					if err != nil {
						zap.L().Error("Customer by identifier in browser cookie not processed", zap.Error(err))
					}

					if customer != nil {
						browserCustomer = decryptedBrowserCustomer
						order.User.Id = customer.Id
						order.User.TechEmail = customer.TechEmail
					} else {
						browserCustomer.VirtualCustomerId = s.getTokenString(s.cfg.Length)
					}
				} else {
					if decryptedBrowserCustomer.VirtualCustomerId == "" {
						browserCustomer.VirtualCustomerId = s.getTokenString(s.cfg.Length)
					} else {
						browserCustomer.VirtualCustomerId = decryptedBrowserCustomer.VirtualCustomerId
					}
				}
			}
		} else {
			browserCustomer.VirtualCustomerId = s.getTokenString(s.cfg.Length)
		}

		if order.User.Id == "" {
			order.User.Id = browserCustomer.VirtualCustomerId
		}

		if order.User.TechEmail == "" {
			order.User.TechEmail = order.User.Id + pkg.TechEmailDomain
		}
	}

	if order.User.Ip == "" || req.Ip != order.User.Ip {
		order.User.Ip = p1.checked.user.Ip
		order.User.Address = &billing.OrderBillingAddress{
			Country:    p1.checked.user.Address.Country,
			City:       p1.checked.user.Address.City,
			PostalCode: p1.checked.user.Address.PostalCode,
			State:      p1.checked.user.Address.State,
		}
	}

	if (order.User.Address != nil && ctr != order.User.Address.Country) || loc != order.User.Locale {
		order.UserAddressDataRequired = true
		rsp.Item.UserAddressDataRequired = order.UserAddressDataRequired

		if loc != order.User.Locale {
			order.User.Locale = loc
		}
	}

	restricted, err := s.applyCountryRestriction(order, order.GetCountry())
	if err != nil {
		zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			rsp.Status = pkg.ResponseStatusSystemError
			rsp.Message = e
			return nil
		}
		return err
	}
	if restricted {
		rsp.Status = pkg.ResponseStatusSystemError
		rsp.Message = orderCountryPaymentRestrictedEmailRequire
		rsp.Item.Id = order.Uuid
		return nil
	}

	switch order.ProductType {
	case billing.OrderType_product:
		err = s.ProcessOrderProducts(ctx, order)
		break
	case billing.OrderType_key:
		rsp.Item.Platforms, err = s.ProcessOrderKeyProducts(ctx, order)
	case billing.OrderTypeVirtualCurrency:
		err = s.ProcessOrderVirtualCurrency(ctx, order)
	}

	if err != nil {
		if pid := order.PrivateMetadata["PaylinkId"]; pid != "" {
			s.notifyPaylinkError(ctx, pid, err, req, order)
		}
		zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			rsp.Status = pkg.ResponseStatusSystemError
			rsp.Message = e
			return nil
		}
		return err
	}

	if order.Issuer == nil {
		order.Issuer = &billing.OrderIssuer{
			Embedded: req.IsEmbedded,
		}
	}
	if order.Issuer.Url == "" {
		order.Issuer.Url = req.Referer
	}
	if order.Issuer.ReferenceType == "" {
		order.Issuer.ReferenceType = req.IssuerReferenceType
	}
	if order.Issuer.Reference == "" {
		order.Issuer.Reference = req.IssuerReference
	}
	if order.Issuer.UtmSource == "" {
		order.Issuer.UtmSource = req.UtmSource
	}
	if order.Issuer.UtmCampaign == "" {
		order.Issuer.UtmCampaign = req.UtmCampaign
	}
	if order.Issuer.UtmMedium == "" {
		order.Issuer.UtmMedium = req.UtmMedium
	}
	order.Issuer.ReferrerHost = getHostFromUrl(order.Issuer.Url)

	p1.processOrderVat(order)
	err = s.updateOrder(order)

	if err != nil {
		zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			rsp.Status = pkg.ResponseStatusSystemError
			rsp.Message = e
			return nil
		}
		return err
	}

	project, err := s.project.GetById(order.Project.Id)

	if err != nil {
		rsp.Status = pkg.ResponseStatusNotFound
		rsp.Message = orderErrorProjectNotFound
		return nil
	}

	pms, err := p.processRenderFormPaymentMethods(project)

	if err != nil {
		zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			rsp.Status = pkg.ResponseStatusSystemError
			rsp.Message = e

			if e == orderErrorPaymentMethodNotAllowed {
				rsp.Status = pkg.ResponseStatusNotFound
			}

			return nil
		}
		return err
	}

	projectName, ok := order.Project.Name[order.User.Locale]

	if !ok {
		projectName = order.Project.Name[DefaultLanguage]
	}

	expire := time.Now().Add(time.Minute * 30).Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": order.Uuid, "exp": expire})

	rsp.Item.Id = order.Uuid
	rsp.Item.Account = order.ProjectAccount
	rsp.Item.Description = order.Description
	rsp.Item.HasVat = order.Tax.Amount > 0
	rsp.Item.Vat = order.Tax.Amount
	rsp.Item.Currency = order.Currency
	rsp.Item.Project = &grpc.PaymentFormJsonDataProject{
		Name:       projectName,
		UrlSuccess: order.Project.UrlSuccess,
		UrlFail:    order.Project.UrlFail,
	}
	rsp.Item.PaymentMethods = pms
	rsp.Item.Token, _ = token.SignedString([]byte(s.cfg.CentrifugoSecret))
	rsp.Item.Amount = order.OrderAmount
	rsp.Item.TotalAmount = order.TotalPaymentAmount
	rsp.Item.Items = order.Items
	rsp.Item.Email = order.User.Email

	if order.CountryRestriction != nil {
		rsp.Item.CountryPaymentsAllowed = order.CountryRestriction.PaymentsAllowed
		rsp.Item.CountryChangeAllowed = order.CountryRestriction.ChangeAllowed
	} else {
		rsp.Item.CountryPaymentsAllowed = true
		rsp.Item.CountryChangeAllowed = true
	}

	rsp.Item.UserIpData = &grpc.UserIpData{
		Country: order.User.Address.Country,
		City:    order.User.Address.City,
		Zip:     order.User.Address.PostalCode,
	}

	cookie, err := s.generateBrowserCookie(browserCustomer)

	if err == nil {
		rsp.Item.Cookie = cookie
	}

	return nil
}

func (s *Service) PaymentCreateProcess(
	ctx context.Context,
	req *grpc.PaymentCreateRequest,
	rsp *grpc.PaymentCreateResponse,
) error {
	processor := &PaymentCreateProcessor{
		service:        s,
		data:           req.Data,
		ip:             req.Ip,
		acceptLanguage: req.AcceptLanguage,
		userAgent:      req.UserAgent,
	}

	err := processor.processPaymentFormData()
	if err != nil {
		zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			rsp.Status = pkg.ResponseStatusBadData
			rsp.Message = e
			return nil
		}
		return err
	}

	order := processor.checked.order

	if !order.CountryRestriction.PaymentsAllowed {
		rsp.Message = orderCountryPaymentRestrictedError
		rsp.Status = pkg.ResponseStatusForbidden
		return nil
	}

	if order.ProductType == billing.OrderType_product {
		err = s.ProcessOrderProducts(ctx, order)
	} else if order.ProductType == billing.OrderType_key {
		// We should reserve keys only before payment
		if _, err = s.ProcessOrderKeyProducts(ctx, order); err == nil {
			err = processor.reserveKeysForOrder(ctx, order)
		}
	} else if order.ProductType == billing.OrderTypeVirtualCurrency {
		err = s.ProcessOrderVirtualCurrency(ctx, order)
	}

	if err != nil {
		if pid := order.PrivateMetadata["PaylinkId"]; pid != "" {
			s.notifyPaylinkError(ctx, pid, err, req, order)
		}

		zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			rsp.Status = pkg.ResponseStatusBadData
			rsp.Message = e
			return nil
		}
		return err
	}

	settings, err := s.paymentMethod.GetPaymentSettings(
		processor.checked.paymentMethod,
		order.Currency,
		processor.checked.project,
	)

	if err != nil {
		rsp.Status = pkg.ResponseStatusSystemError
		rsp.Message = err.(*grpc.ResponseErrorMessage)

		return nil
	}

	ps, err := s.paymentSystem.GetById(processor.checked.paymentMethod.PaymentSystemId)
	if err != nil {
		rsp.Message = orderErrorPaymentSystemInactive
		rsp.Status = pkg.ResponseStatusBadData

		return nil
	}

	order.PaymentMethod = &billing.PaymentMethodOrder{
		Id:              processor.checked.paymentMethod.Id,
		Name:            processor.checked.paymentMethod.Name,
		Params:          settings,
		PaymentSystemId: ps.Id,
		Group:           processor.checked.paymentMethod.Group,
		ExternalId:      processor.checked.paymentMethod.ExternalId,
		Handler:         ps.Handler,
	}

	p1 := &OrderCreateRequestProcessor{Service: s}
	p1.processOrderVat(order)

	if _, ok := order.PaymentRequisites[pkg.PaymentCreateFieldRecurringId]; ok {
		req.Data[pkg.PaymentCreateFieldRecurringId] = order.PaymentRequisites[pkg.PaymentCreateFieldRecurringId]
		delete(order.PaymentRequisites, pkg.PaymentCreateFieldRecurringId)
	}

	err = s.updateOrder(order)

	if err != nil {
		zap.L().Error(
			"s.updateOrder method failed",
			zap.Error(err),
			zap.Any("order", order),
		)
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			rsp.Status = pkg.ResponseStatusSystemError
			rsp.Message = e
			return nil
		} else {
			rsp.Message = orderErrorUnknown
			rsp.Status = pkg.ResponseStatusSystemError
		}
		return nil
	}

	if !s.hasPaymentCosts(order) {
		rsp.Status = pkg.ResponseStatusBadData
		rsp.Message = orderErrorCostsRatesNotFound
		return nil
	}

	h, err := s.NewPaymentSystem(s.cfg.PaymentSystemConfig, order)

	if err != nil {
		zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			rsp.Status = pkg.ResponseStatusSystemError
			rsp.Message = e
			return nil
		}
		return err
	}

	url, err := h.CreatePayment(order, req.Data)

	if err != nil {
		zap.L().Error(
			"h.CreatePayment method failed",
			zap.Error(err),
			zap.Any("order", order),
		)
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			rsp.Status = pkg.ResponseStatusSystemError
			rsp.Message = e
			return nil
		} else {
			rsp.Message = orderErrorUnknown
			rsp.Status = pkg.ResponseStatusBadData
		}
		return nil
	}

	err = s.updateOrder(order)
	if err != nil {
		zap.S().Errorf("Order create in payment system failed", "err", err.Error(), "order", order)

		zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			rsp.Status = pkg.ResponseStatusSystemError
			rsp.Message = e
			return nil
		}
		return err
	}

	rsp.Status = pkg.ResponseStatusOk
	rsp.RedirectUrl = url
	rsp.NeedRedirect = true

	if _, ok := req.Data[pkg.PaymentCreateFieldRecurringId]; ok && url == "" {
		rsp.NeedRedirect = false
	}

	return nil
}

func (s *Service) PaymentCallbackProcess(
	ctx context.Context,
	req *grpc.PaymentNotifyRequest,
	rsp *grpc.PaymentNotifyResponse,
) error {
	order, err := s.getOrderById(req.OrderId)

	if err != nil {
		return orderErrorNotFound
	}

	var data protobuf.Message

	ps, err := s.paymentSystem.GetById(order.PaymentMethod.PaymentSystemId)
	if err != nil {
		return orderErrorPaymentSystemInactive
	}

	switch ps.Handler {
	case pkg.PaymentSystemHandlerCardPay, paymentSystemHandlerCardPayMock:
		data = &billing.CardPayPaymentCallback{}
		err := json.Unmarshal(req.Request, data)

		if err != nil {
			return errors.New(paymentRequestIncorrect)
		}
		break
	default:
		return orderErrorPaymentMethodNotFound
	}

	h, err := s.NewPaymentSystem(s.cfg.PaymentSystemConfig, order)

	if err != nil {
		return err
	}

	pErr := h.ProcessPayment(order, data, string(req.Request), req.Signature)

	if pErr != nil {
		pErr, _ := pErr.(*grpc.ResponseError)

		rsp.Error = pErr.Error()
		rsp.Status = pErr.Status

		if pErr.Status == pkg.StatusTemporary {
			return nil
		}
	}

	switch order.PaymentMethod.ExternalId {
	case constant.PaymentSystemGroupAliasBankCard:
		if err := s.fillPaymentDataCard(order); err != nil {
			return err
		}
		break
	case constant.PaymentSystemGroupAliasQiwi,
		constant.PaymentSystemGroupAliasWebMoney,
		constant.PaymentSystemGroupAliasNeteller,
		constant.PaymentSystemGroupAliasAlipay:
		if err := s.fillPaymentDataEwallet(order); err != nil {
			return err
		}
		break
	case constant.PaymentSystemGroupAliasBitcoin:
		if err := s.fillPaymentDataCrypto(order); err != nil {
			return err
		}
		break
	}

	err = s.updateOrder(order)

	if err != nil {
		zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			rsp.Status = pkg.StatusErrorSystem
			rsp.Error = e.Message
			return nil
		}
		return err
	}

	if pErr == nil {
		err = s.onPaymentNotify(ctx, order)

		if err != nil {
			zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
			if e, ok := err.(*grpc.ResponseErrorMessage); ok {
				rsp.Status = pkg.StatusErrorSystem
				rsp.Error = e.Message
				return nil
			}
			return err
		}

		if h.IsRecurringCallback(data) {
			s.saveRecurringCard(order, h.GetRecurringId(data))
		}

		rsp.Status = pkg.StatusOK
	}

	return nil
}

func (s *Service) PaymentFormLanguageChanged(
	ctx context.Context,
	req *grpc.PaymentFormUserChangeLangRequest,
	rsp *grpc.PaymentFormDataChangeResponse,
) error {
	order, err := s.getOrderByUuidToForm(req.OrderId)

	if err != nil {
		zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			rsp.Status = pkg.ResponseStatusBadData
			rsp.Message = e
			return nil
		}
		return err
	}

	rsp.Status = pkg.ResponseStatusOk
	rsp.Item = &grpc.PaymentFormDataChangeResponseItem{
		UserAddressDataRequired: false,
	}

	if order.User.Locale == req.Lang {
		return nil
	}

	if order.User.IsIdentified() == true {
		s.updateCustomerFromRequestLocale(order, req.Ip, req.AcceptLanguage, req.UserAgent, req.Lang)
	}

	order.User.Locale = req.Lang
	order.UserAddressDataRequired = true

	if order.ProductType == billing.OrderType_product {
		err = s.ProcessOrderProducts(ctx, order)
	} else if order.ProductType == billing.OrderType_key {
		_, err = s.ProcessOrderKeyProducts(ctx, order)
	}

	if err != nil {
		if pid := order.PrivateMetadata["PaylinkId"]; pid != "" {
			s.notifyPaylinkError(ctx, pid, err, req, order)
		}
		zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			rsp.Status = pkg.ResponseStatusBadData
			rsp.Message = e
			return nil
		}
		return err
	}

	err = s.updateOrder(order)

	if err != nil {
		zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			rsp.Status = pkg.ResponseStatusSystemError
			rsp.Message = e
			return nil
		}
		return err
	}

	rsp.Item.UserAddressDataRequired = true
	rsp.Item.UserIpData = &grpc.UserIpData{
		Country: order.User.Address.Country,
		City:    order.User.Address.City,
		Zip:     order.User.Address.PostalCode,
	}

	return nil
}

func (s *Service) PaymentFormPaymentAccountChanged(
	ctx context.Context,
	req *grpc.PaymentFormUserChangePaymentAccountRequest,
	rsp *grpc.PaymentFormDataChangeResponse,
) error {
	order, err := s.getOrderByUuidToForm(req.OrderId)

	if err != nil {
		zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			rsp.Status = pkg.ResponseStatusBadData
			rsp.Message = e
			return nil
		}
		return err
	}

	pm, err := s.paymentMethod.GetById(req.MethodId)

	if err != nil {
		rsp.Status = pkg.ResponseStatusBadData
		rsp.Message = orderErrorPaymentMethodNotFound

		return nil
	}

	regex := pm.AccountRegexp

	if pm.ExternalId == constant.PaymentSystemGroupAliasBankCard {
		regex = "^\\d{6,18}$"
	}

	match, err := regexp.MatchString(regex, req.Account)

	if match == false || err != nil {
		rsp.Status = pkg.ResponseStatusBadData
		rsp.Message = orderErrorPaymentAccountIncorrect

		return nil
	}

	brand := ""
	country := ""

	rsp.Status = pkg.ResponseStatusOk
	rsp.Item = &grpc.PaymentFormDataChangeResponseItem{}

	switch pm.ExternalId {
	case constant.PaymentSystemGroupAliasBankCard:
		data := s.getBinData(req.Account)

		if data == nil {
			rsp.Status = pkg.ResponseStatusBadData
			rsp.Message = orderErrorCountryByPaymentAccountNotFound

			return nil
		}

		brand = data.CardBrand
		country = data.BankCountryIsoCode
		break
	case constant.PaymentSystemGroupAliasQiwi:
		req.Account = "+" + req.Account
		num, err := libphonenumber.Parse(req.Account, CountryCodeUSA)

		if err != nil || num.CountryCode == nil {
			rsp.Status = pkg.ResponseStatusBadData
			rsp.Message = orderErrorPaymentAccountIncorrect
			return nil
		}

		ok := false
		country, ok = pkg.CountryPhoneCodes[*num.CountryCode]

		if !ok {
			rsp.Status = pkg.ResponseStatusBadData
			rsp.Message = orderErrorCountryByPaymentAccountNotFound

			return nil
		}

		break
	default:
		return nil
	}

	if order.User.Address.Country == country {
		return nil
	}

	order.User.Address.Country = country
	order.UserAddressDataRequired = true

	restricted, err := s.applyCountryRestriction(order, country)

	if err != nil {
		zap.L().Error(
			"s.applyCountryRestriction method failed",
			zap.Error(err),
			zap.Any("order", order),
		)
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			rsp.Status = pkg.ResponseStatusSystemError
			rsp.Message = e
			return nil
		} else {
			rsp.Message = orderErrorUnknown
			rsp.Status = pkg.ResponseStatusSystemError
		}
		return nil
	}

	if restricted == true {
		rsp.Status = pkg.ResponseStatusForbidden
		rsp.Message = orderCountryPaymentRestrictedError

		return nil
	}

	err = s.updateOrder(order)

	if err != nil {
		zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			rsp.Status = pkg.ResponseStatusSystemError
			rsp.Message = e
			return nil
		}
		return err
	}

	rsp.Item.UserAddressDataRequired = true
	rsp.Item.UserIpData = &grpc.UserIpData{
		Country: order.User.Address.Country,
		City:    order.User.Address.City,
		Zip:     order.User.Address.PostalCode,
	}
	rsp.Item.Brand = brand

	if order.CountryRestriction != nil {
		rsp.Item.CountryPaymentsAllowed = order.CountryRestriction.PaymentsAllowed
		rsp.Item.CountryChangeAllowed = order.CountryRestriction.ChangeAllowed
	} else {
		rsp.Item.CountryPaymentsAllowed = true
		rsp.Item.CountryChangeAllowed = true
	}

	return nil
}

func (s *Service) ProcessBillingAddress(
	ctx context.Context,
	req *grpc.ProcessBillingAddressRequest,
	rsp *grpc.ProcessBillingAddressResponse,
) error {
	var err error
	var zip *billing.ZipCode

	if req.Country == CountryCodeUSA {
		if req.Zip == "" {
			rsp.Status = pkg.ResponseStatusBadData
			rsp.Message = orderErrorCreatePaymentRequiredFieldUserZipNotFound

			return nil
		}

		zip, err = s.zipCode.getByZipAndCountry(req.Zip, req.Country)

		if err != nil {
			rsp.Status = pkg.ResponseStatusBadData
			rsp.Message = orderErrorZipCodeNotFound

			return nil
		}
	}

	order, err := s.getOrderByUuidToForm(req.OrderId)

	if err != nil {
		zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			rsp.Status = pkg.ResponseStatusBadData
			rsp.Message = e
			return nil
		}
		return err
	}

	if order.CountryRestriction != nil && order.CountryRestriction.ChangeAllowed != true {
		rsp.Status = pkg.ResponseStatusForbidden
		rsp.Message = orderCountryPaymentRestrictedError
		return nil
	}

	order.BillingAddress = &billing.OrderBillingAddress{
		Country: req.Country,
	}

	if zip != nil {
		order.BillingAddress.PostalCode = zip.Zip
		order.BillingAddress.City = zip.City
		order.BillingAddress.State = zip.State.Code
	}

	restricted, err := s.applyCountryRestriction(order, req.Country)
	if err != nil {
		zap.L().Error(
			"s.applyCountryRestriction method failed",
			zap.Error(err),
			zap.Any("order", order),
		)
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			rsp.Status = pkg.ResponseStatusSystemError
			rsp.Message = e
			return nil
		} else {
			rsp.Message = orderErrorUnknown
			rsp.Status = pkg.ResponseStatusSystemError
		}
		return nil
	}
	if restricted {
		rsp.Status = pkg.ResponseStatusForbidden
		rsp.Message = orderCountryPaymentRestrictedError
		return nil
	}

	if order.ProductType == billing.OrderType_product {
		err = s.ProcessOrderProducts(ctx, order)
	} else if order.ProductType == billing.OrderType_key {
		_, err = s.ProcessOrderKeyProducts(ctx, order)
	}

	if err != nil {
		if pid := order.PrivateMetadata["PaylinkId"]; pid != "" {
			s.notifyPaylinkError(ctx, pid, err, req, order)
		}
		return err
	}

	processor := &OrderCreateRequestProcessor{Service: s}
	processor.processOrderVat(order)

	err = s.updateOrder(order)

	if err != nil {
		zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			rsp.Status = pkg.ResponseStatusSystemError
			rsp.Message = e
			return nil
		}
		return err
	}

	rsp.Status = pkg.ResponseStatusOk
	rsp.Item = &grpc.ProcessBillingAddressResponseItem{
		HasVat:      order.Tax.Amount > 0,
		Vat:         order.Tax.Amount,
		Amount:      tools.FormatAmount(order.OrderAmount),
		TotalAmount: tools.FormatAmount(order.TotalPaymentAmount),
		Currency:    order.Currency,
		Items:       order.Items,
	}

	return nil
}

func (s *Service) saveRecurringCard(order *billing.Order, recurringId string) {
	req := &repo.SavedCardRequest{
		Token:      order.User.Id,
		ProjectId:  order.Project.Id,
		MerchantId: order.Project.MerchantId,
		MaskedPan:  order.PaymentMethodTxnParams[pkg.PaymentCreateFieldPan],
		CardHolder: order.PaymentMethodTxnParams[pkg.PaymentCreateFieldHolder],
		Expire: &entity.CardExpire{
			Month: order.PaymentRequisites[pkg.PaymentCreateFieldMonth],
			Year:  order.PaymentRequisites[pkg.PaymentCreateFieldYear],
		},
		RecurringId: recurringId,
	}

	_, err := s.rep.InsertSavedCard(context.TODO(), req)

	if err != nil {
		s.logError(
			"Call repository service to save recurring card failed",
			[]interface{}{
				"err", err.Error(),
				"request", req,
			},
		)
	} else {
		order.PaymentRequisites["saved"] = "1"
		err = s.updateOrder(order)
		if err != nil {
			zap.S().Errorf("Failed to update order after save recurruing card", "err", err.Error())
		}
	}
}

func (s *Service) updateOrder(order *billing.Order) error {
	ps := order.GetPublicStatus()

	zap.S().Debug("[updateOrder] updating order", "order_id", order.Id, "status", ps)

	originalOrder, _ := s.getOrderById(order.Id)

	statusChanged := false
	if originalOrder != nil {
		ops := originalOrder.GetPublicStatus()
		zap.S().Debug("[updateOrder] no original order status", "order_id", order.Id, "status", ops)
		statusChanged = ops != ps
	} else {
		zap.S().Debug("[updateOrder] no original order found", "order_id", order.Id)
	}

	err := s.db.Collection(collectionOrder).UpdateId(bson.ObjectIdHex(order.Id), order)

	if err != nil {
		s.logError(orderErrorUpdateOrderDataFailed, []interface{}{"error", err.Error(), "order", order})
		if err == mgo.ErrNotFound {
			return orderErrorNotFound
		}
		return orderErrorUnknown
	}

	zap.S().Debug("[updateOrder] updating order success", "order_id", order.Id, "status_changed", statusChanged, "type", order.ProductType)

	if order.ProductType == billing.OrderType_key {
		s.orderNotifyKeyProducts(context.TODO(), order)
	}

	if statusChanged && ps != constant.OrderPublicStatusCreated && ps != constant.OrderPublicStatusPending {
		zap.S().Infow("[updateOrder] notify merchant", "order_id", order.Id, "status", ps)

		switch ps {
		case constant.OrderPublicStatusRefunded:
			s.sendMailWithRefund(order)
		case constant.OrderPublicStatusProcessed:
			s.sendMailWithReceipt(order)
		}

		s.orderNotifyMerchant(order)
	}

	return nil
}

func (s *Service) orderNotifyKeyProducts(ctx context.Context, order *billing.Order) {
	zap.S().Debug("[orderNotifyKeyProducts] called", "order_id", order.Id, "status", order.GetPublicStatus(), "is product notified: ", order.IsKeyProductNotified)

	if order.IsKeyProductNotified {
		return
	}

	keys := order.Keys
	var err error
	switch order.GetPublicStatus() {
	case constant.OrderPublicStatusCanceled, constant.OrderPublicStatusRejected:
		for _, key := range keys {
			zap.S().Infow("[orderNotifyKeyProducts] trying to cancel reserving key", "order_id", order.Id, "key", key)
			rsp := &grpc.EmptyResponseWithStatus{}
			err = s.CancelRedeemKeyForOrder(ctx, &grpc.KeyForOrderRequest{KeyId: key}, rsp)
			if err != nil {
				zap.S().Error("internal error during canceling reservation for key", "err", err, "key", key)
				continue
			}
			if rsp.Status != pkg.ResponseStatusOk {
				zap.S().Error("could not cancel reservation for key", "key", key, "message", rsp.Message)
				continue
			}
		}
		order.IsKeyProductNotified = true
		break
	case constant.OrderPublicStatusProcessed:
		for _, key := range keys {
			zap.S().Infow("[orderNotifyKeyProducts] trying to finish reserving key", "order_id", order.Id, "key", key)
			rsp := &grpc.GetKeyForOrderRequestResponse{}
			err = s.FinishRedeemKeyForOrder(ctx, &grpc.KeyForOrderRequest{KeyId: key}, rsp)
			if err != nil {
				zap.S().Errorw("internal error during finishing reservation for key", "err", err, "key", key)
				continue
			}
			if rsp.Status != pkg.ResponseStatusOk {
				zap.S().Errorw("could not finish reservation for key", "key", key, "message", rsp.Message)
				continue
			}

			s.sendMailWithCode(ctx, order, rsp.Key)
		}
		order.IsKeyProductNotified = true
		break
	}
}

func (s *Service) sendMailWithRefund(order *billing.Order) {
	payload := s.getPayloadForReceipt(order)
	payload.TemplateAlias = s.cfg.EmailRefundTransactionTemplate

	zap.S().Infow("sending receipt to broker", "order_id", order.Id)
	err := s.postmarkBroker.Publish(postmarkSdrPkg.PostmarkSenderTopicName, payload, amqp.Table{})
	if err != nil {
		zap.S().Errorw(
			"Publication refund transaction to user email queue is failed",
			"err", err, "email", order.ReceiptEmail, "order_id", order.Id)
	}
}

func (s *Service) sendMailWithReceipt(order *billing.Order) {
	payload := s.getPayloadForReceipt(order)

	zap.S().Infow("sending receipt to broker", "order_id", order.Id, "topic", postmarkSdrPkg.PostmarkSenderTopicName)
	err := s.postmarkBroker.Publish(postmarkSdrPkg.PostmarkSenderTopicName, payload, amqp.Table{})
	if err != nil {
		zap.S().Errorw(
			"Publication receipt to user email queue is failed",
			"err", err, "email", order.ReceiptEmail, "order_id", order.Id, "topic", postmarkSdrPkg.PostmarkSenderTopicName)
	}
}

func (s *Service) getReceiptModel(name string, price string) *structpb.Value {
	return &structpb.Value{
		Kind: &structpb.Value_StructValue{
			StructValue: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"name": {
						Kind: &structpb.Value_StringValue{StringValue: name},
					},
					"price": {
						Kind: &structpb.Value_StringValue{StringValue: price},
					},
				},
			},
		},
	}
}

func (s *Service) getPayloadForReceipt(order *billing.Order) *postmarkSdrPkg.Payload {
	totalPrice, err := s.formatter.FormatCurrency(DefaultLanguage, order.OrderAmount, order.Currency)
	if err != nil {
		zap.S().Errorw("Error during formatting currency", "price", order.OrderAmount, "locale", DefaultLanguage, "currency", order.Currency)
	}

	date, err := s.formatter.FormatDateTime(DefaultLanguage, time.Unix(order.CreatedAt.Seconds, 0))
	if err != nil {
		zap.S().Errorw("Error during formatting date", "date", order.CreatedAt, "locale", DefaultLanguage)
	}

	merchantName := order.GetMerchantId()
	merchant, err := s.merchant.GetById(order.GetMerchantId())
	if err != nil {
		zap.S().Errorw("Error during getting merchant", "merchant_id", order.GetMerchantId(), "order.uuid", order.Uuid, "err", err)
	} else {
		merchantName = merchant.Company.Name
	}

	payload := &postmarkSdrPkg.Payload{
		TemplateAlias: s.cfg.EmailSuccessTransactionTemplate,
		TemplateModel: map[string]string{
			"total_price":      totalPrice,
			"transaction_id":   order.Uuid,
			"transaction_date": date,
			"project_name":     order.Project.Name[DefaultLanguage],
			"receipt_id":       order.ReceiptId,
			"merchant_name":    merchantName,
		},
		To: order.ReceiptEmail,
	}

	var items []*structpb.Value

	for _, item := range order.Items {
		price, err := s.formatter.FormatCurrency("en", item.Amount, item.Currency)
		if err != nil {
			zap.S().Errorw("Error during formatting currency", "price", item.Amount, "locale", "en", "currency", item.Currency)
		}
		items = append(items, s.getReceiptModel(item.Name, price))
	}

	payload.TemplateObjectModel = &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"items": {
				Kind: &structpb.Value_ListValue{ListValue: &structpb.ListValue{
					Values: items,
				}},
			},
		},
	}

	if platform, ok := availablePlatforms[order.PlatformId]; ok {
		payload.TemplateObjectModel.Fields["platform"] = &structpb.Value{
			Kind: &structpb.Value_StructValue{
				StructValue: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"name": {
							Kind: &structpb.Value_StringValue{
								StringValue: platform.Name,
							},
						},
					},
				},
			},
		}
	}

	return payload
}

func (s *Service) sendMailWithCode(ctx context.Context, order *billing.Order, key *billing.Key) {
	var platformIconUrl = ""
	if platform, ok := availablePlatforms[order.PlatformId]; ok {
		platformIconUrl = platform.Icon
	}

	for _, item := range order.Items {
		if item.Id == key.KeyProductId {
			item.Code = key.Code
			payload := &postmarkSdrPkg.Payload{
				TemplateAlias: s.cfg.EmailGameCodeTemplate,
				TemplateModel: map[string]string{
					"code":          key.Code,
					"platform_icon": platformIconUrl,
					"product_name":  item.Name,
				},
				To: order.ReceiptEmail,
			}

			if len(item.Images) > 0 {
				payload.TemplateModel["product_image"] = item.Images[0]
			}

			err := s.postmarkBroker.Publish(postmarkSdrPkg.PostmarkSenderTopicName, payload, amqp.Table{})
			if err != nil {
				zap.S().Errorw(
					"Publication activation code to user email queue is failed",
					"err", err, "email", order.ReceiptEmail, "order_id", order.Id, "key_id", key.Id)

			} else {
				zap.S().Infow("Sent payload to broker", "email", order.ReceiptEmail, "order_id", order.Id, "key_id", key.Id, "topic", postmarkSdrPkg.PostmarkSenderTopicName)
			}
			return
		}
	}

	zap.S().Errorw("Mail not sent because no items found for key", "order_id", order.Id, "key_id", key.Id, "email", order.ReceiptEmail)
}

func (s *Service) orderNotifyMerchant(order *billing.Order) {
	zap.S().Debug("[orderNotifyMerchant] try to send notify merchant to rmq", "order_id", order.Id, "status", order.GetPublicStatus())

	err := s.broker.Publish(constant.PayOneTopicNotifyPaymentName, order, amqp.Table{"x-retry-count": int32(0)})
	if err != nil {
		zap.S().Debug("[orderNotifyMerchant] send notify merchant to rmq failed", "order_id", order.Id)
		s.logError(orderErrorPublishNotificationFailed, []interface{}{
			"err", err.Error(), "order", order, "topic", constant.PayOneTopicNotifyPaymentName,
		})
	} else {
		zap.S().Debug("[orderNotifyMerchant] send notify merchant to rmq failed", "order_id", order.Id)
	}
	order.SetNotificationStatus(order.GetPublicStatus(), err == nil)
	err = s.db.Collection(collectionOrder).UpdateId(bson.ObjectIdHex(order.Id), order)
	if err != nil {
		zap.S().Debug("[orderNotifyMerchant] notification status update failed", "order_id", order.Id)
		s.logError(orderErrorUpdateOrderDataFailed, []interface{}{"error", err.Error(), "order", order})
	} else {
		zap.S().Debug("[orderNotifyMerchant] notification status updated succesfully", "order_id", order.Id)
	}
}

func (s *Service) getOrderById(id string) (order *billing.Order, err error) {
	err = s.db.Collection(collectionOrder).FindId(bson.ObjectIdHex(id)).One(&order)

	if err != nil && err != mgo.ErrNotFound {
		zap.S().Errorf("Order not found in payment create process", "err", err.Error(), "order_id", id)
	}

	if order == nil {
		return order, orderErrorNotFound
	}

	return
}

func (s *Service) getOrderByUuid(uuid string) (order *billing.Order, err error) {
	order, err = s.orderRepository.GetByUuid(uuid)

	if err != nil && err != mgo.ErrNotFound {
		zap.S().Errorf("Order not found in payment create process", "err", err.Error(), "uuid", uuid)
	}

	if order == nil {
		return order, orderErrorNotFound
	}

	return
}

func (s *Service) getOrderByUuidToForm(uuid string) (*billing.Order, error) {
	order, err := s.getOrderByUuid(uuid)

	if err != nil {
		return nil, orderErrorNotFound
	}

	if order.HasEndedStatus() == true {
		return nil, orderErrorOrderAlreadyComplete
	}

	if order.FormInputTimeIsEnded() == true {
		return nil, orderErrorFormInputTimeExpired
	}

	return order, nil
}

func (s *Service) getBinData(pan string) (data *BinData) {
	if len(pan) < 6 {
		zap.S().Errorf("Incorrect PAN to get BIN data", "pan", pan)
		return
	}

	i, err := strconv.ParseInt(pan[:6], 10, 32)

	if err != nil {
		zap.S().Errorf("Parse PAN to int failed", "error", err.Error(), "pan", pan)
		return
	}

	err = s.db.Collection(collectionBinData).Find(bson.M{"card_bin": int32(i)}).One(&data)

	if err != nil {
		zap.S().Errorf("Query to get bank card BIN data failed", "error", err.Error(), "pan", pan)
		return
	}

	return
}

func (v *OrderCreateRequestProcessor) prepareOrder() (*billing.Order, error) {
	id := bson.NewObjectId().Hex()
	amount := tools.FormatAmount(v.checked.amount)

	if (v.request.UrlVerify != "" || v.request.UrlNotify != "") && v.checked.project.AllowDynamicNotifyUrls == false {
		return nil, orderErrorDynamicNotifyUrlsNotAllowed
	}

	if (v.request.UrlSuccess != "" || v.request.UrlFail != "") && v.checked.project.AllowDynamicRedirectUrls == false {
		return nil, orderErrorDynamicRedirectUrlsNotAllowed
	}

	order := &billing.Order{
		Id:   id,
		Type: pkg.OrderTypeOrder,
		Project: &billing.ProjectOrder{
			Id:                      v.checked.project.Id,
			Name:                    v.checked.project.Name,
			UrlSuccess:              v.checked.project.UrlRedirectSuccess,
			UrlFail:                 v.checked.project.UrlRedirectFail,
			SendNotifyEmail:         v.checked.project.SendNotifyEmail,
			NotifyEmails:            v.checked.project.NotifyEmails,
			SecretKey:               v.checked.project.SecretKey,
			UrlCheckAccount:         v.checked.project.UrlCheckAccount,
			UrlProcessPayment:       v.checked.project.UrlProcessPayment,
			UrlChargebackPayment:    v.checked.project.UrlChargebackPayment,
			UrlCancelPayment:        v.checked.project.UrlCancelPayment,
			UrlRefundPayment:        v.checked.project.UrlRefundPayment,
			UrlFraudPayment:         v.checked.project.UrlFraudPayment,
			CallbackProtocol:        v.checked.project.CallbackProtocol,
			MerchantId:              v.checked.merchant.Id,
			Status:                  v.checked.project.Status,
			MerchantRoyaltyCurrency: v.checked.merchant.GetPayoutCurrency(),
		},
		Description:    fmt.Sprintf(orderDefaultDescription, id),
		ProjectOrderId: v.request.OrderId,
		ProjectAccount: v.request.Account,
		ProjectParams:  v.request.Other,
		PrivateStatus:  constant.OrderStatusNew,
		CreatedAt:      ptypes.TimestampNow(),
		IsJsonRequest:  v.request.IsJson,

		Uuid:               uuid.New().String(),
		ReceiptId:          uuid.New().String(),
		User:               v.checked.user,
		OrderAmount:        amount,
		TotalPaymentAmount: amount,
		Currency:           v.checked.currency,
		Products:           v.checked.products,
		Items:              v.checked.items,
		Metadata:           v.checked.metadata,
		PrivateMetadata:    v.checked.privateMetadata,
		Issuer: &billing.OrderIssuer{
			Url:           v.request.IssuerUrl,
			Embedded:      v.request.IsEmbedded,
			ReferenceType: v.request.IssuerReferenceType,
			Reference:     v.request.IssuerReference,
			UtmSource:     v.request.UtmSource,
			UtmCampaign:   v.request.UtmCampaign,
			UtmMedium:     v.request.UtmMedium,
			ReferrerHost:  getHostFromUrl(v.request.IssuerUrl),
		},
		CountryRestriction: &billing.CountryRestriction{
			IsoCodeA2:       "",
			PaymentsAllowed: true,
			ChangeAllowed:   true,
		},
		PlatformId:  v.request.PlatformId,
		ProductType: v.request.Type,
	}

	if v.checked.virtualAmount > 0 {
		order.VirtualCurrencyAmount = v.checked.virtualAmount
	}

	if order.User == nil {
		order.User = &billing.OrderUser{
			Object: pkg.ObjectTypeUser,
		}
	} else {
		if order.User.Address != nil {
			v.processOrderVat(order)

			restricted, err := v.applyCountryRestriction(order, order.GetCountry())
			if err != nil {
				return nil, err
			}
			if restricted {
				return nil, orderCountryPaymentRestrictedError
			}
		}
	}

	if v.request.Description != "" {
		order.Description = v.request.Description
	}

	if v.request.UrlSuccess != "" {
		order.Project.UrlSuccess = v.request.UrlSuccess
	}

	if v.request.UrlFail != "" {
		order.Project.UrlFail = v.request.UrlFail
	}

	if v.checked.paymentMethod != nil {
		ps, err := v.paymentSystem.GetById(v.checked.paymentMethod.PaymentSystemId)
		if err != nil {
			return nil, err
		}

		settings, err := v.paymentMethod.GetPaymentSettings(
			v.checked.paymentMethod,
			v.checked.currency,
			v.checked.project,
		)

		if err != nil {
			return nil, err
		}

		order.PaymentMethod = &billing.PaymentMethodOrder{
			Id:              v.checked.paymentMethod.Id,
			Name:            v.checked.paymentMethod.Name,
			Params:          settings,
			PaymentSystemId: ps.Id,
			Group:           v.checked.paymentMethod.Group,
		}
	}

	order.ExpireDateToFormInput, _ = ptypes.TimestampProto(time.Now().Add(time.Minute * defaultExpireDateToFormInput))

	return order, nil
}

func (v *OrderCreateRequestProcessor) processMerchant() error {
	if v.checked.merchant.HasTariff() == false {
		return orderErrorMerchantBadTariffs
	}

	return nil
}

func (v *OrderCreateRequestProcessor) processProject() error {
	project, err := v.project.GetById(v.request.ProjectId)

	if err != nil {
		zap.S().Errorw("Order create get project error", "err", err, "request", v.request)
		return orderErrorProjectNotFound
	}

	if project.IsDeleted() == true {
		return orderErrorProjectInactive
	}

	if project.MerchantId == "" || bson.IsObjectIdHex(project.MerchantId) == false {
		return orderErrorProjectMerchantNotFound
	}

	merchant, err := v.merchant.GetById(project.MerchantId)
	if err != nil {
		return orderErrorProjectMerchantNotFound
	}

	if merchant.IsDeleted() == true {
		return orderErrorProjectMerchantInactive
	}

	v.checked.project = project
	v.checked.merchant = merchant

	return nil
}

func (v *OrderCreateRequestProcessor) processCurrency() error {
	if !contains(v.supportedCurrencies, v.request.Currency) {
		return orderErrorCurrencyNotFound
	}

	v.checked.currency = v.request.Currency
	return nil
}

func (v *OrderCreateRequestProcessor) processAmount() {
	v.checked.amount = v.request.Amount
}

func (v *OrderCreateRequestProcessor) processMetadata() {
	v.checked.metadata = v.request.Metadata
}

func (v *OrderCreateRequestProcessor) processPrivateMetadata() {
	v.checked.privateMetadata = v.request.PrivateMetadata
}

func (v *OrderCreateRequestProcessor) processPayerIp() error {
	rsp, err := v.geo.GetIpData(context.TODO(), &proto.GeoIpDataRequest{IP: v.checked.user.Ip})

	if err != nil {
		zap.L().Error(
			"Order create get payer data error",
			zap.Error(err),
			zap.String("ip", v.checked.user.Ip),
		)

		return orderErrorPayerRegionUnknown
	}

	if v.checked.user.Address == nil {
		v.checked.user.Address = &billing.OrderBillingAddress{}
	}

	if v.checked.user.Address.Country == "" {
		v.checked.user.Address.Country = rsp.Country.IsoCode
	}

	if v.checked.user.Address.City == "" {
		v.checked.user.Address.City = rsp.City.Names["en"]
	}

	if v.checked.user.Address.PostalCode == "" && rsp.Postal != nil {
		v.checked.user.Address.PostalCode = rsp.Postal.Code
	}

	if v.checked.user.Address.State == "" && len(rsp.Subdivisions) > 0 {
		v.checked.user.Address.State = rsp.Subdivisions[0].IsoCode
	}

	return nil
}

func (v *OrderCreateRequestProcessor) processPaylinkKeyProducts() error {
	if len(v.request.Products) == 0 {
		return nil
	}

	orderProducts, err := v.GetOrderKeyProducts(context.TODO(), v.checked.project.Id, v.request.Products)
	if err != nil {
		return err
	}

	pid := v.request.PrivateMetadata["PaylinkId"]
	currency := v.checked.merchant.GetPayoutCurrency()
	logInfo := processPaylinkKeyProductsTemplate

	if currency == "" {
		zap.S().Errorw(fmt.Sprintf(logInfo, "merchant payout currency not found"), "paylink", pid)
		return orderErrorNoProductsCommonCurrency
	}

	zap.S().Infow(fmt.Sprintf(logInfo, "use currency"), "currency", currency, "paylink", pid)

	priceGroup, err := v.priceGroup.GetByRegion(currency)

	if err != nil {
		zap.S().Errorw("Price group not found", "currency", currency)
		return priceGroupErrorNotFound
	}

	platformId := v.request.PlatformId

	if len(platformId) == 0 {
		platforms := v.filterPlatforms(orderProducts)
		if len(platforms) == 0 {
			zap.S().Errorw("No available platformIds")
			return orderErrorNoPlatforms
		}
		sort.Slice(platforms, func(i, j int) bool {
			return availablePlatforms[platforms[i]].Order < availablePlatforms[platforms[j]].Order
		})
		platformId = platforms[0]
	}

	amount, err := v.GetOrderKeyProductsAmount(orderProducts, priceGroup, platformId)
	if err != nil {
		return err
	}

	items, err := v.GetOrderKeyProductsItems(orderProducts, DefaultLanguage, priceGroup, platformId)
	if err != nil {
		return err
	}

	v.checked.products = v.request.Products
	v.checked.currency = currency
	v.checked.amount = amount
	v.checked.items = items

	return nil
}

func (v *OrderCreateRequestProcessor) processPaylinkProducts() error {
	if len(v.request.Products) == 0 {
		return nil
	}

	orderProducts, err := v.GetOrderProducts(v.checked.project.Id, v.request.Products)
	if err != nil {
		return err
	}

	logInfo := "[processPaylinkProducts] %s"
	pid := v.request.PrivateMetadata["PaylinkId"]
	currency := v.checked.merchant.GetPayoutCurrency()

	if currency == "" {
		zap.S().Errorw(fmt.Sprintf(logInfo, "merchant payout currency not found"), "paylink", pid)
		return orderErrorNoProductsCommonCurrency
	}

	zap.S().Infow(fmt.Sprintf(logInfo, "use currency"), "currency", currency, "paylink", pid)

	priceGroup, err := v.priceGroup.GetByRegion(currency)

	if err != nil {
		zap.S().Errorw("Price group not found", "currency", currency)
		return priceGroupErrorNotFound
	}

	//v.request.Currency

	amount, err := v.GetOrderProductsAmount(orderProducts, priceGroup)
	if err != nil {
		return err
	}

	items, err := v.GetOrderProductsItems(orderProducts, DefaultLanguage, priceGroup)
	if err != nil {
		return err
	}

	v.checked.products = v.request.Products
	v.checked.currency = currency
	v.checked.amount = amount
	v.checked.items = items

	return nil
}

func (v *OrderCreateRequestProcessor) processProjectOrderId() error {
	var order *billing.Order

	filter := bson.M{
		"project._id":      bson.ObjectIdHex(v.checked.project.Id),
		"project_order_id": v.request.OrderId,
	}

	err := v.db.Collection(collectionOrder).Find(filter).One(&order)

	if err != nil && err != mgo.ErrNotFound {
		zap.S().Errorw("Order create check project order id unique", "err", err, "filter", filter)
		return orderErrorCanNotCreate
	}

	if order != nil {
		return orderErrorProjectOrderIdIsDuplicate
	}

	return nil
}

func (v *OrderCreateRequestProcessor) processPaymentMethod(pm *billing.PaymentMethod) error {
	if pm.IsActive == false {
		return orderErrorPaymentMethodInactive
	}

	if _, err := v.paymentSystem.GetById(pm.PaymentSystemId); err != nil {
		return orderErrorPaymentSystemInactive
	}

	_, err := v.Service.paymentMethod.GetPaymentSettings(pm, v.checked.currency, v.checked.project)

	if err != nil {
		return err
	}

	v.checked.paymentMethod = pm

	return nil
}

func (v *OrderCreateRequestProcessor) processLimitAmounts() (err error) {
	amount := v.checked.amount

	if v.checked.project.LimitsCurrency != "" && v.checked.project.LimitsCurrency != v.checked.currency {
		if !contains(v.supportedCurrencies, v.checked.project.LimitsCurrency) {
			return orderErrorCurrencyNotFound
		}
		req := &currencies.ExchangeCurrencyCurrentForMerchantRequest{
			From:       v.checked.currency,
			To:         v.checked.project.LimitsCurrency,
			MerchantId: v.checked.merchant.Id,
			RateType:   curPkg.RateTypeOxr,
			Amount:     amount,
		}

		rsp, err := v.curService.ExchangeCurrencyCurrentForMerchant(context.TODO(), req)

		if err != nil {
			zap.S().Error(
				pkg.ErrorGrpcServiceCallFailed,
				zap.Error(err),
				zap.String(errorFieldService, "CurrencyRatesService"),
				zap.String(errorFieldMethod, "ExchangeCurrencyCurrentForMerchant"),
			)

			return orderErrorConvertionCurrency
		}

		amount = rsp.ExchangedAmount
	}

	if amount < v.checked.project.MinPaymentAmount {
		return orderErrorAmountLowerThanMinAllowed
	}

	if v.checked.project.MaxPaymentAmount > 0 && amount > v.checked.project.MaxPaymentAmount {
		return orderErrorAmountGreaterThanMaxAllowed
	}

	if v.checked.paymentMethod != nil {
		if v.request.Amount < v.checked.paymentMethod.MinPaymentAmount {
			return orderErrorAmountLowerThanMinAllowedPaymentMethod
		}

		if v.checked.paymentMethod.MaxPaymentAmount > 0 && v.request.Amount > v.checked.paymentMethod.MaxPaymentAmount {
			return orderErrorAmountGreaterThanMaxAllowedPaymentMethod
		}
	}

	return
}

func (v *OrderCreateRequestProcessor) processSignature() error {
	var hashString string

	if v.request.IsJson == false {
		var keys []string
		var elements []string

		for k := range v.request.RawParams {
			keys = append(keys, k)
		}

		sort.Strings(keys)

		for _, k := range keys {
			value := k + "=" + v.request.RawParams[k]
			elements = append(elements, value)
		}

		hashString = strings.Join(elements, "") + v.checked.project.SecretKey
	} else {
		hashString = v.request.RawBody + v.checked.project.SecretKey
	}

	h := sha512.New()
	h.Write([]byte(hashString))

	if hex.EncodeToString(h.Sum(nil)) != v.request.Signature {
		return orderErrorSignatureInvalid
	}

	return nil
}

// Calculate VAT for order
func (v *OrderCreateRequestProcessor) processOrderVat(order *billing.Order) {

	order.Tax = &billing.OrderTax{
		Type:     taxTypeVat,
		Currency: order.Currency,
	}
	req := &tax_service.GetRateRequest{
		IpData:   &tax_service.GeoIdentity{},
		UserData: &tax_service.GeoIdentity{},
	}

	if order.User != nil && order.User.Address != nil {
		req.IpData.Country = order.User.Address.Country
		req.IpData.City = order.User.Address.City

		if order.User.Address.Country == CountryCodeUSA {
			order.Tax.Type = taxTypeSalesTax

			req.IpData.Zip = order.User.Address.PostalCode
			req.IpData.State = order.User.Address.State

			if order.BillingAddress != nil {
				req.UserData.Zip = order.BillingAddress.PostalCode
			}
		}
	}

	if order.BillingAddress != nil {
		req.UserData.Country = order.BillingAddress.Country
		req.UserData.City = order.BillingAddress.City
		req.UserData.State = order.BillingAddress.State
	}

	rsp, err := v.tax.GetRate(context.TODO(), req)

	if err != nil {
		v.logError("Tax service return error", []interface{}{"error", err.Error(), "request", req})
		return
	}

	if order.BillingAddress != nil {
		req.UserData.State = rsp.Rate.State
	}

	order.Tax.Rate = rsp.Rate.Rate
	order.Tax.Amount = tools.FormatAmount(order.OrderAmount * order.Tax.Rate)
	order.TotalPaymentAmount = tools.FormatAmount(order.OrderAmount + order.Tax.Amount)

	return
}

func (v *OrderCreateRequestProcessor) processCustomerToken() error {
	token, err := v.getTokenBy(v.request.Token)

	if err != nil {
		return err
	}

	customer, err := v.getCustomerById(token.CustomerId)

	if err != nil {
		return err
	}

	v.request.Type = token.Settings.Type
	v.request.PlatformId = token.Settings.PlatformId

	v.request.ProjectId = token.Settings.ProjectId
	v.request.Description = token.Settings.Description
	v.request.Amount = token.Settings.Amount
	v.request.Currency = token.Settings.Currency
	v.request.Products = token.Settings.ProductsIds
	v.request.Metadata = token.Settings.Metadata
	v.request.PaymentMethod = token.Settings.PaymentMethod

	if token.Settings.ReturnUrl != nil {
		v.request.UrlSuccess = token.Settings.ReturnUrl.Success
		v.request.UrlFail = token.Settings.ReturnUrl.Fail
	}

	v.checked.user = &billing.OrderUser{
		ExternalId: token.User.Id,
		Address:    token.User.Address,
		Metadata:   token.User.Metadata,
	}

	if token.User.Name != nil {
		v.checked.user.Name = token.User.Name.Value
	}

	if token.User.Email != nil {
		v.checked.user.Email = token.User.Email.Value
		v.checked.user.EmailVerified = token.User.Email.Verified
	}

	if token.User.Phone != nil {
		v.checked.user.Phone = token.User.Phone.Value
		v.checked.user.PhoneVerified = token.User.Phone.Verified
	}

	if token.User.Ip != nil {
		v.checked.user.Ip = token.User.Ip.Value
	}

	if token.User.Locale != nil {
		v.checked.user.Locale = token.User.Locale.Value
	}

	v.checked.user.Id = customer.Id
	v.checked.user.Object = pkg.ObjectTypeUser
	v.checked.user.TechEmail = customer.TechEmail

	return nil
}

func (v *OrderCreateRequestProcessor) processUserData() (err error) {
	customer := new(billing.Customer)
	tokenReq := v.transformOrderUser2TokenRequest(v.request.User)

	if v.request.Token == "" {
		customer, _ = v.findCustomer(tokenReq, v.checked.project)
	}

	if customer != nil {
		customer, err = v.updateCustomer(tokenReq, v.checked.project, customer)
	} else {
		customer, err = v.createCustomer(tokenReq, v.checked.project)
	}

	if err != nil {
		return err
	}

	v.checked.user = v.request.User
	v.checked.user.Id = customer.Id
	v.checked.user.Object = pkg.ObjectTypeUser
	v.checked.user.TechEmail = customer.TechEmail

	return
}

// GetById payment methods of project for rendering in payment form
func (v *PaymentFormProcessor) processRenderFormPaymentMethods(
	project *billing.Project,
) ([]*billing.PaymentFormPaymentMethod, error) {
	var projectPms []*billing.PaymentFormPaymentMethod

	paymentMethods, err := v.service.paymentMethod.ListByCurrency(project, v.order.Currency)

	if err != nil {
		return nil, err
	}

	for _, pm := range paymentMethods {
		if pm.IsActive == false {
			continue
		}

		ps, err := v.service.paymentSystem.GetById(pm.PaymentSystemId)

		if err != nil || ps.IsActive == false {
			continue
		}

		if v.order.OrderAmount < pm.MinPaymentAmount ||
			(pm.MaxPaymentAmount > 0 && v.order.OrderAmount > pm.MaxPaymentAmount) {
			continue
		}

		_, err = v.service.paymentMethod.GetPaymentSettings(pm, v.order.Currency, project)

		if err != nil {
			continue
		}

		formPm := &billing.PaymentFormPaymentMethod{
			Id:            pm.Id,
			Name:          pm.Name,
			Type:          pm.Type,
			Group:         pm.Group,
			AccountRegexp: pm.AccountRegexp,
		}

		err = v.processPaymentMethodsData(formPm)

		if err != nil {
			zap.S().Errorw(
				"Process payment method data failed",
				"error", err,
				"order_id", v.order.Id,
			)
			continue
		}

		projectPms = append(projectPms, formPm)
	}

	if len(projectPms) <= 0 {
		return projectPms, orderErrorPaymentMethodNotAllowed
	}

	return projectPms, nil
}

func (v *PaymentFormProcessor) processPaymentMethodsData(pm *billing.PaymentFormPaymentMethod) error {
	pm.HasSavedCards = false

	if pm.IsBankCard() == true {
		req := &repo.SavedCardRequest{Token: v.order.User.Id}
		rsp, err := v.service.rep.FindSavedCards(context.TODO(), req)

		if err != nil {
			zap.S().Errorw(
				"Get saved cards from repository failed",
				"error", err,
				"token", v.order.User.Id,
				"project_id", v.order.Project.Id,
				"order_id", v.order.Id,
			)
		} else {
			pm.HasSavedCards = len(rsp.SavedCards) > 0
			pm.SavedCards = []*billing.SavedCard{}

			for _, v := range rsp.SavedCards {
				d := &billing.SavedCard{
					Id:         v.Id,
					Pan:        v.MaskedPan,
					CardHolder: v.CardHolder,
					Expire:     &billing.CardExpire{Month: v.Expire.Month, Year: v.Expire.Year},
				}

				pm.SavedCards = append(pm.SavedCards, d)
			}

		}
	}

	return nil
}

func (v *PaymentCreateProcessor) reserveKeysForOrder(ctx context.Context, order *billing.Order) error {
	if len(order.Keys) == 0 {
		zap.S().Infow("[ProcessOrderKeyProducts] reserving keys", "order_id", order.Id)
		keys := make([]string, len(order.Products))
		for i, productId := range order.Products {
			reserveRes := &grpc.PlatformKeyReserveResponse{}
			reserveReq := &grpc.PlatformKeyReserveRequest{
				PlatformId:   order.PlatformId,
				MerchantId:   order.Project.MerchantId,
				OrderId:      order.Id,
				KeyProductId: productId,
				Ttl:          oneDayTtl,
			}

			err := v.service.ReserveKeyForOrder(ctx, reserveReq, reserveRes)
			if err != nil {
				zap.L().Error(
					pkg.ErrorGrpcServiceCallFailed,
					zap.Error(err),
					zap.String(errorFieldService, "KeyService"),
					zap.String(errorFieldMethod, "ReserveKeyForOrder"),
				)
				return err
			}

			if reserveRes.Status != pkg.ResponseStatusOk {
				zap.S().Errorw("[ProcessOrderKeyProducts] can't reserve key. Cancelling reserved before", "message", reserveRes.Message, "order_id", order.Id)

				// we should cancel reservation for keys reserved before
				for _, keyToCancel := range keys {
					if len(keyToCancel) > 0 {
						cancelRes := &grpc.EmptyResponseWithStatus{}
						err := v.service.CancelRedeemKeyForOrder(ctx, &grpc.KeyForOrderRequest{KeyId: keyToCancel}, cancelRes)
						if err != nil {
							zap.L().Error(
								pkg.ErrorGrpcServiceCallFailed,
								zap.Error(err),
								zap.String(errorFieldService, "KeyService"),
								zap.String(errorFieldMethod, "CancelRedeemKeyForOrder"),
							)
						} else if cancelRes.Status != pkg.ResponseStatusOk {
							zap.S().Errorw("[ProcessOrderKeyProducts] error during cancelling reservation", "message", cancelRes.Message, "order_id", order.Id)
						} else {
							zap.S().Infow("[ProcessOrderKeyProducts] successful canceled reservation", "order_id", order.Id, "key_id", keyToCancel)
						}
					}
				}

				return reserveRes.Message
			}
			zap.S().Infow("[ProcessOrderKeyProducts] reserved for product", "product ", productId, "reserveRes ", reserveRes, "order_id", order.Id)
			keys[i] = reserveRes.KeyId
		}

		order.Keys = keys
	}

	return nil
}

// Validate data received from payment form and write validated data to order
func (v *PaymentCreateProcessor) processPaymentFormData() error {
	if _, ok := v.data[pkg.PaymentCreateFieldOrderId]; !ok ||
		v.data[pkg.PaymentCreateFieldOrderId] == "" {
		return orderErrorCreatePaymentRequiredFieldIdNotFound
	}

	if _, ok := v.data[pkg.PaymentCreateFieldPaymentMethodId]; !ok ||
		v.data[pkg.PaymentCreateFieldPaymentMethodId] == "" {
		return orderErrorCreatePaymentRequiredFieldPaymentMethodNotFound
	}

	if _, ok := v.data[pkg.PaymentCreateFieldEmail]; !ok ||
		v.data[pkg.PaymentCreateFieldEmail] == "" {
		return orderErrorCreatePaymentRequiredFieldEmailNotFound
	}

	order, err := v.service.getOrderByUuidToForm(v.data[pkg.PaymentCreateFieldOrderId])

	if err != nil {
		return err
	}

	if order.UserAddressDataRequired == true {
		country, ok := v.data[pkg.PaymentCreateFieldUserCountry]

		if !ok || country == "" {
			return orderErrorCreatePaymentRequiredFieldUserCountryNotFound
		}

		if country == CountryCodeUSA {
			zip, ok := v.data[pkg.PaymentCreateFieldUserZip]

			if !ok || zip == "" {
				return orderErrorCreatePaymentRequiredFieldUserZipNotFound
			}

			zipData, err := v.service.zipCode.getByZipAndCountry(zip, country)

			if err != nil {
				return orderErrorZipCodeNotFound
			}

			v.data[pkg.PaymentCreateFieldUserCity] = zipData.City
			v.data[pkg.PaymentCreateFieldUserState] = zipData.State.Code
		}
	}

	processor := &OrderCreateRequestProcessor{
		Service: v.service,
		request: &billing.OrderCreateRequest{
			ProjectId: order.Project.Id,
			Amount:    order.OrderAmount,
		},
		checked: &orderCreateRequestProcessorChecked{
			currency: order.Currency,
			amount:   order.OrderAmount,
		},
	}

	if err := processor.processProject(); err != nil {
		return err
	}

	pm, err := v.service.paymentMethod.GetById(v.data[pkg.PaymentCreateFieldPaymentMethodId])
	if err != nil {
		return orderErrorPaymentMethodNotFound
	}

	if err = processor.processPaymentMethod(pm); err != nil {
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			return e
		}
		return err
	}

	if err := processor.processLimitAmounts(); err != nil {
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			return e
		}
		return err
	}

	if order.User.Ip != v.ip {
		order.User.Ip = v.ip
	}

	updCustomerReq := &grpc.TokenRequest{User: &billing.TokenUser{}}

	if val, ok := v.data[pkg.PaymentCreateFieldEmail]; ok {
		order.User.Email = val
		updCustomerReq.User.Email = &billing.TokenUserEmailValue{Value: val}
	}

	order.PaymentRequisites = make(map[string]string)

	if order.UserAddressDataRequired == true {
		if order.BillingAddress == nil {
			order.BillingAddress = &billing.OrderBillingAddress{}
		}

		if order.BillingAddress.Country != v.data[pkg.PaymentCreateFieldUserCountry] {
			order.BillingAddress.Country = v.data[pkg.PaymentCreateFieldUserCountry]
		}

		if order.BillingAddress.Country == CountryCodeUSA {
			if order.BillingAddress.City != v.data[pkg.PaymentCreateFieldUserCity] {
				order.BillingAddress.City = v.data[pkg.PaymentCreateFieldUserCity]
			}

			if order.BillingAddress.PostalCode != v.data[pkg.PaymentCreateFieldUserZip] {
				order.BillingAddress.PostalCode = v.data[pkg.PaymentCreateFieldUserZip]
			}

			if order.BillingAddress.State != v.data[pkg.PaymentCreateFieldUserState] {
				order.BillingAddress.State = v.data[pkg.PaymentCreateFieldUserState]
			}
		}

		processor.processOrderVat(order)
		updCustomerReq.User.Address = order.BillingAddress
	}

	restricted, err := v.service.applyCountryRestriction(order, order.GetCountry())
	if err != nil {
		zap.L().Error(
			"v.service.applyCountryRestriction method failed",
			zap.Error(err),
			zap.Any("order", order),
		)
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			return e
		}
		return orderErrorUnknown
	}
	if restricted {
		return orderCountryPaymentRestrictedError
	}

	if order.User.IsIdentified() == true {
		customer, err := v.service.updateCustomerFromRequest(order, updCustomerReq, v.ip, v.acceptLanguage, v.userAgent)

		if err != nil {
			v.service.logError("Update customer data by request failed", []interface{}{"error", err.Error(), "data", updCustomerReq})
		} else {
			if customer.Locale != order.User.Locale {
				order.User.Locale = customer.Locale
			}
		}
	}

	delete(v.data, pkg.PaymentCreateFieldOrderId)
	delete(v.data, pkg.PaymentCreateFieldPaymentMethodId)
	delete(v.data, pkg.PaymentCreateFieldEmail)

	if processor.checked.paymentMethod.IsBankCard() == true {
		if id, ok := v.data[pkg.PaymentCreateFieldStoredCardId]; ok {
			storedCard, err := v.service.rep.FindSavedCardById(context.TODO(), &repo.FindByStringValue{Value: id})

			if err != nil {
				v.service.logError("Get data about stored card failed", []interface{}{"err", err.Error(), "id", id})
			}

			if storedCard == nil {
				v.service.logError("Get data about stored card failed", []interface{}{"id", id})
				return orderGetSavedCardError
			}

			if storedCard.Token != order.User.Id {
				v.service.logError("Alarm: user try use not own bank card for payment", []interface{}{"user_id", order.User.Id, "card_id", id})
				return orderErrorRecurringCardNotOwnToUser
			}

			order.PaymentRequisites[pkg.PaymentCreateFieldPan] = storedCard.MaskedPan
			order.PaymentRequisites[pkg.PaymentCreateFieldMonth] = storedCard.Expire.Month
			order.PaymentRequisites[pkg.PaymentCreateFieldYear] = storedCard.Expire.Year
			order.PaymentRequisites[pkg.PaymentCreateFieldHolder] = storedCard.CardHolder
			order.PaymentRequisites[pkg.PaymentCreateFieldRecurringId] = storedCard.RecurringId
		} else {
			validator := &bankCardValidator{
				Pan:    v.data[pkg.PaymentCreateFieldPan],
				Cvv:    v.data[pkg.PaymentCreateFieldCvv],
				Month:  v.data[pkg.PaymentCreateFieldMonth],
				Year:   v.data[pkg.PaymentCreateFieldYear],
				Holder: v.data[pkg.PaymentCreateFieldHolder],
			}

			if err := validator.Validate(); err != nil {
				return err
			}

			order.PaymentRequisites[pkg.PaymentCreateFieldPan] = tools.MaskBankCardNumber(v.data[pkg.PaymentCreateFieldPan])
			order.PaymentRequisites[pkg.PaymentCreateFieldMonth] = v.data[pkg.PaymentCreateFieldMonth]

			if len(v.data[pkg.PaymentCreateFieldYear]) < 3 {
				v.data[pkg.PaymentCreateFieldYear] = strconv.Itoa(time.Now().UTC().Year())[:2] + v.data[pkg.PaymentCreateFieldYear]
			}

			order.PaymentRequisites[pkg.PaymentCreateFieldYear] = v.data[pkg.PaymentCreateFieldYear]
		}

		bin := v.service.getBinData(order.PaymentRequisites[pkg.PaymentCreateFieldPan])

		if bin != nil {
			order.PaymentRequisites[paymentCreateBankCardFieldBrand] = bin.CardBrand
			order.PaymentRequisites[paymentCreateBankCardFieldType] = bin.CardType
			order.PaymentRequisites[paymentCreateBankCardFieldCategory] = bin.CardCategory
			order.PaymentRequisites[paymentCreateBankCardFieldIssuerName] = bin.BankName
			order.PaymentRequisites[paymentCreateBankCardFieldIssuerCountry] = bin.BankCountryName
		}
	} else {
		account := ""

		if acc, ok := v.data[pkg.PaymentCreateFieldEWallet]; ok {
			account = acc
		}

		if acc, ok := v.data[pkg.PaymentCreateFieldCrypto]; ok {
			account = acc
		}

		if account == "" {
			return paymentSystemErrorEWalletIdentifierIsInvalid
		}

		order.PaymentRequisites = v.data
	}

	v.checked.project = processor.checked.project
	v.checked.paymentMethod = processor.checked.paymentMethod
	v.checked.order = order

	if order.ProjectAccount == "" {
		order.ProjectAccount = order.User.Email
	}

	return nil
}

func (s *Service) GetOrderKeyProducts(ctx context.Context, projectId string, productIds []string) ([]*grpc.KeyProduct, error) {
	if len(productIds) == 0 {
		return nil, orderErrorProductsEmpty
	}

	result := grpc.ListKeyProductsResponse{}

	err := s.GetKeyProductsForOrder(ctx, &grpc.GetKeyProductsForOrderRequest{
		ProjectId: projectId,
		Ids:       productIds,
	}, &result)

	if err != nil {
		zap.L().Error(
			"v.GetKeyProductsForOrder method failed",
			zap.Error(err),
		)
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			return nil, e
		}
		return nil, orderErrorUnknown
	}

	if result.Count != int32(len(productIds)) {
		return nil, orderErrorProductsInvalid
	}

	return result.Products, nil
}

func (s *Service) GetOrderKeyProductsAmount(products []*grpc.KeyProduct, group *billing.PriceGroup, platformId string) (float64, error) {
	if len(products) == 0 {
		return 0, orderErrorProductsEmpty
	}

	sum := float64(0)

	for _, p := range products {
		amount, err := p.GetPriceInCurrencyAndPlatform(group, platformId)

		if err != nil {
			return 0, orderErrorNoProductsCommonCurrency
		}

		sum += amount
	}

	totalAmount := tools.FormatAmount(sum)

	return totalAmount, nil
}

func (s *Service) GetOrderProducts(projectId string, productIds []string) ([]*grpc.Product, error) {
	if len(productIds) == 0 {
		return nil, orderErrorProductsEmpty
	}

	result := grpc.ListProductsResponse{}

	err := s.GetProductsForOrder(context.TODO(), &grpc.GetProductsForOrderRequest{
		ProjectId: projectId,
		Ids:       productIds,
	}, &result)

	if err != nil {
		zap.L().Error(
			"v.GetProductsForOrder method failed",
			zap.Error(err),
		)
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			return nil, e
		}
		return nil, orderErrorUnknown
	}

	if result.Total != int32(len(productIds)) {
		return nil, orderErrorProductsInvalid
	}

	return result.Products, nil
}

func (s *Service) GetOrderProductsAmount(products []*grpc.Product, group *billing.PriceGroup) (float64, error) {
	if len(products) == 0 {
		return 0, orderErrorProductsEmpty
	}

	sum := float64(0)

	for _, p := range products {
		amount, err := p.GetPriceInCurrency(group)

		if err != nil {
			return 0, orderErrorNoProductsCommonCurrency
		}

		sum += amount
	}

	totalAmount := tools.FormatAmount(sum)

	return totalAmount, nil
}

func (s *Service) GetOrderProductsItems(products []*grpc.Product, language string, group *billing.PriceGroup) ([]*billing.OrderItem, error) {
	var result []*billing.OrderItem

	if len(products) == 0 {
		return nil, orderErrorProductsEmpty
	}

	isDefaultLanguage := language == DefaultLanguage

	for _, p := range products {
		var (
			amount      float64
			name        string
			description string
			err         error
		)

		amount, err = p.GetPriceInCurrency(group)
		if err != nil {
			return nil, orderErrorProductsPrice
		}

		name, err = p.GetLocalizedName(language)
		if err != nil {
			if isDefaultLanguage {
				return nil, orderErrorNoNameInRequiredLanguage
			}
			name, err = p.GetLocalizedName(DefaultLanguage)
			if err != nil {
				return nil, orderErrorNoNameInDefaultLanguage
			}
		}

		description, err = p.GetLocalizedDescription(language)
		if err != nil {
			if isDefaultLanguage {
				return nil, orderErrorNoDescriptionInRequiredLanguage
			}
			description, err = p.GetLocalizedDescription(DefaultLanguage)
			if err != nil {
				return nil, orderErrorNoDescriptionInDefaultLanguage
			}
		}

		item := &billing.OrderItem{
			Id:          p.Id,
			Object:      p.Object,
			Sku:         p.Sku,
			Name:        name,
			Description: description,
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
			Images:      p.Images,
			Url:         p.Url,
			Metadata:    p.Metadata,
			Amount:      amount,
			Currency:    group.Currency,
		}
		result = append(result, item)
	}

	return result, nil
}

func (s *Service) GetOrderKeyProductsItems(products []*grpc.KeyProduct, language string, group *billing.PriceGroup, platformId string) ([]*billing.OrderItem, error) {
	var result []*billing.OrderItem

	if len(products) == 0 {
		return nil, orderErrorProductsEmpty
	}

	isDefaultLanguage := language == DefaultLanguage

	for _, p := range products {
		var (
			amount      float64
			name        string
			description string
			err         error
		)

		amount, err = p.GetPriceInCurrencyAndPlatform(group, platformId)
		if err != nil {
			return nil, orderErrorProductsPrice
		}

		name, err = p.GetLocalizedName(language)
		if err != nil {
			if isDefaultLanguage {
				return nil, orderErrorNoNameInRequiredLanguage
			}
			name, err = p.GetLocalizedName(DefaultLanguage)
			if err != nil {
				return nil, orderErrorNoNameInDefaultLanguage
			}
		}

		description, err = p.GetLocalizedDescription(language)
		if err != nil {
			if isDefaultLanguage {
				return nil, orderErrorNoDescriptionInRequiredLanguage
			}
			description, err = p.GetLocalizedDescription(DefaultLanguage)
			if err != nil {
				return nil, orderErrorNoDescriptionInDefaultLanguage
			}
		}

		item := &billing.OrderItem{
			Id:          p.Id,
			Object:      p.Object,
			Sku:         p.Sku,
			Name:        name,
			Description: description,
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
			Images:      []string{getImageByLanguage(DefaultLanguage, p.Cover)},
			Url:         p.Url,
			Metadata:    p.Metadata,
			Amount:      amount,
			Currency:    group.Currency,
			PlatformId:  platformId,
		}
		result = append(result, item)
	}

	return result, nil
}

func (s *Service) filterPlatforms(orderProducts []*grpc.KeyProduct) []string {
	// filter available platformIds for all products in request
	var platformIds []string
	for i, product := range orderProducts {
		var platformsToCheck []string
		for _, pl := range product.Platforms {
			platformsToCheck = append(platformsToCheck, pl.Id)
		}

		if i > 0 {
			platformIds = intersect(platformIds, platformsToCheck)
		} else {
			platformIds = platformsToCheck
		}
	}

	return platformIds
}

func (s *Service) ProcessOrderVirtualCurrency(ctx context.Context, order *billing.Order) error {
	var (
		country    string
		currency   string
		priceGroup *billing.PriceGroup
	)

	merchant, _ := s.merchant.GetById(order.Project.MerchantId)
	defaultCurrency := merchant.GetPayoutCurrency()

	if defaultCurrency == "" {
		zap.S().Infow("merchant payout currency not found", "order.Uuid", order.Uuid)
		return orderErrorNoProductsCommonCurrency
	}

	defaultPriceGroup, err := s.priceGroup.GetByRegion(defaultCurrency)
	if err != nil {
		zap.S().Errorw("Price group not found", "currency", currency)
		return orderErrorUnknown
	}

	currency = defaultCurrency
	priceGroup = defaultPriceGroup

	country = order.GetCountry()

	if country != "" {
		countryData, err := s.country.GetByIsoCodeA2(country)
		if err != nil {
			zap.S().Errorw("Country not found", "country", country)
			return orderErrorUnknown
		}

		priceGroup, err = s.priceGroup.GetById(countryData.PriceGroupId)
		if err != nil {
			zap.S().Errorw("Price group not found", "countryData", countryData)
			return orderErrorUnknown
		}

		currency = priceGroup.Currency
	}

	zap.S().Infow("try to use detected currency for order amount", "currency", currency, "order.Uuid", order.Uuid)

	project, err := s.project.GetById(order.GetProjectId())

	if project == nil || project.VirtualCurrency == nil {
		return orderErrorVirtualCurrencyNotFilled
	}

	amount, err := s.GetAmountForVirtualCurrency(order.VirtualCurrencyAmount, priceGroup, project.VirtualCurrency.Prices)
	if err != nil {
		if priceGroup.Id == defaultPriceGroup.Id {
			return err
		}

		// try to get order Amount in default currency, if it differs from requested one
		amount, err = s.GetAmountForVirtualCurrency(order.VirtualCurrencyAmount, defaultPriceGroup, project.VirtualCurrency.Prices)
		if err != nil {
			return err
		}
	}

	amount = tools.FormatAmount(amount)

	order.Currency = currency
	order.OrderAmount = amount
	order.TotalPaymentAmount = amount

	return nil
}

func (s *Service) GetAmountForVirtualCurrency(virtualAmount float64, group *billing.PriceGroup, prices []*billing.ProductPrice) (float64, error) {
	for _, price := range prices {
		if price.Currency == group.Currency {
			return virtualAmount * price.Amount, nil
		}
	}

	return 0, virtualCurrencyPayoutCurrencyMissed
}

func (s *Service) ProcessOrderKeyProducts(ctx context.Context, order *billing.Order) ([]*grpc.Platform, error) {
	project, err := s.project.GetById(order.Project.Id)
	if err != nil {
		return nil, orderErrorProjectNotFound
	}
	if project.IsDeleted() == true {
		return nil, orderErrorProjectInactive
	}

	if order.ProductType != billing.OrderType_key {
		return nil, nil
	}

	orderProducts, err := s.GetOrderKeyProducts(ctx, project.Id, order.Products)
	if err != nil {
		return nil, err
	}

	var (
		country    string
		currency   string
		priceGroup *billing.PriceGroup
		platformId string
		locale     string
		logInfo    = processProcessOrderKeyProductsTemplate
	)

	country = order.GetCountry()

	// filter available platformIds for all products in request
	platformIds := s.filterPlatforms(orderProducts)

	if len(platformIds) == 0 {
		zap.S().Errorw("No available platformIds", "order.uuid", order.Uuid)
		return nil, orderErrorNoPlatforms
	}

	platforms := make([]*grpc.Platform, len(platformIds))
	for i, v := range platformIds {
		platforms[i] = availablePlatforms[v]
	}

	sort.Slice(platforms, func(i, j int) bool {
		return platforms[i].Order < platforms[j].Order
	})

	if len(order.PlatformId) > 0 {
		platformId = order.PlatformId
	} else {
		// default platform if not specified before
		platformId = platforms[0].Id
	}

	merchant, _ := s.merchant.GetById(order.Project.MerchantId)
	defaultCurrency := merchant.GetPayoutCurrency()

	if defaultCurrency == "" {
		zap.S().Infow(fmt.Sprintf(logInfo, "merchant payout currency not found"), "order.Uuid", order.Uuid)
		return nil, orderErrorNoProductsCommonCurrency
	}

	defaultPriceGroup, err := s.priceGroup.GetByRegion(defaultCurrency)
	if err != nil {
		zap.S().Errorw("Price group not found", "currency", currency)
		return nil, orderErrorUnknown
	}

	currency = defaultCurrency
	priceGroup = defaultPriceGroup

	if country != "" {
		countryData, err := s.country.GetByIsoCodeA2(country)
		if err != nil {
			zap.S().Errorw("Country not found", "country", country)
			return nil, orderErrorUnknown
		}

		priceGroup, err = s.priceGroup.GetById(countryData.PriceGroupId)
		if err != nil {
			zap.S().Errorw("Price group not found", "countryData", countryData)
			return nil, orderErrorUnknown
		}

		currency = priceGroup.Currency
	}

	zap.S().Infow(fmt.Sprintf(logInfo, "try to use detected currency for order amount"), "currency", currency, "order.Uuid", order.Uuid, "platform_id", platformId, "order.PlatformId", order.PlatformId)
	// try to get order Amount in requested currency
	amount, err := s.GetOrderKeyProductsAmount(orderProducts, priceGroup, platformId)
	if err != nil {
		if priceGroup.Id == defaultPriceGroup.Id {
			return nil, err
		}
		// try to get order Amount in default currency, if it differs from requested one
		amount, err = s.GetOrderKeyProductsAmount(orderProducts, defaultPriceGroup, platformId)
		if err != nil {
			return nil, err
		}
		zap.S().Infow(fmt.Sprintf(logInfo, "try to use default currency for order amount"), "currency", defaultCurrency, "order.Uuid", order.Uuid, "platform_id", platformId)

		priceGroup = defaultPriceGroup
		// converting Amount from default currency to requested
		req := &currencies.ExchangeCurrencyCurrentForMerchantRequest{
			From:       defaultCurrency,
			To:         currency,
			MerchantId: order.GetMerchantId(),
			RateType:   curPkg.RateTypeOxr,
			Amount:     amount,
		}

		rsp, err := s.curService.ExchangeCurrencyCurrentForMerchant(ctx, req)

		if err != nil {
			zap.L().Error(
				pkg.ErrorGrpcServiceCallFailed,
				zap.Error(err),
				zap.String(errorFieldService, "CurrencyRatesService"),
				zap.String(errorFieldMethod, "ExchangeCurrencyCurrentForMerchant"),
			)

			return nil, orderErrorConvertionCurrency
		}
		amount = rsp.ExchangedAmount
	}

	if order.User != nil && order.User.Locale != "" {
		locale = order.User.Locale
	} else {
		locale = DefaultLanguage
	}

	items, err := s.GetOrderKeyProductsItems(orderProducts, locale, priceGroup, platformId)
	if err != nil {
		return nil, err
	}

	amount = tools.FormatAmount(amount)

	order.Currency = currency
	order.OrderAmount = amount
	order.TotalPaymentAmount = amount
	order.Items = items

	return platforms, nil
}

func (s *Service) ProcessOrderProducts(ctx context.Context, order *billing.Order) error {
	project, err := s.project.GetById(order.Project.Id)
	if err != nil {
		return orderErrorProjectNotFound
	}
	if project.IsDeleted() == true {
		return orderErrorProjectInactive
	}

	if order.ProductType != billing.OrderType_product {
		return nil
	}

	if project.IsProductsCheckout == false {
		return nil
	}

	orderProducts, err := s.GetOrderProducts(project.Id, order.Products)
	if err != nil {
		return err
	}

	var (
		country    string
		currency   string
		priceGroup *billing.PriceGroup
		locale     string
		logInfo    = "[ProcessOrderProducts] %s"
		amount 	   float64
	)

	if order.BillingAddress != nil && order.BillingAddress.Country != "" {
		country = order.BillingAddress.Country
	} else if order.User.Address != nil && order.User.Address.Country != "" {
		country = order.User.Address.Country
	}

	merchant, _ := s.merchant.GetById(order.Project.MerchantId)
	defaultCurrency := merchant.GetPayoutCurrency()

	if defaultCurrency == "" {
		zap.S().Infow(fmt.Sprintf(logInfo, "merchant payout currency not found"), "order.Uuid", order.Uuid)
		return orderErrorNoProductsCommonCurrency
	}

	defaultPriceGroup, err := s.priceGroup.GetByRegion(defaultCurrency)
	if err != nil {
		zap.S().Errorw("Price group not found", "currency", currency)
		return orderErrorUnknown
	}

	currency = defaultCurrency
	priceGroup = defaultPriceGroup

	if country != "" {
		countryData, err := s.country.GetByIsoCodeA2(country)
		if err != nil {
			zap.S().Errorw("Country not found", "country", country)
			return orderErrorUnknown
		}

		priceGroup, err = s.priceGroup.GetById(countryData.PriceGroupId)
		if err != nil {
			zap.S().Errorw("Price group not found", "countryData", countryData)
			return orderErrorUnknown
		}

		currency = priceGroup.Currency
	}

	zap.S().Infow(fmt.Sprintf(logInfo, "try to use detected currency for order amount"), "currency", currency, "order.Uuid", order.Uuid)

	if order.IsBuyForVirtualCurrency {
		virtualAmount, err := s.GetOrderProductsAmount(orderProducts, &billing.PriceGroup{Currency: "virtual"})
		if err != nil {
			return err
		}

		project, err := s.project.GetById(order.GetProjectId())
		if err != nil {
			return err
		}

		if project.VirtualCurrency == nil || len(project.VirtualCurrency.Prices) == 0 {
			return orderErrorVirtualCurrencyNotFilled
		}

		amount, err = s.GetAmountForVirtualCurrency(virtualAmount, priceGroup, project.VirtualCurrency.Prices)
		if err != nil {
			amount, err = s.GetAmountForVirtualCurrency(virtualAmount, defaultPriceGroup, project.VirtualCurrency.Prices)
			if err != nil {
				return err
			}

			priceGroup = defaultPriceGroup
			// converting Amount from default currency to requested
			req := &currencies.ExchangeCurrencyCurrentForMerchantRequest{
				From:       defaultCurrency,
				To:         currency,
				MerchantId: order.GetMerchantId(),
				RateType:   curPkg.RateTypeOxr,
				Amount:     amount,
			}

			rsp, err := s.curService.ExchangeCurrencyCurrentForMerchant(ctx, req)

			if err != nil {
				zap.S().Error(
					pkg.ErrorGrpcServiceCallFailed,
					zap.Error(err),
					zap.String(errorFieldService, "CurrencyRatesService"),
					zap.String(errorFieldMethod, "ExchangeCurrencyCurrentForMerchant"),
				)

				return orderErrorConvertionCurrency
			}
			amount = rsp.ExchangedAmount
		}
	} else {
		// try to get order Amount in requested currency
		amount, err = s.GetOrderProductsAmount(orderProducts, priceGroup)
		if err != nil {
			if priceGroup.Id == defaultPriceGroup.Id {
				return err
			}
			// try to get order Amount in default currency, if it differs from requested one
			amount, err = s.GetOrderProductsAmount(orderProducts, defaultPriceGroup)
			if err != nil {
				return err
			}
			zap.S().Infow(fmt.Sprintf(logInfo, "try to use default currency for order amount"), "currency", defaultCurrency, "order.Uuid", order.Uuid)

			priceGroup = defaultPriceGroup
			// converting Amount from default currency to requested
			req := &currencies.ExchangeCurrencyCurrentForMerchantRequest{
				From:       defaultCurrency,
				To:         currency,
				MerchantId: order.GetMerchantId(),
				RateType:   curPkg.RateTypeOxr,
				Amount:     amount,
			}

			rsp, err := s.curService.ExchangeCurrencyCurrentForMerchant(ctx, req)

			if err != nil {
				zap.S().Error(
					pkg.ErrorGrpcServiceCallFailed,
					zap.Error(err),
					zap.String(errorFieldService, "CurrencyRatesService"),
					zap.String(errorFieldMethod, "ExchangeCurrencyCurrentForMerchant"),
				)

				return orderErrorConvertionCurrency
			}
			amount = rsp.ExchangedAmount
		}
	}

	if order.User != nil && order.User.Locale != "" {
		locale = order.User.Locale
	} else {
		locale = DefaultLanguage
	}

	items, err := s.GetOrderProductsItems(orderProducts, locale, priceGroup)
	if err != nil {
		return err
	}

	amount = tools.FormatAmount(amount)

	order.Currency = currency
	order.OrderAmount = amount
	order.TotalPaymentAmount = amount

	order.Items = items

	return nil
}

func (s *Service) notifyPaylinkError(ctx context.Context, paylinkId string, err error, req interface{}, order interface{}) {
	msg := map[string]interface{}{
		"event":     "error",
		"paylinkId": paylinkId,
		"message":   "Invalid paylink",
		"error":     err,
		"request":   req,
		"order":     order,
	}
	_ = s.centrifugo.Publish(ctx, centrifugoChannel, msg)
}

func (v *PaymentCreateProcessor) GetMerchantId() string {
	return v.checked.project.MerchantId
}

func (s *Service) processCustomerData(
	customerId string,
	order *billing.Order,
	req *grpc.PaymentFormJsonDataRequest,
	browserCustomer *BrowserCookieCustomer,
	locale string,
) (*billing.Customer, error) {
	customer, err := s.getCustomerById(customerId)

	if err != nil {
		return nil, err
	}

	tokenReq := &grpc.TokenRequest{
		User: &billing.TokenUser{
			Ip:             &billing.TokenUserIpValue{Value: req.Ip},
			Locale:         &billing.TokenUserLocaleValue{Value: locale},
			AcceptLanguage: req.Locale,
			UserAgent:      req.UserAgent,
		},
	}
	project := &billing.Project{
		Id:         order.Project.Id,
		MerchantId: order.Project.MerchantId,
	}

	browserCustomer.CustomerId = customer.Id
	_, err = s.updateCustomer(tokenReq, project, customer)

	return customer, err
}

func (s *Service) IsOrderCanBePaying(
	ctx context.Context,
	req *grpc.IsOrderCanBePayingRequest,
	rsp *grpc.IsOrderCanBePayingResponse,
) error {
	order, err := s.getOrderByUuidToForm(req.OrderId)
	rsp.Status = pkg.ResponseStatusBadData

	if err != nil {
		zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			rsp.Message = e
			return nil
		}
		return err
	}

	if order != nil && order.GetProjectId() != req.ProjectId {
		rsp.Message = orderErrorOrderCreatedAnotherProject
		return nil
	}

	rsp.Status = pkg.ResponseStatusOk
	rsp.Item = order

	return nil
}

func (s *Service) fillPaymentDataCard(order *billing.Order) error {
	first6 := ""
	last4 := ""
	pan, ok := order.PaymentMethodTxnParams[pkg.PaymentCreateFieldPan]
	if !ok || pan == "" {
		pan, ok = order.PaymentRequisites["pan"]
		if !ok {
			pan = ""
		}
	}
	order.PaymentMethodPayerAccount = pan
	if len(pan) >= 6 {
		first6 = pan[0:6]
		last4 = pan[len(pan)-4:]
	}
	cardBrand, ok := order.PaymentRequisites["card_brand"]

	month, ok := order.PaymentRequisites["month"]
	if !ok {
		month = ""
	}
	year, ok := order.PaymentRequisites["year"]
	if !ok {
		year = ""
	}

	order.PaymentMethod.Card = &billing.PaymentMethodCard{
		Masked:      pan,
		First6:      first6,
		Last4:       last4,
		ExpiryMonth: month,
		ExpiryYear:  year,
		Brand:       cardBrand,
		Secure3D:    order.PaymentMethodTxnParams[pkg.TxnParamsFieldBankCardIs3DS] == "1",
	}
	b, err := json.Marshal(order.PaymentMethod.Card)
	if err != nil {
		return err
	}
	fp, err := bcrypt.GenerateFromPassword([]byte(string(b)), bcrypt.MinCost)
	if err == nil {
		order.PaymentMethod.Card.Fingerprint = string(fp)
	}
	return nil
}

func (s *Service) fillPaymentDataEwallet(order *billing.Order) error {
	account := order.PaymentMethodTxnParams[pkg.PaymentCreateFieldEWallet]
	order.PaymentMethodPayerAccount = account
	order.PaymentMethod.Wallet = &billing.PaymentMethodWallet{
		Brand:   order.PaymentMethod.Name,
		Account: account,
	}
	return nil
}

func (s *Service) fillPaymentDataCrypto(order *billing.Order) error {
	address := order.PaymentMethodTxnParams[pkg.PaymentCreateFieldCrypto]
	order.PaymentMethodPayerAccount = address
	order.PaymentMethod.CryptoCurrency = &billing.PaymentMethodCrypto{
		Brand:   order.PaymentMethod.Name,
		Address: address,
	}
	return nil
}

func (s *Service) SetUserNotifySales(
	ctx context.Context,
	req *grpc.SetUserNotifyRequest,
	rsp *grpc.EmptyResponse,
) error {

	order, err := s.getOrderByUuid(req.OrderUuid)

	if err != nil {
		s.logError(orderErrorNotFound.Message, []interface{}{"error", err.Error(), "request", req})
		return orderErrorNotFound
	}

	if req.EnableNotification && req.Email == "" {
		return orderErrorEmailRequired
	}

	order.NotifySale = req.EnableNotification
	order.NotifySaleEmail = req.Email
	err = s.updateOrder(order)
	if err != nil {
		return err
	}

	if !req.EnableNotification {
		return nil
	}

	data := &grpc.NotifyUserSales{
		Email:   req.Email,
		OrderId: order.Id,
		Date:    time.Now().Format(time.RFC3339),
	}
	if order.User != nil {
		data.UserId = order.User.Id
	}
	err = s.db.Collection(collectionNotifySales).Insert(data)
	if err != nil {

		zap.S().Errorf(
			"Save email to collection failed",
			"error", err.Error(),
			"request", req,
			"collection", collectionNotifySales,
		)
		return err
	}

	if order.User.IsIdentified() == true {
		customer, err := s.getCustomerById(order.User.Id)
		if err != nil {
			return err
		}
		project, err := s.project.GetById(order.Project.Id)
		if err != nil {
			return err
		}

		customer.NotifySale = req.EnableNotification
		customer.NotifySaleEmail = req.Email

		tokenReq := s.transformOrderUser2TokenRequest(order.User)
		_, err = s.updateCustomer(tokenReq, project, customer)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) SetUserNotifyNewRegion(
	ctx context.Context,
	req *grpc.SetUserNotifyRequest,
	rsp *grpc.EmptyResponse,
) error {

	order, err := s.getOrderByUuid(req.OrderUuid)

	if err != nil {
		s.logError(orderErrorNotFound.Message, []interface{}{"error", err.Error(), "request", req})
		return orderErrorNotFound
	}

	if order.CountryRestriction.PaymentsAllowed {
		s.logError(orderErrorNotRestricted.Message, []interface{}{"request", req})
		return orderErrorNotRestricted
	}

	if req.EnableNotification && req.Email == "" {
		return orderErrorEmailRequired
	}

	if order.User == nil {
		order.User = &billing.OrderUser{}
	}
	order.User.NotifyNewRegion = req.EnableNotification
	order.User.NotifyNewRegionEmail = req.Email
	err = s.updateOrder(order)
	if err != nil {
		return err
	}

	if !(req.EnableNotification && order.CountryRestriction != nil) {
		return nil
	}

	data := &grpc.NotifyUserNewRegion{
		Email:            req.Email,
		OrderId:          order.Id,
		UserId:           order.User.Id,
		Date:             time.Now().Format(time.RFC3339),
		CountryIsoCodeA2: order.CountryRestriction.IsoCodeA2,
	}
	err = s.db.Collection(collectionNotifyNewRegion).Insert(data)
	if err != nil {
		zap.S().Errorf(
			"Save email to collection failed",
			"error", err.Error(),
			"request", req,
			"collection", collectionNotifyNewRegion,
		)
		return err
	}

	if order.User.IsIdentified() == true {
		customer, err := s.getCustomerById(order.User.Id)
		if err != nil {
			return err
		}
		project, err := s.project.GetById(order.Project.Id)
		if err != nil {
			return err
		}

		customer.NotifyNewRegion = req.EnableNotification
		customer.NotifyNewRegionEmail = req.Email

		tokenReq := s.transformOrderUser2TokenRequest(order.User)
		_, err = s.updateCustomer(tokenReq, project, customer)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) applyCountryRestriction(order *billing.Order, countryCode string) (restricted bool, err error) {
	restricted = false
	if countryCode == "" {
		return
	}
	country, err := s.country.GetByIsoCodeA2(countryCode)
	if err != nil {
		return
	}
	order.CountryRestriction = &billing.CountryRestriction{
		IsoCodeA2:       countryCode,
		PaymentsAllowed: country.PaymentsAllowed,
		ChangeAllowed:   country.ChangeAllowed,
	}
	if country.PaymentsAllowed {
		return
	}
	if country.ChangeAllowed {
		order.UserAddressDataRequired = true
		return
	}
	order.PrivateStatus = constant.OrderStatusPaymentSystemDeclined
	restricted = true
	err = s.updateOrder(order)
	if err != nil && err.Error() == orderErrorNotFound.Error() {
		err = nil
	}
	return
}

func (s *Service) PaymentFormPlatformChanged(ctx context.Context, req *grpc.PaymentFormUserChangePlatformRequest, rsp *grpc.EmptyResponseWithStatus) error {
	order, err := s.getOrderByUuidToForm(req.OrderId)

	if err != nil {
		zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			rsp.Status = pkg.ResponseStatusBadData
			rsp.Message = e
			return nil
		}
		return err
	}

	rsp.Status = pkg.ResponseStatusOk

	order.PlatformId = req.Platform

	if order.ProductType == billing.OrderType_product {
		err = s.ProcessOrderProducts(ctx, order)
	} else if order.ProductType == billing.OrderType_key {
		_, err = s.ProcessOrderKeyProducts(ctx, order)
	}

	if err != nil {
		if pid := order.PrivateMetadata["PaylinkId"]; pid != "" {
			s.notifyPaylinkError(ctx, pid, err, req, order)
		}
		zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			rsp.Status = pkg.ResponseStatusBadData
			rsp.Message = e
			return nil
		}
		return err
	}

	err = s.updateOrder(order)

	if err != nil {
		zap.S().Errorw(pkg.MethodFinishedWithError, "err", err.Error())
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			rsp.Status = pkg.ResponseStatusSystemError
			rsp.Message = e
			return nil
		}
		return err
	}

	return nil
}

func (s *Service) OrderReceipt(
	ctx context.Context,
	req *grpc.OrderReceiptRequest,
	rsp *grpc.OrderReceiptResponse,
) error {
	order, err := s.orderRepository.GetByUuid(req.OrderId)

	if err != nil {
		zap.L().Error(pkg.MethodFinishedWithError, zap.Error(err))

		rsp.Status = pkg.ResponseStatusBadData
		rsp.Message = err.(*grpc.ResponseErrorMessage)

		return nil
	}

	if order.ReceiptId != req.ReceiptId {
		zap.L().Error(
			orderErrorReceiptNotEquals.Message,
			zap.String("Requested receipt", req.ReceiptId),
			zap.String("Order receipt", order.ReceiptId),
		)

		rsp.Status = pkg.ResponseStatusBadData
		rsp.Message = orderErrorReceiptNotEquals

		return nil
	}

	merchant, err := s.merchant.GetById(order.GetMerchantId())

	if err != nil {
		zap.L().Error(orderErrorMerchantForOrderNotFound.Message, zap.Error(err))

		rsp.Status = pkg.ResponseStatusSystemError
		rsp.Message = orderErrorMerchantForOrderNotFound

		return nil
	}

	totalPrice, err := s.formatter.FormatCurrency(DefaultLanguage, order.OrderAmount, order.Currency)

	if err != nil {
		zap.L().Error(
			orderErrorDuringFormattingCurrency.Message,
			zap.Float64("price", order.OrderAmount),
			zap.String("locale", DefaultLanguage),
			zap.String("currency", order.Currency),
		)

		rsp.Status = pkg.ResponseStatusSystemError
		rsp.Message = orderErrorDuringFormattingCurrency

		return nil
	}

	date, err := s.formatter.FormatDateTime(DefaultLanguage, time.Unix(order.CreatedAt.Seconds, 0))

	if err != nil {
		zap.L().Error(
			orderErrorDuringFormattingDate.Message,
			zap.Any("date", order.CreatedAt),
			zap.String("locale", DefaultLanguage),
		)

		rsp.Status = pkg.ResponseStatusSystemError
		rsp.Message = orderErrorDuringFormattingDate

		return nil
	}

	items := make([]*billing.OrderReceiptItem, len(order.Items))

	for i, item := range order.Items {
		price, err := s.formatter.FormatCurrency("en", item.Amount, item.Currency)

		if err != nil {
			zap.L().Error(
				orderErrorDuringFormattingCurrency.Message,
				zap.Float64("price", item.Amount),
				zap.String("locale", DefaultLanguage),
				zap.String("currency", item.Currency),
			)

			rsp.Status = pkg.ResponseStatusSystemError
			rsp.Message = orderErrorDuringFormattingCurrency

			return nil
		}

		items[i] = &billing.OrderReceiptItem{Name: item.Name, Price: price}
	}

	var platformName = ""

	if platform, ok := availablePlatforms[order.PlatformId]; ok {
		platformName = platform.Name
	}

	receipt := &billing.OrderReceipt{
		TotalPrice:      totalPrice,
		TransactionId:   order.Uuid,
		TransactionDate: date,
		ProjectName:     order.Project.Name[DefaultLanguage],
		MerchantName:    merchant.Company.Name,
		OrderType:       order.Type,
		Items:           items,
		PlatformName:    platformName,
	}

	rsp.Status = pkg.ResponseStatusOk
	rsp.Receipt = receipt

	return nil
}

type OrderRepositoryInterface interface {
	GetByUuid(string) (*billing.Order, error)
}

func newOrderRepository(svc *Service) OrderRepositoryInterface {
	s := &OrderRepository{svc: svc}
	return s
}

func (h *OrderRepository) GetByUuid(uuid string) (*billing.Order, error) {
	order := &billing.Order{}
	err := h.svc.db.Collection(collectionOrder).Find(bson.M{"uuid": uuid}).One(order)

	if err != nil {
		return nil, err
	}

	return order, nil
}

func (v *OrderCreateRequestProcessor) UserCountryExists() bool {
	return v.checked != nil && v.checked.user != nil && v.checked.user.Address != nil &&
		v.checked.user.Address.Country != ""
}

func intersect(a []string, b []string) []string {
	set := make([]string, 0)
	hash := make(map[string]bool)

	for _, v := range a {
		hash[v] = true
	}

	for _, v := range b {
		if _, found := hash[v]; found {
			set = append(set, v)
		}
	}

	return set
}

func (s *Service) hasPaymentCosts(order *billing.Order) bool {
	country, err := s.country.GetByIsoCodeA2(order.GetCountry())

	if err != nil {
		return false
	}

	methodName, err := order.GetCostPaymentMethodName()

	if err != nil {
		return false
	}

	_, err = s.paymentChannelCostSystem.Get(methodName, country.Region, country.IsoCodeA2)

	if err != nil {
		return false
	}

	data := &billing.PaymentChannelCostMerchantRequest{
		MerchantId:     order.GetMerchantId(),
		Name:           methodName,
		PayoutCurrency: order.GetMerchantRoyaltyCurrency(),
		Amount:         order.TotalPaymentAmount,
		Region:         country.Region,
		Country:        country.IsoCodeA2,
	}
	_, err = s.getPaymentChannelCostMerchant(data)
	return err == nil
}

func (v *OrderCreateRequestProcessor) processVirtualCurrency() error {
	amount := v.request.Amount
	virtualCurrency := v.checked.project.VirtualCurrency

	if virtualCurrency == nil || len(virtualCurrency.Prices) <= 0 {
		return orderErrorVirtualCurrencyNotFilled
	}

	_, frac := math.Modf(amount)

	if virtualCurrency.SellCountType == pkg.ProjectSellCountTypeIntegral && frac > 0 {
		return orderErrorVirtualCurrencyFracNotSupported
	}

	if v.checked.amount < virtualCurrency.MinPurchaseValue ||
		(virtualCurrency.MaxPurchaseValue > 0 && amount > virtualCurrency.MaxPurchaseValue) {
		return orderErrorVirtualCurrencyLimits
	}

	v.checked.virtualAmount = amount
	return nil
}
