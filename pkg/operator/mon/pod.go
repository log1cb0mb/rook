/*
Copyright 2016 The Rook Authors. All rights reserved.

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
package mon

import (
	"fmt"
	"strings"

	"github.com/rook/rook/pkg/cephmgr/mon"
	"github.com/rook/rook/pkg/operator/k8sutil"
	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/api/v1"
	"k8s.io/client-go/1.5/pkg/labels"
)

func (c *Cluster) makeMonPod(config *MonConfig) *v1.Pod {

	container := c.monContainer(config)
	// TODO: container.LivenessProbe = config.livenessProbe()

	pod := &v1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name: config.Name,
			Labels: map[string]string{
				k8sutil.AppAttr: monApp,
				monNodeAttr:     config.Name,
				monClusterAttr:  config.Info.Name,
			},
			Annotations: map[string]string{},
		},
		Spec: v1.PodSpec{
			Containers:    []v1.Container{container},
			RestartPolicy: v1.RestartPolicyAlways,
			Volumes: []v1.Volume{
				{Name: k8sutil.DataDirVolume, VolumeSource: v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}}},
			},
		},
	}

	k8sutil.SetPodVersion(pod, versionAttr, c.Version)

	if c.AntiAffinity {
		k8sutil.PodWithAntiAffinity(pod, monClusterAttr, config.Info.Name)
	}

	if len(c.NodeSelector) != 0 {
		pod.Spec.NodeSelector = c.NodeSelector
	}

	return pod
}

func (c *Cluster) monContainer(config *MonConfig) v1.Container {
	command := fmt.Sprintf("/usr/bin/rookd mon --data-dir=%s --name=%s --port=%d --fsid=%s --mon-secret=%s --admin-secret=%s --cluster-name=%s",
		k8sutil.DataDir, config.Name, config.Port, config.Info.FSID, config.Info.MonitorSecret, config.Info.AdminSecret, config.Info.Name)

	return v1.Container{
		// TODO: fix "sleep 5".
		// Without waiting some time, there is highly probable flakes in network setup.
		Command: []string{"/bin/sh", "-c", fmt.Sprintf("sleep 5; %s", command)},
		Name:    "cephmon",
		Image:   k8sutil.MakeRookImage(),
		Ports: []v1.ContainerPort{
			{
				Name:          "client",
				ContainerPort: config.Port,
				Protocol:      v1.ProtocolTCP,
			},
		},
		VolumeMounts: []v1.VolumeMount{
			{Name: k8sutil.DataDirVolume, MountPath: k8sutil.DataDir},
		},
		Env: []v1.EnvVar{
			{Name: mon.IPAddressEnvVar, ValueFrom: &v1.EnvVarSource{FieldRef: &v1.ObjectFieldSelector{FieldPath: "status.podIP"}}},
		},
	}
}

func (m *MonConfig) livenessProbe() *v1.Probe {
	// simple query of the REST api locally to see if the pod is alive
	return &v1.Probe{
		Handler: v1.Handler{
			Exec: &v1.ExecAction{
				Command: []string{"/bin/sh", "-c", "curl localhost:8124"},
			},
		},
		InitialDelaySeconds: 10,
		TimeoutSeconds:      10,
		PeriodSeconds:       60,
		FailureThreshold:    3,
	}
}

func flattenMonEndpoints(mons []*mon.CephMonitorConfig) string {
	endpoints := []string{}
	for _, m := range mons {
		endpoints = append(endpoints, fmt.Sprintf("%s=%s", m.Name, m.Endpoint))
	}
	return strings.Join(endpoints, ",")
}

func (c *Cluster) pollPods(clientset *kubernetes.Clientset, cluster *mon.ClusterInfo) ([]*v1.Pod, []*v1.Pod, error) {
	podList, err := clientset.Core().Pods(c.Namespace).List(clusterListOpt(cluster.Name))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list running pods: %v", err)
	}

	var running []*v1.Pod
	var pending []*v1.Pod
	for i := range podList.Items {
		pod := &podList.Items[i]
		/*if len(pod.OwnerReferences) < 1 {
			logger.Warningf("pollPods: ignore pod %v: no owner", pod.Name)
			continue
		}
		if pod.OwnerReferences[0].UID != c.cluster.UID {
			logger.Warningf("pollPods: ignore pod %v: owner (%v) is not %v", pod.Name, pod.OwnerReferences[0].UID, c.cluster.UID)
			continue
		}*/
		switch pod.Status.Phase {
		case v1.PodRunning:
			running = append(running, pod)
		case v1.PodPending:
			pending = append(pending, pod)
		}
	}

	return running, pending, nil
}

func clusterListOpt(clusterName string) api.ListOptions {
	return api.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			monClusterAttr:  clusterName,
			k8sutil.AppAttr: monApp,
		}),
	}
}

func podsToMonEndpoints(pods []*v1.Pod) []*mon.CephMonitorConfig {
	mons := []*mon.CephMonitorConfig{}
	for _, pod := range pods {
		mon := &mon.CephMonitorConfig{Name: pod.Name, Endpoint: fmt.Sprintf("%s:%d", pod.Status.PodIP, MonPort)}
		mons = append(mons, mon)
	}
	return mons
}
