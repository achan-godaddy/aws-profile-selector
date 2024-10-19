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

remember in the `aws-okta-processor` command line you can also add the `--application` url to reduce having to choose an application if there are multiple.


