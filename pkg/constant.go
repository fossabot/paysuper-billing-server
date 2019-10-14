package pkg

const (
	ServiceName    = "p1paybilling"
	ServiceVersion = "latest"

	LoggerName = "PAYONE_BILLING_SERVER"

	CardPayPaymentResponseStatusInProgress = "IN_PROGRESS"
	CardPayPaymentResponseStatusPending    = "PENDING"
	CardPayPaymentResponseStatusRefunded   = "REFUNDED"
	CardPayPaymentResponseStatusDeclined   = "DECLINED"
	CardPayPaymentResponseStatusAuthorized = "AUTHORIZED"
	CardPayPaymentResponseStatusCompleted  = "COMPLETED"
	CardPayPaymentResponseStatusCancelled  = "CANCELLED"

	PaymentCreateFieldOrderId         = "order_id"
	PaymentCreateFieldPaymentMethodId = "payment_method_id"
	PaymentCreateFieldEmail           = "email"
	PaymentCreateFieldPan             = "pan"
	PaymentCreateFieldCvv             = "cvv"
	PaymentCreateFieldMonth           = "month"
	PaymentCreateFieldYear            = "year"
	PaymentCreateFieldHolder          = "card_holder"
	PaymentCreateFieldEWallet         = "ewallet"
	PaymentCreateFieldCrypto          = "address"
	PaymentCreateFieldStoreData       = "store_data"
	PaymentCreateFieldRecurringId     = "recurring_id"
	PaymentCreateFieldStoredCardId    = "stored_card_id"
	PaymentCreateFieldUserCountry     = "country"
	PaymentCreateFieldUserCity        = "city"
	PaymentCreateFieldUserState       = "state"
	PaymentCreateFieldUserZip         = "zip"

	TxnParamsFieldBankCardEmissionCountry = "emission_country"
	TxnParamsFieldBankCardToken           = "token"
	TxnParamsFieldBankCardIs3DS           = "is_3ds"
	TxnParamsFieldBankCardRrn             = "rrn"
	TxnParamsFieldDeclineCode             = "decline_code"
	TxnParamsFieldDeclineReason           = "decline_reason"
	TxnParamsFieldCryptoTransactionId     = "transaction_id"
	TxnParamsFieldCryptoAmount            = "amount_crypto"
	TxnParamsFieldCryptoCurrency          = "currency_crypto"

	StatusOK              = int32(0)
	StatusErrorValidation = int32(1)
	StatusErrorSystem     = int32(2)
	StatusTemporary       = int32(4)

	MerchantStatusDraft            = int32(0)
	MerchantStatusAgreementSigning = int32(3)
	MerchantStatusAgreementSigned  = int32(4)
	MerchantStatusDeleted          = int32(5)
	MerchantStatusRejected         = int32(6)

	MerchantMinimalPayoutLimit = float32(1000)

	ResponseStatusOk          = int32(200)
	ResponseStatusNotModified = int32(304)
	ResponseStatusBadData     = int32(400)
	ResponseStatusNotFound    = int32(404)
	ResponseStatusForbidden   = int32(403)
	ResponseStatusSystemError = int32(500)
	ResponseStatusTemporary   = int32(410)

	RefundStatusCreated               = int32(0)
	RefundStatusRejected              = int32(1)
	RefundStatusInProgress            = int32(2)
	RefundStatusCompleted             = int32(3)
	RefundStatusPaymentSystemDeclined = int32(4)
	RefundStatusPaymentSystemCanceled = int32(5)

	PaymentSystemErrorCreateRefundFailed   = "refund can't be create. try request later"
	PaymentSystemErrorCreateRefundRejected = "refund create request rejected"

	PaymentSystemHandlerCardPay = "cardpay"

	MerchantAgreementTypeESign = 2

	SignerTypeMerchant = int32(0)
	SignerTypePs       = int32(1)

	ProjectStatusDraft         = int32(0)
	ProjectStatusTestCompleted = int32(1)
	ProjectStatusTestFailed    = int32(2)
	ProjectStatusInProduction  = int32(3)
	ProjectStatusDeleted       = int32(4)

	ProjectCallbackProtocolEmpty   = "empty"
	ProjectCallbackProtocolDefault = "default"

	ObjectTypeUser = "user"

	UserIdentityTypeEmail    = "email"
	UserIdentityTypePhone    = "phone"
	UserIdentityTypeExternal = "external"

	TechEmailDomain        = "@paysuper.com"
	OrderInlineFormUrlMask = "%s://%s/order/%s"

	MigrationSource = "file://./migrations"

	ErrorGrpcServiceCallFailed       = "gRPC call failed"
	ErrorVatReportDateCantBeInFuture = "vat report date cant be in future"
	MethodFinishedWithError          = "method finished with error"
	LogFieldRequest                  = "request"
	LogFieldResponse                 = "request"
	LogFieldHandler                  = "handler"

	ObjectTypeBalanceTransaction = "balance_transaction"

	AccountingEntryTypeRealGrossRevenue                    = "real_gross_revenue"
	AccountingEntryTypeRealTaxFee                          = "real_tax_fee"
	AccountingEntryTypeCentralBankTaxFee                   = "central_bank_tax_fee"
	AccountingEntryTypeRealTaxFeeTotal                     = "real_tax_fee_total"
	AccountingEntryTypePsGrossRevenueFx                    = "ps_gross_revenue_fx"
	AccountingEntryTypePsGrossRevenueFxTaxFee              = "ps_gross_revenue_fx_tax_fee"
	AccountingEntryTypePsGrossRevenueFxProfit              = "ps_gross_revenue_fx_profit"
	AccountingEntryTypeMerchantGrossRevenue                = "merchant_gross_revenue"
	AccountingEntryTypeMerchantTaxFeeCostValue             = "merchant_tax_fee_cost_value"
	AccountingEntryTypeMerchantTaxFeeCentralBankFx         = "merchant_tax_fee_central_bank_fx"
	AccountingEntryTypeMerchantTaxFee                      = "merchant_tax_fee"
	AccountingEntryTypePsMethodFee                         = "ps_method_fee"
	AccountingEntryTypeMerchantMethodFee                   = "merchant_method_fee"
	AccountingEntryTypeMerchantMethodFeeCostValue          = "merchant_method_fee_cost_value"
	AccountingEntryTypePsMarkupMerchantMethodFee           = "ps_markup_merchant_method_fee"
	AccountingEntryTypeMerchantMethodFixedFee              = "merchant_method_fixed_fee"
	AccountingEntryTypeRealMerchantMethodFixedFee          = "real_merchant_method_fixed_fee"
	AccountingEntryTypeMarkupMerchantMethodFixedFeeFx      = "markup_merchant_method_fixed_fee_fx"
	AccountingEntryTypeRealMerchantMethodFixedFeeCostValue = "real_merchant_method_fixed_fee_cost_value"
	AccountingEntryTypePsMethodFixedFeeProfit              = "ps_method_fixed_fee_profit"
	AccountingEntryTypeMerchantPsFixedFee                  = "merchant_ps_fixed_fee"
	AccountingEntryTypeRealMerchantPsFixedFee              = "real_merchant_ps_fixed_fee"
	AccountingEntryTypeMarkupMerchantPsFixedFee            = "markup_merchant_ps_fixed_fee"
	AccountingEntryTypePsMethodProfit                      = "ps_method_profit"
	AccountingEntryTypeMerchantNetRevenue                  = "merchant_net_revenue"
	AccountingEntryTypePsProfitTotal                       = "ps_profit_total"

	AccountingEntryTypeRealRefund                      = "real_refund"
	AccountingEntryTypeRealRefundTaxFee                = "real_refund_tax_fee"
	AccountingEntryTypeRealRefundFee                   = "real_refund_fee"
	AccountingEntryTypeRealRefundFixedFee              = "real_refund_fixed_fee"
	AccountingEntryTypeMerchantRefund                  = "merchant_refund"
	AccountingEntryTypePsMerchantRefundFx              = "ps_merchant_refund_fx"
	AccountingEntryTypeMerchantRefundFee               = "merchant_refund_fee"
	AccountingEntryTypePsMarkupMerchantRefundFee       = "ps_markup_merchant_refund_fee"
	AccountingEntryTypeMerchantRefundFixedFeeCostValue = "merchant_refund_fixed_fee_cost_value"
	AccountingEntryTypeMerchantRefundFixedFee          = "merchant_refund_fixed_fee"
	AccountingEntryTypePsMerchantRefundFixedFeeFx      = "ps_merchant_refund_fixed_fee_fx"
	AccountingEntryTypePsMerchantRefundFixedFeeProfit  = "ps_merchant_refund_fixed_fee_profit"
	AccountingEntryTypeReverseTaxFee                   = "reverse_tax_fee"
	AccountingEntryTypeReverseTaxFeeDelta              = "reverse_tax_fee_delta"
	AccountingEntryTypePsReverseTaxFeeDelta            = "ps_reverse_tax_fee_delta"
	AccountingEntryTypeMerchantReverseTaxFee           = "merchant_reverse_tax_fee"
	AccountingEntryTypeMerchantReverseRevenue          = "merchant_reverse_revenue"
	AccountingEntryTypePsRefundProfit                  = "ps_refund_profit"
	AccountingEntryTypeMerchantRollingReserveCreate    = "merchant_rolling_reserve_create"
	AccountingEntryTypeMerchantRollingReserveRelease   = "merchant_rolling_reserve_release"
	AccountingEntryTypeMerchantRoyaltyCorrection       = "merchant_royalty_correction"

	BalanceTransactionStatusAvailable = "available"

	ErrorDatabaseQueryFailed          = "Query to database collection failed"
	ErrorDatabaseFieldCollection      = "collection"
	ErrorDatabaseFieldDocumentId      = "document_id"
	ErrorDatabaseFieldQuery           = "query"
	ErrorDatabaseFieldSet             = "set"
	ErrorDatabaseFieldSorts           = "sorts"
	ErrorDatabaseFieldLimit           = "limit"
	ErrorDatabaseFieldOffset          = "offset"
	ErrorDatabaseFieldOperation       = "operation"
	ErrorDatabaseFieldOperationInsert = "insert"
	ErrorDatabaseFieldOperationUpdate = "update"
	ErrorDatabaseFieldOperationUpsert = "upsert"
	ErrorDatabaseFieldDocument        = "document"

	ErrorJsonMarshallingFailed = "json marshalling failed"

	ErrorCacheQueryFailed = "Query to cache storage failed"
	ErrorCacheFieldKey    = "key"
	ErrorCacheFieldData   = "data"
	ErrorCacheFieldCmd    = "command"

	ErrorRoyaltyReportGenerationFailed = "royalty report generation failed"
	ErrorRoyaltyReportFieldMerchantId  = "merchant_id"
	ErrorRoyaltyReportFieldFrom        = "from"
	ErrorRoyaltyReportFieldTo          = "to"

	CardPayDeclineCodeSystemMalfunction                = "01"
	CardPayDeclineCodeCancelledByCustomer              = "02"
	CardPayDeclineCodeDeclinedByAntiFraud              = "03"
	CardPayDeclineCodeDeclinedBy3DSecure               = "04"
	CardPayDeclineCodeOnly3DSecureTransactionsAllowed  = "05"
	CardPayDeclineCode3DSecureAvailabilityIsUnknown    = "06"
	CardPayDeclineCodeLimitReached                     = "07"
	CardPayDeclineCodeRequestedOperationIsNotSupported = "08"
	CardPayDeclineCodeDeclinedByBankWithoutReason      = "10"
	CardPayDeclineCodeCommonDeclineByBank              = "11"
	CardPayDeclineCodeInsufficientFunds                = "13"
	CardPayDeclineCodeCardLimitReached                 = "14"
	CardPayDeclineCodeIncorrectCardData                = "15"
	CardPayDeclineCodeDeclinedByBankAntiFraud          = "16"
	CardPayDeclineCodeBanksMalfunction                 = "17"
	CardPayDeclineCodeConnectionProblem                = "18"
	CardPayDeclineCodeNoPaymentWasReceived             = "21"
	CardPayDeclineCodeWrongPaymentWasReceived          = "22"
	CardPayDeclineCodeConfirmationsPaymentTimeout      = "23"

	PaySuperDeclineCodeSystemMalfunction                = "ps000001"
	PaySuperDeclineCodeCancelledByCustomer              = "ps000002"
	PaySuperDeclineCodeDeclinedByAntiFraud              = "ps000003"
	PaySuperDeclineCodeDeclinedBy3DSecure               = "ps000004"
	PaySuperDeclineCodeOnly3DSecureTransactionsAllowed  = "ps000005"
	PaySuperDeclineCode3DSecureAvailabilityIsUnknown    = "ps000006"
	PaySuperDeclineCodeLimitReached                     = "ps000007"
	PaySuperDeclineCodeRequestedOperationIsNotSupported = "ps000008"
	PaySuperDeclineCodeDeclinedByBankWithoutReason      = "ps000009"
	PaySuperDeclineCodeCommonDeclineByBank              = "ps000010"
	PaySuperDeclineCodeInsufficientFunds                = "ps000011"
	PaySuperDeclineCodeCardLimitReached                 = "ps000012"
	PaySuperDeclineCodeIncorrectCardData                = "ps000013"
	PaySuperDeclineCodeDeclinedByBankAntiFraud          = "ps000014"
	PaySuperDeclineCodeBanksMalfunction                 = "ps000015"
	PaySuperDeclineCodeConnectionProblem                = "ps000016"
	PaySuperDeclineCodeNoPaymentWasReceived             = "ps000017"
	PaySuperDeclineCodeWrongPaymentWasReceived          = "ps000018"
	PaySuperDeclineCodeConfirmationsPaymentTimeout      = "ps000019"

	OrderTypeOrder  = "order"
	OrderTypeRefund = "refund"

	PaymentCreateBankCardFieldBrand         = "card_brand"
	PaymentCreateBankCardFieldType          = "card_type"
	PaymentCreateBankCardFieldCategory      = "card_category"
	PaymentCreateBankCardFieldIssuerName    = "bank_issuer_name"
	PaymentCreateBankCardFieldIssuerCountry = "bank_issuer_country"

	RoyaltyReportStatusPending        = "pending"
	RoyaltyReportStatusAccepted       = "accepted"
	RoyaltyReportStatusCanceled       = "canceled"
	RoyaltyReportStatusDispute        = "dispute"
	RoyaltyReportStatusWaitForPayment = "waiting_payment"
	RoyaltyReportStatusPaid           = "paid"

	EmailRoyaltyReportMessage = "New royalty report wait for merchant owner approve"

	RoyaltyReportChangeSourceAuto      = "auto"
	RoyaltyReportChangeSourceMerchant  = "merchant"
	RoyaltyReportChangeSourceHellosign = "hellosign"
	RoyaltyReportChangeSourceAdmin     = "admin"

	VatCurrencyRatesPolicyOnDay    = "on-day"
	VatCurrencyRatesPolicyLastDay  = "last-day"
	VatCurrencyRatesPolicyAvgMonth = "avg-month"

	VatReportStatusThreshold = "threshold"
	VatReportStatusExpired   = "expired"
	VatReportStatusPending   = "pending"
	VatReportStatusNeedToPay = "need_to_pay"
	VatReportStatusPaid      = "paid"
	VatReportStatusOverdue   = "overdue"
	VatReportStatusCanceled  = "canceled"

	UndoReasonReversal   = "reversal"
	UndoReasonChargeback = "chargeback"

	DashboardPeriodCurrentDay      = "current_day"
	DashboardPeriodPreviousDay     = "previous_day"
	DashboardPeriodTwoDaysAgo      = "two_days_ago"
	DashboardPeriodCurrentWeek     = "current_week"
	DashboardPeriodPreviousWeek    = "previous_week"
	DashboardPeriodTwoWeeksAgo     = "two_weeks_ago"
	DashboardPeriodCurrentMonth    = "current_month"
	DashboardPeriodPreviousMonth   = "previous_month"
	DashboardPeriodTwoMonthsAgo    = "two_months_ago"
	DashboardPeriodCurrentQuarter  = "current_quarter"
	DashboardPeriodPreviousQuarter = "previous_quarter"
	DashboardPeriodTwoQuarterAgo   = "two_quarter_ago"
	DashboardPeriodCurrentYear     = "current_year"
	DashboardPeriodPreviousYear    = "previous_year"
	DashboardPeriodTwoYearsAgo     = "two_years_ago"

	PayoutDocumentStatusSkip       = "skip"
	PayoutDocumentStatusPending    = "pending"
	PayoutDocumentStatusInProgress = "in_progress"
	PayoutDocumentStatusPaid       = "paid"
	PayoutDocumentStatusCanceled   = "canceled"
	PayoutDocumentStatusFailed     = "failed"

	ProjectSellCountTypeFractional = "fractional"
	ProjectSellCountTypeIntegral   = "integral"

	FallbackCurrency = "USD"
)

