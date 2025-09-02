# Full authentication flow test
GET http://localhost:8080/health
HTTP/1.1 200

POST http://localhost:8080/v1/accounts/register
Content-Type: application/json
[Options]
variable: test_email="test-{{newUuid}}@example.com"

{
  "email": "{{test_email}}",
  "password": "TestPass123!"
}

HTTP/1.1 201
[Asserts]
jsonpath "$.message" == "Account created successfully"
jsonpath "$.account_id" exists

[Captures]
account_id: jsonpath "$.account_id"

# Login with the created account  
POST http://localhost:8080/v1/accounts/login
Content-Type: application/json

{
  "email": "{{test_email}}",
  "password": "TestPass123!"
}

HTTP/1.1 200
[Asserts]
jsonpath "$.message" == "Login successful"
jsonpath "$.account_id" == "{{account_id}}"
jsonpath "$.access_token" exists
jsonpath "$.refresh_token" exists
jsonpath "$.token_type" == "Bearer"
jsonpath "$.expires_in" > 0

[Captures]
access_token: jsonpath "$.access_token"
refresh_token: jsonpath "$.refresh_token"

# Refresh the tokens
POST http://localhost:8080/v1/accounts/refresh
Content-Type: application/json

{
  "refresh_token": "{{refresh_token}}"
}

HTTP/1.1 200
[Asserts]
jsonpath "$.message" == "Login successful"
jsonpath "$.account_id" == "{{account_id}}"
jsonpath "$.access_token" exists
jsonpath "$.refresh_token" exists
jsonpath "$.token_type" == "Bearer"
jsonpath "$.expires_in" > 0

[Captures]
new_refresh_token: jsonpath "$.refresh_token"

# Logout using the new refresh token
POST http://localhost:8080/v1/accounts/logout
Content-Type: application/json

{
  "refresh_token": "{{new_refresh_token}}"
}

HTTP/1.1 200

# Try to refresh with the revoked token (should fail)
POST http://localhost:8080/v1/accounts/refresh
Content-Type: application/json

{
  "refresh_token": "{{new_refresh_token}}"
}

HTTP/1.1 401