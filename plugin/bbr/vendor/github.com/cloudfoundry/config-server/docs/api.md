# API Docs
This document describes the APIs exposed by the **Config Server**.

- GET
  - [Get by ID](#11-get-by-id) 
  - [Get by Name](#12-get-by-name)
- PUT
  - [Set Name Value](#21-set-name-value)
- POST:   
  - [Generate Password](#31-generate-password)
  - [Generate Certificate](#32-generate-certificate)
  - [Generate SSH Keys](#33-generate-ssh-key)
  - [Generate RSA keys](#34-generate-rsa-key)
- DELETE  
  - [Delete Name](#41-delete-name)

## 1. GET

### 1.1 Get By ID
```
GET /v1/data/:id
```

#### Response Schema
`Content-Type: application/json`

``` JSON 
{
  "$schema": "http://json-schema.org/draft-04/schema#",
  "title": "Get by ID response",
  "description": "Get by ID response",
  "type": "object",
  "properties": {
    "id": {
      "description": "Unique identifier",
      "type": "string",
    },
    "name": {
      "description": "Name for the value",
      "type": "string",
    },
    "value": {
      "description": "Value stored against ID",
      "anyOf": [
        {"type": "string"},
        {"type": "number"},
        {"type": "object"},
        {"type": "array"},
        {"type": "boolean"},
        {"type": "null"}
      ]
    }
  }
}
```

#### Response Codes
| Code   | Description |
| ------ | ----------- |
| 200 | Status OK |
| 400 | Bad Request |
| 401 | Not Authorized |
| 404 | Not Found |
| 500 | Server Error |

#### Sample Request/Response

Request URL:
```
 GET /v1/data/some_id
``` 

Response Body:
``` JSON
{
  "id": "some_id",
  "name": "color",
  "value": "blue"
}
```

### 1.2 Get By Name

`GET /v1/data?name=":key_name"`

#### Response Schema
`Content-Type: application/json`

``` JSON 
{
  "$schema": "http://json-schema.org/draft-04/schema#",
  "title": "Get by Name response",
  "description": "Get by Name response",
  "type": "object",
  "properties": {
    "data": {
      "description": "Array of values stored for the name",
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "id": {
            "description": "Unique identifier",
            "type": "string",
          },
          "name": {
            "description": "Name for the value",
            "type": "string",
          },
          "value": {
            "description": "Value stored against ID",
            "anyOf": [
              {"type": "string"},
              {"type": "number"},
              {"type": "object"},
              {"type": "array"},
              {"type": "boolean"},
              {"type": "null"}
            ]
          }
        }
      }
    }
  }
}
```

#### Response Codes
| Code   | Description |
| ------ | ----------- |
| 200 | Status OK |
| 400 | Bad Request |
| 401 | Not Authorized |
| 404 | Not found |
| 500 | Server Error |


#### Sample Request/Response

Request URL: 
```
GET /v1/data?name="/server/tomcat/port"
```

Response Body:

``` JSON
{
  "data": [
    {
      "id": "10",
      "name": "server/tomcat/port",
      "value": 8080
    },
    {
      "id": "11",
      "name": "server/tomcat/port",
      "value": 9090
    }
  ]
}
```

## 2. PUT

### 2.1 Set Name Value

```
PUT /v1/data
```

##### Request Schema
`Content-Type: application/json`

``` JSON 
{
  "$schema": "http://json-schema.org/draft-04/schema#",
  "title": "Set name and value",
  "description": "Set name and value",
  "type": "object",
  "properties": {
    "name": {
      "description": "Name to use for value",
      "type": "string",
    },
    "value": {
      "description": "The value to store against name",
      "anyOf": [
        {"type": "string"},
        {"type": "number"},
        {"type": "object"},
        {"type": "array"},
        {"type": "boolean"},
        {"type": "null"}
      ]
    }
  }
}
```

##### Response Schema
`Content-Type: application/json`

``` JSON 
{
  "$schema": "http://json-schema.org/draft-04/schema#",
  "title": "Response for setting name and value",
  "description": "Response for setting name and value",
  "type": "object",
  "properties": {
    "id": {
      "description": "Unique identifier",
      "type": "string",
    },
    "name": {
      "description": "Name used for value",
      "type": "string",
    },
    "value": {
      "description": "Value stored against ID",
      "anyOf": [
        {"type": "string"},
        {"type": "number"},
        {"type": "object"},
        {"type": "array"},
        {"type": "boolean"},
        {"type": "null"}
      ]
    }
  }
}
```

##### Response Codes
| Code | Description |
| ---- | ----------- |
| 200 | Call successful - name value was added |
| 400 | Bad Request |
| 401 | Not Authorized |
| 415 | Unsupported Media Type |
| 500 | Server Error |


##### Sample Request/Response

Request URL:
```
PUT /v1/data
```

Request Body:
``` JSON
{
  "name": "some_name",
  "value": "happy value"
}
```

Response Body:
``` JSON
{
  "id": "123",
  "name": "some_name",
  "value": "happy value"
}
```


## 3. POST

### 3.1 Generate Password

```
POST /v1/data
```

##### Request Schema
`Content-Type: application/json`

``` JSON 
{
  "$schema": "http://json-schema.org/draft-04/schema#",
  "title": "Request to generate password",
  "description": "Request to generate password",
  "type": "object",
  "properties": {
    "name": {
      "description": "Name to use for generated password",
      "type": "string",
    },
    "type": {
      "type": "string",
      "enum": ["password"]
    }
  }
}
```

##### Response Schema
`Content-Type: application/json`

``` JSON 
{
  "$schema": "http://json-schema.org/draft-04/schema#",
  "title": "Response for generate password",
  "description": "Response for generate password",
  "type": "object",
  "properties": {
    "id": {
      "description": "Unique identifier",
      "type": "string",
    },
    "name": {
      "description": "Password name",
      "type": "string",
    },
    "value": {
      "description": "Generated password value",
      "type": "string"
    }
  }
}
```

##### Response Codes
| Code | Description |
| ---- | ----------- |
| 201 | Call successful |
| 400 | Bad Request |
| 401 | Not Authorized |
| 415 | Unsupported Media Type |
| 500 | Server Error |


##### Sample Request/Response

Request URL:
```
POST /v1/data
```

Request Body:
``` JSON
{
  "name": "mypasswd",
  "type": "password"
}
```

Response Body:
``` JSON
{
  "id": "10",
  "name": "mypasswd",
  "value": "AXZ123TYghK"
}
```

### 3.2 Generate Certificate

The Generate Certificate API can be used for generating different types of certificates:
- Root CA
- Intermediate CA
- Regular certificate

The request parameters vary based on the type of certificate to generate. The `Sample Request/Response` section has examples for each of them

```
POST /v1/data/
```

##### Request Schema 
`Content-Type: application/json`

``` JSON 
{
  "$schema": "http://json-schema.org/draft-04/schema#",
  "title": "Request to generate certificate",
  "description": "Request to generate certificate",
  "type": "object",
  "properties": {
    "name": {
      "description": "Name to use for generated certificate",
      "type": "string"
    },
    "type": {
      "type": "string",
      "enum": [
        "certificate"
      ]
    },
    "parameters": {
      "type": "object",
      "properties": {
        "is_ca": {
          "description": "Indicates if the certificate to be generated is a CA or not",
          "type": "boolean"
        },
        "ca": {
          "description": "Name of the CA to sign the generated cert with. The is required when the generated cert is NOT a ROOT CA.",
          "type": "string"
        },
        "common_name": {
          "description": "Common Name used for the generated certificate",
          "type": "string"
        },
        "alternative_names": {
          "description": "List of alternative names used for the generated certificate",
          "type": "array",
          "items": {
            "type": "string"
          }
        },
        "extended_key_usage": {
          "type": "array",
          "items": {
            "type": "string",
            "enum": ["client_auth", "server_auth"]
          }
        }
      }
    }
  }
}
```

##### Response Schema
`Content-Type: application/json`

``` JSON
{
  "$schema": "http://json-schema.org/draft-04/schema#",
  "title": "Response for generate certificate",
  "type": "object",
  "properties": {
    "id": {
      "description": "ID of the generated value",
      "type": "string"
    },
    "name": {
      "description": "Certificate name",
      "type": "string"
    },
    "value": {
      "description": "Generated certificate",
      "type": "object",
      "properties": {
        "certificate": {
          "description": "String value of the generated certificate",
          "type": "string"
        },
        "private_key": {
          "description": "Private key of the generated certificate",
          "type": "string"
        },
        "ca": {
          "description": "CA used to sign the generated certificate",
          "type": "string"
        }
      }
    }
  }
}
```

##### Response Codes
| Code | Description |
| ---- | ----------- |
| 201 | Call successful |
| 400 | Bad Request |
| 401 | Not Authorized |
| 415 | Unsupported Media Type |
| 500 | Server Error |

##### Sample Request/Response

Request URL:
```
POST /v1/data
```

Request Body (root CA):
``` JSON
{
  "name": "my_cert",
  "type": "certificate",
  "parameters": {
    "is_ca": true
    "common_name": "bosh.io",
  }
}
```

Request Body (intermediate CA):
``` JSON
{
  "name": "my_cert",
  "type": "certificate",
  "parameters": {
    "is_ca": true,
    "ca": "my_ca",
    "common_name": "bosh.io",
  }
}
```

Request Body (regular certificate):
``` JSON
{
  "name": "my_cert",
  "type": "certificate",
  "parameters": {
    "ca": "my_ca",
    "common_name": "bosh.io",
    "alternative_names": ["bosh.io", "blah.bosh.io"],
  }
}
```

Response Body:
``` 
{
  "id": "some_id",
  "name":"my_cert",
  "value": {
    "ca" : "CA Certificate....",
    "certificate": "Generated Certificate ....",
    "private_key": "Private Key...."
  }
}
```

### 3.3 Generate SSH Key

```
POST /v1/data
```

##### Request Body
`Content-Type: application/json`

``` JSON 
{
  "$schema": "http://json-schema.org/draft-04/schema#",
  "title": "Request to generate SSH key",
  "description": "Request to generate SSH key",
  "type": "object",
  "properties": {
      "name": {
          "description": "Name to use for generated SSH key",
          "type": "string",
      },
      "type": {
          "type": "string",
          "enum": ["ssh"]
      }
  }
}
```

##### Response Body
`Content-Type: application/json`

``` JSON 
{
  "$schema": "http://json-schema.org/draft-04/schema#",
  "title": "Generate SSH key response",
  "description": "Generate SSH key response",
  "type": "object",
  "properties": {
    "id": {
      "description": "The unique identifier",
      "type": "string",
    },
    "name": {
      "description": "The SSH key name",
      "type": "string",
    },
    "value": {
      "description": "Generated SSH key",
      "type": "object",
      "properties": {
        "private_key": {
          "type": "string"
        },
        "public_key": {
          "type": "string"
        },
        "public_key_fingerprint": {
          "type": "string"
        }
      }
    }
  }
}
```

##### Response Codes
| Code | Description |
| ---- | ----------- |
| 201 | Call successful |
| 400 | Bad Request |
| 401 | Not Authorized |
| 415 | Unsupported Media Type |
| 500 | Server Error |


##### Sample Request/Response

Request URL:
```
POST /v1/data
```

Request Body:
``` JSON
{
  "name": "my_ssh_key",
  "type": "ssh"
}
```

Response Body:
``` JSON
{
  "id": "10",
  "name": "my_ssh_key",
  "value": {
    "private_key" : "Private key....",
    "public_key" : "Public key....",
    "public_key_fingerprint" : "Public key fingerprint...."
  }
}
```

### 3.4 Generate RSA Key

```
POST /v1/data
```

##### Request Schema
`Content-Type: application/json`

``` JSON 
{
  "$schema": "http://json-schema.org/draft-04/schema#",
  "title": "Request to generate RSA key",
  "description": "Request to generate RSA key",
  "type": "object",
  "properties": {
    "name": {
      "description": "Name to use for generated RSA key",
      "type": "string",
    },
    "type": {
      "type": "string",
      "enum": ["rsa"]
    }
  }
}
```

##### Response Schema
`Content-Type: application/json`

``` JSON 
{
  "$schema": "http://json-schema.org/draft-04/schema#",
  "title": "Generate RSA key response",
  "description": "Generate RSA key response",
  "type": "object",
  "properties": {
    "id": {
      "description": "The unique identifier",
      "type": "string",
    },
    "name": {
      "description": "The RSA key name",
      "type": "string",
    },
    "value": {
      "description": "Generated RSA key",
      "type": "object",
      "properties": {
        "private_key": {
          "type": "string"
        },
        "public_key": {
          "type": "string"
        }
      }
    }
  }
}
```

##### Response Codes
| Code | Description |
| ---- | ----------- |
| 201 | Call successful |
| 400 | Bad Request |
| 401 | Not Authorized |
| 415 | Unsupported Media Type |
| 500 | Server Error |


##### Sample Request/Response

Request URL:
```
POST /v1/data
```

Request Body:
``` JSON
{
  "name": "my_rsa_key",
  "type": "rsa"
}
```

Response Body:
``` JSON
{
  "id": "10",
  "name": "my_rsa_key",
  "value": {
    "private_key" : "Private key....",
    "public_key" : "Public key...."
  }
}
```

## 4. DELETE

### 4.1 Delete Name
```
DELETE /v1/data?name=":name_to_delete"
```

##### Sample Request

`DELETE /v1/data?name="some_name_to_delete"`

##### Response Codes
| Code | Description |
| ---- | ----------- |
| 204 | Call successful - name was deleted |
| 400 | Bad Request |
| 401 | Not Authorized |
| 404 | Not Found |
| 500 | Server Error |
