PaySuper Billing Server
=====
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-brightgreen.svg)](https://www.gnu.org/licenses/gpl-3.0)
[![Build Status](https://travis-ci.org/paysuper/paysuper-billing-server.svg?branch=master)](https://travis-ci.org/paysuper/paysuper-billing-server) [![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fpaysuper%2Fpaysuper-billing-server.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Fpaysuper%2Fpaysuper-billing-server?ref=badge_shield)

[![codecov](https://codecov.io/gh/paysuper/paysuper-billing-server/branch/master/graph/badge.svg)](https://codecov.io/gh/paysuper/paysuper-billing-server)
[![Go Report Card](https://goreportcard.com/badge/github.com/paysuper/paysuper-billing-server)](https://goreportcard.com/report/github.com/paysuper/paysuper-billing-server) 

PaySuper is a unique, simple payment toolkit designed to make developers self-reliant. It’s an open-source payment service 
with a highly customizable payment form, an intuitive API, and comprehensible, eye-catching reports.

Billing Server is a core and heart of all PaySuper for payment processing business logic. It designed to be mediator 
micro-service for [Management API](https://github.com/paysuper/paysuper-management-api) covering all REST API based 
interfaces for [dashboard](https://github.com/paysuper/paysuper-management-server) and [payment form](https://github.com/paysuper/paysuper-payment-form).

## Features

* A nice looking dashboard with graph and key metrics.
* Support for payment tokens and next payment without asking payment information.
* VAT/Sales tax calculating. 
* Payment Link feature.
* Detailed statistics for each transaction.
* Transparent payout calculation.  
* Reports like `Live Transaction`, `Country Distribution`, `Revenue By Client`, `Items Sales` and more to come.
* Export all reports to PDF, XLSX or CSV format.
* Customizable payment form.
* Recurring invoices (work in progress)  

# Getting Started 

Billing Server is designed to be cloud agnostic application. In general we use Kubernetes based installation to launch 
PaySuper in a dev environment cloud and AWS for production. 

## Docker Deployment

```bash
docker build -f Dockerfile -t paysuper_billing_service .
docker run -d -e "MONGO_HOST=127.0.0.1:27017" -e "MONGO_DB="paysuper" ... e="CACHE_PROJECT_PAYMENT_METHOD_TIMEOUT=600" paysuper_billing_service
```

## Starting the app

Billing Server application can be started in 2 modes:

* as microservice, to maintain rates requests from other components of system. This mode does not requests any rates
* as console app, to run the tasks, that passed as command line argument

Console mode can be used with cron schedule.

To start app in console mode you must set `-task` flag in command line to one of these values:

- `vat_reports` - to update vat reports data. This task must be run every day, at the end of day.
- `royalty_reports` - to build royalty reports for merchants. This task must be run once on a week.
- `royalty_reports_accept` - to auto-accept toyalty reports. This task must be run daily.

Notice: for `vat-reports` task you may pass an report date (from past only!) for that you need get an report. 
Date passed as `date` parameter, in YYYY-MM-DD format 

Example: `$ paysuper-billing-server.exe -task=vat_reports -date="2018-12-31"` runs VAT reports calculation for 
last day of December, 2018.

To run application as microservice simply don't pass any flags to command line :)  

## Environment variables:

| Name                                 | Description                                                                                                                         |
|:-------------------------------------|:------------------------------------------------------------------------------------------------------------------------------------|
| MONGO_DSN                            | MongoBD DSN connection string                                                                                                       |
| MONGO_DIAL_TIMEOUT                   | MongoBD dial timeout in seconds                                                                                                     |
| PSP_ACCOUNTING_CURRENCY              | PaySuper accounting currency                                                                                                        |
| METRICS_PORT                         | Http server port for health and metrics request                                                                                     |
| CENTRIFUGO_SECRET                    | Centrifugo secret key                                                                                                               |
| CENTRIFUGO_API_SECRET                | Centrifugo API secret key                                                                                                           |
| BROKER_ADDRESS                       | RabbitMQ url address                                                                                                                |
| CARD_PAY_API_URL                     | CardPay API url to process payments, more in [documentation](https://integration.cardpay.com/v3/)                                   | 
| CACHE_REDIS_ADDRESS                  | A seed list of host:port addresses of cluster nodes                                                                                 |
| CACHE_REDIS_PASSWORD                 | Password for connection string                                                                                                      |
| CACHE_REDIS_POOL_SIZE                | PoolSize applies per cluster node and not for the whole cluster                                                                     |
| CACHE_REDIS_MAX_RETRIES              | Maximum retries for connection                                                                                                      |
| CACHE_REDIS_MAX_REDIRECTS            | The maximum number of retries before giving up                                                                                      |
| CUSTOMER_COOKIE_PUBLIC_KEY           | Base64 encoded RSA public key - used for encrypt customer browser cookies content. Minimal length of RSA public key must be 4096    |
| CUSTOMER_COOKIE_PRIVATE_KEY          | Base64 encoded RSA private key - used for decrypt customer browser cookies content. Minimal length of RSA private key must be 4096  |
| REDIS_HOST                           | Redis server host                                                                                                                   |
| REDIS_PASSWORD                       | Password to access to Redis server                                                                                                  |
| CENTRIFUGO_MERCHANT_CHANNEL          | Centrifugo channel name to send notifications to merchant                                                                           |
| CENTRIFUGO_FINANCIER_CHANNEL         | Centrifugo channel name to send notifications to financier                                                                          |
| EMAIL_NOTIFICATION_FINANCIER_RECIPIENT|Email of financier, to get vat reports notification                                                                                 |
| EMAIL_CONFIRM_URL                    | Url to use in template of confirmation email                                                                                        |
| EMAIL_CONFIRM_TEMPLATE               | Confirmation email template name                                                                                                    |
| EMAIL_NEW_ROYALTY_REPORT_TEMPLATE    | New royalty report notification email template name                                                                                 |
| EMAIL_UPDATE_ROYALTY_REPORT_TEMPLATE | Royalty report update notification email template name                                                                              |
| EMAIL_VAT_REPORT_TEMPLATE            | New vat report notification email template name                                                                                     |
| EMAIL_NEW_PAYOUT_TEMPLATE            | New payout notification email template name                                                                                         |
| PAYSUPER_DOCUMENT_SIGNER_EMAIL       | Paysuper signer email for sign license agreements                                                                                   |
| PAYSUPER_DOCUMENT_SIGNER_NAME        | Paysuper signer name for sign license agreements                                                                                    |
| HELLO_SIGN_DEFAULT_TEMPLATE          | License agreement template identifier in HelloSign                                                                                  |
| HELLO_SIGN_AGREEMENT_CLIENT_ID       | Client application identifier in HelloSign for Merchant Agreement sign                                                              |
| HELLO_SIGN_PAYOUTS_CLIENT_ID         | Client application identifier in HelloSign for Payout Invoice sign                                                                  |
| KEY_DAEMON_RESTART_INTERVAL          | Starting frequency in seconds of the script to check the locked keys and return them to the stack.                                  |
| EMAIL_ACTIVATION_CODE_TEMPLATE       | Postmark Email template id for sending to user with activation code                                                                 |
| HELLO_SIGN_PAYOUTS_TITLE             | Hellosign title for payout invoices                                                                                                 |
| HELLO_SIGN_PAYOUTS_SUBJECT           | Hellosign subject for payout invoices                                                                                               |
| HELLO_SIGN_PAYOUTS_MESSAGE           | Hellosign prompt message for payout invoices                                                                                        |
| PAYLINK_MIN_PRODUCTS                 | Minimum number of products allowed for one payment link (must be >= 1)                                                              |
| PAYLINK_MAX_PRODUCTS                 | Maximum number of products allowed for one payment link                                                                             |
| ROYALTY_REPORTS_URL                  | Url to use in template of royalty reports                                                                                           |
| PAYOUTS_URL                          | Url to use in template of payouts                                                                                                   |


## Architecture

WIP. Draft described in docs folder.

## Contributing
We feel that a welcoming community is important and we ask that you follow PaySuper's [Open Source Code of Conduct](https://github.com/paysuper/code-of-conduct/blob/master/README.md) in all interactions with the community.

PaySuper welcomes contributions from anyone and everyone. Please refer to each project's style and contribution guidelines for submitting patches and additions. In general, we follow the "fork-and-pull" Git workflow.

The master branch of this repository contains the latest stable release of this component.

 


## License
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fpaysuper%2Fpaysuper-billing-server.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Fpaysuper%2Fpaysuper-billing-server?ref=badge_large)