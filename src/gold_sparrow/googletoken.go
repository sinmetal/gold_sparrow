package gold_sparrow

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/zenazn/goji/web"

	"appengine"
	"appengine/datastore"
	"appengine/memcache"
	"github.com/mjibson/goon"
	gengine "google.golang.org/appengine"
	"google.golang.org/appengine/log"
	gmemcache "google.golang.org/appengine/memcache"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	drive "github.com/google/google-api-go-client/drive/v2"
)

const (
	googleTokenId = "google-token-id"
)

type requestParam struct {
	Host       string
	Method     string
	UrlHost    string
	Fragment   string
	Path       string
	Scheme     string
	Opaque     string
	RawQuery   string
	RemoteAddr string
	RequestURI string
	UserAgent  string
}

const (
	randStateForAuthToMemcacheKey = "randStateForAuthToMemcacheKey"
)

type GoogleToken struct {
	Id           string    `datastore:"-" goon:"id" json:"id"` // google-token-id 固定
	RefreshToken string    `datastore:",noindex"`              // リフレッシュトークン
	CreatedAt    time.Time `json:"createdAt"`                  // 作成日時
	UpdatedAt    time.Time `json:"updatedAt"`                  // 更新日時
}

type GoogleTokenApi struct {
}

func SetUpGoogleToken(m *web.Mux) {
	api := GoogleTokenApi{}

	m.Get("/api/1/login", api.Login)
	m.Get("/oauth2callback", api.OAuth2Callback)
}

func (a *GoogleTokenApi) getConfig(r *http.Request) (*oauth2.Config, error) {
	ac := appengine.NewContext(r)

	acs := &AppConfigService{}
	config, err := acs.Get(ac)
	if err != nil {
		return &oauth2.Config{}, err
	}

	protocol := "https"
	if appengine.IsDevAppServer() {
		protocol = "http"
	}
	redirectUrl := fmt.Sprintf("%s://%s/oauth2callback", protocol, r.Host)

	return &oauth2.Config{
		ClientID:     config.ClientId,
		ClientSecret: config.ClientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  redirectUrl,
		Scopes:       []string{drive.DriveScope, drive.DriveFileScope},
	}, nil
}

func (a *GoogleTokenApi) Login(c web.C, w http.ResponseWriter, r *http.Request) {
	ac := appengine.NewContext(r)

	config, err := a.getConfig(r)
	if err != nil {
		ac.Errorf("pug config get error, %v", err)
		er := ErrorResponse{
			http.StatusInternalServerError,
			[]string{"config get error"},
		}
		er.Write(w)
		return
	}

	randState := fmt.Sprintf("st%d", time.Now().UnixNano())
	authUrl := config.AuthCodeURL(randState, oauth2.AccessTypeOffline)
	ac.Infof("auth url = %s", authUrl)

	item := &memcache.Item{
		Key:        fmt.Sprintf("%s-_-%s", randStateForAuthToMemcacheKey, randState),
		Value:      []byte(randState),
		Expiration: 3 * time.Minute,
	}
	memcache.Add(ac, item)

	http.Redirect(w, r, authUrl, http.StatusFound)
}

func (a *GoogleTokenApi) OAuth2Callback(c web.C, w http.ResponseWriter, r *http.Request) {
	ac := gengine.NewContext(r)

	p := &requestParam{
		r.Host,
		r.Method,
		r.URL.Host,
		r.URL.Fragment,
		r.URL.Path,
		r.URL.Scheme,
		r.URL.Opaque,
		r.URL.RawQuery,
		r.RemoteAddr,
		r.RequestURI,
		r.UserAgent(),
	}

	_, err := json.Marshal(p)
	if err != nil {
		log.Errorf(ac, "handler error: %#v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	config, err := a.getConfig(r)
	if err != nil {
		log.Errorf(ac, "pug config get error, %v", err)
		er := ErrorResponse{
			http.StatusInternalServerError,
			[]string{"config get error"},
		}
		er.Write(w)
		return
	}

	stateMemKey := fmt.Sprintf("%s-_-%s", randStateForAuthToMemcacheKey, r.FormValue("state"))
	item, err := gmemcache.Get(ac, stateMemKey)
	if err != nil {
		log.Errorf(ac, "memcache get error, %v", err)
		er := ErrorResponse{
			http.StatusUnauthorized,
			[]string{"unauthorized"},
		}
		er.Write(w)
		return
	}

	if r.FormValue("state") != string(item.Value) {
		log.Warningf(ac, "State doesn't match: req = %#v", "")
		er := ErrorResponse{
			http.StatusUnauthorized,
			[]string{"unauthorized"},
		}
		er.Write(w)
		return
	}

	code := r.FormValue("code")
	if code == "" {
		log.Errorf(ac, "token not found.")
	}
	token, err := config.Exchange(ac, code)
	if err != nil {
		log.Errorf(ac, "Token exchange error: %v", err)
	}
	_, err = json.Marshal(&token)
	if err != nil {
		log.Errorf(ac, "token json marshal error: %v", err)
	}

	t := &GoogleToken{
		Id:           googleTokenId,
		RefreshToken: token.RefreshToken,
	}
	g := goon.NewGoon(r)
	err = t.PutByLogin(g)
	if err != nil {
		log.Errorf(ac, "google token put error, %v", err)
		er := ErrorResponse{
			http.StatusInternalServerError,
			[]string{"google token put error"},
		}
		er.Write(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(t)
}

func (t *GoogleToken) PutByLogin(g *goon.Goon) error {
	return g.RunInTransaction(func(g *goon.Goon) error {
		stored := &GoogleToken{
			Id: googleTokenId,
		}
		err := g.Get(stored)
		if err == nil {
			stored.RefreshToken = t.RefreshToken

			_, err = g.Put(stored)
			if err != nil {
				return err
			}
			*t = *stored

			return nil
		} else if err == datastore.ErrNoSuchEntity {
			_, err = g.Put(t)
			if err != nil {
				return err
			}

			return nil
		} else {
			return err
		}
	}, nil)
}

func (t *GoogleToken) Load(c <-chan datastore.Property) error {
	if err := datastore.LoadStruct(t, c); err != nil {
		return err
	}

	return nil
}

func (t *GoogleToken) Save(c chan<- datastore.Property) error {
	now := time.Now()
	t.UpdatedAt = now

	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}

	if err := datastore.SaveStruct(t, c); err != nil {
		return err
	}
	return nil
}
