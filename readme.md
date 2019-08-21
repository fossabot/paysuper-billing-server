Billing service
=====
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-brightgreen.svg)](https://www.gnu.org/licenses/gpl-3.0)
[![Build Status](https://travis-ci.org/paysuper/paysuper-billing-server.svg?branch=master)](https://travis-ci.org/paysuper/paysuper-billing-server) 
[![codecov](https://codecov.io/gh/paysuper/paysuper-billing-server/branch/master/graph/badge.svg)](https://codecov.io/gh/paysuper/paysuper-billing-server)
[![Go Report Card](https://goreportcard.com/badge/github.com/paysuper/paysuper-billing-server)](https://goreportcard.com/report/github.com/paysuper/paysuper-billing-server) [![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=paysuper_paysuper-billing-server&metric=alert_status)](https://sonarcloud.io/dashboard?id=paysuper_paysuper-billing-server)

This service contain all business logic for payment processing

## Environment variables:

| Name                                 | Required | Default                                             | Description                                                                                                                         |
|:-------------------------------------|:--------:|:----------------------------------------------------|:------------------------------------------------------------------------------------------------------------------------------------|
| MONGO_DSN                            | true     | -                                                   | MongoBD DSN connection string                                                                                                       |
| MONGO_DIAL_TIMEOUT                   | -        | 10                                                  | MongoBD dial timeout in seconds                                                                                                     |
| PSP_ACCOUNTING_CURRENCY              | -        | EUR                                                 | PaySuper accounting currency                                                                                                        |
| METRICS_PORT                         | -        | 8086                                                | Http server port for health and metrics request                                                                                     |
| CENTRIFUGO_SECRET                    | true     | -                                                   | Centrifugo secret key                                                                                                               |
| CENTRIFUGO_API_SECRET                | true     | -                                                   | Centrifugo API secret key                                                                                                           |
| BROKER_ADDRESS                       | -        | amqp://127.0.0.1:5672                               | RabbitMQ url address                                                                                                                |
| CARD_PAY_API_URL                     | true     | -                                                   | CardPay API url to process payments, more in [documentation](https://integration.cardpay.com/v3/)                                   | 
| CACHE_REDIS_ADDRESS                  | true     |                                                     | A seed list of host:port addresses of cluster nodes                                                                                 |
| CACHE_REDIS_PASSWORD                 | -        |                                                     | Password for connection string                                                                                                      |
| CACHE_REDIS_POOL_SIZE                | -        | 1                                                   | PoolSize applies per cluster node and not for the whole cluster                                                                     |
| CACHE_REDIS_MAX_RETRIES              | -        | 10                                                  | Maximum retries for connection                                                                                                      |
| CACHE_REDIS_MAX_REDIRECTS            | -        | 8                                                   | The maximum number of retries before giving up                                                                                      |
| CUSTOMER_COOKIE_PUBLIC_KEY           | true     | -                                                   | Base64 encoded RSA public key - used for encrypt customer browser cookies content. Minimal length of RSA public key must be 4096    |
| CUSTOMER_COOKIE_PRIVATE_KEY          | true     | -                                                   | Base64 encoded RSA private key - used for decrypt customer browser cookies content. Minimal length of RSA private key must be 4096  |
| REDIS_HOST                           | -        | 127.0.0.1:6379                                      | Redis server host                                                                                                                   |
| REDIS_PASSWORD                       | -        | ""                                                  | Password to access to Redis server                                                                                                  |
| CENTRIFUGO_MERCHANT_CHANNEL          | -        | paysuper:merchant#%s                                | Centrifugo channel name to send notifications to merchant                                                                           |
| CENTRIFUGO_FINANCIER_CHANNEL         | -        | paysuper:financier                                  | Centrifugo channel name to send notifications to financier                                                                          |
| EMAIL_NOTIFICATION_FINANCIER_RECIPIENT| true    |                                                     | Email of financier, to get vat reports notification                                                                                 |
| EMAIL_CONFIRM_URL                    | -        | https://paysupermgmt.tst.protocol.one/confirm_email | Url to use in template of confirmation email                                                                                        |
| EMAIL_CONFIRM_TEMPLATE               | -        | sidmal_test_email_confirm                           | Confirmation email template name                                                                                                    |
| EMAIL_NEW_ROYALTY_REPORT_TEMPLATE    | -        | p1_new_royalty_report                               | New royalty report notification email template name                                                                                 |
| EMAIL_VAT_REPORT_TEMPLATE            | -        | p1_vat_report                                       | New vat report notification email template name                                                                                     |
| EMAIL_NEW_ROYALTY_REPORT_TEMPLATE    | -        | p1_new_royalty_report                               | New royalty report notification email template name                                                                                 |
| PAYSUPER_DOCUMENT_SIGNER_EMAIL       | true     | -                                                   | Paysuper signer email for sign license agreements                                                                                   |
| PAYSUPER_DOCUMENT_SIGNER_NAME        | true     | -                                                   | Paysuper signer name for sign license agreements                                                                                    |
| HELLO_SIGN_DEFAULT_TEMPLATE          | true     | -                                                   | License agreement template identifier in HelloSign                                                                                  |
| HELLO_SIGN_CLIENT_ID                 | true     | -                                                   | Client application identifier in HelloSign                                                                                          |

## Docker Deployment

```bash
docker build -f Dockerfile -t paysuper_billing_service .
docker run -d -e "MONGO_HOST=127.0.0.1:27017" -e "MONGO_DB="paysuper" ... e="CACHE_PROJECT_PAYMENT_METHOD_TIMEOUT=600" paysuper_billing_service
```

## Starting the app

This application can be started in 2 modes:

* as microservice, to maintain rates requests from other components of system. This mode does not requests any rates
* as console app, to run the tasks, that passed as command line argument

Console mode can be used with cron schedule.

To start app in console mode you must set `-task` flag in command line to one of these values:

- `vat_reports` - to update vat reports data. This task must be run every day, at the end of day.
- `royalty_reports` - to build royalty reports for merchants. This task must be run once on a week.
- `royalty_reports_accept` - to auto-accept toyalty reports. This task must be run daily.

Notice: for `vat-reports` task you may pass an report date (from past only!) for that you need get an report. 
Date passed as `date` parameter, in YYYY-MM-DD format 

Example: `$ paysuper-billing-server.exe -task=vat_reports -date="2018-12-31"` runs VAT reports calculation for last day of December, 2018.

To run application as microservice simply don't pass any flags to command line :)  

## Architecture

Described in docs folder.

## Contributing
We feel that a welcoming community is important and we ask that you follow PaySuper's [Open Source Code of Conduct](https://github.com/paysuper/code-of-conduct/blob/master/README.md) in all interactions with the community.

PaySuper welcomes contributions from anyone and everyone. Please refer to each project's style and contribution guidelines for submitting patches and additions. In general, we follow the "fork-and-pull" Git workflow.

The master branch of this repository contains the latest stable release of this component.

