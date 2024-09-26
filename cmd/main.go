package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

func main() {
	kubeconfig, err := restclient.InClusterConfig()
	if err != nil {
		klog.Error(err)
		return
	}
	clientset, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		klog.Error(err)
		return
	}
	cmdline := os.Getenv("EARLYOOM_CMDLINE")
	pid := os.Getenv("EARLYOOM_PID")

	// TODO get from /run/secrets/kubernetes.io/serviceaccount/namespace
	namespace := "kube-system"

	nodeName := os.Getenv("NODE_NAME")
	involvedObject := v1.ObjectReference{
		Namespace: metav1.NamespaceSystem,
	}
	if len(nodeName) > 0 {
		involvedObject = v1.ObjectReference{
			Kind:      "Node",
			Name:      nodeName,
			UID:       types.UID(nodeName),
			Namespace: namespace,
		}
	}

	cgroupData := os.Getenv("EARLYOOM_CGROUP")
	podNs, podObject := parseCgroup(cgroupData, clientset, nodeName)
	if podObject != nil {
		namespace = podNs
		involvedObject = *podObject
	}

	t := metav1.Now()
	event := &v1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%v.%x", "earlyoom", t.UnixNano()),
			Namespace: namespace,
		},
		InvolvedObject: involvedObject,
		Reason:         "early-oomkill",
		Message:        truncateMessage(fmt.Sprintf("killed pid:%s, cmdline:%s", pid, cmdline)),
		FirstTimestamp: t,
		LastTimestamp:  t,
		Count:          1,
		Type:           v1.EventTypeWarning,
	}
	_, err = clientset.CoreV1().Events(namespace).Create(context.TODO(), event, metav1.CreateOptions{})
	if err != nil {
		klog.Error(err)
		return
	}
}

func parseCgroup(cgroupData string, clientset kubernetes.Interface, nodeName string) (string, *v1.ObjectReference) {
	if len(cgroupData) > 0 {
		cgroupInfo := strings.Split(cgroupData, ":")
		if len(cgroupInfo) >= 2 {
			cgroupPath := cgroupInfo[2]
			cgroupHierarch := strings.Split(cgroupPath, "/")
			if len(cgroupHierarch) >= 4 {
				podCgroup := cgroupHierarch[3]
				i := strings.LastIndex(podCgroup, "pod")
				if i != -1 {
					podId := podCgroup[i+3 : len(podCgroup)-6]
					podId = strings.ReplaceAll(podId, "_", "-")
					podlist, err := clientset.CoreV1().Pods(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{
						FieldSelector:   fields.OneTermEqualSelector("spec.nodeName", string(nodeName)).String(),
						ResourceVersion: "0",
					})
					if err != nil {
						klog.Error(err)
					} else {
						for _, p := range podlist.Items {
							if p.UID == types.UID(podId) {
								involvedObject := &v1.ObjectReference{
									Kind:      "Pod",
									Name:      p.Name,
									UID:       p.UID,
									Namespace: p.Namespace,
								}
								return p.Namespace, involvedObject
							}
						}
						klog.Infof("failed to get match pod:%v", podId)
					}
				} else {
					klog.Infof("failed to parse podCgroup:%v", cgroupHierarch[3])
				}
			} else {
				klog.Infof("failed to parse cgroupHierarch:%v", cgroupHierarch)
			}
		} else {
			klog.Infof("failed to parse cgroupData:%v", cgroupInfo)
		}
	}
	klog.Infof("failed to parse cgroup:%s", cgroupData)
	return "", nil
}

func truncateMessage(message string) string {
	max := 1024
	if len(message) <= max {
		return message
	}
	suffix := " ..."
	return message[:max-len(suffix)] + suffix
}
