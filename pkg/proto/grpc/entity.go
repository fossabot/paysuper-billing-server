package grpc

import (
	"errors"
	"fmt"
	"github.com/paysuper/paysuper-billing-server/pkg/proto/billing"
)

var (
	productNoPriceInCurrency           = "no price in currency %s"
	productNoNameInLanguage            = "no name in language %s"
	productNoDescriptionInLanguage     = "no description in language %s"
	productNoLongDescriptionInLanguage = "no long description in language %s"
)

func (m *MerchantPaymentMethodRequest) GetPerTransactionCurrency() string {
	return m.Commission.PerTransaction.Currency
}

func (m *MerchantPaymentMethodRequest) GetPerTransactionFee() float64 {
	return m.Commission.PerTransaction.Fee
}

func (m *MerchantPaymentMethodRequest) HasPerTransactionCurrency() bool {
	return m.Commission.PerTransaction.Currency != ""
}

func (m *MerchantPaymentMethodRequest) HasIntegration() bool {
	return m.Integration.TerminalId != "" && m.Integration.TerminalPassword != "" &&
		m.Integration.TerminalCallbackPassword != ""
}

func (p *Product) IsPricesContainDefaultCurrency() bool {
	_, err := p.GetPriceInCurrency(&billing.PriceGroup{Currency: p.DefaultCurrency})
	return err == nil
}

func (p *Product) GetPriceInCurrency(group *billing.PriceGroup) (float64, error) {
	for _, price := range p.Prices {
		if group.Currency == "virtual" && price.IsVirtualCurrency {
			return price.Amount, nil
		}

		if group.Region != "" && price.Region == group.Region {
			return price.Amount, nil
		}

		if group.Region == "" && price.Region == group.Currency {
			return price.Amount, nil
		}
	}
	return 0, errors.New(fmt.Sprintf(productNoPriceInCurrency, group.Region))
}

func (p *Product) GetLocalizedName(lang string) (string, error) {
	v, ok := p.Name[lang]
	if !ok {
		return "", errors.New(fmt.Sprintf(productNoNameInLanguage, lang))
	}
	return v, nil
}

func (p *Product) GetLocalizedDescription(lang string) (string, error) {
	v, ok := p.Description[lang]
	if !ok {
		return "", errors.New(fmt.Sprintf(productNoDescriptionInLanguage, lang))
	}
	return v, nil
}

func (p *Product) GetLocalizedLongDescription(lang string) (string, error) {
	v, ok := p.LongDescription[lang]
	if !ok {
		return "", errors.New(fmt.Sprintf(productNoLongDescriptionInLanguage, lang))
	}
	return v, nil
}

func (r *ResponseErrorMessage) Error() string {
	return r.Message
}

func (r *ResponseError) Error() string {
	return r.Message.Message
}

func (m *UserProfile) HasPersonChanges(profile *UserProfile) bool {
	return m.Personal != nil && profile.Personal != m.Personal
}

func (m *UserProfile) HasHelpChanges(profile *UserProfile) bool {
	return m.Help != nil && profile.Help != m.Help
}

func (m *UserProfile) HasCompanyChanges(profile *UserProfile) bool {
	return m.Company != nil && profile.Company != m.Company
}

func (m *UserProfile) HasCompanyAnnualIncomeChanges(profile *UserProfile) bool {
	return m.Company.AnnualIncome != nil && profile.Company.AnnualIncome != m.Company.AnnualIncome
}

func (m *UserProfile) HasCompanyNumberOfEmployeesChanges(profile *UserProfile) bool {
	return m.Company.NumberOfEmployees != nil && profile.Company.NumberOfEmployees != m.Company.NumberOfEmployees
}

func (m *UserProfile) HasCompanyMonetizationChanges(profile *UserProfile) bool {
	return m.Company.Monetization != nil && profile.Company.Monetization != m.Company.Monetization
}

func (m *UserProfile) HasCompanyPlatformsChanges(profile *UserProfile) bool {
	return m.Company.Platforms != nil && profile.Company.Platforms != m.Company.Platforms
}

func (m *UserProfile) IsEmailVerified() bool {
	return m.Email.Confirmed
}

func (m *UserProfile) IsPersonalComplete() bool {
	return m.Personal != nil && m.Personal.FirstName != "" && m.Personal.LastName != "" && m.Personal.Position != ""
}

func (m *UserProfile) IsHelpCompleted() bool {
	return m.Help != nil && (m.Help.ProductPromotionAndDevelopment || m.Help.ReleasedGamePromotion || m.Help.InternationalSales || m.Help.Other)
}

func (m *UserProfile) IsCompanyCompleted() bool {
	return m.Company != nil && m.Company.CompanyName != "" && m.Company.Website != "" &&
		m.Company.AnnualIncome != nil && m.Company.NumberOfEmployees != nil && m.Company.KindOfActivity != "" &&
		m.Company.Monetization != nil && m.Company.Monetization.IsComplete() && m.Company.Platforms != nil &&
		m.Company.Platforms.IsComplete()
}

func (m *UserProfile) NeedConfirmEmail() bool {
	return m.IsPersonalComplete() && m.IsHelpCompleted() && m.IsCompanyCompleted() && !m.Email.IsConfirmationEmailSent
}

func (m *UserProfileCompanyMonetization) IsComplete() bool {
	return m.PaidSubscription || m.InGameAdvertising || m.InGamePurchases || m.PremiumAccess || m.Other
}

func (m *UserProfileCompanyPlatforms) IsComplete() bool {
	return m.PcMac || m.GameConsole || m.MobileDevice || m.WebBrowser || m.Other
}

func (m *OnboardingRequest) HasIdentificationFields() bool {
	return m.Id != "" || (m.User != nil && m.User.Id != "")
}

func (p *KeyProduct) GetLocalizedName(lang string) (string, error) {
	v, ok := p.Name[lang]
	if !ok {
		return "", errors.New(fmt.Sprintf(productNoNameInLanguage, lang))
	}
	return v, nil
}

func (p *KeyProduct) GetLocalizedDescription(lang string) (string, error) {
	v, ok := p.Description[lang]
	if !ok {
		return "", errors.New(fmt.Sprintf(productNoDescriptionInLanguage, lang))
	}
	return v, nil
}

func (p *KeyProduct) GetLocalizedLongDescription(lang string) (string, error) {
	v, ok := p.LongDescription[lang]
	if !ok {
		return "", errors.New(fmt.Sprintf(productNoLongDescriptionInLanguage, lang))
	}
	return v, nil
}

func (p *KeyProduct) GetPriceInCurrencyAndPlatform(group *billing.PriceGroup, platformId string) (float64, error) {
	// todo: add support for virtual currencies here?
	// note: virtual currency has IsVirtualCurrency=true && Currency=""
	for _, platform := range p.Platforms {
		if platform.Id == platformId {
			for _, price := range platform.Prices {
				if group.Region != "" && price.Region == group.Region {
					return price.Amount, nil
				}

				if group.Region == "" && price.Region == group.Currency {
					return price.Amount, nil
				}
			}
		}
	}

	return 0, errors.New(fmt.Sprintf(productNoPriceInCurrency, group.Region))
}
