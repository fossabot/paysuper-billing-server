// Code generated by protoc-gen-micro. DO NOT EDIT.
// source: grpc/grpc.proto

/*
Package grpc is a generated protocol buffer package.

It is generated from these files:
	grpc/grpc.proto

It has these top-level messages:
	EmptyRequest
	EmptyResponse
	PaymentCreateRequest
	PaymentCreateResponse
	PaymentFormJsonDataRequest
	PaymentFormJsonDataProject
	PaymentFormJsonDataResponse
	PaymentNotifyRequest
	PaymentNotifyResponse
	ConvertRateRequest
	ConvertRateResponse
	OnboardingBanking
	OnboardingRequest
	FindByIdRequest
	MerchantListingRequest
	MerchantListingResponse
	MerchantChangeStatusRequest
	NotificationRequest
	Notifications
	ListingNotificationRequest
	ListingMerchantPaymentMethod
	GetMerchantPaymentMethodRequest
	ListMerchantPaymentMethodsRequest
	MerchantPaymentMethodRequest
	MerchantPaymentMethodResponse
	MerchantGetMerchantResponse
	GetNotificationRequest
	CreateRefundRequest
	CreateRefundResponse
	ListRefundsRequest
	ListRefundsResponse
	GetRefundRequest
	CallbackRequest
	PaymentFormDataChangedRequest
	PaymentFormUserChangeLangRequest
	PaymentFormUserChangePaymentAccountRequest
	UserIpData
	PaymentFormDataChangeResponseItem
	PaymentFormDataChangeResponse
	ProcessBillingAddressRequest
	ProcessBillingAddressResponseItem
	ProcessBillingAddressResponse
	GetMerchantByRequest
	ChangeMerchantDataRequest
	ChangeMerchantDataResponse
	SetMerchantS3AgreementRequest
	Product
	ProductPrice
	ListProductsRequest
	GetProductsForOrderRequest
	ListProductsResponse
	RequestProduct
	I18NTextSearchable
*/
package grpc

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import _ "github.com/golang/protobuf/ptypes/timestamp"
import billing "github.com/paysuper/paysuper-billing-server/pkg/proto/billing"

