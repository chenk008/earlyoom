package main

import (
	"reflect"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
)

func Test_parseCgroup(t *testing.T) {
	nodeName := "test1"
	namespace := "test-namepsace"
	pod := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo-two-containers",
			Namespace: namespace,
			Labels:    map[string]string{"foo": "bar"},
			UID:       types.UID("001f250a_78a9_461c_af9c_6eec9b821ab2"),
		},
		Spec: v1.PodSpec{
			NodeName: nodeName,
		},
	}
	clientset := fake.NewSimpleClientset(pod)
	tests := []struct {
		name       string
		cgroupData string
		want       string
		want1      v1.ObjectReference
	}{
		{
			name: "test1",
			cgroupData: `13:devices:/kubepods.slice/kubepods-besteffort.slice/kubepods-besteffort-pod001f250a_78a9_461c_af9c_6eec9b821ab2.slice/cri-containerd-12c037ca514069bea8c968f8b21fa70d17242335aad3f68cb9ec8a0d9a61274d.scope
12:memory:/kubepods.slice/kubepods-besteffort.sli`,
			want: namespace,
			want1: v1.ObjectReference{
				Kind:      "Pod",
				Namespace: namespace,
				Name:      pod.Name,
				UID:       pod.UID,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := parseCgroup(tt.cgroupData, clientset, nodeName)
			if got != tt.want {
				t.Errorf("parseCgroup() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(*got1, tt.want1) {
				t.Errorf("parseCgroup() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
