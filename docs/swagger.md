# Trackerr
API for Trackerr service.

## Version: 1.0

### Security
**ApiKeyAuth**  

| apiKey | *API Key* |
| ------ | --------- |
| Name | X-API-Key |
| In | header |

**Schemes:** http, https

---
### /models

#### GET
##### Summary

Get list of tracker models

##### Description

Get a list of all tracker models currently supported

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | OK | [ [model.Model](#modelmodel) ] |
| 400 | API key required | [api.StringResultRes](#apistringresultres) |
| 401 | Invalid API key | [api.StringResultRes](#apistringresultres) |

##### Security

| Security Schema | Scopes |
| --------------- | ------ |
| ApiKeyAuth |  |

#### POST
##### Summary

Create model

##### Description

Add support for new model, by specifing which SMS messages should be sent when the tracker model is provisioned. The tracker model, must support GT06 or JT808, to work.

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ------ |
| body | body | Register model payload | Yes | [api.CreateModelReq](#apicreatemodelreq) |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | success | [api.StringResultRes](#apistringresultres) |
| 400 | failed to parse OR API key required | [api.StringResultRes](#apistringresultres) |
| 401 | Invalid API key OR You don't have access to this feature | [api.StringResultRes](#apistringresultres) |
| 500 | failed | [api.StringResultRes](#apistringresultres) |

##### Security

| Security Schema | Scopes |
| --------------- | ------ |
| ApiKeyAuth |  |

### /models/{name}

#### GET
##### Summary

Get model by name

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ------ |
| name | path | Model name | Yes | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | OK | [model.Model](#modelmodel) |
| 400 | API key required | [api.StringResultRes](#apistringresultres) |
| 401 | Invalid API key | [api.StringResultRes](#apistringresultres) |
| 404 | Model was not found | [api.StringResultRes](#apistringresultres) |

##### Security

| Security Schema | Scopes |
| --------------- | ------ |
| ApiKeyAuth |  |

#### DELETE
##### Summary

Delete model

##### Description

Remove support for tracker model

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ------ |
| name | path | Model name | Yes | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | success | [api.StringResultRes](#apistringresultres) |
| 400 | API key required | [api.StringResultRes](#apistringresultres) |
| 401 | Invalid API key OR You don't have access to this feature | [api.StringResultRes](#apistringresultres) |
| 500 | failed | [api.StringResultRes](#apistringresultres) |

##### Security

| Security Schema | Scopes |
| --------------- | ------ |
| ApiKeyAuth |  |

---
### /trackers

#### GET
##### Summary

Get list of trackers

##### Description

If the user is a admin, it will respond with a list of all trackers in the system, and if the user is a regular user, it will return all trackers owned by the user,

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | OK | [ [api.TrackerResponse](#apitrackerresponse) ] |
| 400 | API key required | [api.StringResultRes](#apistringresultres) |
| 401 | Invalid API key | [api.StringResultRes](#apistringresultres) |

##### Security

| Security Schema | Scopes |
| --------------- | ------ |
| ApiKeyAuth |  |

#### POST
##### Summary

Register tracker

##### Description

Register tracking by supplying all properties of a tracker

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ------ |
| body | body | Register tracker payload | Yes | [api.RegisterTrackerReq](#apiregistertrackerreq) |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 201 | Success | [api.StringResultRes](#apistringresultres) |
| 400 | failed to parse OR API key required | [api.StringResultRes](#apistringresultres) |
| 401 | Invalid API key | [api.StringResultRes](#apistringresultres) |
| 403 | This action requires admin permissions | [api.StringResultRes](#apistringresultres) |
| 409 | Tracker with identical id or name already exists | [api.StringResultRes](#apistringresultres) |
| 500 | Failed | [api.StringResultRes](#apistringresultres) |

##### Security

| Security Schema | Scopes |
| --------------- | ------ |
| ApiKeyAuth |  |

### /trackers/{id}

#### GET
##### Summary

Get tracker by id

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ------ |
| id | path | TrackerID | Yes | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | OK | [api.TrackerResponse](#apitrackerresponse) |
| 400 | API key required | [api.StringResultRes](#apistringresultres) |
| 401 | Invalid API key OR not allowed to access tracker | [api.StringResultRes](#apistringresultres) |
| 403 | You don't have a tracker registered with the specified id | [api.StringResultRes](#apistringresultres) |
| 404 | Tracker not found | [api.StringResultRes](#apistringresultres) |

##### Security

| Security Schema | Scopes |
| --------------- | ------ |
| ApiKeyAuth |  |

#### DELETE
##### Summary

Deregister tracker by id

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ------ |
| id | path | TrackerID | Yes | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | success | [api.StringResultRes](#apistringresultres) |
| 400 | API key required | [api.StringResultRes](#apistringresultres) |
| 401 | Invalid API key OR not allowed to access tracker | [api.StringResultRes](#apistringresultres) |
| 500 | failed | [api.StringResultRes](#apistringresultres) |

##### Security

| Security Schema | Scopes |
| --------------- | ------ |
| ApiKeyAuth |  |

---
### /trackers/{id}/command

#### POST
##### Summary

Send command

##### Description

Send upstream command to specified tracker, and get tracker response. The request will fail if the tracker is not currently connected. Additionally the request may timeout, if the tracker is connected but does not response. This can happen if the tracker has entered sleep mode without first closing the TCP connection

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ------ |
| id | path | TrackerID | Yes | string |
| body | body | Command | Yes | [api.CommandReq](#apicommandreq) |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | RESPONSE | [api.StringResultRes](#apistringresultres) |
| 400 | failed to parse OR API key required | [api.StringResultRes](#apistringresultres) |
| 401 | Invalid API key OR not allowed to access tracker | [api.StringResultRes](#apistringresultres) |
| 503 | The tracker is not connected | [api.StringResultRes](#apistringresultres) |

##### Security

| Security Schema | Scopes |
| --------------- | ------ |
| ApiKeyAuth |  |

---
### /trackers/{id}/location

#### GET
##### Summary

Get tracker location

##### Description

Get the latest location data event reported by specified tracker

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ------ |
| id | path | TrackerID | Yes | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | OK | [api.LocationResponse](#apilocationresponse) |
| 400 | API key required | [api.StringResultRes](#apistringresultres) |
| 401 | Invalid API key OR not allowed to access tracker | [api.StringResultRes](#apistringresultres) |
| 404 | No location entry found | [api.StringResultRes](#apistringresultres) |

##### Security

| Security Schema | Scopes |
| --------------- | ------ |
| ApiKeyAuth |  |

### /trackers/{id}/locations

#### GET
##### Summary

Get tracker locations

##### Description

Get a array with all location data events reported by specified tracker

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ------ |
| id | path | TrackerID | Yes | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | OK | [ [ [api.LocationResponse](#apilocationresponse) ] ] |
| 400 | API key required | [api.StringResultRes](#apistringresultres) |
| 401 | Invalid API key OR not allowed to access tracker | [api.StringResultRes](#apistringresultres) |
| 404 | No location entry found | [api.StringResultRes](#apistringresultres) |

##### Security

| Security Schema | Scopes |
| --------------- | ------ |
| ApiKeyAuth |  |

---
### /whoami

#### GET
##### Summary

Whoami

##### Description

Fetch the user/organization name associated with the used API key. This can be used to detect if a api-key is valid

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | NAME OF USER/ORGANISATION | [api.NameRes](#apinameres) |
| 400 | API key required | [api.StringResultRes](#apistringresultres) |
| 401 | Invalid API key | [api.StringResultRes](#apistringresultres) |

##### Security

| Security Schema | Scopes |
| --------------- | ------ |
| ApiKeyAuth |  |

---
### Models

#### api.CommandReq

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| command | string |  | Yes |

#### api.CreateModelReq

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| init_commands | string |  | Yes |
| name | string |  | Yes |
| success_keywords | string |  | Yes |

#### api.LocationResponse

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| heading | integer |  | No |
| lat | integer |  | No |
| lon | integer |  | No |
| speed | integer |  | No |
| timestamp | string |  | No |

#### api.NameRes

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| name | string |  | No |

#### api.RegisterTrackerReq

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| enabled | boolean |  | Yes |
| id | string |  | Yes |
| model | string |  | Yes |
| name | string |  | Yes |
| owner | integer |  | No |
| phoneNumber | string |  | Yes |

#### api.StringResultRes

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| result | string |  | No |

#### api.TrackerResponse

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| connected | boolean |  | No |
| enabled | boolean |  | No |
| heading | integer |  | No |
| id | string |  | No |
| lastConnected | string |  | No |
| lat | integer |  | No |
| lon | integer |  | No |
| model | string |  | No |
| name | string |  | No |
| owner | integer |  | No |
| phoneNumber | string |  | No |
| speed | integer |  | No |
| timestamp | string |  | No |

#### model.Model

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| init_commands | string |  | No |
| name | string |  | No |
| success_keywords | string |  | No |
