/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"strconv"

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubebuilderv1 "github.com/apricote/kubebuilder-poke-sync/api/v1"
	"github.com/apricote/kubebuilder-poke-sync/pokeapi"
)

// PokemonReconciler reconciles a Pokemon object
type PokemonReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func ignoreNotFound(err error) error {
	if apierrs.IsNotFound(err) {
		return nil
	}
	return err
}

// +kubebuilder:rbac:groups=kubebuilder.meetup.apricote.de,resources=pokemons,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kubebuilder.meetup.apricote.de,resources=pokemons/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
func (r *PokemonReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("pokemon", req.NamespacedName)

	// retrieve api object
	var pokemonSync kubebuilderv1.Pokemon
	if err := r.Get(ctx, req.NamespacedName, &pokemonSync); err != nil {
		log.Error(err, "unable to fetch pokemon")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, ignoreNotFound(err)
	}

	// retrieve config data from external api
	pokemonData, err := pokeapi.GetPokemon(ctx, pokemonSync.Spec.PokemonName)
	if err != nil {
		log.Error(err, "unable to fetch pokemon data from pokeapi")
		return ctrl.Result{}, err
	}

	configMap := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: pokemonSync.Spec.ConfigMapName, Namespace: pokemonSync.Namespace}}
	if _, err := ctrl.CreateOrUpdate(ctx, r.Client, configMap, func() error {

		if configMap.Data == nil {
			// Initialize Data map for new objects
			configMap.Data = make(map[string]string)
		}

		configMap.Data["ID"] = strconv.Itoa(pokemonData.ID)
		configMap.Data["Name"] = pokemonData.Name
		configMap.Data["Height"] = strconv.Itoa(pokemonData.Height)
		configMap.Data["Weight"] = strconv.Itoa(pokemonData.Weight)
		configMap.Data["BaseExperience"] = strconv.Itoa(pokemonData.BaseExperience)

		// Set ownership for automatic clean up
		if err := ctrl.SetControllerReference(&pokemonSync, configMap, r.Scheme); err != nil {
			return err
		}

		return nil
	}); err != nil {
		log.Error(err, "unable to update config map")
	}

	return ctrl.Result{}, nil
}

func (r *PokemonReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kubebuilderv1.Pokemon{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}
