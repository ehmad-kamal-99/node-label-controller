/*
Copyright 2022.

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
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	addUbuntuLabelAnnotation  = "k8c.io/uses-ubuntu"
	addCentOSLabelAnnotation  = "k8c.io/uses-centos"
	addFlatCarLabelAnnotation = "k8c.io/uses-flatcar"
	addFedoraLabelAnnotation  = "k8c.io/uses-fedora"
	addRHELLabelAnnotation    = "k8c.io/uses-rhel"
)

const (
	addOSLabel = "k8c.io/uses-%s"
	trueValue  = "true"
)

// NodeReconciler reconciles a Node object
type NodeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Node object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *NodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the Node from the Kubernetes API.

	var node corev1.Node
	if err := r.Get(ctx, req.NamespacedName, &node); err != nil {
		if apierrors.IsNotFound(err) {
			// we'll ignore not-found errors, since we can get them on deleted requests.
			return ctrl.Result{}, nil
		}

		logger.Error(err, "unable to fetch Node")
		return ctrl.Result{}, err
	}

	// Holds name of a node's operating system with space character removed (space not allowed in labels).
	nodeOS := strings.ToLower(strings.ReplaceAll(node.Status.NodeInfo.OSImage, " ", ""))

	labelIsPresent := node.Labels[fmt.Sprintf(addOSLabel, nodeOS)] == trueValue

	if labelIsPresent {
		// The desired state and actual state of the Node are the same.
		// No further action is required by the operator at this moment.
		logger.Info("no update required")
		return ctrl.Result{}, nil
	}

	// Set the label for the Node.
	if node.Labels == nil {
		node.Labels = make(map[string]string)
	}

	node.Labels[fmt.Sprintf(addOSLabel, nodeOS)] = trueValue
	logger.Info("adding label")

	// Update the Node in the Kubernetes API.

	if err := r.Update(ctx, &node); err != nil {
		if apierrors.IsConflict(err) {
			// The Node has been updated since we read it.
			// Requeue the Node to try to reconciliate again.
			return ctrl.Result{Requeue: true}, nil
		}
		if apierrors.IsNotFound(err) {
			// The Node has been deleted since we read it.
			// Requeue the Node to try to reconciliate again.
			return ctrl.Result{Requeue: true}, nil
		}
		logger.Error(err, "unable to update Node")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Node{}).
		Complete(r)
}
