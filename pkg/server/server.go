package server

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rancher/channelserver/pkg/config"
	"github.com/rancher/channelserver/pkg/model"
	"github.com/rancher/channelserver/pkg/server/store"
	"github.com/rancher/channelserver/pkg/server/store/release"
	"github.com/rancher/steve/pkg/schemaserver/server"
	"github.com/rancher/steve/pkg/schemaserver/store/apiroot"
	"github.com/rancher/steve/pkg/schemaserver/types"
)

func ListenAndServe(ctx context.Context, address string, configs map[string]*config.Config) error {
	router := mux.NewRouter()
	for prefix, config := range configs {
		server := server.DefaultAPIServer()
		server.Schemas.MustImportAndCustomize(model.Channel{}, func(schema *types.APISchema) {
			schema.Store = store.New(config)
			schema.CollectionMethods = []string{http.MethodGet}
			schema.ResourceMethods = []string{http.MethodGet}
		})
		server.Schemas.MustImportAndCustomize(model.Release{}, func(schema *types.APISchema) {
			schema.Store = release.New(config)
			schema.CollectionMethods = []string{http.MethodGet}
		})
		prefix = strings.Trim(prefix, "/")
		apiroot.Register(server.Schemas, []string{prefix}, nil)
		router.MatcherFunc(setType("apiRoot", prefix)).Path("/").Handler(server)
		router.MatcherFunc(setType("apiRoot", prefix)).Path("/{name}").Handler(server)
		router.Path("/{prefix:" + prefix + "}/{type}").Handler(server)
		router.Path("/{prefix:" + prefix + "}/{type}/{name}").Handler(server)
	}
	next := handlers.LoggingHandler(os.Stdout, router)
	handler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		user := req.Header.Get("X-SUC-Cluster-ID")
		if user != "" && req.URL != nil {
			req.URL.User = url.User(user)
		}
		next.ServeHTTP(rw, req)
	})
	return http.ListenAndServe(address, handler)
}

func setType(t string, pathPrefix string) mux.MatcherFunc {
	return func(request *http.Request, match *mux.RouteMatch) bool {
		if match.Vars == nil {
			match.Vars = map[string]string{}
		}
		match.Vars["type"] = t
		match.Vars["prefix"] = pathPrefix
		return true
	}
}
