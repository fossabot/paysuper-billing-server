package service

import (
	"context"
	cryptoRand "crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/golang/protobuf/ptypes"
	"github.com/paysuper/paysuper-billing-server/pkg"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/billing"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/grpc"
	"go.uber.org/zap"
	"math/rand"
	"net"
	"strings"
	"time"
)

const (
	tokenStorageMask   = "paysuper:token:%s"
	tokenLetterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	tokenLetterIdxBits = uint(6)
	tokenLetterIdxMask = uint64(1<<tokenLetterIdxBits - 1)
	tokenLetterIdxMax  = 63 / tokenLetterIdxBits

	collectionCustomer = "customer"
)

var (
	tokenErrorUnknown              = newBillingServerErrorMsg("tk000001", "unknown token error")
	customerNotFound               = newBillingServerErrorMsg("tk000002", "customer by specified data not found")
	tokenErrorNotFound             = newBillingServerErrorMsg("tk000003", "token not found")
	tokenErrorUserIdentityRequired = newBillingServerErrorMsg("tk000004", "request must contain one or more parameters with user information")

	tokenErrorSettingsTypeRequired                            = newBillingServerErrorMsg("tk000005", `field settings.type is required`)
	tokenErrorSettingsSimpleCheckoutParamsRequired            = newBillingServerErrorMsg("tk000006", `fields settings.amount and settings.currency is required for creating payment token with type "simple"`)
	tokenErrorSettingsProductAndKeyProductIdsParamsRequired   = newBillingServerErrorMsg("tk000007", `field settings.product_ids is required for creating payment token with type "product" or "key"`)
	tokenErrorSettingsAmountAndCurrencyParamNotAllowedForType = newBillingServerErrorMsg("tk000008", `fields settings.amount and settings.currency not allowed for creating payment token with types "product" or "key"`)
	tokenErrorSettingsProductIdsParamNotAllowedForType        = newBillingServerErrorMsg("tk000009", `fields settings.product_ids not allowed for creating payment token with type "simple"`)

	tokenRandSource = rand.NewSource(time.Now().UnixNano())
)

type Token struct {
	CustomerId string                 `json:"customer_id"`
	User       *billing.TokenUser     `json:"user"`
	Settings   *billing.TokenSettings `json:"settings"`
}

type tokenRepository struct {
	token   *Token
	service *Service
}

