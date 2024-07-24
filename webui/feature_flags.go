package webui

import (
	"context"
	"fmt"
	http2 "github.com/daedaleanai/git-ticket/webui/http"
	"github.com/gorilla/mux"
	"net/http"
)

const enabledFeaturesContextKey = "feature_flags"

type FeatureFlag int

const (
	_ FeatureFlag = iota
)

func (f FeatureFlag) name() string {
	for k, v := range featureFlagMap() {
		if v == f {
			return k
		}
	}

	return ""
}

func (f FeatureFlag) IsEnabled(ctx context.Context) bool {
	val, ok := http2.FindInContext(FeatureList{}, ctx)
	if !ok {
		return false
	}

	enabled := val.(FeatureList)

	return enabled.contains(f)
}

type FeatureList map[string]FeatureFlag

func (l FeatureList) ContextKey() string {
	return enabledFeaturesContextKey
}

func (l FeatureList) names() []string {
	var names []string
	for _, v := range l {
		names = append(names, v.name())
	}

	return names
}

func (l FeatureList) contains(f FeatureFlag) bool {
	_, ok := l[f.name()]
	return ok
}

func featureFlagFromName(name string) *FeatureFlag {
	for k, v := range featureFlagMap() {
		if name == k {
			return &v
		}
	}

	return nil
}

func featureFlagMap() FeatureList {
	features := make(map[string]FeatureFlag)

	return features
}

func getFeatureFlagNames() []string {
	var strings []string

	for k := range featureFlagMap() {
		strings = append(strings, k)
	}

	return strings
}

func featureFlagMiddleware(features []string) func(handler http.Handler) http.Handler {
	var enabledFeatures FeatureList
	enabledFeatures = make(map[string]FeatureFlag)

	for _, f := range features {
		feature := featureFlagFromName(f)

		if feature == nil {
			fmt.Println("Warning: ", f, " is not a feature flag")
			fmt.Println("Did you mean any of these?\n", getFeatureFlagNames())
		} else {
			enabledFeatures[feature.name()] = *feature
		}
	}

	if len(enabledFeatures) > 0 {
		fmt.Println("Enabling features:", enabledFeatures.names())
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			route := mux.CurrentRoute(r).GetName()
			featureFlag := featureFlagFromName(route)

			if featureFlag != nil && !enabledFeatures.contains(*featureFlag) {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			r = http2.LoadIntoContext(r, enabledFeatures)

			next.ServeHTTP(w, r)
		})
	}
}
