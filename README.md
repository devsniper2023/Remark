# remark [![Build Status](http://drone.umputun.com:9080/api/badges/umputun/remark/status.svg)](http://drone.umputun.com:9080/umputun/remark)

Comment engine

## Install

### Backend

- copy provided docker-compose.yml and customize for your needs
- make sure you **don't keep** `DEV=true` for any non-development deployments
- pull and start `docker compose pull && docker compose up`

#### Run modes

- `server` activates regular, server mode
- `import` performs import from external providers (disqus and internal json, see `/api/v1/admin/export`)

#### Register oauth2 providers

TBD

### Frontend

TBD

## API

### Authorization

- `GET /login/{provider}?from=http://url` - perform "social" login with one of supported providers and redirect to `url`
- `GET /logout` - logout 
- `GET /api/v1/user` - get user info, _auth required_

```go
type User struct {
    Name    string `json:"name"`
    ID      string `json:"id"`
    Picture string `json:"picture"`
    Profile string `json:"profile"`
    Admin   bool   `json:"admin"`
}
```

_currently supported providers are `google` and `github`_

### Commenting

- `POST /api/v1/comment` - add a comment. _auth required_

```go
type Comment struct {
    ID        string          `json:"id"`      // read only
    ParentID  string          `json:"pid"`
    Text      string          `json:"text"`
    User      User            `json:"user"`    // read only
    Locator   Locator         `json:"locator"`
    Score     int             `json:"score"`   // read only
    Votes     map[string]bool `json:"votes"`   // read only
    Timestamp time.Time       `json:"time"`    // read only
}

type Locator struct {
    SiteID string `json:"site"`
    URL    string `json:"url"`
}
```

- `GET /api/v1/find?url=post-url` - find all comments for given post, returns flat list of `Comment`
- `GET /api/v1/last/{max}` - get up to `{max}` last comments
- `GET /api/v1/id/{id}` - get comment by `id`
- `GET /api/v1/count?url=post-url` - get comment's count for `{url}`
- `PUT /api/v1/vote/{id}?url=post-url&vote=1` - vote for comment. `vote`=1 will increase score, -1 decreases. _auth required_
- `DELETE /api/v1/admin/comment/{id}?url=post-url` - delete comment by `id`. _auth and admin required_
- `PUT /api/v1/admin/user/{userid}?site=side-id&block=1` - block or unblock user. _auth and admin required_
- `GET /api/v1/admin/export?site=side-id&block=1` - export all comments. _auth and admin required_
