package origin

import (
	"reflect"
	"strings"
	"testing"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/rest"
	extapi "k8s.io/kubernetes/pkg/apis/extensions"
	kubeletclient "k8s.io/kubernetes/pkg/kubelet/client"
	etcdstorage "k8s.io/kubernetes/pkg/storage/etcd"
	"k8s.io/kubernetes/pkg/util/sets"

	_ "github.com/openshift/origin/pkg/api/install"
	"github.com/openshift/origin/pkg/api/validation"
	authorizationapi "github.com/openshift/origin/pkg/authorization/api"
	"github.com/openshift/origin/pkg/util/restoptions"
)

// KnownUpdateValidationExceptions is the list of types that are known to not have an update validation function registered
// If you add something to this list, explain why it doesn't need update validation.
var KnownUpdateValidationExceptions = []reflect.Type{
	reflect.TypeOf(&extapi.Scale{}), // scale operation uses the ValidateScale() function for both create and update
}

// TestValidationRegistration makes sure that any RESTStorage that allows create or update has the correct validation register.
// It doesn't guarantee that it's actually called, but it does guarantee that it at least exists
func TestValidationRegistration(t *testing.T) {
	config := fakeMasterConfig()

	storageMap := config.GetRestStorage()
	for key, storage := range storageMap {
		obj := storage.New()
		kindType := reflect.TypeOf(obj)

		validationInfo, validatorExists := validation.Validator.GetInfo(obj)

		if _, ok := storage.(rest.Creater); ok {
			// if we're a creater, then we must have a validate method registered
			if !validatorExists {
				t.Errorf("No validator registered for %v (used by %v).  Register in pkg/api/validation/register.go.", kindType, key)
			}
		}

		if _, ok := storage.(rest.Updater); ok {
			exempted := false
			for _, t := range KnownUpdateValidationExceptions {
				if t == kindType {
					exempted = true
					break
				}
			}

			// if we're an updater, then we must have a validateUpdate method registered
			if !validatorExists && !exempted {
				t.Errorf("No validator registered for %v (used by %v).  Register in pkg/api/validation/register.go.", kindType, key)
			}

			if !validationInfo.UpdateAllowed && !exempted {
				t.Errorf("No validateUpdate method registered for %v (used by %v).  Register in pkg/api/validation/register.go.", kindType, key)
			}
		}

	}
}

// TestAllOpenShiftResourceCoverage checks to make sure that the openshift all group actually contains all openshift resources
func TestAllOpenShiftResourceCoverage(t *testing.T) {
	allOpenshift := authorizationapi.NormalizeResources(sets.NewString(authorizationapi.GroupsToResources[authorizationapi.OpenshiftAllGroupName]...))

	config := fakeMasterConfig()

	storageMap := config.GetRestStorage()
	for key := range storageMap {
		if allOpenshift.Has(strings.ToLower(key)) {
			continue
		}

		t.Errorf("authorizationapi.GroupsToResources[authorizationapi.OpenshiftAllGroupName] is missing %v.  Check pkg/authorization/api/types.go.", strings.ToLower(key))
	}
}

// fakeMasterConfig creates a new fake master config with an empty kubelet config and dummy storage.
func fakeMasterConfig() *MasterConfig {
	etcdHelper := etcdstorage.NewEtcdStorage(nil, api.Codecs.LegacyCodec(), "", false)

	return &MasterConfig{
		KubeletClientConfig: &kubeletclient.KubeletClientConfig{},
		RESTOptionsGetter:   restoptions.NewSimpleGetter(etcdHelper),
		EtcdHelper:          etcdHelper,
	}
}
