## an aws profile selector

## Installation

`go install github.com/achan-godaddy/aws-profile-selector/cmd/aws-login`

## Local installation

```sh
git checkout https://github.com/achan-godaddy/aws-profile-selector/cmd/aws-login
cd aws-login
go install ./cmd/aws-login
```

## Usage

```
$ aws-login
```

Uses the profiles defined in ~/.aws/credentials

```
[default]

[example-prod]
aws_account_id=123456789012
credential_process = aws-okta-processor authenticate -u USER_ID_GOES_HERE -o godaddy.okta.com -k default -d 7200 --role arn:aws:iam::123456789012:role/THE_ROLE
```

## Use the 1password cli

set the environment variable `export USE_ONEPASS_CLI=true`. This will use the 1password cli to get the password for the user.

```
AWS_OKTA_PASS=op://vault/identifier/password
```
