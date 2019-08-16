// Code generated by mockery v1.0.0. DO NOT EDIT.

package mock

import billing "github.com/paysuper/paysuper-billing-server/pkg/proto/billing"
import grpc "github.com/paysuper/paysuper-billing-server/pkg/proto/grpc"
import mock "github.com/stretchr/testify/mock"

// MerchantTariffRatesInterface is an autogenerated mock type for the MerchantTariffRatesInterface type
type MerchantTariffRatesInterface struct {
	mock.Mock
}

// GetBy provides a mock function with given fields: _a0
func (_m *MerchantTariffRatesInterface) GetBy(_a0 *grpc.GetMerchantTariffRatesRequest) (*billing.MerchantTariffRates, error) {
	ret := _m.Called(_a0)

	var r0 *billing.MerchantTariffRates
	if rf, ok := ret.Get(0).(func(*grpc.GetMerchantTariffRatesRequest) *billing.MerchantTariffRates); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*billing.MerchantTariffRates)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*grpc.GetMerchantTariffRatesRequest) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetCacheKeyForGetBy provides a mock function with given fields: _a0
func (_m *MerchantTariffRatesInterface) GetCacheKeyForGetBy(_a0 *grpc.GetMerchantTariffRatesRequest) (string, error) {
	ret := _m.Called(_a0)

	var r0 string
	if rf, ok := ret.Get(0).(func(*grpc.GetMerchantTariffRatesRequest) string); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*grpc.GetMerchantTariffRatesRequest) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