import (
	context "context"
	client "github.com/micro/go-micro/client"
	server "github.com/micro/go-micro/server"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf
var _ = billing.SystemFeesList{}

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ client.Option
var _ server.Option

// Client API for BillingService service

type BillingService interface {
	OrderCreateProcess(ctx context.Context, in *billing.OrderCreateRequest, opts ...client.CallOption) (*billing.Order, error)
	PaymentFormJsonDataProcess(ctx context.Context, in *PaymentFormJsonDataRequest, opts ...client.CallOption) (*PaymentFormJsonDataResponse, error)
	PaymentCreateProcess(ctx context.Context, in *PaymentCreateRequest, opts ...client.CallOption) (*PaymentCreateResponse, error)
	PaymentCallbackProcess(ctx context.Context, in *PaymentNotifyRequest, opts ...client.CallOption) (*PaymentNotifyResponse, error)
	RebuildCache(ctx context.Context, in *EmptyRequest, opts ...client.CallOption) (*EmptyResponse, error)
	UpdateOrder(ctx context.Context, in *billing.Order, opts ...client.CallOption) (*EmptyResponse, error)
	UpdateMerchant(ctx context.Context, in *billing.Merchant, opts ...client.CallOption) (*EmptyResponse, error)
	GetConvertRate(ctx context.Context, in *ConvertRateRequest, opts ...client.CallOption) (*ConvertRateResponse, error)
	GetMerchantBy(ctx context.Context, in *GetMerchantByRequest, opts ...client.CallOption) (*MerchantGetMerchantResponse, error)
	ListMerchants(ctx context.Context, in *MerchantListingRequest, opts ...client.CallOption) (*MerchantListingResponse, error)
	ChangeMerchant(ctx context.Context, in *OnboardingRequest, opts ...client.CallOption) (*billing.Merchant, error)
	ChangeMerchantStatus(ctx context.Context, in *MerchantChangeStatusRequest, opts ...client.CallOption) (*billing.Merchant, error)
	ChangeMerchantData(ctx context.Context, in *ChangeMerchantDataRequest, opts ...client.CallOption) (*ChangeMerchantDataResponse, error)
	SetMerchantS3Agreement(ctx context.Context, in *SetMerchantS3AgreementRequest, opts ...client.CallOption) (*ChangeMerchantDataResponse, error)
	CreateNotification(ctx context.Context, in *NotificationRequest, opts ...client.CallOption) (*billing.Notification, error)
	GetNotification(ctx context.Context, in *GetNotificationRequest, opts ...client.CallOption) (*billing.Notification, error)
	ListNotifications(ctx context.Context, in *ListingNotificationRequest, opts ...client.CallOption) (*Notifications, error)
	MarkNotificationAsRead(ctx context.Context, in *GetNotificationRequest, opts ...client.CallOption) (*billing.Notification, error)
	ListMerchantPaymentMethods(ctx context.Context, in *ListMerchantPaymentMethodsRequest, opts ...client.CallOption) (*ListingMerchantPaymentMethod, error)
	GetMerchantPaymentMethod(ctx context.Context, in *GetMerchantPaymentMethodRequest, opts ...client.CallOption) (*billing.MerchantPaymentMethod, error)
	ChangeMerchantPaymentMethod(ctx context.Context, in *MerchantPaymentMethodRequest, opts ...client.CallOption) (*MerchantPaymentMethodResponse, error)
	CreateRefund(ctx context.Context, in *CreateRefundRequest, opts ...client.CallOption) (*CreateRefundResponse, error)
	ListRefunds(ctx context.Context, in *ListRefundsRequest, opts ...client.CallOption) (*ListRefundsResponse, error)
	GetRefund(ctx context.Context, in *GetRefundRequest, opts ...client.CallOption) (*CreateRefundResponse, error)
	ProcessRefundCallback(ctx context.Context, in *CallbackRequest, opts ...client.CallOption) (*PaymentNotifyResponse, error)
	PaymentFormLanguageChanged(ctx context.Context, in *PaymentFormUserChangeLangRequest, opts ...client.CallOption) (*PaymentFormDataChangeResponse, error)
	PaymentFormPaymentAccountChanged(ctx context.Context, in *PaymentFormUserChangePaymentAccountRequest, opts ...client.CallOption) (*PaymentFormDataChangeResponse, error)
	ProcessBillingAddress(ctx context.Context, in *ProcessBillingAddressRequest, opts ...client.CallOption) (*ProcessBillingAddressResponse, error)
	CreateOrUpdateProduct(ctx context.Context, in *Product, opts ...client.CallOption) (*Product, error)
	ListProducts(ctx context.Context, in *ListProductsRequest, opts ...client.CallOption) (*ListProductsResponse, error)
	GetProduct(ctx context.Context, in *RequestProduct, opts ...client.CallOption) (*Product, error)
	DeleteProduct(ctx context.Context, in *RequestProduct, opts ...client.CallOption) (*EmptyResponse, error)
	GetProductsForOrder(ctx context.Context, in *GetProductsForOrderRequest, opts ...client.CallOption) (*ListProductsResponse, error)
	AddSystemFees(ctx context.Context, in *billing.AddSystemFeesRequest, opts ...client.CallOption) (*EmptyResponse, error)
	GetSystemFeesForPayment(ctx context.Context, in *billing.GetSystemFeesRequest, opts ...client.CallOption) (*billing.FeeSet, error)
	GetActualSystemFeesList(ctx context.Context, in *EmptyRequest, opts ...client.CallOption) (*billing.SystemFeesList, error)
}

type billingService struct {
	c    client.Client
	name string
}

func NewBillingService(name string, c client.Client) BillingService {
	if c == nil {
		c = client.NewClient()
	}
	if len(name) == 0 {
		name = "grpc"
	}
	return &billingService{
		c:    c,
		name: name,
	}
}

func (c *billingService) OrderCreateProcess(ctx context.Context, in *billing.OrderCreateRequest, opts ...client.CallOption) (*billing.Order, error) {
	req := c.c.NewRequest(c.name, "BillingService.OrderCreateProcess", in)
	out := new(billing.Order)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) PaymentFormJsonDataProcess(ctx context.Context, in *PaymentFormJsonDataRequest, opts ...client.CallOption) (*PaymentFormJsonDataResponse, error) {
	req := c.c.NewRequest(c.name, "BillingService.PaymentFormJsonDataProcess", in)
	out := new(PaymentFormJsonDataResponse)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) PaymentCreateProcess(ctx context.Context, in *PaymentCreateRequest, opts ...client.CallOption) (*PaymentCreateResponse, error) {
	req := c.c.NewRequest(c.name, "BillingService.PaymentCreateProcess", in)
	out := new(PaymentCreateResponse)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) PaymentCallbackProcess(ctx context.Context, in *PaymentNotifyRequest, opts ...client.CallOption) (*PaymentNotifyResponse, error) {
	req := c.c.NewRequest(c.name, "BillingService.PaymentCallbackProcess", in)
	out := new(PaymentNotifyResponse)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) RebuildCache(ctx context.Context, in *EmptyRequest, opts ...client.CallOption) (*EmptyResponse, error) {
	req := c.c.NewRequest(c.name, "BillingService.RebuildCache", in)
	out := new(EmptyResponse)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) UpdateOrder(ctx context.Context, in *billing.Order, opts ...client.CallOption) (*EmptyResponse, error) {
	req := c.c.NewRequest(c.name, "BillingService.UpdateOrder", in)
	out := new(EmptyResponse)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) UpdateMerchant(ctx context.Context, in *billing.Merchant, opts ...client.CallOption) (*EmptyResponse, error) {
	req := c.c.NewRequest(c.name, "BillingService.UpdateMerchant", in)
	out := new(EmptyResponse)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) GetConvertRate(ctx context.Context, in *ConvertRateRequest, opts ...client.CallOption) (*ConvertRateResponse, error) {
	req := c.c.NewRequest(c.name, "BillingService.GetConvertRate", in)
	out := new(ConvertRateResponse)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) GetMerchantBy(ctx context.Context, in *GetMerchantByRequest, opts ...client.CallOption) (*MerchantGetMerchantResponse, error) {
	req := c.c.NewRequest(c.name, "BillingService.GetMerchantBy", in)
	out := new(MerchantGetMerchantResponse)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) ListMerchants(ctx context.Context, in *MerchantListingRequest, opts ...client.CallOption) (*MerchantListingResponse, error) {
	req := c.c.NewRequest(c.name, "BillingService.ListMerchants", in)
	out := new(MerchantListingResponse)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) ChangeMerchant(ctx context.Context, in *OnboardingRequest, opts ...client.CallOption) (*billing.Merchant, error) {
	req := c.c.NewRequest(c.name, "BillingService.ChangeMerchant", in)
	out := new(billing.Merchant)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) ChangeMerchantStatus(ctx context.Context, in *MerchantChangeStatusRequest, opts ...client.CallOption) (*billing.Merchant, error) {
	req := c.c.NewRequest(c.name, "BillingService.ChangeMerchantStatus", in)
	out := new(billing.Merchant)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) ChangeMerchantData(ctx context.Context, in *ChangeMerchantDataRequest, opts ...client.CallOption) (*ChangeMerchantDataResponse, error) {
	req := c.c.NewRequest(c.name, "BillingService.ChangeMerchantData", in)
	out := new(ChangeMerchantDataResponse)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) SetMerchantS3Agreement(ctx context.Context, in *SetMerchantS3AgreementRequest, opts ...client.CallOption) (*ChangeMerchantDataResponse, error) {
	req := c.c.NewRequest(c.name, "BillingService.SetMerchantS3Agreement", in)
	out := new(ChangeMerchantDataResponse)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) CreateNotification(ctx context.Context, in *NotificationRequest, opts ...client.CallOption) (*billing.Notification, error) {
	req := c.c.NewRequest(c.name, "BillingService.CreateNotification", in)
	out := new(billing.Notification)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) GetNotification(ctx context.Context, in *GetNotificationRequest, opts ...client.CallOption) (*billing.Notification, error) {
	req := c.c.NewRequest(c.name, "BillingService.GetNotification", in)
	out := new(billing.Notification)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) ListNotifications(ctx context.Context, in *ListingNotificationRequest, opts ...client.CallOption) (*Notifications, error) {
	req := c.c.NewRequest(c.name, "BillingService.ListNotifications", in)
	out := new(Notifications)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) MarkNotificationAsRead(ctx context.Context, in *GetNotificationRequest, opts ...client.CallOption) (*billing.Notification, error) {
	req := c.c.NewRequest(c.name, "BillingService.MarkNotificationAsRead", in)
	out := new(billing.Notification)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) ListMerchantPaymentMethods(ctx context.Context, in *ListMerchantPaymentMethodsRequest, opts ...client.CallOption) (*ListingMerchantPaymentMethod, error) {
	req := c.c.NewRequest(c.name, "BillingService.ListMerchantPaymentMethods", in)
	out := new(ListingMerchantPaymentMethod)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) GetMerchantPaymentMethod(ctx context.Context, in *GetMerchantPaymentMethodRequest, opts ...client.CallOption) (*billing.MerchantPaymentMethod, error) {
	req := c.c.NewRequest(c.name, "BillingService.GetMerchantPaymentMethod", in)
	out := new(billing.MerchantPaymentMethod)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) ChangeMerchantPaymentMethod(ctx context.Context, in *MerchantPaymentMethodRequest, opts ...client.CallOption) (*MerchantPaymentMethodResponse, error) {
	req := c.c.NewRequest(c.name, "BillingService.ChangeMerchantPaymentMethod", in)
	out := new(MerchantPaymentMethodResponse)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) CreateRefund(ctx context.Context, in *CreateRefundRequest, opts ...client.CallOption) (*CreateRefundResponse, error) {
	req := c.c.NewRequest(c.name, "BillingService.CreateRefund", in)
	out := new(CreateRefundResponse)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) ListRefunds(ctx context.Context, in *ListRefundsRequest, opts ...client.CallOption) (*ListRefundsResponse, error) {
	req := c.c.NewRequest(c.name, "BillingService.ListRefunds", in)
	out := new(ListRefundsResponse)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) GetRefund(ctx context.Context, in *GetRefundRequest, opts ...client.CallOption) (*CreateRefundResponse, error) {
	req := c.c.NewRequest(c.name, "BillingService.GetRefund", in)
	out := new(CreateRefundResponse)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) ProcessRefundCallback(ctx context.Context, in *CallbackRequest, opts ...client.CallOption) (*PaymentNotifyResponse, error) {
	req := c.c.NewRequest(c.name, "BillingService.ProcessRefundCallback", in)
	out := new(PaymentNotifyResponse)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) PaymentFormLanguageChanged(ctx context.Context, in *PaymentFormUserChangeLangRequest, opts ...client.CallOption) (*PaymentFormDataChangeResponse, error) {
	req := c.c.NewRequest(c.name, "BillingService.PaymentFormLanguageChanged", in)
	out := new(PaymentFormDataChangeResponse)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) PaymentFormPaymentAccountChanged(ctx context.Context, in *PaymentFormUserChangePaymentAccountRequest, opts ...client.CallOption) (*PaymentFormDataChangeResponse, error) {
	req := c.c.NewRequest(c.name, "BillingService.PaymentFormPaymentAccountChanged", in)
	out := new(PaymentFormDataChangeResponse)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) ProcessBillingAddress(ctx context.Context, in *ProcessBillingAddressRequest, opts ...client.CallOption) (*ProcessBillingAddressResponse, error) {
	req := c.c.NewRequest(c.name, "BillingService.ProcessBillingAddress", in)
	out := new(ProcessBillingAddressResponse)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) CreateOrUpdateProduct(ctx context.Context, in *Product, opts ...client.CallOption) (*Product, error) {
	req := c.c.NewRequest(c.name, "BillingService.CreateOrUpdateProduct", in)
	out := new(Product)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) ListProducts(ctx context.Context, in *ListProductsRequest, opts ...client.CallOption) (*ListProductsResponse, error) {
	req := c.c.NewRequest(c.name, "BillingService.ListProducts", in)
	out := new(ListProductsResponse)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) GetProduct(ctx context.Context, in *RequestProduct, opts ...client.CallOption) (*Product, error) {
	req := c.c.NewRequest(c.name, "BillingService.GetProduct", in)
	out := new(Product)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) DeleteProduct(ctx context.Context, in *RequestProduct, opts ...client.CallOption) (*EmptyResponse, error) {
	req := c.c.NewRequest(c.name, "BillingService.DeleteProduct", in)
	out := new(EmptyResponse)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) GetProductsForOrder(ctx context.Context, in *GetProductsForOrderRequest, opts ...client.CallOption) (*ListProductsResponse, error) {
	req := c.c.NewRequest(c.name, "BillingService.GetProductsForOrder", in)
	out := new(ListProductsResponse)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) AddSystemFees(ctx context.Context, in *billing.AddSystemFeesRequest, opts ...client.CallOption) (*EmptyResponse, error) {
	req := c.c.NewRequest(c.name, "BillingService.AddSystemFees", in)
	out := new(EmptyResponse)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) GetSystemFeesForPayment(ctx context.Context, in *billing.GetSystemFeesRequest, opts ...client.CallOption) (*billing.FeeSet, error) {
	req := c.c.NewRequest(c.name, "BillingService.GetSystemFeesForPayment", in)
	out := new(billing.FeeSet)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *billingService) GetActualSystemFeesList(ctx context.Context, in *EmptyRequest, opts ...client.CallOption) (*billing.SystemFeesList, error) {
	req := c.c.NewRequest(c.name, "BillingService.GetActualSystemFeesList", in)
	out := new(billing.SystemFeesList)
	err := c.c.Call(ctx, req, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for BillingService service

type BillingServiceHandler interface {
	OrderCreateProcess(context.Context, *billing.OrderCreateRequest, *billing.Order) error
	PaymentFormJsonDataProcess(context.Context, *PaymentFormJsonDataRequest, *PaymentFormJsonDataResponse) error
	PaymentCreateProcess(context.Context, *PaymentCreateRequest, *PaymentCreateResponse) error
	PaymentCallbackProcess(context.Context, *PaymentNotifyRequest, *PaymentNotifyResponse) error
	RebuildCache(context.Context, *EmptyRequest, *EmptyResponse) error
	UpdateOrder(context.Context, *billing.Order, *EmptyResponse) error
	UpdateMerchant(context.Context, *billing.Merchant, *EmptyResponse) error
	GetConvertRate(context.Context, *ConvertRateRequest, *ConvertRateResponse) error
	GetMerchantBy(context.Context, *GetMerchantByRequest, *MerchantGetMerchantResponse) error
	ListMerchants(context.Context, *MerchantListingRequest, *MerchantListingResponse) error
	ChangeMerchant(context.Context, *OnboardingRequest, *billing.Merchant) error
	ChangeMerchantStatus(context.Context, *MerchantChangeStatusRequest, *billing.Merchant) error
	ChangeMerchantData(context.Context, *ChangeMerchantDataRequest, *ChangeMerchantDataResponse) error
	SetMerchantS3Agreement(context.Context, *SetMerchantS3AgreementRequest, *ChangeMerchantDataResponse) error
	CreateNotification(context.Context, *NotificationRequest, *billing.Notification) error
	GetNotification(context.Context, *GetNotificationRequest, *billing.Notification) error
	ListNotifications(context.Context, *ListingNotificationRequest, *Notifications) error
	MarkNotificationAsRead(context.Context, *GetNotificationRequest, *billing.Notification) error
	ListMerchantPaymentMethods(context.Context, *ListMerchantPaymentMethodsRequest, *ListingMerchantPaymentMethod) error
	GetMerchantPaymentMethod(context.Context, *GetMerchantPaymentMethodRequest, *billing.MerchantPaymentMethod) error
	ChangeMerchantPaymentMethod(context.Context, *MerchantPaymentMethodRequest, *MerchantPaymentMethodResponse) error
	CreateRefund(context.Context, *CreateRefundRequest, *CreateRefundResponse) error
	ListRefunds(context.Context, *ListRefundsRequest, *ListRefundsResponse) error
	GetRefund(context.Context, *GetRefundRequest, *CreateRefundResponse) error
	ProcessRefundCallback(context.Context, *CallbackRequest, *PaymentNotifyResponse) error
	PaymentFormLanguageChanged(context.Context, *PaymentFormUserChangeLangRequest, *PaymentFormDataChangeResponse) error
	PaymentFormPaymentAccountChanged(context.Context, *PaymentFormUserChangePaymentAccountRequest, *PaymentFormDataChangeResponse) error
	ProcessBillingAddress(context.Context, *ProcessBillingAddressRequest, *ProcessBillingAddressResponse) error
	CreateOrUpdateProduct(context.Context, *Product, *Product) error
	ListProducts(context.Context, *ListProductsRequest, *ListProductsResponse) error
	GetProduct(context.Context, *RequestProduct, *Product) error
	DeleteProduct(context.Context, *RequestProduct, *EmptyResponse) error
	GetProductsForOrder(context.Context, *GetProductsForOrderRequest, *ListProductsResponse) error
	AddSystemFees(context.Context, *billing.AddSystemFeesRequest, *EmptyResponse) error
	GetSystemFeesForPayment(context.Context, *billing.GetSystemFeesRequest, *billing.FeeSet) error
	GetActualSystemFeesList(context.Context, *EmptyRequest, *billing.SystemFeesList) error
}

func RegisterBillingServiceHandler(s server.Server, hdlr BillingServiceHandler, opts ...server.HandlerOption) error {
	type billingService interface {
		OrderCreateProcess(ctx context.Context, in *billing.OrderCreateRequest, out *billing.Order) error
		PaymentFormJsonDataProcess(ctx context.Context, in *PaymentFormJsonDataRequest, out *PaymentFormJsonDataResponse) error
		PaymentCreateProcess(ctx context.Context, in *PaymentCreateRequest, out *PaymentCreateResponse) error
		PaymentCallbackProcess(ctx context.Context, in *PaymentNotifyRequest, out *PaymentNotifyResponse) error
		RebuildCache(ctx context.Context, in *EmptyRequest, out *EmptyResponse) error
		UpdateOrder(ctx context.Context, in *billing.Order, out *EmptyResponse) error
		UpdateMerchant(ctx context.Context, in *billing.Merchant, out *EmptyResponse) error
		GetConvertRate(ctx context.Context, in *ConvertRateRequest, out *ConvertRateResponse) error
		GetMerchantBy(ctx context.Context, in *GetMerchantByRequest, out *MerchantGetMerchantResponse) error
		ListMerchants(ctx context.Context, in *MerchantListingRequest, out *MerchantListingResponse) error
		ChangeMerchant(ctx context.Context, in *OnboardingRequest, out *billing.Merchant) error
		ChangeMerchantStatus(ctx context.Context, in *MerchantChangeStatusRequest, out *billing.Merchant) error
		ChangeMerchantData(ctx context.Context, in *ChangeMerchantDataRequest, out *ChangeMerchantDataResponse) error
		SetMerchantS3Agreement(ctx context.Context, in *SetMerchantS3AgreementRequest, out *ChangeMerchantDataResponse) error
		CreateNotification(ctx context.Context, in *NotificationRequest, out *billing.Notification) error
		GetNotification(ctx context.Context, in *GetNotificationRequest, out *billing.Notification) error
		ListNotifications(ctx context.Context, in *ListingNotificationRequest, out *Notifications) error
		MarkNotificationAsRead(ctx context.Context, in *GetNotificationRequest, out *billing.Notification) error
		ListMerchantPaymentMethods(ctx context.Context, in *ListMerchantPaymentMethodsRequest, out *ListingMerchantPaymentMethod) error
		GetMerchantPaymentMethod(ctx context.Context, in *GetMerchantPaymentMethodRequest, out *billing.MerchantPaymentMethod) error
		ChangeMerchantPaymentMethod(ctx context.Context, in *MerchantPaymentMethodRequest, out *MerchantPaymentMethodResponse) error
		CreateRefund(ctx context.Context, in *CreateRefundRequest, out *CreateRefundResponse) error
		ListRefunds(ctx context.Context, in *ListRefundsRequest, out *ListRefundsResponse) error
		GetRefund(ctx context.Context, in *GetRefundRequest, out *CreateRefundResponse) error
		ProcessRefundCallback(ctx context.Context, in *CallbackRequest, out *PaymentNotifyResponse) error
		PaymentFormLanguageChanged(ctx context.Context, in *PaymentFormUserChangeLangRequest, out *PaymentFormDataChangeResponse) error
		PaymentFormPaymentAccountChanged(ctx context.Context, in *PaymentFormUserChangePaymentAccountRequest, out *PaymentFormDataChangeResponse) error
		ProcessBillingAddress(ctx context.Context, in *ProcessBillingAddressRequest, out *ProcessBillingAddressResponse) error
		CreateOrUpdateProduct(ctx context.Context, in *Product, out *Product) error
		ListProducts(ctx context.Context, in *ListProductsRequest, out *ListProductsResponse) error
		GetProduct(ctx context.Context, in *RequestProduct, out *Product) error
		DeleteProduct(ctx context.Context, in *RequestProduct, out *EmptyResponse) error
		GetProductsForOrder(ctx context.Context, in *GetProductsForOrderRequest, out *ListProductsResponse) error
		AddSystemFees(ctx context.Context, in *billing.AddSystemFeesRequest, out *EmptyResponse) error
		GetSystemFeesForPayment(ctx context.Context, in *billing.GetSystemFeesRequest, out *billing.FeeSet) error
		GetActualSystemFeesList(ctx context.Context, in *EmptyRequest, out *billing.SystemFeesList) error
	}
	type BillingService struct {
		billingService
	}
	h := &billingServiceHandler{hdlr}
	return s.Handle(s.NewHandler(&BillingService{h}, opts...))
}

type billingServiceHandler struct {
	BillingServiceHandler
}

func (h *billingServiceHandler) OrderCreateProcess(ctx context.Context, in *billing.OrderCreateRequest, out *billing.Order) error {
	return h.BillingServiceHandler.OrderCreateProcess(ctx, in, out)
}

func (h *billingServiceHandler) PaymentFormJsonDataProcess(ctx context.Context, in *PaymentFormJsonDataRequest, out *PaymentFormJsonDataResponse) error {
	return h.BillingServiceHandler.PaymentFormJsonDataProcess(ctx, in, out)
}

func (h *billingServiceHandler) PaymentCreateProcess(ctx context.Context, in *PaymentCreateRequest, out *PaymentCreateResponse) error {
	return h.BillingServiceHandler.PaymentCreateProcess(ctx, in, out)
}

func (h *billingServiceHandler) PaymentCallbackProcess(ctx context.Context, in *PaymentNotifyRequest, out *PaymentNotifyResponse) error {
	return h.BillingServiceHandler.PaymentCallbackProcess(ctx, in, out)
}

func (h *billingServiceHandler) RebuildCache(ctx context.Context, in *EmptyRequest, out *EmptyResponse) error {
	return h.BillingServiceHandler.RebuildCache(ctx, in, out)
}

func (h *billingServiceHandler) UpdateOrder(ctx context.Context, in *billing.Order, out *EmptyResponse) error {
	return h.BillingServiceHandler.UpdateOrder(ctx, in, out)
}

func (h *billingServiceHandler) UpdateMerchant(ctx context.Context, in *billing.Merchant, out *EmptyResponse) error {
	return h.BillingServiceHandler.UpdateMerchant(ctx, in, out)
}

func (h *billingServiceHandler) GetConvertRate(ctx context.Context, in *ConvertRateRequest, out *ConvertRateResponse) error {
	return h.BillingServiceHandler.GetConvertRate(ctx, in, out)
}

func (h *billingServiceHandler) GetMerchantBy(ctx context.Context, in *GetMerchantByRequest, out *MerchantGetMerchantResponse) error {
	return h.BillingServiceHandler.GetMerchantBy(ctx, in, out)
}

func (h *billingServiceHandler) ListMerchants(ctx context.Context, in *MerchantListingRequest, out *MerchantListingResponse) error {
	return h.BillingServiceHandler.ListMerchants(ctx, in, out)
}

func (h *billingServiceHandler) ChangeMerchant(ctx context.Context, in *OnboardingRequest, out *billing.Merchant) error {
	return h.BillingServiceHandler.ChangeMerchant(ctx, in, out)
}

func (h *billingServiceHandler) ChangeMerchantStatus(ctx context.Context, in *MerchantChangeStatusRequest, out *billing.Merchant) error {
	return h.BillingServiceHandler.ChangeMerchantStatus(ctx, in, out)
}

func (h *billingServiceHandler) ChangeMerchantData(ctx context.Context, in *ChangeMerchantDataRequest, out *ChangeMerchantDataResponse) error {
	return h.BillingServiceHandler.ChangeMerchantData(ctx, in, out)
}

func (h *billingServiceHandler) SetMerchantS3Agreement(ctx context.Context, in *SetMerchantS3AgreementRequest, out *ChangeMerchantDataResponse) error {
	return h.BillingServiceHandler.SetMerchantS3Agreement(ctx, in, out)
}

func (h *billingServiceHandler) CreateNotification(ctx context.Context, in *NotificationRequest, out *billing.Notification) error {
	return h.BillingServiceHandler.CreateNotification(ctx, in, out)
}

func (h *billingServiceHandler) GetNotification(ctx context.Context, in *GetNotificationRequest, out *billing.Notification) error {
	return h.BillingServiceHandler.GetNotification(ctx, in, out)
}

func (h *billingServiceHandler) ListNotifications(ctx context.Context, in *ListingNotificationRequest, out *Notifications) error {
	return h.BillingServiceHandler.ListNotifications(ctx, in, out)
}

func (h *billingServiceHandler) MarkNotificationAsRead(ctx context.Context, in *GetNotificationRequest, out *billing.Notification) error {
	return h.BillingServiceHandler.MarkNotificationAsRead(ctx, in, out)
}

func (h *billingServiceHandler) ListMerchantPaymentMethods(ctx context.Context, in *ListMerchantPaymentMethodsRequest, out *ListingMerchantPaymentMethod) error {
	return h.BillingServiceHandler.ListMerchantPaymentMethods(ctx, in, out)
}

func (h *billingServiceHandler) GetMerchantPaymentMethod(ctx context.Context, in *GetMerchantPaymentMethodRequest, out *billing.MerchantPaymentMethod) error {
	return h.BillingServiceHandler.GetMerchantPaymentMethod(ctx, in, out)
}

func (h *billingServiceHandler) ChangeMerchantPaymentMethod(ctx context.Context, in *MerchantPaymentMethodRequest, out *MerchantPaymentMethodResponse) error {
	return h.BillingServiceHandler.ChangeMerchantPaymentMethod(ctx, in, out)
}

func (h *billingServiceHandler) CreateRefund(ctx context.Context, in *CreateRefundRequest, out *CreateRefundResponse) error {
	return h.BillingServiceHandler.CreateRefund(ctx, in, out)
}

func (h *billingServiceHandler) ListRefunds(ctx context.Context, in *ListRefundsRequest, out *ListRefundsResponse) error {
	return h.BillingServiceHandler.ListRefunds(ctx, in, out)
}

func (h *billingServiceHandler) GetRefund(ctx context.Context, in *GetRefundRequest, out *CreateRefundResponse) error {
	return h.BillingServiceHandler.GetRefund(ctx, in, out)
}

func (h *billingServiceHandler) ProcessRefundCallback(ctx context.Context, in *CallbackRequest, out *PaymentNotifyResponse) error {
	return h.BillingServiceHandler.ProcessRefundCallback(ctx, in, out)
}

func (h *billingServiceHandler) PaymentFormLanguageChanged(ctx context.Context, in *PaymentFormUserChangeLangRequest, out *PaymentFormDataChangeResponse) error {
	return h.BillingServiceHandler.PaymentFormLanguageChanged(ctx, in, out)
}

func (h *billingServiceHandler) PaymentFormPaymentAccountChanged(ctx context.Context, in *PaymentFormUserChangePaymentAccountRequest, out *PaymentFormDataChangeResponse) error {
	return h.BillingServiceHandler.PaymentFormPaymentAccountChanged(ctx, in, out)
}

func (h *billingServiceHandler) ProcessBillingAddress(ctx context.Context, in *ProcessBillingAddressRequest, out *ProcessBillingAddressResponse) error {
	return h.BillingServiceHandler.ProcessBillingAddress(ctx, in, out)
}

func (h *billingServiceHandler) CreateOrUpdateProduct(ctx context.Context, in *Product, out *Product) error {
	return h.BillingServiceHandler.CreateOrUpdateProduct(ctx, in, out)
}

func (h *billingServiceHandler) ListProducts(ctx context.Context, in *ListProductsRequest, out *ListProductsResponse) error {
	return h.BillingServiceHandler.ListProducts(ctx, in, out)
}

func (h *billingServiceHandler) GetProduct(ctx context.Context, in *RequestProduct, out *Product) error {
	return h.BillingServiceHandler.GetProduct(ctx, in, out)
}

func (h *billingServiceHandler) DeleteProduct(ctx context.Context, in *RequestProduct, out *EmptyResponse) error {
	return h.BillingServiceHandler.DeleteProduct(ctx, in, out)
}

func (h *billingServiceHandler) GetProductsForOrder(ctx context.Context, in *GetProductsForOrderRequest, out *ListProductsResponse) error {
	return h.BillingServiceHandler.GetProductsForOrder(ctx, in, out)
}

func (h *billingServiceHandler) AddSystemFees(ctx context.Context, in *billing.AddSystemFeesRequest, out *EmptyResponse) error {
	return h.BillingServiceHandler.AddSystemFees(ctx, in, out)
}

func (h *billingServiceHandler) GetSystemFeesForPayment(ctx context.Context, in *billing.GetSystemFeesRequest, out *billing.FeeSet) error {
	return h.BillingServiceHandler.GetSystemFeesForPayment(ctx, in, out)
}

func (h *billingServiceHandler) GetActualSystemFeesList(ctx context.Context, in *EmptyRequest, out *billing.SystemFeesList) error {
	return h.BillingServiceHandler.GetActualSystemFeesList(ctx, in, out)
}