type BrowserCookieCustomer struct {
	CustomerId        string    `json:"customer_id"`
	VirtualCustomerId string    `json:"virtual_customer_id"`
	Ip                string    `json:"ip"`
	UserAgent         string    `json:"user_agent"`
	AcceptLanguage    string    `json:"accept_language"`
	SessionCount      int32     `json:"session_count"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (s *Service) CreateToken(
	ctx context.Context,
	req *grpc.TokenRequest,
	rsp *grpc.TokenResponse,
) error {
	identityExist := req.User.Id != "" || (req.User.Email != nil && req.User.Email.Value != "") ||
		(req.User.Phone != nil && req.User.Phone.Value != "")

	if identityExist == false {
		rsp.Status = pkg.ResponseStatusBadData
		rsp.Message = tokenErrorUserIdentityRequired

		return nil
	}

	processor := &OrderCreateRequestProcessor{
		Service: s,
		request: &billing.OrderCreateRequest{
			ProjectId:  req.Settings.ProjectId,
			Amount:     req.Settings.Amount,
			Currency:   req.Settings.Currency,
			Products:   req.Settings.ProductsIds,
			PlatformId: req.Settings.PlatformId,
			IsBuyForVirtualCurrency: req.Settings.IsBuyForVirtualCurrency,
		},
		checked: &orderCreateRequestProcessorChecked{},
	}

	err := processor.processProject()

	if err != nil {
		rsp.Status = pkg.ResponseStatusBadData
		rsp.Message = err.(*grpc.ResponseErrorMessage)
		return nil
	}

	err = processor.processMerchant()

	if err != nil {
		rsp.Status = pkg.ResponseStatusBadData
		rsp.Message = err.(*grpc.ResponseErrorMessage)
		return nil
	}

	if req.Settings.Type == billing.OrderType_product || req.Settings.Type == billing.OrderType_key {
		if req.Settings.Amount > 0 || req.Settings.Currency != "" {
			rsp.Status = pkg.ResponseStatusBadData
			rsp.Message = tokenErrorSettingsAmountAndCurrencyParamNotAllowedForType
			return nil
		}

		if len(req.Settings.ProductsIds) <= 0 {
			rsp.Status = pkg.ResponseStatusBadData
			rsp.Message = tokenErrorSettingsProductAndKeyProductIdsParamsRequired
			return nil
		}
	}

	switch req.Settings.Type {
	case billing.OrderType_simple:
		if len(req.Settings.ProductsIds) > 0 {
			rsp.Status = pkg.ResponseStatusBadData
			rsp.Message = tokenErrorSettingsProductIdsParamNotAllowedForType
			return nil
		}

		if req.Settings.Amount <= 0 || req.Settings.Currency == "" {
			rsp.Status = pkg.ResponseStatusBadData
			rsp.Message = tokenErrorSettingsSimpleCheckoutParamsRequired
			return nil
		}

		err = processor.processCurrency()

		if err != nil {
			rsp.Status = pkg.ResponseStatusBadData
			rsp.Message = err.(*grpc.ResponseErrorMessage)
			return nil
		}

		processor.processAmount()
		err = processor.processLimitAmounts()

		if err != nil {
			rsp.Status = pkg.ResponseStatusBadData
			rsp.Message = err.(*grpc.ResponseErrorMessage)
			return nil
		}
		break
	case billing.OrderType_product:
		err = processor.processPaylinkProducts()

		if err != nil {
			rsp.Status = pkg.ResponseStatusBadData
			rsp.Message = tokenErrorUnknown

			e, ok := err.(*grpc.ResponseErrorMessage)

			if ok {
				rsp.Message = e
			}

			return nil
		}
		break
	case billing.OrderType_key:
		err = processor.processPaylinkKeyProducts()

		if err != nil {
			rsp.Status = pkg.ResponseStatusBadData
			rsp.Message = tokenErrorUnknown

			e, ok := err.(*grpc.ResponseErrorMessage)

			if ok {
				rsp.Message = e
			}

			return nil
		}
		break
	case billing.OrderTypeVirtualCurrency:
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
	default:
		rsp.Status = pkg.ResponseStatusBadData
		rsp.Message = tokenErrorSettingsTypeRequired
		return nil
	}

	project := processor.checked.project
	customer, err := s.findCustomer(req, project)

	if err != nil && err != customerNotFound {
		rsp.Status = pkg.ResponseStatusBadData
		rsp.Message = err.(*grpc.ResponseErrorMessage)
		return nil
	}

	if customer == nil {
		customer, err = s.createCustomer(req, project)
	} else {
		customer, err = s.updateCustomer(req, project, customer)
	}

	if err != nil {
		zap.S().Errorw(pkg.MethodFinishedWithError, "err", err)
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			rsp.Status = pkg.ResponseStatusBadData
			rsp.Message = e
			return nil
		}
		return err
	}

	token, err := s.createToken(req, customer)

	if err != nil {
		zap.S().Errorw(pkg.MethodFinishedWithError, "err", err)
		if e, ok := err.(*grpc.ResponseErrorMessage); ok {
			rsp.Status = pkg.ResponseStatusBadData
			rsp.Message = e
			return nil
		}
		return err
	}

	rsp.Status = pkg.ResponseStatusOk
	rsp.Token = token

	return nil
}

func (s *Service) createToken(req *grpc.TokenRequest, customer *billing.Customer) (string, error) {
	tokenRep := &tokenRepository{
		service: s,
		token: &Token{
			CustomerId: customer.Id,
			User:       req.User,
			Settings:   req.Settings,
		},
	}
	token := tokenRep.service.getTokenString(s.cfg.GetCustomerTokenLength())
	err := tokenRep.setToken(token)

	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *Service) getTokenBy(token string) (*Token, error) {
	tokenRep := &tokenRepository{
		service: s,
		token:   &Token{},
	}
	err := tokenRep.getToken(token)

	if err != nil {
		return nil, err
	}

	return tokenRep.token, nil
}

func (s *Service) getCustomerById(id string) (*billing.Customer, error) {
	var customer *billing.Customer
	err := s.db.Collection(collectionCustomer).FindId(bson.ObjectIdHex(id)).One(&customer)

	if err != nil {
		if err != mgo.ErrNotFound {
			zap.L().Error(
				"find customer by id failed",
				zap.Error(err),
				zap.String("id", id),
			)
			return nil, orderErrorUnknown
		}

		return nil, customerNotFound
	}

	return customer, nil
}

func (s *Service) findCustomer(
	req *grpc.TokenRequest,
	project *billing.Project,
) (*billing.Customer, error) {
	var subQuery []bson.M
	var subQueryItem bson.M

	if req.User.Id != "" {
		subQueryItem = bson.M{
			"identity": bson.M{
				"$elemMatch": bson.M{
					"type":        pkg.UserIdentityTypeExternal,
					"merchant_id": bson.ObjectIdHex(project.MerchantId),
					"value":       req.User.Id,
				},
			},
		}

		subQuery = append(subQuery, subQueryItem)
	}

	if req.User.Email != nil && req.User.Email.Value != "" {
		subQueryItem = bson.M{
			"identity": bson.M{
				"$elemMatch": bson.M{
					"type":        pkg.UserIdentityTypeEmail,
					"merchant_id": bson.ObjectIdHex(project.MerchantId),
					"value":       req.User.Email.Value,
				},
			},
		}

		subQuery = append(subQuery, subQueryItem)
	}

	if req.User.Phone != nil && req.User.Phone.Value != "" {
		subQueryItem = bson.M{
			"identity": bson.M{
				"$elemMatch": bson.M{
					"type":        pkg.UserIdentityTypePhone,
					"merchant_id": bson.ObjectIdHex(project.MerchantId),
					"value":       req.User.Phone.Value,
				},
			},
		}

		subQuery = append(subQuery, subQueryItem)
	}

	query := make(bson.M)
	customer := new(billing.Customer)

	if subQuery == nil || len(subQuery) <= 0 {
		return nil, customerNotFound
	}

	if len(subQuery) > 1 {
		query["$or"] = subQuery
	} else {
		query = subQuery[0]
	}

	err := s.db.Collection(collectionCustomer).Find(query).One(&customer)

	if err != nil {
		if err != mgo.ErrNotFound {
			zap.L().Error(
				pkg.ErrorDatabaseQueryFailed,
				zap.Error(err),
				zap.String(pkg.ErrorDatabaseFieldCollection, collectionCustomer),
				zap.Any(pkg.ErrorDatabaseFieldQuery, query),
			)
			return nil, orderErrorUnknown
		}

		return nil, customerNotFound
	}

	return customer, nil
}

func (s *Service) createCustomer(
	req *grpc.TokenRequest,
	project *billing.Project,
) (*billing.Customer, error) {
	id := bson.NewObjectId().Hex()

	customer := &billing.Customer{
		Id:        id,
		TechEmail: id + pkg.TechEmailDomain,
		Metadata:  req.User.Metadata,
		CreatedAt: ptypes.TimestampNow(),
		UpdatedAt: ptypes.TimestampNow(),
	}
	s.processCustomer(req, project, customer)

	err := s.db.Collection(collectionCustomer).Insert(customer)

	if err != nil {
		zap.S().Errorf("Query to create new customer failed", "err", err.Error(), "data", customer)
		return nil, tokenErrorUnknown
	}

	return customer, nil
}

func (s *Service) updateCustomer(
	req *grpc.TokenRequest,
	project *billing.Project,
	customer *billing.Customer,
) (*billing.Customer, error) {
	s.processCustomer(req, project, customer)
	err := s.db.Collection(collectionCustomer).UpdateId(bson.ObjectIdHex(customer.Id), customer)

	if err != nil {
		zap.S().Errorf("Query to update customer data failed", "err", err.Error(), "data", customer)
		return nil, tokenErrorUnknown
	}

	return customer, nil
}

func (s *Service) processCustomer(
	req *grpc.TokenRequest,
	project *billing.Project,
	customer *billing.Customer,
) {
	user := req.User

	if user.Id != "" && user.Id != customer.ExternalId {
		customer.ExternalId = user.Id
		identity := &billing.CustomerIdentity{
			MerchantId: project.MerchantId,
			ProjectId:  project.Id,
			Type:       pkg.UserIdentityTypeExternal,
			Value:      user.Id,
			Verified:   true,
			CreatedAt:  ptypes.TimestampNow(),
		}

		customer.Identity = s.processCustomerIdentity(customer.Identity, identity)
	}

	if user.Email != nil && (customer.Email != user.Email.Value || customer.EmailVerified != user.Email.Verified) {
		customer.Email = user.Email.Value
		customer.EmailVerified = user.Email.Verified
		identity := &billing.CustomerIdentity{
			MerchantId: project.MerchantId,
			ProjectId:  project.Id,
			Type:       pkg.UserIdentityTypeEmail,
			Value:      user.Email.Value,
			Verified:   user.Email.Verified,
			CreatedAt:  ptypes.TimestampNow(),
		}

		customer.Identity = s.processCustomerIdentity(customer.Identity, identity)
	}

	if user.Phone != nil && (customer.Phone != user.Phone.Value || customer.PhoneVerified != user.Phone.Verified) {
		customer.Phone = user.Phone.Value
		customer.PhoneVerified = user.Phone.Verified
		identity := &billing.CustomerIdentity{
			MerchantId: project.MerchantId,
			ProjectId:  project.Id,
			Type:       pkg.UserIdentityTypePhone,
			Value:      user.Phone.Value,
			Verified:   user.Phone.Verified,
			CreatedAt:  ptypes.TimestampNow(),
		}

		customer.Identity = s.processCustomerIdentity(customer.Identity, identity)
	}

	if user.Name != nil && customer.Name != user.Name.Value {
		customer.Name = user.Name.Value
	}

	if user.Ip != nil && user.Ip.Value != "" {
		ip := net.IP(customer.Ip)
		customer.Ip = net.ParseIP(user.Ip.Value)

		if len(ip) > 0 && ip.String() != user.Ip.Value {
			history := &billing.CustomerIpHistory{
				Ip:        ip,
				CreatedAt: ptypes.TimestampNow(),
			}
			customer.IpHistory = append(customer.IpHistory, history)
		}
	}

	if user.Locale != nil && user.Locale.Value != "" && customer.Locale != user.Locale.Value {
		history := &billing.CustomerStringValueHistory{
			Value:     customer.Locale,
			CreatedAt: ptypes.TimestampNow(),
		}
		customer.Locale = user.Locale.Value

		if history.Value != "" {
			customer.LocaleHistory = append(customer.LocaleHistory, history)
		}
	}

	if user.Address != nil && customer.Address != user.Address {
		if customer.Address != nil {
			history := &billing.CustomerAddressHistory{
				Country:    customer.Address.Country,
				City:       customer.Address.City,
				PostalCode: customer.Address.PostalCode,
				State:      customer.Address.State,
				CreatedAt:  ptypes.TimestampNow(),
			}
			customer.AddressHistory = append(customer.AddressHistory, history)
		}

		customer.Address = user.Address
	}

	if user.UserAgent != "" && customer.UserAgent != user.UserAgent {
		customer.UserAgent = user.UserAgent
	}

	if user.AcceptLanguage != "" && customer.AcceptLanguage != user.AcceptLanguage {
		history := &billing.CustomerStringValueHistory{
			Value:     customer.AcceptLanguage,
			CreatedAt: ptypes.TimestampNow(),
		}
		customer.AcceptLanguage = user.AcceptLanguage

		if history.Value != "" {
			customer.AcceptLanguageHistory = append(customer.AcceptLanguageHistory, history)
		}
	}
}

func (s *Service) processCustomerIdentity(
	currentIdentities []*billing.CustomerIdentity,
	newIdentity *billing.CustomerIdentity,
) []*billing.CustomerIdentity {
	if len(currentIdentities) <= 0 {
		return append(currentIdentities, newIdentity)
	}

	isNewIdentity := true

	for k, v := range currentIdentities {
		needChange := v.Type == newIdentity.Type && v.ProjectId == newIdentity.ProjectId &&
			v.MerchantId == newIdentity.MerchantId && v.Value == newIdentity.Value && v.Verified != newIdentity.Verified

		if needChange == false {
			continue
		}

		currentIdentities[k] = newIdentity
		isNewIdentity = false
	}

	if isNewIdentity == true {
		currentIdentities = append(currentIdentities, newIdentity)
	}

	return currentIdentities
}

func (s *Service) transformOrderUser2TokenRequest(user *billing.OrderUser) *grpc.TokenRequest {
	tokenReq := &grpc.TokenRequest{User: &billing.TokenUser{}}

	if user.ExternalId != "" {
		tokenReq.User.Id = user.ExternalId
	}

	if user.Name != "" {
		tokenReq.User.Name = &billing.TokenUserValue{Value: user.Name}
	}

	if user.Email != "" {
		tokenReq.User.Email = &billing.TokenUserEmailValue{
			Value:    user.Email,
			Verified: user.EmailVerified,
		}
	}

	if user.Phone != "" {
		tokenReq.User.Phone = &billing.TokenUserPhoneValue{
			Value:    user.Phone,
			Verified: user.PhoneVerified,
		}
	}

	if user.Ip != "" {
		tokenReq.User.Ip = &billing.TokenUserIpValue{Value: user.Ip}
	}

	if user.Locale != "" {
		tokenReq.User.Locale = &billing.TokenUserLocaleValue{Value: user.Locale}
	}

	if user.Address != nil {
		tokenReq.User.Address = user.Address
	}

	if len(user.Metadata) > 0 {
		tokenReq.User.Metadata = user.Metadata
	}

	return tokenReq
}

func (r *tokenRepository) getToken(token string) error {
	data, err := r.service.redis.Get(r.getKey(token)).Bytes()

	if err != nil {
		r.service.logError("Get customer token from Redis failed", []interface{}{"error", err.Error()})
		return tokenErrorNotFound
	}

	err = json.Unmarshal(data, &r.token)

	if err != nil {
		r.service.logError("Unmarshal customer token failed", []interface{}{"error", err.Error()})
		return tokenErrorNotFound
	}

	return nil
}

func (r *tokenRepository) setToken(token string) error {
	b, err := json.Marshal(r.token)

	if err != nil {
		r.service.logError("Marshal customer token failed", []interface{}{"error", err.Error()})
		return tokenErrorUnknown
	}

	return r.service.redis.Set(r.getKey(token), b, r.service.cfg.GetCustomerTokenExpire()).Err()
}

func (r *tokenRepository) getKey(token string) string {
	return fmt.Sprintf(tokenStorageMask, token)
}

func (s *Service) getTokenString(n int) string {
	sb := strings.Builder{}
	sb.Grow(n)

	for i, cache, remain := n-1, tokenRandSource.Int63(), tokenLetterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = tokenRandSource.Int63(), tokenLetterIdxMax
		}

		if idx := int(uint64(cache) & tokenLetterIdxMask); idx < len(tokenLetterBytes) {
			sb.WriteByte(tokenLetterBytes[idx])
			i--
		}

		cache >>= tokenLetterIdxBits
		remain--
	}

	return sb.String()
}

func (s *Service) updateCustomerFromRequest(
	order *billing.Order,
	req *grpc.TokenRequest,
	ip, acceptLanguage, userAgent string,
) (*billing.Customer, error) {
	customer, err := s.getCustomerById(order.User.Id)
	project := &billing.Project{Id: order.Project.Id, MerchantId: order.Project.MerchantId}

	if err != nil {
		return nil, err
	}

	req.User.Ip = &billing.TokenUserIpValue{Value: ip}
	req.User.AcceptLanguage = acceptLanguage
	req.User.UserAgent = userAgent

	req.User.Locale = &billing.TokenUserLocaleValue{}
	req.User.Locale.Value, _ = s.getCountryFromAcceptLanguage(acceptLanguage)

	return s.updateCustomer(req, project, customer)
}

func (s *Service) updateCustomerFromRequestLocale(
	order *billing.Order,
	ip, acceptLanguage, userAgent, locale string,
) {
	tokenReq := &grpc.TokenRequest{
		User: &billing.TokenUser{
			Locale: &billing.TokenUserLocaleValue{Value: locale},
		},
	}

	_, err := s.updateCustomerFromRequest(order, tokenReq, ip, acceptLanguage, userAgent)

	if err != nil {
		zap.S().Errorf("Update customer data by request failed", "err", err.Error())
	}
}

func (s *Service) generateBrowserCookie(customer *BrowserCookieCustomer) (string, error) {
	b, err := json.Marshal(customer)

	if err != nil {
		zap.S().Errorf("Customer cookie generation failed", "err", err.Error())
		return "", err
	}

	hash := sha512.New()
	cookie, err := rsa.EncryptOAEP(hash, cryptoRand.Reader, s.cfg.CookiePublicKey, b, nil)

	if err != nil {
		zap.S().Errorf("Customer cookie generation failed", "err", err.Error())
		return "", err
	}

	return base64.StdEncoding.EncodeToString(cookie), nil
}

func (s *Service) decryptBrowserCookie(cookie string) (*BrowserCookieCustomer, error) {
	bCookie, err := base64.StdEncoding.DecodeString(cookie)

	if err != nil {
		zap.S().Errorf("Customer cookie base64 decode failed", "err", err.Error())
		return nil, err
	}

	hash := sha512.New()
	res, err := rsa.DecryptOAEP(hash, cryptoRand.Reader, s.cfg.CookiePrivateKey, bCookie, nil)

	if err != nil {
		zap.L().Error("Customer cookie decrypt failed", zap.Error(err))
		return nil, err
	}

	customer := &BrowserCookieCustomer{}
	err = json.Unmarshal(res, &customer)

	if err != nil {
		zap.L().Error("Customer cookie decrypt failed", zap.Error(err))
		return nil, err
	}

	return customer, nil
}
