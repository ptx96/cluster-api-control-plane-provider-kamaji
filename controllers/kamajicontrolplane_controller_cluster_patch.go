// Copyright 2023 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"encoding/json"
	"net"
	"strconv"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *KamajiControlPlaneReconciler) patchCluster(ctx context.Context, cluster capiv1beta1.Cluster, hostPort string) error {
	if cluster.Spec.InfrastructureRef == nil {
		return errors.New("capiv1beta1.Cluster has no InfrastructureRef")
	}

	endpoint, strPort, err := net.SplitHostPort(hostPort)
	if err != nil {
		return errors.Wrap(err, "cannot split the Kamaji endpoint host port pair")
	}

	port, err := strconv.ParseInt(strPort, 10, 64)
	if err != nil {
		return errors.Wrap(err, "cannot convert Kamaji endpoint port pair")
	}

	switch cluster.Spec.InfrastructureRef.Kind {
	case "OpenStackCluster":
		return r.patchOpenStackCluster(ctx, cluster, endpoint, port)
	case "KubevirtCluster":
		return r.patchKubeVirtCluster(ctx, cluster, endpoint, port)
	default:
		return errors.New("unsupported infrastructure provider")
	}
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=kubevirtclusters,verbs=patch

func (r *KamajiControlPlaneReconciler) patchKubeVirtCluster(ctx context.Context, cluster capiv1beta1.Cluster, endpoint string, port int64) error {
	kvc := unstructured.Unstructured{}

	kvc.SetGroupVersionKind(cluster.Spec.InfrastructureRef.GroupVersionKind())
	kvc.SetName(cluster.Spec.InfrastructureRef.Name)
	kvc.SetNamespace(cluster.Spec.InfrastructureRef.Namespace)

	specPatch, err := json.Marshal(map[string]interface{}{
		"spec": map[string]interface{}{
			"controlPlaneEndpoint": map[string]interface{}{
				"host": endpoint,
				"port": port,
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "unable to marshal KubeVirtCluster spec patch")
	}

	if err = r.client.Patch(ctx, &kvc, client.RawPatch(types.MergePatchType, specPatch)); err != nil {
		return errors.Wrap(err, "cannot perform PATCH update for the KubeVirtCluster resource")
	}

	statusPatch, err := json.Marshal(map[string]interface{}{
		"status": map[string]interface{}{
			"ready": true,
		},
	})
	if err != nil {
		return errors.Wrap(err, "unable to marshal KubeVirtCluster status patch")
	}

	if err = r.client.Status().Patch(ctx, &kvc, client.RawPatch(types.MergePatchType, statusPatch)); err != nil {
		return errors.Wrap(err, "cannot perform PATCH update for the KubeVirtCluster status")
	}

	return nil
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=openstackclusters,verbs=patch

func (r *KamajiControlPlaneReconciler) patchOpenStackCluster(ctx context.Context, cluster capiv1beta1.Cluster, endpoint string, port int64) error {
	osc := unstructured.Unstructured{}

	osc.SetGroupVersionKind(cluster.Spec.InfrastructureRef.GroupVersionKind())
	osc.SetName(cluster.Spec.InfrastructureRef.Name)
	osc.SetNamespace(cluster.Spec.InfrastructureRef.Namespace)

	mergePatch, err := json.Marshal(map[string]interface{}{
		"spec": map[string]interface{}{
			"apiServerFixedIP": endpoint,
			"apiServerPort":    port,
		},
	})
	if err != nil {
		return errors.Wrap(err, "unable to marshal OpenStackCluster patch")
	}

	if err = r.client.Patch(ctx, &osc, client.RawPatch(types.MergePatchType, mergePatch)); err != nil {
		return errors.Wrap(err, "cannot perform PATCH update for the OpenStackCluster resource")
	}

	return nil
}
