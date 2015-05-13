package gold_sparrow

import (
	"bytes"
	"encoding/json"
	"github.com/zenazn/goji/web"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mjibson/goon"

	"appengine"
	"appengine/aetest"
)

type AppConfigTester struct {
}

func TestPostAppConfig(t *testing.T) {
	t.Parallel()

	opt := &aetest.Options{AppID: "unittest", StronglyConsistentDatastore: true}
	inst, err := aetest.NewInstance(opt)

	defer inst.Close()
	req, err := inst.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal("fatal new request error : %s", err.Error())
	}
	c := appengine.NewContext(req)

	g := goon.FromContext(c)

	con := &AppConfig{
		ClientId:     "hoge-clinet-id",
		ClientSecret: "hoge-client-secret",
	}
	b, err := json.Marshal(con)
	t.Logf("%v", string(b))
	if err != nil {
		t.Fatal(err)
	}

	m := web.New()
	ts := httptest.NewServer(m)
	defer ts.Close()

	r, err := inst.NewRequest("POST", ts.URL+"/admin/api/1/config", bytes.NewReader(b))
	if err != nil {
		t.Error(err.Error())
	}
	w := httptest.NewRecorder()
	http.DefaultServeMux.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status code : %d, %s", w.Code, w.Body)
	}

	var rpc AppConfig
	json.NewDecoder(w.Body).Decode(&rpc)
	if rpc.Id != appConfigId {
		t.Fatalf("unexpected appConfig.id, %s != %s", rpc.Id, appConfigId)
	}
	if rpc.ClientId != con.ClientId {
		t.Fatalf("unexpected appConfig.ClinetId, %s != %s", rpc.ClientId, con.ClientId)
	}
	if rpc.ClientSecret != con.ClientSecret {
		t.Fatalf("unexpected appConfig.ClinetSecret, %s != %s", rpc.ClientSecret, con.ClientSecret)
	}
	if rpc.CreatedAt.IsZero() {
		t.Fatalf("unexpected appConfig.createdAt, IsZero")
	}
	if rpc.UpdatedAt.IsZero() {
		t.Fatalf("unexpected appConfig.updatedAt, IsZero")
	}

	stored := &AppConfig{
		Id: appConfigId,
	}
	err = g.Get(stored)
	if err != nil {
		t.Fatalf("unexpected datastore appConfig, %v", err)
	}
}

func TestGetAppConfig(t *testing.T) {
	t.Parallel()

	opt := &aetest.Options{AppID: "unittest", StronglyConsistentDatastore: true}
	inst, err := aetest.NewInstance(opt)

	defer inst.Close()
	req, err := inst.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal("fatal new request error : %s", err.Error())
	}
	c := appengine.NewContext(req)

	g := goon.FromContext(c)

	con := &AppConfig{
		Id:           appConfigId,
		ClientId:     "hoge-clinet-id",
		ClientSecret: "hoge-client-secret",
	}
	_, err = g.Put(con)
	if err != nil {
		t.Error(err)
	}

	s := &AppConfigService{}
	pc, err := s.Get(c)
	if err != nil {
		t.Error(err)
	}
	if pc.Id != appConfigId {
		t.Fatalf("unexpected appConfig.id, %s != %s", pc.Id, appConfigId)
	}
	if pc.ClientId != con.ClientId {
		t.Fatalf("unexpected appConfig.ClinetId, %s != %s", pc.ClientId, con.ClientId)
	}
	if pc.ClientSecret != con.ClientSecret {
		t.Fatalf("unexpected appConfig.ClinetSecret, %s != %s", pc.ClientSecret, con.ClientSecret)
	}
	if pc.CreatedAt.IsZero() {
		t.Fatalf("unexpected appConfig.createdAt, IsZero")
	}
	if pc.UpdatedAt.IsZero() {
		t.Fatalf("unexpected appConfig.updatedAt, IsZero")
	}
}
