# Vault SSO with Azure AD

## Usage

```text
Usage:
   [flags]
   [command]

Available Commands:
  help        Help about any command
  version     Print the version.

Flags:
      --client-id string      Application ID in App registrations
      --default-role string   Default role for vault login (default "default")
  -h, --help                  help for this command
      --port int              HTTP port (default 3000)
      --tenant-id string      Directory ID in Azure AD Properties
      --vault-url string      Vault URL (default "http://127.0.0.1:8200")
  -v, --verbose               Enable verbose

Use " [command] --help" for more information about a command.
```

## App Registrations in Azure AD Portal

- Name: Vault
- Application type: Web app / API
- Sign-on URL: https://YOUR_VAULT_AZURE_SSO_ENDPOINT
- Edit the Manifest to change : `"groupMembershipClaims": "All"` and ` "oauth2AllowImplicitFlow": true,`

## Configure Vault

### Enable JWT auth

```bash
$ vault auth enable jwt
```

### Configure JWT auth plugin

First, convert JWKs (https://login.microsoftonline.com/${YOUR_TENANT_ID}/discovery/keys) in RSA keys

Then, configure JWT with Azure AD keys

```bash
$ vault write auth/jwt/config jwt_validation_pubkeys=@azure_keys.pem bound_issuer="https://login.microsoftonline.com/${YOUR_TENANT_ID}/v2.0"
```

### Create an JWT role in vault

```bash
vault write auth/jwt/role/default \
    bound_audiences="${CLIENT_ID}" \
    user_claim="email" \
    groups_claim="groups" \
    max_ttl=10m \
    ttl=10m
```

## Associate AD group on Vault group 

- Create a Vault external group
- Set policies for this group
- Add an alias group with name as Azure Group ID and auth backend as `jwt/`