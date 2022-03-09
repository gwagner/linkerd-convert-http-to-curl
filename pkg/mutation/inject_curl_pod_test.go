package mutation

import (
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"testing"
)

func Test_buildExecCommand(t *testing.T) {
	type args struct {
		probe *corev1.HTTPGetAction
		ports []corev1.ContainerPort
	}
	tests := []struct {
		name string
		args args
		want *corev1.ExecAction
	}{
		{
			name: "test port conversion",
			args: args{
				probe: &corev1.HTTPGetAction{
					Path: "/",
					Port: intstr.FromString("http"),
				},
				ports: []corev1.ContainerPort{
					{
						Name:          "http",
						ContainerPort: 8080,
					},
				},
			},
			want: &corev1.ExecAction{
				Command: []string{"curl", "http://127.0.0.1:8080/", "--fail", "-o", "/dev/null"},
			},
		},
		{
			name: "test natural port",
			args: args{
				probe: &corev1.HTTPGetAction{
					Path: "/",
					Port: intstr.FromInt(8080),
				},
				ports: []corev1.ContainerPort{
					{
						Name:          "http",
						ContainerPort: 8080,
					},
				},
			},
			want: &corev1.ExecAction{
				Command: []string{"curl", "http://127.0.0.1:8080/", "--fail", "-o", "/dev/null"},
			},
		},
		{
			name: "test https scheme",
			args: args{
				probe: &corev1.HTTPGetAction{
					Path:   "/",
					Port:   intstr.FromInt(8080),
					Scheme: "https",
				},
				ports: []corev1.ContainerPort{
					{
						Name:          "http",
						ContainerPort: 8080,
					},
				},
			},
			want: &corev1.ExecAction{
				Command: []string{"curl", "https://127.0.0.1:8080/", "--fail", "-o", "/dev/null"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, buildExecCommand(tt.args.probe, tt.args.ports), "buildExecCommand(%v, %v)", tt.args.probe, tt.args.ports)
		})
	}
}