var (
	CountryPhoneCodes = map[int32]string{
		7:    "RU",
		375:  "BY",
		994:  "AZ",
		91:   "IN",
		77:   "KZ",
		380:  "UA",
		44:   "GB",
		9955: "GE",
		370:  "LT",
		992:  "TJ",
		66:   "TH",
		998:  "UZ",
		507:  "PA",
		374:  "AM",
		371:  "LV",
		90:   "TR",
		373:  "MD",
		972:  "IL",
		84:   "VN",
		372:  "EE",
		82:   "KR",
		996:  "KG",
	}

	DeclineCodeMap = map[string]string{
		CardPayDeclineCodeSystemMalfunction:                PaySuperDeclineCodeSystemMalfunction,
		CardPayDeclineCodeCancelledByCustomer:              PaySuperDeclineCodeCancelledByCustomer,
		CardPayDeclineCodeDeclinedByAntiFraud:              PaySuperDeclineCodeDeclinedByAntiFraud,
		CardPayDeclineCodeDeclinedBy3DSecure:               PaySuperDeclineCodeDeclinedBy3DSecure,
		CardPayDeclineCodeOnly3DSecureTransactionsAllowed:  PaySuperDeclineCodeOnly3DSecureTransactionsAllowed,
		CardPayDeclineCode3DSecureAvailabilityIsUnknown:    PaySuperDeclineCode3DSecureAvailabilityIsUnknown,
		CardPayDeclineCodeLimitReached:                     PaySuperDeclineCodeLimitReached,
		CardPayDeclineCodeRequestedOperationIsNotSupported: PaySuperDeclineCodeRequestedOperationIsNotSupported,
		CardPayDeclineCodeDeclinedByBankWithoutReason:      PaySuperDeclineCodeDeclinedByBankWithoutReason,
		CardPayDeclineCodeCommonDeclineByBank:              PaySuperDeclineCodeCommonDeclineByBank,
		CardPayDeclineCodeInsufficientFunds:                PaySuperDeclineCodeInsufficientFunds,
		CardPayDeclineCodeCardLimitReached:                 PaySuperDeclineCodeCardLimitReached,
		CardPayDeclineCodeIncorrectCardData:                PaySuperDeclineCodeIncorrectCardData,
		CardPayDeclineCodeDeclinedByBankAntiFraud:          PaySuperDeclineCodeDeclinedByBankAntiFraud,
		CardPayDeclineCodeBanksMalfunction:                 PaySuperDeclineCodeBanksMalfunction,
		CardPayDeclineCodeConnectionProblem:                PaySuperDeclineCodeConnectionProblem,
		CardPayDeclineCodeNoPaymentWasReceived:             PaySuperDeclineCodeNoPaymentWasReceived,
		CardPayDeclineCodeWrongPaymentWasReceived:          PaySuperDeclineCodeWrongPaymentWasReceived,
		CardPayDeclineCodeConfirmationsPaymentTimeout:      PaySuperDeclineCodeConfirmationsPaymentTimeout,
	}
)
