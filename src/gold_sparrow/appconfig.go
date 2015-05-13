package gold_sparrow

import (
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "github.com/zenazn/goji/web"

    "appengine"
    "appengine/datastore"
    "github.com/mjibson/goon"
)

type AppConfig struct {
    Id           string    `datastore:"-" goon:"id" json:"id"`        // app-config-id 固定
    ClientId     string    `json:"clientId" datastore:",noindex"`     // GCP Client Id
    ClientSecret string    `json:"clientSecret" datastore:",noindex"` // GCP Client Secret
    CreatedAt    time.Time `json:"createdAt"`                         // 作成日時
    UpdatedAt    time.Time `json:"updatedAt"`                         // 更新日時
}

const (
    appConfigId = "app-config-id"
)

type AppConfigApi struct {
}

type AppConfigService struct {
}

func SetUpAppConfig(m *web.Mux) {
    api := AppConfigApi{}

    m.Post("/admin/api/1/config", api.Put)
}

func (a *AppConfigApi) Put(c web.C, w http.ResponseWriter, r *http.Request) {
    ac := appengine.NewContext(r)
    g := goon.FromContext(ac)

    var appCon AppConfig
    err := json.NewDecoder(r.Body).Decode(&appCon)
    if err != nil {
        ac.Infof("rquest body, %v", r.Body)
        er := ErrorResponse{
            http.StatusBadRequest,
            []string{"invalid request"},
        }
        er.Write(w)
        return
    }
    defer r.Body.Close()

    appCon.Id = appConfigId
    _, err = g.Put(&appCon)
    if err != nil {
        er := ErrorResponse{
            http.StatusInternalServerError,
            []string{err.Error()},
        }
        ac.Errorf(fmt.Sprintf("datastore put error : ", err.Error()))
        er.Write(w)
        return
    }

    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(appCon)
}

func (s *AppConfigService) Get(ac appengine.Context) (AppConfig, error) {
    g := goon.FromContext(ac)

    appCon := AppConfig{
        Id: appConfigId,
    }

    err := g.Get(&appCon)
    return appCon, err
}

func (appCon *AppConfig) Load(c <-chan datastore.Property) error {
    if err := datastore.LoadStruct(appCon, c); err != nil {
        return err
    }

    return nil
}

func (appCon *AppConfig) Save(c chan<- datastore.Property) error {
    now := time.Now()
    appCon.UpdatedAt = now

    if appCon.CreatedAt.IsZero() {
        appCon.CreatedAt = now
    }

    if err := datastore.SaveStruct(appCon, c); err != nil {
        return err
    }
    return nil
}