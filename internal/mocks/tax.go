package mocks

import (
	"context"
	"github.com/micro/go-micro/client"
	"github.com/paysuper/paysuper-tax-service/proto"
)

type TaxServiceOkMock struct{}

func NewTaxServiceOkMock() tax_service.TaxService {
	return &TaxServiceOkMock{}
}

func (m *TaxServiceOkMock) GetRate(
	ctx context.Context,
	in *tax_service.GetRateRequest,
	opts ...client.CallOption,
) (*tax_service.GetRateResponse, error) {
	rate := &tax_service.GetRateResponse{
		Rate: &tax_service.TaxRate{
			Id:      0,
			Zip:     "190000",
			Country: "RU",
			State:   "SPE",
			City:    "St.Petersburg",
			Rate:    0.20,
		},
		UserDataPriority: false,
	}

	if in.UserData != nil && in.UserData.Country == "US" {
		rate = &tax_service.GetRateResponse{
			Rate: &tax_service.TaxRate{
				Id:      1,
				Zip:     "98001",
				Country: "US",
				State:   "NY",
				City:    "Washington",
				Rate:    0.15,
			},
			UserDataPriority: true,
		}
	}

	return rate, nil
}

func (m *TaxServiceOkMock) GetRates(
	ctx context.Context,
	in *tax_service.GetRatesRequest,
	opts ...client.CallOption,
) (*tax_service.GetRatesResponse, error) {
	return &tax_service.GetRatesResponse{}, nil
}

func (m *TaxServiceOkMock) CreateOrUpdate(
	ctx context.Context,
	in *tax_service.TaxRate,
	opts ...client.CallOption,
) (*tax_service.TaxRate, error) {
	return &tax_service.TaxRate{}, nil
}

func (m *TaxServiceOkMock) DeleteRateById(
	ctx context.Context,
	in *tax_service.DeleteRateRequest,
	opts ...client.CallOption,
) (*tax_service.DeleteRateResponse, error) {
	return &tax_service.DeleteRateResponse{}, nil
}
